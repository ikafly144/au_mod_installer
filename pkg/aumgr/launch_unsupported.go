//go:build !windows

package aumgr

import "errors"

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string, exchangeCode string, directJoinInfo DirectJoinInfo, onStarted func(pid int) error) error {
	return errors.New("launching Among Us is unsupported on this platform")
}
