package assetstools

import (
	"errors"
	"fmt"
)

type AssetsFile struct {
	Header   AssetsFileHeader
	Metadata AssetsFileMetadata
}

func ReadAssetsFile(reader *AssetsFileReader) (*AssetsFile, error) {
	if reader == nil {
		return nil, errors.New("reader is nil")
	}
	file := &AssetsFile{}
	if err := file.Header.Read(reader); err != nil {
		return nil, err
	}
	if err := file.Metadata.Read(reader, file.Header.Version); err != nil {
		return nil, err
	}
	return file, nil
}

type AssetsFileHeader struct {
	MetadataSize int64
	FileSize     int64
	Version      uint32
	DataOffset   int64
	Endianness   bool
}

func (h *AssetsFileHeader) Read(r *AssetsFileReader) error {
	r.BigEndian = true

	metadataSize, err := r.ReadUint32()
	if err != nil {
		return err
	}
	fileSize, err := r.ReadUint32()
	if err != nil {
		return err
	}
	version, err := r.ReadUint32()
	if err != nil {
		return err
	}
	dataOffset, err := r.ReadUint32()
	if err != nil {
		return err
	}
	endianness, err := r.ReadBoolean()
	if err != nil {
		return err
	}
	if err := r.SeekOffset(3); err != nil {
		return err
	}

	h.MetadataSize = int64(metadataSize)
	h.FileSize = int64(fileSize)
	h.Version = version
	h.DataOffset = int64(dataOffset)
	h.Endianness = endianness

	if h.Version >= 22 {
		metadataSize, err := r.ReadUint32()
		if err != nil {
			return err
		}
		fileSize, err := r.ReadInt64()
		if err != nil {
			return err
		}
		dataOffset, err := r.ReadInt64()
		if err != nil {
			return err
		}
		if err := r.SeekOffset(8); err != nil {
			return err
		}
		h.MetadataSize = int64(metadataSize)
		h.FileSize = fileSize
		h.DataOffset = dataOffset
	}

	r.BigEndian = h.Endianness
	return nil
}

type AssetsFileMetadata struct {
	UnityVersion    string
	TargetPlatform  uint32
	TypeTreeEnabled bool
	TypeTreeTypes   []*TypeTreeType
	AssetInfos      []AssetFileInfo

	byPathID map[int64]*AssetFileInfo
}

func (m *AssetsFileMetadata) Read(r *AssetsFileReader, version uint32) error {
	m.byPathID = make(map[int64]*AssetFileInfo)

	unityVersion, err := r.ReadNullTerminatedString()
	if err != nil {
		return err
	}
	m.UnityVersion = unityVersion

	targetPlatform, err := r.ReadUint32()
	if err != nil {
		return err
	}
	m.TargetPlatform = targetPlatform

	if version >= 13 {
		typeTreeEnabled, err := r.ReadBoolean()
		if err != nil {
			return err
		}
		m.TypeTreeEnabled = typeTreeEnabled
	}

	typeCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if typeCount < 0 {
		return fmt.Errorf("invalid type count: %d", typeCount)
	}
	m.TypeTreeTypes = make([]*TypeTreeType, typeCount)
	for i := 0; i < int(typeCount); i++ {
		t := &TypeTreeType{}
		if err := t.Read(r, version, m.TypeTreeEnabled, false); err != nil {
			return err
		}
		m.TypeTreeTypes[i] = t
	}

	assetInfoCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if assetInfoCount < 0 {
		return fmt.Errorf("invalid asset info count: %d", assetInfoCount)
	}
	if err := r.Align(); err != nil {
		return err
	}
	m.AssetInfos = make([]AssetFileInfo, assetInfoCount)
	for i := 0; i < int(assetInfoCount); i++ {
		info := &AssetFileInfo{}
		if err := info.Read(r, version); err != nil {
			return err
		}
		typeID, err := info.GetTypeID(m, version)
		if err != nil {
			return err
		}
		info.TypeID = typeID
		m.AssetInfos[i] = *info
		m.byPathID[info.PathID] = info
	}

	scriptTypeCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	for i := 0; i < int(scriptTypeCount); i++ {
		if _, err := r.ReadInt32(); err != nil {
			return err
		}
		if _, err := r.ReadInt64(); err != nil {
			return err
		}
	}

	externalCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	for i := 0; i < int(externalCount); i++ {
		if _, err := r.ReadNullTerminatedString(); err != nil {
			return err
		}
		if _, err := r.ReadBytes(16); err != nil {
			return err
		}
		if _, err := r.ReadInt32(); err != nil {
			return err
		}
		if _, err := r.ReadNullTerminatedString(); err != nil {
			return err
		}
	}

	if version >= 20 {
		refTypeCount, err := r.ReadInt32()
		if err != nil {
			return err
		}
		for i := 0; i < int(refTypeCount); i++ {
			t := &TypeTreeType{}
			if err := t.Read(r, version, m.TypeTreeEnabled, true); err != nil {
				return err
			}
		}
	}

	if version >= 5 {
		if _, err := r.ReadNullTerminatedString(); err != nil {
			return err
		}
	}

	return nil
}

