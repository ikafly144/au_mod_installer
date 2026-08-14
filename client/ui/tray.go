//go:build windows

package ui

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"github.com/zzl/go-win32api/v2/win32"
)

func setupSystemTray(w fyne.Window, onQuit func()) {
	desk, ok := fyne.CurrentApp().(desktop.App)
	if !ok {
		slog.Warn("Desktop system tray is not supported by current app driver")
		return
	}

	showItem := fyne.NewMenuItem(lang.LocalizeKey("tray.show", "Show Mod of Us"), func() {
		fyne.Do(func() {
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
		})
	})

	quitItem := fyne.NewMenuItem(lang.LocalizeKey("tray.quit", "Quit"), func() {
		slog.Info("Quitting application from system tray")
		if onQuit != nil {
			onQuit()
		}
		w.SetCloseIntercept(nil)
		w.Close()
		fyne.CurrentApp().Quit()
	})

	menu := fyne.NewMenu(lang.LocalizeKey("app.name", "Mod of Us"), showItem, fyne.NewMenuItemSeparator(), quitItem)
	desk.SetSystemTrayMenu(menu)

	if icon := fyne.CurrentApp().Metadata().Icon; icon != nil {
		desk.SetSystemTrayIcon(icon)
	}
}
