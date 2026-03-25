//go:build windows

package uicommon

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	webview2 "github.com/jchv/go-webview2"
)

var epicWebViewCodePattern = regexp.MustCompile(`(?i)\b[a-f0-9]{32}\b`)

type epicWebViewPayload struct {
	Code string `json:"code"`
	Raw  string `json:"raw"`
	URL  string `json:"url"`
}

func parseEpicCodeFromWebViewPayload(payload epicWebViewPayload) (string, bool) {
	try := func(value string) (string, bool) {
		value = strings.TrimSpace(value)
		if value == "" {
			return "", false
		}
		if matched := epicWebViewCodePattern.FindString(value); matched != "" {
			return strings.ToLower(matched), true
		}
		return "", false
	}

	if code, ok := try(payload.Code); ok {
		return code, true
	}

	if payload.URL != "" {
		if u, err := url.Parse(payload.URL); err == nil {
			for _, key := range []string{"exchange_code", "code", "authorization_code"} {
				if code, ok := try(u.Query().Get(key)); ok {
					return code, true
				}
			}
		}
	}

	if payload.Raw != "" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(payload.Raw), &obj); err == nil {
			for _, key := range []string{"exchange_code", "code", "authorization_code", "authorizationCode"} {
				if v, ok := obj[key].(string); ok {
					if code, found := try(v); found {
						return code, true
					}
				}
			}
		}
		if code, ok := try(payload.Raw); ok {
			return code, true
		}
	}

	return "", false
}

const epicWebView2BridgeScript = `(function () {
  const sent = new Set();
  let lastHref = "";
  let lastBodySample = "";

  function sendPayload(payload) {
    if (!payload) return;
    const key = JSON.stringify(payload);
    if (sent.has(key)) return;
    sent.add(key);
    if (typeof window.epicReportCode === 'function') {
      try { window.epicReportCode(payload); } catch (_) {}
    }
  }

  function sendCandidate(value, raw) {
    if (!value) return;
    const code = String(value).trim();
    if (!code) return;
    sendPayload({ code: code, raw: raw || "", url: window.location.href || "" });
  }

  function sendRaw(raw) {
    if (!raw) return;
    sendPayload({ code: "", raw: String(raw), url: window.location.href || "" });
  }

  function inspectText(text) {
    if (!text) return;
    const raw = String(text);
    sendRaw(raw);
    try {
      const obj = JSON.parse(raw);
      const keys = ['exchange_code', 'code', 'authorization_code', 'authorizationCode'];
      for (const key of keys) {
        if (obj && typeof obj[key] === 'string') {
          sendCandidate(obj[key], raw);
        }
      }
    } catch (_) {}
    const m = raw.match(/[a-f0-9]{32}/i);
    if (m) sendCandidate(m[0], raw);
  }

  function inspectUrl(href) {
    if (!href || href === lastHref) return;
    lastHref = href;
    try {
      const u = new URL(href, window.location.origin);
      sendPayload({ code: "", raw: "", url: u.href });
      ['exchange_code', 'code', 'authorization_code'].forEach((key) => {
        const v = u.searchParams.get(key);
        if (v) sendCandidate(v, "");
      });
      if (u.hash && u.hash.length > 1) {
        inspectText(decodeURIComponent(u.hash.slice(1)));
      }
    } catch (_) {}
  }

  function inspectBodySample() {
    try {
      if (!document || !document.body) return;
      const text = (document.body.textContent || '').trim();
      if (!text) return;
      const sample = text.length > 4096 ? text.slice(0, 4096) : text;
      if (sample === lastBodySample) return;
      lastBodySample = sample;
      inspectText(sample);
    } catch (_) {}
  }

  function tick() {
    inspectUrl(window.location.href);
    inspectBodySample();
  }

  window.addEventListener('load', tick);
  window.addEventListener('hashchange', tick);
  window.addEventListener('popstate', tick);
  setInterval(tick, 1200);
  tick();
})();`

func startEpicWebView2Login(authURL string) (<-chan string, <-chan error, func()) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	var mu sync.Mutex
	var w webview2.WebView

	stop := func() {
		mu.Lock()
		defer mu.Unlock()
		if w != nil {
			w.Destroy()
			w.Terminate()
		}
	}

	go func() {
		defer close(codeCh)
		defer close(errCh)

		tmpPath, err := os.MkdirTemp("", "au_mod_installer_webview_*")
		if err != nil {
			errCh <- err
			return
		}
		defer os.RemoveAll(tmpPath)

		wv := webview2.NewWithOptions(webview2.WebViewOptions{
			Debug:     false,
			AutoFocus: true,
			DataPath:  tmpPath,
			WindowOptions: webview2.WindowOptions{
				Title:  "Epic Games Login",
				Width:  960,
				Height: 720,
				IconId: 1,
				Center: true,
			},
		})
		if wv == nil {
			errCh <- errors.New("failed to initialize WebView2")
			return
		}
		mu.Lock()
		w = wv
		mu.Unlock()
		defer func() {
			mu.Lock()
			w = nil
			mu.Unlock()
		}()

		var delivered atomic.Bool
		if err := wv.Bind("epicReportCode", func(payload epicWebViewPayload) {
			code, ok := parseEpicCodeFromWebViewPayload(payload)
			if !ok {
				return
			}
			if delivered.CompareAndSwap(false, true) {
				select {
				case codeCh <- code:
				default:
				}
				wv.Destroy()
				wv.Terminate()
			}
		}); err != nil {
			errCh <- err
			return
		}

		wv.Init(epicWebView2BridgeScript)
		wv.Navigate(authURL)
		wv.Run()
	}()

	return codeCh, errCh, stop
}
