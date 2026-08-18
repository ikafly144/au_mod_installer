//go:build windows

package uicommon

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"runtime"
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
			for _, key := range []string{"exchange_code", "code", "authorization_code", "authorizationCode"} {
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

  var lastReportedHref = "";
  var sentCode = false;

  function reportNav(href) {
    if (!href || href === lastReportedHref) return;
    lastReportedHref = href;
    if (typeof window.epicReportNav === 'function') {
      try { window.epicReportNav(href); } catch (_) {}
    }
  }

  function sendCode(code, raw, url) {
    if (!code || sentCode) return;
    var m = String(code).trim().match(/[a-f0-9]{32}/i);
    if (!m) return;
    sentCode = true;
    if (typeof window.epicReportCode === 'function') {
      try {
        window.epicReportCode({
          code: m[0].toLowerCase(),
          raw: raw || "",
          url: url || window.location.href || ""
        });
      } catch (_) {}
    }
  }

  function checkUrl(href) {
    if (!href) return;
    reportNav(href);
    try {
      var u = new URL(href, window.location.origin);
      var params = ['exchange_code', 'code', 'authorization_code', 'authorizationCode'];
      for (var i = 0; i < params.length; i++) {
        var v = u.searchParams.get(params[i]);
        if (v) {
          sendCode(v, "", href);
          return;
        }
      }
      if (u.hash && u.hash.length > 1) {
        var hashText = decodeURIComponent(u.hash.slice(1));
        var m = hashText.match(/[a-f0-9]{32}/i);
        if (m) {
          sendCode(m[0], hashText, href);
          return;
        }
      }
    } catch (_) {}
  }

  function checkBody() {
    if (sentCode) return;
    try {
      if (!document || !document.body) return;
      var text = (document.body.innerText || document.body.textContent || '').trim();
      if (!text) return;
      if (text.indexOf('{') !== -1 || text.indexOf('code') !== -1 || text.indexOf('Code') !== -1 || text.indexOf('exchange') !== -1) {
        try {
          var obj = JSON.parse(text);
          var keys = ['exchange_code', 'code', 'authorization_code', 'authorizationCode'];
          for (var i = 0; i < keys.length; i++) {
            if (obj && typeof obj[keys[i]] === 'string') {
              sendCode(obj[keys[i]], text, window.location.href);
              return;
            }
          }
        } catch (_) {}
        var m = text.match(/[a-f0-9]{32}/i);
        if (m) {
          sendCode(m[0], text, window.location.href);
        }
      }
    } catch (_) {}
  }

  function onPageCheck() {
    checkUrl(window.location.href);
    checkBody();
  }

  window.addEventListener('load', onPageCheck);
  window.addEventListener('pageshow', onPageCheck);
  window.addEventListener('DOMContentLoaded', onPageCheck);
  window.addEventListener('hashchange', function() { checkUrl(window.location.href); });
  window.addEventListener('popstate', function() { checkUrl(window.location.href); });

  onPageCheck();
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
				slog.Info("Destroying WebView2 instance from stop callback")
				target.Destroy()
			})
		}
	}

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		defer close(codeCh)
		defer close(errCh)

		tmpPath, err := os.MkdirTemp("", "au_mod_installer_webview_*")
		if err != nil {
			slog.Error("Failed to create temporary data directory for WebView2", "error", err)
			errCh <- err
			return
		}
		slog.Info("Created temporary data directory for WebView2", "path", tmpPath)
		defer func() {
			slog.Info("Removing temporary data directory for WebView2", "path", tmpPath)
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
		if err := wv.Bind("epicReportNav", func(url string) {
			slog.Info("WebView2 navigated to URL", "url", url)
		}); err != nil {
			slog.Warn("Failed to bind epicReportNav to WebView2", "error", err)
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
