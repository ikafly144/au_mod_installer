//go:build windows

package uicommon

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/TheTitanrain/w32"
)

type dropFiles struct {
	PFiles uint32
	X      int32
	Y      int32
	FNC    uint32
	FWide  uint32
}

func (s *State) ClipboardSetFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute file path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("failed to access clipboard file: %w", err)
	}

	utf16Path, err := syscall.UTF16FromString(absPath)
	if err != nil {
		return fmt.Errorf("failed to encode clipboard file path: %w", err)
	}
	// Double null terminator for CF_HDROP file list.
	fileList := append(utf16Path, 0)

	headerSize := int(unsafe.Sizeof(dropFiles{}))
	totalBytes := headerSize + len(fileList)*2
	mem := w32.GlobalAlloc(w32.GMEM_MOVEABLE|w32.GMEM_ZEROINIT, uint32(totalBytes))
	if mem == 0 {
		return fmt.Errorf("failed to allocate clipboard memory")
	}
	memOwned := true
	defer func() {
		if memOwned {
			w32.GlobalFree(mem)
		}
	}()

	ptr := w32.GlobalLock(mem)
	if ptr == nil {
		return fmt.Errorf("failed to lock clipboard memory")
	}
	header := (*dropFiles)(ptr)
	header.PFiles = uint32(headerSize)
	header.FWide = 1

	dataPtr := unsafe.Pointer(uintptr(ptr) + uintptr(header.PFiles))
	dst := unsafe.Slice((*uint16)(dataPtr), len(fileList))
	copy(dst, fileList)
	_ = w32.GlobalUnlock(mem)

	if !w32.OpenClipboard(0) {
		return fmt.Errorf("failed to open clipboard")
	}
	defer w32.CloseClipboard()

	if !w32.EmptyClipboard() {
		return fmt.Errorf("failed to clear clipboard")
	}
	if w32.SetClipboardData(w32.CF_HDROP, w32.HANDLE(mem)) == 0 {
		return fmt.Errorf("failed to set file clipboard data")
	}

	memOwned = false // ownership transferred to the OS clipboard
	return nil
}
