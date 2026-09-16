//go:build !windows

package repomgr

import "fmt"

func GetVersion(gamePath string) (string, error) {
	return "", fmt.Errorf("reading game version is only supported on Windows")
}
