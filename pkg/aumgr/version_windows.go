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

/*
#include <stdlib.h>
#include <stdio.h>
#include <windows.h>

typedef int(*ReadVersionFileFn)(const char *path, char *buffer, int bufferSize);

HMODULE hModule = NULL;

int init(char *path) {
	hModule = LoadLibraryA(path);
	if (hModule == NULL) {
		return -1;
	}
	return 0;
}

int ReadVersionFile(const char *path, char *buffer, int bufferSize) {
	ReadVersionFileFn readVersionFile = (ReadVersionFileFn)GetProcAddress(hModule, "ReadVersionFile");
	if (readVersionFile == NULL) {
		return -2;
	}
	return readVersionFile(path, buffer, bufferSize);
}
*/
import "C"

//go:embed lib/dump-version.dll
var dumpVersionDLL []byte

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
	cPath := C.CString(filepath.Join(filepath.Dir(path), "lib", "dump-version.dll"))
	defer C.free(unsafe.Pointer(cPath))
	if res := C.init(cPath); res != 0 {
		panic(fmt.Sprintf("failed to load dump-version.dll: error code %d", res))
	}
}

func readVersionFile(path string) (string, error) {
	const bufferSize = 128
	var buffer [bufferSize]C.char
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", fmt.Errorf("failed to convert path to C string: %w", err)
	}
	result := C.ReadVersionFile((*C.char)(unsafe.Pointer(pathPtr)), &buffer[0], C.int(bufferSize))
	if result != 0 {
		return "", fmt.Errorf("failed to read version file: error code %d", result)
	}
	return C.GoString(&buffer[0]), nil
}

func GetVersion(gamePath string) (string, error) {
	versionFilePath := filepath.Join(gamePath, "Among Us_Data", "globalgamemanagers")
	return readVersionFile(versionFilePath)
}
