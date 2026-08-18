//go:build !windows

package uicommon

import "errors"

func (s *State) EnableNativeTextDrop() (func(), error) {
	return nil, errors.New("native text drop is not supported on this platform")
}
