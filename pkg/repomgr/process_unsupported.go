//go:build !windows

package repomgr

import "fmt"

func IsRepoRunning() (pid int, err error) {
	return 0, fmt.Errorf("process check is only supported on Windows")
}

func IsProcessRunning(pid int) (bool, error) {
	return false, fmt.Errorf("process check is only supported on Windows")
}
