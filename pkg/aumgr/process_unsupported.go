//go:build !windows

package aumgr

import "errors"

func IsAmongUsRunning() (pid int, err error) {
	return 0, errors.New("checking if Among Us is running is unsupported on this platform")
}
