//go:build !windows

package aumgr

import "errors"

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string, args ...string) error {
	return errors.New("launching Among Us is only supported on Windows")
}
