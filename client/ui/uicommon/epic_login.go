package uicommon

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var epicExchangeCodePattern = regexp.MustCompile(`(?i)\b[a-f0-9]{32}\b`)

func parseEpicCodeFromClipboard(content string) (string, bool) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", false
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err == nil {
		for _, key := range []string{"exchange_code", "code", "authorization_code", "authorizationCode"} {
			value, ok := payload[key].(string)
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			if value != "" {
				return value, true
			}
		}
	}

	if u, err := url.Parse(content); err == nil {
		for _, key := range []string{"exchange_code", "code", "authorization_code"} {
			value := strings.TrimSpace(u.Query().Get(key))
			if value != "" {
				return value, true
			}
		}
	}

	code := epicExchangeCodePattern.FindString(content)
	if code != "" {
		return code, true
	}

	return "", false
}

func (s *State) ShowEpicLoginWindow(onSuccess func(), onCancel func()) {
	var popup dialog.Dialog
	var success atomic.Bool
	var flowCancel context.CancelFunc

	statusLabel := widget.NewLabel(lang.LocalizeKey("settings.epic_login_waiting", "Please complete Epic Games login in your browser."))
	statusLabel.Wrapping = fyne.TextWrapWord

	setStatus := func(text string) {
		fyne.Do(func() {
			statusLabel.SetText(text)
		})
	}

	authURL := s.Core.EpicApi.GetAuthUrl()
	openLoginPage := func() {
		u, err := url.Parse(authURL)
		if err != nil {
			slog.Error("Failed to parse Epic auth URL", "error", err)
			dialog.ShowError(err, s.Window)
			return
		}
		slog.Info("Opening Epic login page in external browser", "url", authURL)
		if err := fyne.CurrentApp().OpenURL(u); err != nil {
			slog.Error("Failed to open Epic auth URL in external browser", "error", err)
			dialog.ShowError(err, s.Window)
		}
	}

	instructionLabel := widget.NewLabel(lang.LocalizeKey("settings.epic_login_instruction", "Logging in with Epic Games will complete the connection automatically."))
	instructionLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		instructionLabel,
		statusLabel,
		container.NewHBox(layout.NewSpacer()),
	)

	popup = dialog.NewCustom(
		lang.LocalizeKey("settings.epic_games_account", "Epic Games Account"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		content,
		s.Window,
	)

	var stopWebView func()
	popup.SetOnClosed(func() {
		slog.Info("Epic login popup closed", "success", success.Load())
		if stopWebView != nil {
			stopWebView()
		}
		if flowCancel != nil {
			flowCancel()
		}
		if !success.Load() && onCancel != nil {
			onCancel()
		}
	})

	popup.Resize(fyne.NewSize(420, 260))
	popup.Show()

	ctx, cancel := context.WithCancel(context.Background())
	flowCancel = cancel

	setStatus(lang.LocalizeKey("settings.epic_login_waiting", "Please complete Epic Games login in your browser."))

	slog.Info("Initiating Epic login flow", "authURL", authURL)
	var webViewCodeCh <-chan string
	var webViewErrCh <-chan error
	webViewCodeCh, webViewErrCh, stopWebView = startEpicWebView2Login(authURL)
	clipboardFallbackEnabled := webViewCodeCh == nil || webViewErrCh == nil
	if clipboardFallbackEnabled {
		slog.Warn("WebView2 login unavailable, falling back immediately to clipboard / browser")
	}

	go func() {
		defer func() {
			if stopWebView != nil {
				stopWebView()
			}
		}()
		ticker := time.NewTicker(700 * time.Millisecond)
		timeout := time.NewTimer(5 * time.Minute)
		defer ticker.Stop()
		defer timeout.Stop()

		triedCodes := map[string]struct{}{}
		lastClipboard := ""

		for {
			select {
			case <-ctx.Done():
				slog.Info("Epic login context cancelled")
				return
			case <-timeout.C:
				slog.Warn("Epic login timed out after 5 minutes")
				fyne.Do(func() {
					dialog.ShowError(errors.New(lang.LocalizeKey("settings.epic_login_timeout", "Epicログインがタイムアウトしました。もう一度お試しください。")), s.Window)
				})
				return
			case err, ok := <-webViewErrCh:
				if !ok {
					webViewErrCh = nil
					continue
				}
				if err != nil {
					slog.Warn("WebView2 login error encountered", "error", err)
					goto clipboardFallback
				}
			case code, ok := <-webViewCodeCh:
				if !ok || code == "" {
					webViewCodeCh = nil
					goto clipboardFallback
				}
				slog.Info("Received auth code from WebView2", "codeLength", len(code))
				clipboardContent := code
				lastClipboard = clipboardContent
				if _, exists := triedCodes[clipboardContent]; exists {
					continue
				}
				triedCodes[clipboardContent] = struct{}{}

				setStatus(lang.LocalizeKey("settings.epic_login_code_detected", "Auth code detected. Completing login..."))

				session, err := s.Core.EpicApi.LoginWithCode(clipboardContent)
				if err != nil {
					slog.Warn("Failed to login with auth code from WebView2", "error", err)
					setStatus(lang.LocalizeKey("settings.epic_login_code_failed", "Code verification failed. Please try again after logging in on your browser."))
					continue
				}

				if err := s.Core.EpicSessionManager.Save(session); err != nil {
					slog.Error("Failed to save Epic session", "error", err)
					fyne.Do(func() {
						dialog.ShowError(err, s.Window)
					})
					return
				}

				slog.Info("Successfully logged in with Epic Games via WebView2")
				success.Store(true)
				cancel()
				stopWebView()
				fyne.Do(func() {
					popup.Hide()
					if onSuccess != nil {
						onSuccess()
					}
				})
				return
			case <-ticker.C:
				if !clipboardFallbackEnabled {
					continue
				}
				clipboardContent := fyne.CurrentApp().Clipboard().Content()
				if clipboardContent == "" || clipboardContent == lastClipboard {
					continue
				}
				lastClipboard = clipboardContent

				code, ok := parseEpicCodeFromClipboard(clipboardContent)
				if !ok {
					continue
				}
				if _, exists := triedCodes[code]; exists {
					continue
				}
				triedCodes[code] = struct{}{}

				slog.Info("Detected auth code in clipboard", "codeLength", len(code))
				setStatus(lang.LocalizeKey("settings.epic_login_code_detected", "Auth code detected. Completing login..."))

				session, err := s.Core.EpicApi.LoginWithCode(code)
				if err != nil {
					slog.Warn("Failed to login with auth code from clipboard", "error", err)
					setStatus(lang.LocalizeKey("settings.epic_login_code_failed", "Code verification failed. Please try again after logging in on your browser."))
					continue
				}

				if err := s.Core.EpicSessionManager.Save(session); err != nil {
					slog.Error("Failed to save Epic session from clipboard login", "error", err)
					fyne.Do(func() {
						dialog.ShowError(err, s.Window)
					})
					return
				}

				slog.Info("Successfully logged in with Epic Games via clipboard code")
				success.Store(true)
				cancel()
				stopWebView()
				fyne.Do(func() {
					popup.Hide()
					if onSuccess != nil {
						onSuccess()
					}
				})
				return
			}

		clipboardFallback:
			slog.Info("Prompting user for external browser login fallback")
			fyne.Do(func() {
				dialog.ShowConfirm(
					lang.LocalizeKey("settings.epic_login_fallback_title", "WebView Login Failed"),
					lang.LocalizeKey("settings.epic_login_fallback_message", "Failed to log in via WebView. Do you want to continue login in an external browser?"),
					func(confirm bool) {
						if !confirm {
							slog.Info("User declined browser fallback login")
							cancel()
							return
						}
						slog.Info("User accepted browser fallback login")
						clipboardFallbackEnabled = true
						setStatus(lang.LocalizeKey("settings.epic_login_waiting", "Please complete Epic Games login in your browser."))
						openLoginPage()
					},
					s.Window,
				)
			})
		}
	}()
}
