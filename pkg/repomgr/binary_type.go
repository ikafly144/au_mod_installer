package repomgr

type BinaryType string

const (
	BinaryTypeUnknown BinaryType = "unknown"
	BinaryType64Bit   BinaryType = "x64"
)

func (bt BinaryType) String() string {
	return string(bt)
}

func GetBinaryType(gamePath string) (BinaryType, error) {
	return BinaryType64Bit, nil
}
