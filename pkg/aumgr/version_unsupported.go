//go:build !windows

package aumgr

import "errors"

func GetVersion(gamePath string) (string, error) {
	return "", errors.New("unsupported platform for reading Among Us version")
}
