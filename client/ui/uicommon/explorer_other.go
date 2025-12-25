//go:build !windows

package uicommon

import (
	"errors"
)

func (s *State) ExplorerOpenFile(fileType, ext string) (path string, err error) {
	return "", errors.New("file explorer is not implemented on this platform")
}
