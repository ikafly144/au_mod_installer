//go:build windows

package uicommon

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"github.com/zzl/go-win32api/win32"
)

func nudgeWindowPosition(w fyne.Window) {
	hwndValue, err := nativeHWNDFromFyneWindow(w)
	if err != nil || hwndValue == 0 {
		if err != nil {
			slog.Debug("Failed to get native HWND for window nudge", "error", err)
		}
		return
	}
	hwnd := win32.HWND(hwndValue)

	var rect win32.RECT
	if ok, winErr := win32.GetWindowRect(hwnd, &rect); ok == win32.FALSE {
		slog.Debug("GetWindowRect failed for window nudge", "error", winErr)
		return
	}

	x := rect.Left
	y := rect.Top
	flags := win32.SWP_NOSIZE | win32.SWP_NOZORDER | win32.SWP_NOACTIVATE | win32.SWP_NOSENDCHANGING
	if ok, winErr := win32.SetWindowPos(hwnd, 0, x+1, y+1, 0, 0, flags); ok == win32.FALSE {
		slog.Debug("SetWindowPos first nudge failed", "error", winErr)
		return
	}
	if ok, winErr := win32.SetWindowPos(hwnd, 0, x, y, 0, 0, flags); ok == win32.FALSE {
		slog.Debug("SetWindowPos restore nudge failed", "error", winErr)
	}
}
