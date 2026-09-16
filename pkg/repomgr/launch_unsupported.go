//go:build !windows

package repomgr

import "fmt"

func LaunchRepo(gameDir string, dllDir string, onStarted func(pid int) error, args ...string) error {
	return fmt.Errorf("launching R.E.P.O. is only supported on Windows")
}
