//go:build windows

package ui

import (
	"context"
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zzl/go-win32api/v2/win32"

	"github.com/ikafly144/au_mod_installer/client/ui/tab/launcher"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/repo"
	servertab "github.com/ikafly144/au_mod_installer/client/ui/tab/server"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/settings"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"

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

	var (
		initOnce        sync.Once
		textDropCleanup func()
		launcherTabInst *launcher.Launcher
	)

	// Set fallback handlers for IPC before full UI is built
	state.OnActivateReceived = func() {
		fyne.Do(func() {
			if state.ShowWindow != nil {
				state.ShowWindow()
			}
		})
	}
	state.OnSharedURIReceived = func(uri string) {
		state.SharedURI = uri
		fyne.Do(func() {
			if state.ShowWindow != nil {
				state.ShowWindow()
			}
		})
	}
	state.OnSharedArchiveReceived = func(path string) {
		state.SharedArchive = path
		fyne.Do(func() {
			if state.ShowWindow != nil {
				state.ShowWindow()
			}
		})
	}

	buildFullUI := func() {
		l := launcher.NewLauncherTab(state)
		launcherTabInst = l
		launcherTab, err := l.Tab()
		if err != nil {
			slog.Error("Failed to create launcher tab", "error", err)
			return
		}

		repoPlaceholder := container.NewStack()
		repoItem := container.NewTabItem(lang.LocalizeKey("repository.tab_name", "Repository"), repoPlaceholder)
		var loadRepoOnce sync.Once

		serverPlaceholder := container.NewStack()
		serverItem := container.NewTabItem(lang.LocalizeKey("server.tab_name", "Servers"), serverPlaceholder)
		var loadServerOnce sync.Once

		settingsPlaceholder := container.NewStack()
		settingsItem := container.NewTabItem(lang.LocalizeKey("settings.title", "Settings"), settingsPlaceholder)
		var loadSettingsOnce sync.Once

		tabs := container.NewAppTabs(
			launcherTab,
			repoItem,
			serverItem,
			settingsItem,
		)
		tabs.OnSelected = func(item *container.TabItem) {
			switch item {
			case repoItem:
				loadRepoOnce.Do(func() {
					r := repo.NewRepository(state)
					t, err := r.Tab()
					if err == nil {
						repoPlaceholder.Objects = []fyne.CanvasObject{t.Content}
						repoPlaceholder.Refresh()
					} else {
						slog.Error("Failed to create repo tab", "error", err)
					}
				})
			case serverItem:
				loadServerOnce.Do(func() {
					st := servertab.NewServerTab(state)
					t, err := st.Tab()
					if err == nil {
						serverPlaceholder.Objects = []fyne.CanvasObject{t.Content}
						serverPlaceholder.Refresh()
					} else {
						slog.Error("Failed to create server tab", "error", err)
					}
				})
			case settingsItem:
				loadSettingsOnce.Do(func() {
					s := settings.NewSettings(state)
					t, err := s.Tab()
					if err == nil {
						settingsPlaceholder.Objects = []fyne.CanvasObject{t.Content}
						settingsPlaceholder.Refresh()
					} else {
						slog.Error("Failed to create settings tab", "error", err)
					}
				})
			}
		}

		w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
			if state.OnDroppedURIs != nil {
				state.OnDroppedURIs(uris)
			}
		})
		w.SetContent(tabs)
		w.SetFixedSize(false)

		for s, ok := state.Core.DiscordService.PopQueue(); ok; s, ok = state.Core.DiscordService.PopQueue() {
			l.HandleJoinLink(s)
		}
	}

	state.ShowWindow = func() {
		initOnce.Do(func() {
			buildFullUI()
			w.Show()
			state.SetWindowVisible(true)
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
		state.SetWindowVisible(true)
		w.RequestFocus()
		if state.Core != nil && state.Core.DiscordService != nil {
			state.Core.DiscordService.SetIdleActivityEnabled(true)
		}
		if nw, ok := w.(driver.NativeWindow); ok {
			nw.RunNative(func(context any) {
				if winCtx, ok := context.(driver.WindowsWindowContext); ok {
					hwnd := win32.HWND(winCtx.HWND)
					win32.ShowWindow(hwnd, win32.SW_RESTORE)
					win32.SetWindowPos(hwnd, win32.HWND_TOPMOST, 0, 0, 0, 0, win32.SWP_NOMOVE|win32.SWP_NOSIZE|win32.SWP_SHOWWINDOW)
					win32.SetWindowPos(hwnd, win32.HWND_NOTOPMOST, 0, 0, 0, 0, win32.SWP_NOMOVE|win32.SWP_NOSIZE|win32.SWP_SHOWWINDOW)
					win32.SetForegroundWindow(hwnd)
					win32.BringWindowToTop(hwnd)
				}
			})
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	onClosed := func() {
		state.SetWindowVisible(false)
		if state.CloseIPC != nil {
			state.CloseIPC()
		}
		if state.Core != nil && state.Core.DiscordService != nil {
			state.Core.DiscordService.SetIdleActivityEnabled(false)
		}
		uicommon.SaveMainWindowSize(w)
		if textDropCleanup != nil {
			textDropCleanup()
		}
		cancel()
	}

	setupSystemTray(w, state, onClosed)

	w.SetCloseIntercept(func() {
		resident := fyne.CurrentApp().Preferences().BoolWithFallback("tray_resident", true)
		gameRunning := state.Core != nil && state.Core.IsAnyProfileBusy()
		if resident || gameRunning {
			uicommon.SaveMainWindowSize(w)
			w.Hide()
			state.SetWindowVisible(false)
			if state.Core != nil && state.Core.DiscordService != nil {
				state.Core.DiscordService.SetIdleActivityEnabled(false)
			}
			slog.Info("Main window hidden to system tray", "tray_resident", resident, "game_running", gameRunning)
			go func() {
				time.Sleep(300 * time.Millisecond)
				runtime.GC()
				debug.FreeOSMemory()
			}()
		} else {
			if onClosed != nil {
				onClosed()
			}
			w.SetCloseIntercept(nil)
			w.Close()
			fyne.CurrentApp().Quit()
		}
	})

	state.AddOnGameExitedListener(func(profileID uuid.UUID) {
		fyne.Do(func() {
			resident := fyne.CurrentApp().Preferences().BoolWithFallback("tray_resident", true)
			if !resident && !state.IsWindowVisible() && (state.Core == nil || !state.Core.IsAnyProfileBusy()) {
				slog.Info("Game exited and tray residency is disabled while window is hidden; quitting application")
				if onClosed != nil {
					onClosed()
				}
				w.SetCloseIntercept(nil)
				w.Close()
				fyne.CurrentApp().Quit()
			}
		})
	})

	if !config.silent || sharedURI != "" || sharedArchive != "" {
		state.ShowWindow()
	} else {
		// In silent mode, perform a prompt GC/FreeOSMemory after setting up tray
		go func() {
			time.Sleep(300 * time.Millisecond)
			runtime.GC()
			debug.FreeOSMemory()
		}()
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
	state.StartPeriodicUpdateChecker(ctx)
	w.SetOnClosed(onClosed)
	fyne.Do(func() {
		slog.Info("Application started", "silent", config.silent)
		if launcherTabInst != nil {
			for s, ok := state.Core.DiscordService.PopQueue(); ok; s, ok = state.Core.DiscordService.PopQueue() {
				launcherTabInst.HandleJoinLink(s)
			}
		}
	})
	runtime.LockOSThread()
	fyne.CurrentApp().Run()
	return nil
}
