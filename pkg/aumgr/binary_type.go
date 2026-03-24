package aumgr

import "github.com/ikafly144/au_mod_installer/common/rest/model"

type BinaryType string

const (
	BinaryTypeUnknown BinaryType = "unknown"
	BinaryType32Bit   BinaryType = "x86"
	BinaryType64Bit   BinaryType = "x64"
)

func (bt BinaryType) IsCompatibleWith(target model.TargetPlatform) bool {
	switch target {
	case model.TargetPlatformAny:
		return true
	case model.TargetPlatformX86:
		return bt == BinaryType32Bit
	case model.TargetPlatformX64:
		return bt == BinaryType64Bit
	default:
		return false
	}
}
