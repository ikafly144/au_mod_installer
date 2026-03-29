package assetstools

import (
	"encoding/binary"
	"io"
)

type AssetsFileReader struct {
	reader    io.ReadSeeker
	BigEndian bool
}

func NewAssetsFileReader(r io.ReadSeeker) *AssetsFileReader {
	return &AssetsFileReader{reader: r}
}

func (r *AssetsFileReader) byteOrder() binary.ByteOrder {
	if r.BigEndian {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

func (r *AssetsFileReader) ReadBoolean() (bool, error) {
	var b [1]byte
	if _, err := io.ReadFull(r.reader, b[:]); err != nil {
		return false, err
	}
	return b[0] != 0, nil
}

func (r *AssetsFileReader) ReadByte() (byte, error) {
	var b [1]byte
	if _, err := io.ReadFull(r.reader, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *AssetsFileReader) ReadInt16() (int16, error) {
	var v int16
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadUint16() (uint16, error) {
	var v uint16
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadInt32() (int32, error) {
	var v int32
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadUint32() (uint32, error) {
	var v uint32
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadInt64() (int64, error) {
	var v int64
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadUint64() (uint64, error) {
	var v uint64
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadFloat32() (float32, error) {
	var v float32
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadFloat64() (float64, error) {
	var v float64
	return v, binary.Read(r.reader, r.byteOrder(), &v)
}

func (r *AssetsFileReader) ReadUint24() (uint32, error) {
	var b [3]byte
	if _, err := io.ReadFull(r.reader, b[:]); err != nil {
		return 0, err
	}
	if r.BigEndian {
		return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]), nil
	}
	return uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0]), nil
}

func (r *AssetsFileReader) ReadBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := io.ReadFull(r.reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (r *AssetsFileReader) ReadNullTerminatedString() (string, error) {
	var out []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == 0 {
			return string(out), nil
		}
		out = append(out, b)
	}
}

func (r *AssetsFileReader) ReadCountStringInt32() (string, error) {
	length, err := r.ReadInt32()
	if err != nil {
		return "", err
	}
	if length < 0 {
		return "", io.ErrUnexpectedEOF
	}
	b, err := r.ReadBytes(int(length))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *AssetsFileReader) Align() error {
	return r.alignN(4)
}

func (r *AssetsFileReader) Align8() error {
	return r.alignN(8)
}

func (r *AssetsFileReader) alignN(n int64) error {
	cur, err := r.Position()
	if err != nil {
		return err
	}
	padding := n - (cur % n)
	if padding == n {
		return nil
	}
	_, err = r.reader.Seek(padding, io.SeekCurrent)
	return err
}

func (r *AssetsFileReader) Position() (int64, error) {
	return r.reader.Seek(0, io.SeekCurrent)
}

func (r *AssetsFileReader) SeekAt(position int64) error {
	_, err := r.reader.Seek(position, io.SeekStart)
	return err
}

func (r *AssetsFileReader) SeekOffset(offset int64) error {
	_, err := r.reader.Seek(offset, io.SeekCurrent)
	return err
}
