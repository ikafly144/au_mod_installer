package common

import (
	"errors"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"fyne.io/fyne/v2/driver"
	"github.com/TheTitanrain/w32"
)

func (s *State) ExplorerOpenFile(fileType, ext string) (path string, err error) {
	var hwnd uintptr
	if w, ok := s.Window.(driver.NativeWindow); ok {
		w.RunNative(func(context any) {
			if window, ok := context.(driver.WindowsWindowContext); ok {
				hwnd = window.HWND
			}
		})
	}
	buf := make([]uint16, w32.MAX_PATH)
	filter := unsafe.SliceData(utf16.Encode([]rune(fileType + " (" + ext + ")\000" + ext + "\000\000")))
	title, err := syscall.UTF16PtrFromString(fileType + "を選択")
	if err != nil {
		return "", err
	}
	ofn := &w32.OPENFILENAME{
		Owner:   w32.HWND(hwnd),
		File:    unsafe.SliceData(buf),
		MaxFile: uint32(len(buf)),
		Flags:   w32.OFN_FILEMUSTEXIST | w32.OFN_PATHMUSTEXIST | w32.OFN_NOCHANGEDIR,
		Filter:  filter,
		Title:   title,
	}
	ofn.StructSize = uint32(unsafe.Sizeof(*ofn))
	if !w32.GetOpenFileName(ofn) {
		return "", errors.New("file selection cancelled")
	}
	i := 0
	for i < len(buf) && buf[i] != 0 {
		i++
	}
	path = syscall.UTF16ToString(buf[:i])
	return path, nil
}
