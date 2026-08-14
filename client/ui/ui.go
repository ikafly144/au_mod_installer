//go:build windows

package ui

import (
	"context"
	"log/slog"
	"runtime"
	"sync"

	"github.com/ikafly144/au_mod_installer/client/ui/tab/launcher"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/repo"
	servertab "github.com/ikafly144/au_mod_installer/client/ui/tab/server"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/settings"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/zzl/go-win32api/v2/win32"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type Config struct {
	stateOptions []uicommon.Option
	stateInits   []func(*uicommon.State)
	silent       bool
}

func WithSilent(silent bool) func(*Config) {
	return func(cfg *Config) {
		cfg.silent = silent
	}
}

func WithStateOptions(options ...uicommon.Option) func(*Config) {
	return func(cfg *Config) {
		cfg.stateOptions = options
	}
}

func WithStateInit(init func(*uicommon.State)) func(*Config) {
	return func(cfg *Config) {
		cfg.stateInits = append(cfg.stateInits, init)
	}
}

func Main(w fyne.Window, version string, sharedURI string, sharedArchive string, cfg ...func(*Config)) error {
	var config Config

	for _, c := range cfg {
		c(&config)
	}

	state, err := uicommon.NewState(w, version, config.stateOptions...)
	if err != nil {
		return err
	}
	state.SharedURI = sharedURI
	state.SharedArchive = sharedArchive

	for _, init := range config.stateInits {
		init(state)
	}

	l := launcher.NewLauncherTab(state)
	launcherTab, err := l.Tab()
	if err != nil {
		return err
	}

	r := repo.NewRepository(state)
	repoTab, err := r.Tab()
	if err != nil {
		return err
	}

	s := settings.NewSettings(state)
	settingsTab, err := s.Tab()
	if err != nil {
		return err
	}

	st := servertab.NewServerTab(state)
	serverTab, err := st.Tab()
	if err != nil {
		return err
	}

	canvas := container.NewAppTabs(
		launcherTab,
		repoTab,
		serverTab,
		settingsTab,
	)
	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		if state.OnDroppedURIs != nil {
			state.OnDroppedURIs(uris)
		}
	})
	w.SetContent(canvas)
	w.SetFixedSize(false)

	var (
		initOnce        sync.Once
		textDropCleanup func()
	)

	state.ShowWindow = func() {
		initOnce.Do(func() {
			w.Show()
			if _, err := state.EnableNativeCustomWindowFrame(); err != nil {
				slog.Warn("Failed to enable native custom window frame", "error", err)
			}
			if cleanup, err := state.EnableNativeTextDrop(); err != nil {
				slog.Warn("Failed to enable native OLE text drop", "error", err)
			} else {
				textDropCleanup = cleanup
			}
			if uicommon.RestoreMainWindowSize(w) {
				w.CenterOnScreen()
			}
		})

		w.Show()
		w.RequestFocus()
		if nw, ok := w.(driver.NativeWindow); ok {
			nw.RunNative(func(context any) {
				if winCtx, ok := context.(driver.WindowsWindowContext); ok {
					win32.ShowWindow(win32.HWND(winCtx.HWND), win32.SW_RESTORE)
					win32.SetForegroundWindow(win32.HWND(winCtx.HWND))
				}
			})
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	onClosed := func() {
		uicommon.SaveMainWindowSize(w)
		if textDropCleanup != nil {
			textDropCleanup()
		}
		cancel()
	}

	setupSystemTray(w, state, onClosed)

	w.SetCloseIntercept(func() {
		if fyne.CurrentApp().Preferences().BoolWithFallback("tray_resident", true) {
			uicommon.SaveMainWindowSize(w)
			w.Hide()
			slog.Info("Main window hidden to system tray")
		} else {
			if onClosed != nil {
				onClosed()
			}
			w.SetCloseIntercept(nil)
			w.Close()
			fyne.CurrentApp().Quit()
		}
	})

	if !config.silent {
		state.ShowWindow()
	}

	if state.Core.DiscordService != nil {
		ds := state.Core.DiscordService
		ds.Connect()
		if !config.silent && !fyne.CurrentApp().Preferences().Bool("tried_discord_login") {
			go func() {
				ds.WaitReady()
				if !ds.IsLoggedIn() {
					fyne.Do(func() {
						var loginDialog *dialog.CustomDialog
						if ds.StartSignIn(func(success bool) {
							fyne.Do(func() {
								if loginDialog != nil {
									loginDialog.Hide()
								}
								if success {
									fyne.CurrentApp().Preferences().SetBool("tried_discord_login", true)
								}
							})
						}) {
							progress := widget.NewProgressBarInfinite()
							content := container.NewVBox(
								widget.NewLabel(lang.LocalizeKey("settings.discord_login_waiting", "Please complete the Discord login in your browser.")),
								progress,
							)
							loginDialog = dialog.NewCustom(
								lang.LocalizeKey("settings.discord_login_in_progress_title", "Login in progress"),
								lang.LocalizeKey("common.cancel", "Cancel"),
								content,
								w,
							)
							loginDialog.SetOnClosed(func() {
								if ds.IsSigningIn() {
									ds.AbortSignIn()
								}
							})
							loginDialog.Resize(fyne.NewSize(420, 160))
							loginDialog.Show()
						}
					})
				}
			}()
		}
	}
	state.Core.StartActivityPolling(ctx)
	w.SetOnClosed(onClosed)
	fyne.Do(func() {
		slog.Info("Application started", "silent", config.silent)
		for s, ok := state.Core.DiscordService.PopQueue(); ok; s, ok = state.Core.DiscordService.PopQueue() {
			l.HandleJoinLink(s)
		}
	})
	runtime.LockOSThread()
	fyne.CurrentApp().Run()
	return nil
}
