//go:build windows

package aumgr

import (
	"debug/pe"
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/v2/win32"
)

func GetBinaryType(amongUsDir string) (BinaryType, error) {
	path, err := syscall.UTF16PtrFromString(filepath.Join(amongUsDir, "Among Us.exe"))
	if err != nil {
		return BinaryTypeUnknown, err
	}
	var binaryType uint32
	isExe, winErr := win32.GetBinaryType(path, &binaryType)
	if winErr != win32.NO_ERROR {
		var fileinfo win32.SHFILEINFO
		flag := win32.SHGetFileInfo(path, 0, &fileinfo, uint32(unsafe.Sizeof(fileinfo)), win32.SHGFI_EXETYPE)
		if flag == 0 {
			if b, err := getBinaryTypeFallback(amongUsDir); err == nil {
				return b, nil
			}
			return BinaryTypeUnknown, fmt.Errorf("GetBinaryType failed with error: %v", winErr)
		}
		binaryType = uint32(flag)
	} else if isExe == 0 {
		return BinaryTypeUnknown, errors.New("given path is not an executable")
	}
	switch binaryType {
	case win32.SCS_32BIT_BINARY:
		return BinaryType32Bit, nil
	case win32.SCS_64BIT_BINARY:
		return BinaryType64Bit, nil
	default:
		return BinaryTypeUnknown, fmt.Errorf("unknown binary type: %d", binaryType)
	}
}

func getBinaryTypeFallback(amongUsDir string) (BinaryType, error) {
	dllPath := filepath.Join(amongUsDir, "GameAssembly.dll")
	file, err := pe.Open(dllPath)
	if err != nil {
		return BinaryTypeUnknown, err
	}
	defer file.Close()
	// Check the architecture of the loaded DLL
	switch file.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return BinaryType32Bit, nil
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return BinaryType64Bit, nil
	default:
		return BinaryTypeUnknown, fmt.Errorf("unknown architecture: %d", file.Machine)
	}
}
