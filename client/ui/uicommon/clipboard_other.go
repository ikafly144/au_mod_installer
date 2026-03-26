//go:build !windows

package uicommon

import "errors"

func (s *State) ClipboardSetFile(path string) error {
	return errors.New("native file clipboard is not implemented on this platform")
}
