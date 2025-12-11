//go:build !windows

package aumgr

import "errors"

func DetectBinaryType(amongUsDir string) (BinaryType, error) {
	return BinaryTypeUnknown, errors.New("binary type detection is only supported on Windows")
}
