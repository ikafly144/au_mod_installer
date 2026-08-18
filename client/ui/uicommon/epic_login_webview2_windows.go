//go:build windows

package uicommon

import (
	"encoding/json"
	"errors"
	"log/slog"
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
  if (window.top !== window.self) return;

  function reportState(event, details) {
    if (typeof window.epicReportState === 'function') {
      try { window.epicReportState(event, String(details || "")); } catch (_) {}
    }
  }

  reportState("script_injected", window.location.href);

  const sent = new Set();
  let lastHref = "";
  let lastBodySample = "";

  function sendPayload(payload) {
    if (!payload || !payload.code) return;
    const key = payload.code;
    if (sent.has(key)) return;
    sent.add(key);
    reportState("code_detected", "code_len=" + key.length + ", url=" + (payload.url || window.location.href));
    if (typeof window.epicReportCode === 'function') {
      try { window.epicReportCode(payload); } catch (_) {}
    }
  }

  function sendCandidate(value, raw) {
    if (!value) return;
    const code = String(value).trim();
    if (!code) return;
    const m = code.match(/[a-f0-9]{32}/i);
    if (m) {
      sendPayload({ code: m[0].toLowerCase(), raw: raw || "", url: window.location.href || "" });
    }
  }

  function inspectText(text) {
    if (!text) return;
    const raw = String(text);
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
    reportState("navigated", href);
    try {
      const u = new URL(href, window.location.origin);
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
      reportState("body_updated", "len=" + text.length + ", sample=" + (sample.length > 128 ? sample.slice(0, 128) + "..." : sample));
      inspectText(sample);
    } catch (_) {}
  }

  function tick() {
    inspectUrl(window.location.href);
    inspectBodySample();
  }

  window.addEventListener('load', function() {
    reportState("load_event", window.location.href);
    tick();
  });
  window.addEventListener('DOMContentLoaded', function() {
    reportState("dom_content_loaded", window.location.href);
    tick();
  });
  window.addEventListener('hashchange', function() {
    reportState("hash_change", window.location.href);
    tick();
  });
  window.addEventListener('popstate', function() {
    reportState("pop_state", window.location.href);
    tick();
  });
  setInterval(tick, 1000);
  tick();
})();`

func startEpicWebView2Login(authURL string) (<-chan string, <-chan error, func()) {
	slog.Info("Starting Epic Games WebView2 login flow", "authURL", authURL)
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	var mu sync.Mutex
	var w webview2.WebView

	stop := func() {
		mu.Lock()
		defer mu.Unlock()
		if w != nil {
			slog.Info("Stopping WebView2 login on request")
			target := w
			w = nil
			target.Dispatch(func() {
				slog.Debug("Destroying WebView2 instance from stop callback")
				target.Destroy()
			})
		}
	}

	go func() {
		defer close(codeCh)
		defer close(errCh)

		tmpPath, err := os.MkdirTemp("", "au_mod_installer_webview_*")
		if err != nil {
			slog.Error("Failed to create temporary data directory for WebView2", "error", err)
			errCh <- err
			return
		}
		slog.Debug("Created temporary data directory for WebView2", "path", tmpPath)
		defer func() {
			slog.Debug("Removing temporary data directory for WebView2", "path", tmpPath)
			_ = os.RemoveAll(tmpPath)
		}()

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
			slog.Error("Failed to create WebView2 instance (NewWithOptions returned nil)")
			errCh <- errors.New("failed to initialize WebView2")
			return
		}
		slog.Info("WebView2 instance created successfully")
		mu.Lock()
		w = wv
		mu.Unlock()
		defer func() {
			mu.Lock()
			w = nil
			mu.Unlock()
		}()

		var delivered atomic.Bool
		if err := wv.Bind("epicReportState", func(event string, details string) {
			slog.Debug("WebView2 state event", "event", event, "details", details)
		}); err != nil {
			slog.Warn("Failed to bind epicReportState to WebView2", "error", err)
		}

		if err := wv.Bind("epicReportCode", func(payload epicWebViewPayload) {
			slog.Info("WebView2 reported code payload", "url", payload.URL, "hasCode", payload.Code != "", "rawLen", len(payload.Raw))
			code, ok := parseEpicCodeFromWebViewPayload(payload)
			if !ok {
				slog.Debug("WebView2 payload could not be parsed into a valid Epic auth code")
				return
			}
			slog.Info("Extracted valid Epic auth code from WebView2 payload", "codeLength", len(code))
			if delivered.CompareAndSwap(false, true) {
				select {
				case codeCh <- code:
				default:
				}
				mu.Lock()
				target := w
				w = nil
				mu.Unlock()
				if target != nil {
					target.Dispatch(func() {
						slog.Info("Destroying WebView2 instance after code delivery")
						target.Destroy()
					})
				}
			}
		}); err != nil {
			slog.Error("Failed to bind epicReportCode to WebView2", "error", err)
			errCh <- err
			return
		}

		wv.Init(epicWebView2BridgeScript)
		slog.Info("Navigating WebView2 to Epic authorization URL", "url", authURL)
		wv.Navigate(authURL)
		slog.Info("Starting WebView2 message loop")
		wv.Run()
		slog.Info("WebView2 message loop finished")
	}()

	return codeCh, errCh, stop
}
