//go:build !windows

package aumgr

func IsAmongUsRunning() (pid int, err error) {
	panic("not implemented yet")
}
