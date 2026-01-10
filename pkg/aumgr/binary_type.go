package aumgr

type BinaryType string

const (
	BinaryTypeUnknown BinaryType = "unknown"
	BinaryType32Bit   BinaryType = "x86"
	BinaryType64Bit   BinaryType = "x64"
)
