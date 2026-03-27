//go:build !windows

package aumgr

import "errors"

func IsAmongUsRunning() (pid int, err error) {
	return 0, errors.New("checking if Among Us is running is unsupported on this platform")
}

func IsProcessRunning(pid int) (bool, error) {
	return false, errors.New("checking process state is unsupported on this platform")
}
