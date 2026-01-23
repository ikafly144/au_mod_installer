package assetstools

import (
	"encoding/binary"
	"io"
)

func NewAssetsFileReader(r io.ReadSeeker) *AssetsFileReader {
	return &AssetsFileReader{
		reader: r,
	}
}

type AssetsFileReader struct {
	reader    io.ReadSeeker
	BigEndian bool
}

func (r *AssetsFileReader) byteOrder() binary.ByteOrder {
	if r.BigEndian {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

func (r *AssetsFileReader) ReadBoolean() (bool, error) {
	var buf [1]byte
	_, err := r.reader.Read(buf[:])
	if err != nil {
		return false, err
	}
	return buf[0] != 0, nil
}

func (r *AssetsFileReader) ReadInt16() (int16, error) {
	var value int16
	return value, binary.Read(r.reader, r.byteOrder(), &value)
}

func (r *AssetsFileReader) ReadUint16() (uint16, error) {
	var value uint16
	return value, binary.Read(r.reader, r.byteOrder(), &value)
}

func (r *AssetsFileReader) ReadInt24() (int32, error) {
	var bytes [3]byte
	_, err := io.ReadFull(r.reader, bytes[:])
	if err != nil {
		return 0, err
	}
	var value int32
	if r.BigEndian {
		value = int32(bytes[0])<<16 | int32(bytes[1])<<8 | int32(bytes[2])
		if value&(1<<23) != 0 {
			value |= ^0 << 24
		}
	} else {
		value = int32(bytes[2])<<16 | int32(bytes[1])<<8 | int32(bytes[0])
		if value&(1<<23) != 0 {
			value |= ^0 << 24
		}
	}
	return value, nil
}

func (r *AssetsFileReader) ReadUint24() (uint32, error) {
	var bytes [3]byte
	_, err := io.ReadFull(r.reader, bytes[:])
	if err != nil {
		return 0, err
	}
	var value uint32
	if r.BigEndian {
		value = uint32(bytes[0])<<16 | uint32(bytes[1])<<8 | uint32(bytes[2])
	} else {
		value = uint32(bytes[2])<<16 | uint32(bytes[1])<<8 | uint32(bytes[0])
	}
	return value, nil
}

func (r *AssetsFileReader) ReadInt32() (int32, error) {
	var value int32
	return value, binary.Read(r.reader, r.byteOrder(), &value)
}

func (r *AssetsFileReader) ReadUint32() (uint32, error) {
	var value uint32
	return value, binary.Read(r.reader, r.byteOrder(), &value)
}

func (r *AssetsFileReader) ReadInt64() (int64, error) {
	var value int64
	return value, binary.Read(r.reader, r.byteOrder(), &value)
}

func (r *AssetsFileReader) ReadUint64() (uint64, error) {
	var value uint64
	return value, binary.Read(r.reader, r.byteOrder(), &value)
}

func (r *AssetsFileReader) Align() error {
	return r.alignN(4)
}

func (r *AssetsFileReader) Align8() error {
	return r.alignN(8)
}

func (r *AssetsFileReader) Align16() error {
	return r.alignN(16)
}

func (r *AssetsFileReader) alignN(n int64) error {
	currentOffset, err := r.reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	padding := (n - (currentOffset % n))
	if padding != n {
		_, err = r.reader.Seek(padding, io.SeekCurrent)
		if err != nil {
			return err
		}
	}
	return err
}

func (r *AssetsFileReader) ReadStringsLength(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := io.ReadFull(r.reader, bytes)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (r *AssetsFileReader) ReadNullTerminatedString() (string, error) {
	var bytes []byte
	for {
		var b [1]byte
		_, err := r.reader.Read(b[:])
		if err != nil {
			return "", err
		}
		if b[0] == 0 {
			break
		}
		bytes = append(bytes, b[0])
	}
	return string(bytes), nil
}

func (r *AssetsFileReader) ReadNullTerminatedArray(bytes []byte, pos int) (string, error) {
	var result []byte
	for {
		if pos >= len(bytes) {
			return "", io.EOF
		}
		b := bytes[pos]
		pos++
		if b == 0 {
			break
		}
		result = append(result, b)
	}
	return string(result), nil
}

func (r *AssetsFileReader) ReadCountString() (string, error) {
	var buf [1]byte
	_, err := r.reader.Read(buf[:])
	if err != nil {
		return "", err
	}
	length := int(buf[0])
	return r.ReadStringsLength(length)
}

func (r *AssetsFileReader) ReadCountStringInt16() (string, error) {
	length, err := r.ReadUint16()
	if err != nil {
		return "", err
	}
	return r.ReadStringsLength(int(length))
}

func (r *AssetsFileReader) ReadCountStringInt32() (string, error) {
	length, err := r.ReadInt32()
	if err != nil {
		return "", err
	}
	return r.ReadStringsLength(int(length))
}

func (r *AssetsFileReader) Position() (int64, error) {
	return r.reader.Seek(0, io.SeekCurrent)
}

func (r *AssetsFileReader) SeekOffset(offset int64) error {
	_, err := r.reader.Seek(offset, io.SeekCurrent)
	return err
}

func (r *AssetsFileReader) SeekAt(position int64) error {
	_, err := r.reader.Seek(position, io.SeekStart)
	return err
}

func (r *AssetsFileReader) ReadBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := io.ReadFull(r.reader, bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (r *AssetsFileReader) ReadByte() (byte, error) {
	var buf [1]byte
	_, err := r.reader.Read(buf[:])
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}
