package assetstools

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type AssetFileInfo struct {
	// Path ID of the asset.
	PathId int64
	// Address of the asset's data from the header's DataOffset.
	// Use GetAbsoluteByteOffset(AssetsFile) for the real file position.
	// If the asset has a replacer, this field is ignored.
	ByteOffset int64
	// Byte size of the asset data. If the asset has a replacer, this field is ignored.
	ByteSize uint32
	// Before version 16 this is the type ID of the asset. In version 16 and later this is the
	// index into the type tree list. In versions 15 and below, this is the same as TypeId
	// except in MonoBehaviours where this acts similar to ScriptTypeIndex (negative).
	// You should use TypeId for the type ID in either version. AssetClassID
	TypeIdOrIndex int32
	// Old Type ID of the asset (officially called class ID). This field is only used in versions
	// 15 and below and is the same as TypeId, except when TypeId is negative, in which case
	// the old type ID will be a MonoBehaviour (0x72) and TypeId will be the same as TypeIdOrIndex.
	// You should use TypeId for the type ID in either version. AssetClassID
	OldTypeId uint16
	// Script type index of the asset. Assets other than MonoBehaviours will have 0xffff for
	// this field. This value is stored in the type tree starting at version 17. You should use
	// GetScriptIndex(AssetsFile) instead.
	ScriptTypeIndex uint16
	// Marks if the type in the type tree has been stripped (?)
	Stripped uint8

	// The type ID of the asset. This field works in both versions. This field is only for
	// convenience; modifying the type ID in the type tree in later versions will not update the
	// ID here, and modifying this field will not update the type ID when saved.
	TypeId int32
}

