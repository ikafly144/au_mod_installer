//go:build windows

package aumgr

import (
	"errors"
	"path/filepath"
	"syscall"

	"github.com/zzl/go-win32api/win32"
)

func DetectBinaryType(amongUsDir string) (BinaryType, error) {
	path, err := syscall.UTF16PtrFromString(filepath.Join(amongUsDir, "Among Us.exe"))
	if err != nil {
		return BinaryTypeUnknown, err
	}
	var binaryType uint32
	isExe, winErr := win32.GetBinaryType(path, &binaryType)
	if isExe == 0 {
		return BinaryTypeUnknown, errors.New("given path is not an executable")
	}
	if winErr != win32.NO_ERROR {
		return BinaryTypeUnknown, winErr
	}
	switch binaryType {
	case win32.SCS_32BIT_BINARY:
		return BinaryType32Bit, nil
	case win32.SCS_64BIT_BINARY:
		return BinaryType64Bit, nil
	default:
		return BinaryTypeUnknown, nil
	}
}
