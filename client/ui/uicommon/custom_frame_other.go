//go:build !windows

package uicommon

func (s *State) EnableNativeCustomWindowFrame() (func(), error) {
	return nil, nil
}
