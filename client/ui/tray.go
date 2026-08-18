//go:build windows

package ui

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
)

func setupSystemTray(w fyne.Window, state *uicommon.State, onQuit func()) {
	desk, ok := fyne.CurrentApp().(desktop.App)
	if !ok {
		slog.Warn("Desktop system tray is not supported by current app driver")
		return
	}

	showItem := fyne.NewMenuItem(lang.LocalizeKey("tray.show", "Show Mod of Us"), func() {
		fyne.Do(func() {
			if state != nil && state.ShowWindow != nil {
				state.ShowWindow()
			} else {
				w.Show()
				w.RequestFocus()
			}
		})
	})

	quitItem := fyne.NewMenuItem(lang.LocalizeKey("tray.quit", "Quit"), func() {
		slog.Info("Quitting application from system tray")
		if state != nil && state.CloseIPC != nil {
			state.CloseIPC()
		}
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