func (m *AssetsFileMetadata) FindTypeTreeTypeByID(typeID int32, scriptTypeIndex int16) *TypeTreeType {
	for _, tt := range m.TypeTreeTypes {
		if tt.TypeID != typeID {
			continue
		}
		if scriptTypeIndex == -1 || tt.ScriptTypeIndex == scriptTypeIndex {
			return tt
		}
	}
	return nil
}

func (m *AssetsFileMetadata) GetAssetInfo(pathID int64) *AssetFileInfo {
	if m.byPathID == nil {
		return nil
	}
	return m.byPathID[pathID]
}

type AssetFileInfo struct {
	PathID          int64
	ByteOffset      int64
	ByteSize        uint32
	TypeIDOrIndex   int32
	OldTypeID       uint16
	ScriptTypeIndex uint16
	Stripped        uint8
	TypeID          int32
}

func (a *AssetFileInfo) Read(r *AssetsFileReader, version uint32) error {
	if err := r.Align(); err != nil {
		return err
	}
	if version >= 14 {
		pathID, err := r.ReadInt64()
		if err != nil {
			return err
		}
		a.PathID = pathID
	} else {
		pathID, err := r.ReadUint32()
		if err != nil {
			return err
		}
		a.PathID = int64(pathID)
	}

	if version >= 22 {
		byteOffset, err := r.ReadInt64()
		if err != nil {
			return err
		}
		a.ByteOffset = byteOffset
	} else {
		byteOffset, err := r.ReadUint32()
		if err != nil {
			return err
		}
		a.ByteOffset = int64(byteOffset)
	}

	byteSize, err := r.ReadUint32()
	if err != nil {
		return err
	}
	a.ByteSize = byteSize

	typeIDOrIndex, err := r.ReadInt32()
	if err != nil {
		return err
	}
	a.TypeIDOrIndex = typeIDOrIndex

	if version <= 15 {
		oldTypeID, err := r.ReadUint16()
		if err != nil {
			return err
		}
		a.OldTypeID = oldTypeID
	}
	if version <= 16 {
		scriptTypeIndex, err := r.ReadUint16()
		if err != nil {
			return err
		}
		a.ScriptTypeIndex = scriptTypeIndex
	}
	if version >= 15 && version <= 16 {
		stripped, err := r.ReadByte()
		if err != nil {
			return err
		}
		a.Stripped = stripped
	}
	return nil
}

func (a *AssetFileInfo) GetTypeID(metadata *AssetsFileMetadata, version uint32) (int32, error) {
	if version < 16 {
		return a.TypeIDOrIndex, nil
	}
	if a.TypeIDOrIndex < 0 || int(a.TypeIDOrIndex) >= len(metadata.TypeTreeTypes) {
		return 0, fmt.Errorf("type index %d out of range", a.TypeIDOrIndex)
	}
	return metadata.TypeTreeTypes[a.TypeIDOrIndex].TypeID, nil
}

func (a *AssetFileInfo) GetAbsoluteByteOffset(file *AssetsFile) int64 {
	return file.Header.DataOffset + a.ByteOffset
}

func (a *AssetFileInfo) GetScriptIndex(file *AssetsFile) int16 {
	if file.Header.Version < 16 {
		return int16(a.ScriptTypeIndex)
	}
	if a.TypeIDOrIndex < 0 || int(a.TypeIDOrIndex) >= len(file.Metadata.TypeTreeTypes) {
		return -1
	}
	return file.Metadata.TypeTreeTypes[a.TypeIDOrIndex].ScriptTypeIndex
}
