//go:build windows

package aumgr

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	_ "embed"
)

//go:embed lib/dump-version.dll
var dumpVersionDLL []byte

var dll *syscall.LazyDLL

func init() {
	path, err := os.Executable()
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "lib", "dump-version.dll")); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(path), "lib"), os.ModePerm); err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(path), "lib", "dump-version.dll"), dumpVersionDLL, 0644); err != nil {
			panic(err)
		}
	}
	dll = syscall.NewLazyDLL(filepath.Join(filepath.Dir(path), "lib", "dump-version.dll"))
}

func readVersionFile(path string) (string, error) {
	const bufferSize = 128
	var buffer [bufferSize]byte
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", fmt.Errorf("failed to convert path to UTF16: %w", err)
	}
	proc := dll.NewProc("ReadVersionFile")
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(bufferSize))
	if ret != 0 {
		return "", fmt.Errorf("failed to read version file: error code %d", ret)
	}
	return string(buffer[:]), nil
}

func GetVersion(gamePath string) (string, error) {
	versionFilePath := filepath.Join(gamePath, "Among Us_Data", "globalgamemanagers")
	return readVersionFile(versionFilePath)
}
