//go:build !windows

package aumgr

import "errors"

func GetBinaryType(amongUsDir string) (BinaryType, error) {
	return BinaryTypeUnknown, errors.New("binary type detection is unsupported on this platform")
}
