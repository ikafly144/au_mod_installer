//go:build !windows

package aumgr

import "errors"

func GetAmongUsDir() (string, error) {
	return "", errors.New("automatic Among Us directory detection is unsupported on this platform")
}
