//go:build windows

package aumgr

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
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
	dllPath := filepath.Join(filepath.Dir(path), "lib", "dump-version.dll")
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(dllPath), "lib"), os.ModePerm); err != nil {
			panic(err)
		}
		if err := os.WriteFile(dllPath, dumpVersionDLL, 0644); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	} else {
		// Check hashsum to make sure the existing file is the same as the embedded one, if not, overwrite it
		existingDLL, err := os.OpenFile(dllPath, os.O_RDONLY, 0444)
		if err != nil {
			panic(err)
		}
		defer existingDLL.Close()
		hasher := sha256.New()
		_, err = io.Copy(hasher, existingDLL)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(hasher.Sum(nil), new(sha256.Sum256(dumpVersionDLL))[:]) {
			if err := os.WriteFile(dllPath, dumpVersionDLL, 0644); err != nil {
				panic(err)
			}
		}
	}
	dll = syscall.NewLazyDLL(dllPath)
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
	// truncate at the first null byte
	for i, b := range buffer {
		if b == 0 {
			return string(buffer[:i]), nil
		}
	}
	return string(buffer[:]), nil
}

func GetVersion(gamePath string) (string, error) {
	versionFilePath := filepath.Join(gamePath, "Among Us_Data", "globalgamemanagers")
	return readVersionFile(versionFilePath)
}
