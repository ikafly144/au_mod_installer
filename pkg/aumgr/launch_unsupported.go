//go:build !windows

package aumgr

import "errors"

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string, exchangeCode string, lobbyCode string, serverIP string, serverPort uint16, onStarted func(pid int) error) error {
	return errors.New("launching Among Us is unsupported on this platform")
}
