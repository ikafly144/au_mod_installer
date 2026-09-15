//go:build !windows

package repomgr

import "fmt"

func GetRepoDir() (string, error) {
	return "", fmt.Errorf("auto-detection of R.E.P.O. is only supported on Windows")
}
