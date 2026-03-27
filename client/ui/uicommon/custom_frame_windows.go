//go:build windows

package uicommon

import (
	"fmt"
	"image/color"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/theme"
	"golang.org/x/sys/windows"
)

const (
	dwmwaNCRenderingPolicy    = 2
	dwmwaUseImmersiveDarkMode = 20
	dwmwaWindowCornerPref     = 33
	dwmwaBorderColor          = 34
	dwmwaCaptionColor         = 35
	dwmwaTextColor            = 36
	dwmwaSystemBackdropType   = 38

	dwmncrpEnabled      = 2
	dwmwcpRound         = 2
	dwmsbtMainWindow    = 2
	dwmColorDefault     = 0xFFFFFFFF
	dwmColorTransparent = 0xFFFFFFFE
)

type dwmMargins struct {
	CxLeftWidth    int32
	CxRightWidth   int32
	CyTopHeight    int32
	CyBottomHeight int32
}

var (
	dwmapiDLL                        = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmSetWindowAttribute        = dwmapiDLL.NewProc("DwmSetWindowAttribute")
	procDwmExtendFrameIntoClientArea = dwmapiDLL.NewProc("DwmExtendFrameIntoClientArea")
)

func (s *State) EnableNativeCustomWindowFrame() (func(), error) {
	if s.Window == nil {
		return nil, fmt.Errorf("window is nil")
	}

	hwnd, err := nativeHWNDFromFyneWindow(s.Window)
	if err != nil {
		return nil, err
	}

	if err := applyDWMCustomFrame(hwnd); err != nil {
		return nil, err
	}

	return nil, nil
}

func nativeHWNDFromFyneWindow(w fyne.Window) (uintptr, error) {
	nativeWindow, ok := w.(driver.NativeWindow)
	if !ok {
		return 0, fmt.Errorf("window does not support native handle")
	}
	var hwnd uintptr
	nativeWindow.RunNative(func(context any) {
		if windowCtx, ok := context.(driver.WindowsWindowContext); ok {
			hwnd = windowCtx.HWND
		}
	})
	if hwnd == 0 {
		return 0, fmt.Errorf("failed to get HWND from native window context")
	}
	return hwnd, nil
}

func applyDWMCustomFrame(hwnd uintptr) error {
	var applied bool
	var lastErr error

	if err := setDWMUIntAttribute(hwnd, dwmwaNCRenderingPolicy, dwmncrpEnabled); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	darkMode := uint32(0)
	if fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantDark {
		darkMode = 1
	}
	if err := setDWMUIntAttribute(hwnd, dwmwaUseImmersiveDarkMode, darkMode); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	if err := setDWMUIntAttribute(hwnd, dwmwaWindowCornerPref, dwmwcpRound); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	if err := setDWMUIntAttribute(hwnd, dwmwaSystemBackdropType, dwmsbtMainWindow); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	if err := setDWMColorAttribute(hwnd, dwmwaCaptionColor, themeColorRef(theme.Color(theme.ColorNameBackground))); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	if err := setDWMColorAttribute(hwnd, dwmwaTextColor, themeColorRef(theme.Color(theme.ColorNameForeground))); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	if err := setDWMColorAttribute(hwnd, dwmwaBorderColor, dwmColorTransparent); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	margins := dwmMargins{CxLeftWidth: 0, CxRightWidth: 0, CyTopHeight: 1, CyBottomHeight: 0}
	if err := extendFrameIntoClientArea(hwnd, margins); err != nil {
		lastErr = err
	} else {
		applied = true
	}

	if applied {
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("failed to apply DWM custom frame: %w", lastErr)
	}
	return fmt.Errorf("failed to apply DWM custom frame: no DWM operations succeeded")
}

func setDWMUIntAttribute(hwnd uintptr, attr uint32, value uint32) error {
	return callDwmSetWindowAttribute(hwnd, attr, unsafe.Pointer(&value), unsafe.Sizeof(value))
}

func setDWMColorAttribute(hwnd uintptr, attr uint32, colorRef uint32) error {
	return callDwmSetWindowAttribute(hwnd, attr, unsafe.Pointer(&colorRef), unsafe.Sizeof(colorRef))
}

func callDwmSetWindowAttribute(hwnd uintptr, attr uint32, value unsafe.Pointer, size uintptr) error {
	hr, _, callErr := procDwmSetWindowAttribute.Call(
		hwnd,
		uintptr(attr),
		uintptr(value),
		size,
	)
	if int32(hr) < 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("DwmSetWindowAttribute(attr=%d) failed: hr=0x%08X, err=%v", attr, uint32(hr), callErr)
		}
		return fmt.Errorf("DwmSetWindowAttribute(attr=%d) failed: hr=0x%08X", attr, uint32(hr))
	}
	return nil
}

func extendFrameIntoClientArea(hwnd uintptr, margins dwmMargins) error {
	hr, _, callErr := procDwmExtendFrameIntoClientArea.Call(
		hwnd,
		uintptr(unsafe.Pointer(&margins)),
	)
	if int32(hr) < 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("DwmExtendFrameIntoClientArea failed: hr=0x%08X, err=%v", uint32(hr), callErr)
		}
		return fmt.Errorf("DwmExtendFrameIntoClientArea failed: hr=0x%08X", uint32(hr))
	}
	return nil
}

func themeColorRef(c color.Color) uint32 {
	r16, g16, b16, _ := c.RGBA()
	r8 := uint32(r16 >> 8)
	g8 := uint32(g16 >> 8)
	b8 := uint32(b16 >> 8)
	return b8<<16 | g8<<8 | r8
}