func (a *AssetFileInfo) Read(r *AssetsFileReader, version uint32) error {
	if err := r.Align(); err != nil {
		return err
	}
	if version >= 14 {
		pathId, err := r.ReadInt64()
		if err != nil {
			return err
		}
		a.PathId = pathId
	} else {
		pathId, err := r.ReadUint32()
		if err != nil {
			return err
		}
		a.PathId = int64(pathId)
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

	typeIdOrIndex, err := r.ReadInt32()
	if err != nil {
		return err
	}
	a.TypeIdOrIndex = typeIdOrIndex

	if version <= 15 {
		oldTypeId, err := r.ReadUint16()
		if err != nil {
			return err
		}
		a.OldTypeId = oldTypeId
	}
	if version <= 16 {
		scriptTypeIndex, err := r.ReadUint16()
		if err != nil {
			return err
		}
		a.ScriptTypeIndex = scriptTypeIndex
	}
	if 15 <= version && version <= 16 {
		stripped, err := r.ReadByte()
		if err != nil {
			return err
		}
		a.Stripped = stripped
	}
	return nil
}

func (a *AssetFileInfo) GetTypeId(m *AssetsFileMetadata, version uint32) (int32, error) {
	if version < 16 {
		return a.TypeIdOrIndex, nil
	}
	if int(a.TypeIdOrIndex) >= len(m.TypeTreeTypes) {
		return 0, fmt.Errorf("type index %d out of range", a.TypeIdOrIndex)
	}
	return m.TypeTreeTypes[a.TypeIdOrIndex].TypeId, nil
}

func NewAssetPPtr(fileId int32, pathId int64) *AssetPPtr {
	return &AssetPPtr{
		FileId: fileId,
		PathId: pathId,
	}
}

type AssetPPtr struct {
	// FileId of the pointer. If empty or zero, FilePath will be used.
	FilePath string
	// FileId of the pointer.
	FileId int32
	// PathId of the pointer.
	PathId int64
}

type AssetsFileHeader struct {
	MetadataSize int64
	FileSize     int64
	Version      uint32
	DataOffset   int64
	Endianness   bool
}

func (h *AssetsFileHeader) IsBigEndian() bool {
	return h.Endianness
}

func (h *AssetsFileHeader) Read(r *AssetsFileReader) (err error) {
	r.BigEndian = true
	metaDataSize, err := r.ReadUint32()
	if err != nil {
		return
	}
	h.MetadataSize = int64(metaDataSize)
	fileSize, err := r.ReadUint32()
	if err != nil {
		return
	}
	h.FileSize = int64(fileSize)
	h.Version, err = r.ReadUint32()
	if err != nil {
		return
	}
	dataOffset, err := r.ReadUint32()
	if err != nil {
		return
	}
	h.DataOffset = int64(dataOffset)
	h.Endianness, err = r.ReadBoolean()
	if err != nil {
		return
	}
	// unused bytes
	if err := r.SeekOffset(3); err != nil {
		return err
	}

	if h.Version >= 22 {
		metaDataSize, err := r.ReadUint32()
		if err != nil {
			return err
		}
		h.MetadataSize = int64(metaDataSize)
		fileSize, err := r.ReadInt64()
		if err != nil {
			return err
		}
		h.FileSize = fileSize
		dataOffset, err := r.ReadInt64()
		if err != nil {
			return err
		}
		h.DataOffset = dataOffset
		// unused bytes
		if err := r.SeekOffset(8); err != nil {
			return err
		}
	}

	r.BigEndian = h.Endianness
	return nil
}

type AssetsFileMetadata struct {
	// Engine version this file uses.
	UnityVersion string
	// Target platform this file uses.
	TargetPlatform uint32
	// Marks whether the type info contains type tree data.
	TypeTreeEnabled bool
	// List of type tree types.
	TypeTreeTypes []*TypeTreeType
	// List of asset infos. Do not add or remove from this list directly, instead use the
	// AddAssetInfo or RemoveAssetInfo methods.
	AssetInfos []AssetFileInfo
	// List of script type pointers. This list should match up with ScriptTypeIndex in the type
	// tree types list.
	ScriptTypes []*AssetPPtr
	// List of externals (references to other files).
	Externals []AssetFileExternal
	// List of reference types.
	RefTypes []*TypeTreeType
	// Unknown.
	UserInformation string

	quickLookupAssetInfo map[int64]*AssetFileInfo
}

func (m *AssetsFileMetadata) Read(r *AssetsFileReader, header *AssetsFileHeader) (err error) {
	return m.read(r, header.Version)
}

func (m *AssetsFileMetadata) read(r *AssetsFileReader, version uint32) (err error) {
	m.quickLookupAssetInfo = make(map[int64]*AssetFileInfo)

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

	fieldCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	m.TypeTreeTypes = make([]*TypeTreeType, fieldCount)
	for i := int32(0); i < fieldCount; i++ {
		typeTreeType := new(TypeTreeType)
		if err := typeTreeType.Read(r, version, m.TypeTreeEnabled, false); err != nil {
			return err
		}
		m.TypeTreeTypes[i] = typeTreeType
	}

	assetInfoCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if err := r.Align(); err != nil {
		return err
	}
	m.AssetInfos = make([]AssetFileInfo, assetInfoCount)
	for i := int32(0); i < assetInfoCount; i++ {
		fileInfo := new(AssetFileInfo)
		if err := fileInfo.Read(r, version); err != nil {
			return err
		}

		typeId, err := fileInfo.GetTypeId(m, version)
		if err != nil {
			return err
		}
		fileInfo.TypeId = typeId

		m.AssetInfos[i] = *fileInfo
		m.quickLookupAssetInfo[fileInfo.PathId] = fileInfo
	}

	scriptTypeCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	m.ScriptTypes = make([]*AssetPPtr, scriptTypeCount)
	for i := int32(0); i < scriptTypeCount; i++ {
		fileId, err := r.ReadInt32()
		if err != nil {
			return err
		}
		pathId, err := r.ReadInt64()
		if err != nil {
			return err
		}
		m.ScriptTypes[i] = NewAssetPPtr(fileId, pathId)
	}

	externalCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	m.Externals = make([]AssetFileExternal, externalCount)
	for i := int32(0); i < externalCount; i++ {
		external := new(AssetFileExternal)
		if err := external.Read(r, version); err != nil {
			return err
		}
		m.Externals[i] = *external
	}

	if version >= 20 {
		refTypeCount, err := r.ReadInt32()
		if err != nil {
			return err
		}
		m.RefTypes = make([]*TypeTreeType, refTypeCount)
		for i := int32(0); i < refTypeCount; i++ {
			refType := new(TypeTreeType)
			if err := refType.Read(r, version, m.TypeTreeEnabled, true); err != nil {
				return err
			}
			m.RefTypes[i] = refType
		}
	}

	if version >= 5 {
		userInformation, err := r.ReadNullTerminatedString()
		if err != nil {
			return err
		}
		m.UserInformation = userInformation
	}
	return nil
}

type AssetFileExternal struct {
	// Unknown.
	VirtualAssetPathName string
	// GUID for dependencies used in editor. Otherwise this is 0.
	Guid uuid.UUID
	// Dependency type.
	Type AssetsFileExternalType
	// Real path name to the other file.
	PathName string
	// Original path name listed in the assets file (if it was changed).
	// You shouldn't modify this.
	OriginalPathName string
}

func (a *AssetFileExternal) Read(r *AssetsFileReader, version uint32) error {
	var err error
	a.VirtualAssetPathName, err = r.ReadNullTerminatedString()
	if err != nil {
		return err
	}
	guid, err := r.ReadBytes(16)
	if err != nil {
		return err
	}
	a.Guid = uuid.UUID(guid)
	externalType, err := r.ReadInt32()
	if err != nil {
		return err
	}
	a.Type = AssetsFileExternalType(externalType)
	a.PathName, err = r.ReadNullTerminatedString()
	if err != nil {
		return err
	}
	a.OriginalPathName = a.PathName

	// Because lowercase "resources" is read by Unity fine on Linux, it either uses
	// hardcoded replaces like below or it has case insensitive pathing somehow.
	// This isn't consistent with the original AssetsTools but it only supported
	// Windows anyway, so this will only create issues if more than these three
	// pop up in the future. Also, the reason I don't just replace all "library"
	// with "Resources" is so that when saving, I can change it back to the original
	// (like how unity_builtin_extra goes back to "resources", not "library")
	switch a.PathName {
	case "resources/unity_builtin_extra":
		a.PathName = "Resources/unity_builtin_extra"
	case "library/unity default resources", "Library/unity default resources":
		a.PathName = "Resources/unity default resources"
	case "library/unity editor resources", "Library/unity editor resources":
		a.PathName = "Resources/unity editor resources"
	}
	return nil
}

type AssetsFileExternalType int32

const (
	AssetsFileExternalTypeNormal AssetsFileExternalType = iota
	AssetsFileExternalTypeCached
	AssetsFileExternalTypeSerialized
	AssetsFileExternalTypeMeta
)

func NewAssetTypeReference(className, namespace, asmName string) *AssetTypeReference {
	return &AssetTypeReference{
		ClassName: className,
		Namespace: namespace,
		AsmName:   asmName,
	}
}

func (ref *AssetTypeReference) ReadMetadata(r *AssetsFileReader) error {
	className, err := r.ReadNullTerminatedString()
	if err != nil {
		return err
	}
	namespace, err := r.ReadNullTerminatedString()
	if err != nil {
		return err
	}
	asmName, err := r.ReadNullTerminatedString()
	if err != nil {
		return err
	}
	ref.ClassName = className
	ref.Namespace = namespace
	ref.AsmName = asmName
	return nil
}

func (ref *AssetTypeReference) ReadAsset(r *AssetsFileReader) error {
	className, err := r.ReadCountStringInt32()
	if err != nil {
		return err
	}
	if err := r.Align(); err != nil {
		return err
	}
	namespace, err := r.ReadCountStringInt32()
	if err != nil {
		return err
	}
	if err := r.Align(); err != nil {
		return err
	}
	asmName, err := r.ReadCountStringInt32()
	if err != nil {
		return err
	}
	if err := r.Align(); err != nil {
		return err
	}
	ref.ClassName = className
	ref.Namespace = namespace
	ref.AsmName = asmName
	return nil
}

type AssetTypeReference struct {
	ClassName string
	Namespace string
	AsmName   string
}

type TypeTreeNode struct {
	// Version of the node.
	Version uint16
	// Level of the node (0 for root, 1 for child, etc.)
	Level byte
	// Information about whether the node is an array, registry, etc.
	TypeFlags TypeTreeNodeTypeFlags
	// Offset of the type string in the string table.
	TypeStrOffset uint32
	// Offset of the name string in the string table.
	NameStrOffset uint32
	// Byte size of the field's type (for example, int is 4).
	// If the field isn't a value type, then this value is a sum of all children sizes.
	// If the size is variable, this is set to -1.
	ByteSize int32
	// Index in the type tree. This should always be the same as the index in the array.
	Index uint32
	// 0x4000 if aligned.
	MetaFlags uint32
	// Unknown.
	RefTypeHash uint64
}

func (n *TypeTreeNode) Read(r *AssetsFileReader, version uint32) (err error) {
	n.Version, err = r.ReadUint16()
	if err != nil {
		return err
	}
	n.Level, err = r.ReadByte()
	if err != nil {
		return err
	}
	typeFlags, err := r.ReadByte()
	if err != nil {
		return err
	}
	n.TypeFlags = TypeTreeNodeTypeFlags(typeFlags)
	n.TypeStrOffset, err = r.ReadUint32()
	if err != nil {
		return err
	}
	n.NameStrOffset, err = r.ReadUint32()
	if err != nil {
		return err
	}
	n.ByteSize, err = r.ReadInt32()
	if err != nil {
		return err
	}
	n.Index, err = r.ReadUint32()
	if err != nil {
		return err
	}
	n.MetaFlags, err = r.ReadUint32()
	if err != nil {
		return err
	}
	if version >= 0x12 {
		n.RefTypeHash, err = r.ReadUint64()
		if err != nil {
			return err
		}
	}
	return nil
}

type TypeTreeNodeTypeFlags byte

const (
	None TypeTreeNodeTypeFlags = 0
	// Type tree node is an array.
	Array TypeTreeNodeTypeFlags = 1
	// Type tree node is a ref type. For example, "managedRefArrayItem" would be an
	// array item that is a reference to an object in the registry.
	Ref TypeTreeNodeTypeFlags = 2
	// Type tree node is a registry. Should just be "ManagedReferencesRegistry references".
	Registry TypeTreeNodeTypeFlags = 4
	// Type tree node is an array of ref types. This occurs if the SerializeReference was
	// added to a list or array instead of just a single field. This is not applied to the
	// Array child of the field, just the field itself.
	ArrayOfRefs TypeTreeNodeTypeFlags = 8
)

func (f TypeTreeNodeTypeFlags) Has(flag TypeTreeNodeTypeFlags) bool {
	return f&flag != 0
}

type TypeTreeType struct {
	// ID for this type.
	TypeId int32
	// Marks whether the type is stripped or not. Stripped types do not have any fields.
	IsStrippedType bool
	// Script index for this type.Only used in MonoBehaviours, and MonoBehaviours of the same
	// script have the same index.
	ScriptTypeIndex int16
	// Hash of the script's fields. Two different scripts with the same fields can have the same hash.
	ScriptIdHash Hash128
	// Hash of the script's fields.
	TypeHash Hash128
	// Nodes for this type. This list will be empty if the type is stripped.
	Nodes []*TypeTreeNode
	// String table bytes for this type.
	StringBufferBytes []byte
	// Is the type a reference type?
	IsRefType bool
	/// Type dependencies for this type. Used by MonoBehaviours referencing ref types. Only used
	/// when IsRefType is false.
	TypeDependencies []int32
	/// Type reference information. Only used when IsRefType is true.
	TypeReference *AssetTypeReference
}

func (t *TypeTreeType) Read(r *AssetsFileReader, version uint32, hasTypeTree bool, isRefType bool) (err error) {
	typeId, err := r.ReadInt32()
	if err != nil {
		return err
	}
	t.TypeId = typeId
	if version >= 16 {
		isStrippedType, err := r.ReadBoolean()
		if err != nil {
			return err
		}
		t.IsStrippedType = isStrippedType
	}

	if version >= 17 {
		scriptTypeIndex, err := r.ReadInt16()
		if err != nil {
			return err
		}
		t.ScriptTypeIndex = scriptTypeIndex
	} else {
		t.ScriptTypeIndex = -1
	}

	if (version < 17 && t.TypeId < 0) ||
		(version >= 17 && t.TypeId == int32(ClassID_MonoBehaviour)) ||
		(isRefType && t.ScriptTypeIndex != -1) {
		scriptIdHash, err := NewHash128(r)
		if err != nil {
			return err
		}
		t.ScriptIdHash = scriptIdHash
	}

	typeHash, err := NewHash128(r)
	if err != nil {
		return err
	}
	t.TypeHash = typeHash
	t.IsRefType = isRefType

	if hasTypeTree {
		typeTreeNodeCount, err := r.ReadInt32()
		if err != nil {
			return err
		}
		stringBufferLength, err := r.ReadInt32()
		if err != nil {
			return err
		}

		t.Nodes = make([]*TypeTreeNode, typeTreeNodeCount)
		for i := int32(0); i < typeTreeNodeCount; i++ {
			node := new(TypeTreeNode)
			if err := node.Read(r, version); err != nil {
				return err
			}
			t.Nodes[i] = node
		}

		stringBufferBytes, err := r.ReadBytes(int(stringBufferLength))
		if err != nil {
			return err
		}
		t.StringBufferBytes = stringBufferBytes

		if version >= 21 {
			if !isRefType {
				dependenciesCount, err := r.ReadInt32()
				if err != nil {
					return err
				}
				t.TypeDependencies = make([]int32, dependenciesCount)
				for i := int32(0); i < dependenciesCount; i++ {
					dep, err := r.ReadInt32()
					if err != nil {
						return err
					}
					t.TypeDependencies[i] = dep
				}
			} else {
				typeRef := new(AssetTypeReference)
				if err := typeRef.ReadMetadata(r); err != nil {
					return err
				}
				t.TypeReference = typeRef
			}
		}
	}
	return nil
}

func NewHash128(r *AssetsFileReader) (Hash128, error) {
	bytes, err := r.ReadBytes(16)
	if err != nil {
		return Hash128{}, err
	}
	return Hash128{data: bytes}, nil
}

func NewHash128FromBytes(data []byte) Hash128 {
	return Hash128{data: data}
}

func NewBlankHash128() Hash128 {
	return Hash128{data: make([]byte, 16)}
}

type Hash128 struct {
	data []byte // 16 bytes
}

func (h Hash128) IsZero() bool {
	if h.data == nil {
		return true
	}
	for _, b := range h.data {
		if b != 0 {
			return false
		}
	}
	return true
}

func (h Hash128) String() string {
	hex := new(strings.Builder)
	for _, b := range h.data {
		fmt.Fprintf(hex, "%02x", b)
	}
	return hex.String()
}
