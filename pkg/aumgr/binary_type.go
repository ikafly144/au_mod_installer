package aumgr

type BinaryType string

const (
	BinaryTypeUnknown BinaryType = ""
	BinaryType32Bit   BinaryType = "x86"
	BinaryType64Bit   BinaryType = "x64"
)
