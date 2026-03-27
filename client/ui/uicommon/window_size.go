package uicommon

import (
	"fyne.io/fyne/v2"
)

const (
	BaseWindowWidth  = float32(440)
	BaseWindowHeight = float32(720)

	preferenceMainWindowWidth  = "ui.main_window_width"
	preferenceMainWindowHeight = "ui.main_window_height"
)

func ScaledMainWindowSize(scale float32) fyne.Size {
	if scale <= 0 {
		scale = 1
	}
	return fyne.NewSize(BaseWindowWidth*scale, BaseWindowHeight*scale)
}

func ApplyScaledMainWindowSize(w fyne.Window, scale float32) {
	if w == nil {
		return
	}
	size := ScaledMainWindowSize(scale)
	fyne.DoAndWait(func() {
		if c := w.Canvas(); c != nil {
			c.Refresh(w.Content())
		}
		w.Resize(size)
		nudgeWindowPosition(w)
	})
}

func RestoreMainWindowSize(w fyne.Window) (defaultSize bool) {
	if w == nil {
		return false
	}
	prefs := fyne.CurrentApp().Preferences()
	width := float32(prefs.FloatWithFallback(preferenceMainWindowWidth, 0))
	height := float32(prefs.FloatWithFallback(preferenceMainWindowHeight, 0))
	if width <= 0 || height <= 0 {
		defaultSize := w.Content().MinSize()
		defaultSize.Height = max(defaultSize.Height, BaseWindowHeight)
		defaultSize.Width = max(defaultSize.Width, BaseWindowWidth)
		w.Resize(defaultSize)
		return true
	}
	size := fyne.NewSize(width, height)
	if content := w.Content(); content != nil {
		min := content.MinSize()
		if size.Width < min.Width {
			size.Width = min.Width
		}
		if size.Height < min.Height {
			size.Height = min.Height
		}
	}
	w.Resize(size)
	return false
}

func SaveMainWindowSize(w fyne.Window) {
	if w == nil {
		return
	}
	var size fyne.Size
	if canvas := w.Canvas(); canvas != nil {
		size = canvas.Size()
	}
	if (size.Width <= 0 || size.Height <= 0) && w.Content() != nil {
		size = w.Content().Size()
	}
	if size.Width <= 0 || size.Height <= 0 {
		return
	}
	prefs := fyne.CurrentApp().Preferences()
	prefs.SetFloat(preferenceMainWindowWidth, float64(size.Width))
	prefs.SetFloat(preferenceMainWindowHeight, float64(size.Height))
}
