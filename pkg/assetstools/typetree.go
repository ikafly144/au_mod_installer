package assetstools

import (
	"errors"
	"fmt"
	"strings"
)

type TypeTreeNodeTypeFlags byte

const (
	TypeTreeNodeFlagArray TypeTreeNodeTypeFlags = 1
)

func (f TypeTreeNodeTypeFlags) Has(flag TypeTreeNodeTypeFlags) bool {
	return f&flag != 0
}

type TypeTreeNode struct {
	Version       uint16
	Level         byte
	TypeFlags     TypeTreeNodeTypeFlags
	TypeStrOffset uint32
	NameStrOffset uint32
	ByteSize      int32
	Index         uint32
	MetaFlags     uint32
	RefTypeHash   uint64
}

func (n *TypeTreeNode) Read(r *AssetsFileReader, version uint32) error {
	var err error
	if n.Version, err = r.ReadUint16(); err != nil {
		return err
	}
	if n.Level, err = r.ReadByte(); err != nil {
		return err
	}
	flags, err := r.ReadByte()
	if err != nil {
		return err
	}
	n.TypeFlags = TypeTreeNodeTypeFlags(flags)
	if n.TypeStrOffset, err = r.ReadUint32(); err != nil {
		return err
	}
	if n.NameStrOffset, err = r.ReadUint32(); err != nil {
		return err
	}
	if n.ByteSize, err = r.ReadInt32(); err != nil {
		return err
	}
	if n.Index, err = r.ReadUint32(); err != nil {
		return err
	}
	if n.MetaFlags, err = r.ReadUint32(); err != nil {
		return err
	}
	if version >= 18 {
		if n.RefTypeHash, err = r.ReadUint64(); err != nil {
			return err
		}
	}
	return nil
}

func (n *TypeTreeNode) GetTypeString(localStringBuffer []byte) string {
	return resolveTypeTreeString(n.TypeStrOffset, localStringBuffer)
}

func (n *TypeTreeNode) GetNameString(localStringBuffer []byte) string {
	return resolveTypeTreeString(n.NameStrOffset, localStringBuffer)
}

type TypeTreeType struct {
	TypeID          int32
	IsStrippedType  bool
	ScriptTypeIndex int16
	Nodes           []*TypeTreeNode
	StringBuffer    []byte
}

func (t *TypeTreeType) Read(r *AssetsFileReader, version uint32, hasTypeTree bool, isRefType bool) error {
	typeID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	t.TypeID = typeID

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

	needsScriptHash := (version < 17 && t.TypeID < 0) ||
		(version >= 17 && t.TypeID == int32(ClassIDMonoBehaviour)) ||
		(isRefType && t.ScriptTypeIndex != -1)
	if needsScriptHash {
		if _, err := r.ReadBytes(16); err != nil {
			return err
		}
	}
	if _, err := r.ReadBytes(16); err != nil {
		return err
	}

	if !hasTypeTree {
		return nil
	}

	nodeCount, err := r.ReadInt32()
	if err != nil {
		return err
	}
	stringBufLen, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if nodeCount < 0 || stringBufLen < 0 {
		return fmt.Errorf("invalid type tree section sizes: nodes=%d strings=%d", nodeCount, stringBufLen)
	}

	t.Nodes = make([]*TypeTreeNode, nodeCount)
	for i := 0; i < int(nodeCount); i++ {
		node := &TypeTreeNode{}
		if err := node.Read(r, version); err != nil {
			return err
		}
		t.Nodes[i] = node
	}
	stringBuffer, err := r.ReadBytes(int(stringBufLen))
	if err != nil {
		return err
	}
	t.StringBuffer = stringBuffer

	if version >= 21 {
		if !isRefType {
			dependenciesCount, err := r.ReadInt32()
			if err != nil {
				return err
			}
			if dependenciesCount < 0 {
				return fmt.Errorf("invalid dependencies count: %d", dependenciesCount)
			}
			for i := 0; i < int(dependenciesCount); i++ {
				if _, err := r.ReadInt32(); err != nil {
					return err
				}
			}
		} else {
			if _, err := r.ReadNullTerminatedString(); err != nil {
				return err
			}
			if _, err := r.ReadNullTerminatedString(); err != nil {
				return err
			}
			if _, err := r.ReadNullTerminatedString(); err != nil {
				return err
			}
		}
	}

	return nil
}

func resolveTypeTreeString(offset uint32, localStringBuffer []byte) string {
	const commonFlag = 0x80000000
	if offset&commonFlag != 0 {
		return readNullTerminatedFrom(commonTypeTreeStringData, int(offset-commonFlag))
	}
	return readNullTerminatedFrom(localStringBuffer, int(offset))
}

func readNullTerminatedFrom(b []byte, start int) string {
	if start < 0 || start >= len(b) {
		return ""
	}
	end := start
	for end < len(b) && b[end] != 0 {
		end++
	}
	return string(b[start:end])
}

const commonTypeTreeStrings = "AABB\x00AnimationClip\x00AnimationCurve\x00AnimationState\x00Array\x00Base\x00BitField\x00bitset\x00bool\x00char\x00ColorRGBA\x00Component\x00data\x00deque\x00double\x00dynamic_array\x00FastPropertyName\x00first\x00float\x00Font\x00GameObject\x00Generic Mono\x00GradientNEW\x00GUID\x00GUIStyle\x00int\x00list\x00long long\x00map\x00Matrix4x4f\x00MdFour\x00MonoBehaviour\x00MonoScript\x00m_ByteSize\x00m_Curve\x00m_EditorClassIdentifier\x00m_EditorHideFlags\x00m_Enabled\x00m_ExtensionPtr\x00m_GameObject\x00m_Index\x00m_IsArray\x00m_IsStatic\x00m_MetaFlag\x00m_Name\x00m_ObjectHideFlags\x00m_PrefabInternal\x00m_PrefabParentObject\x00m_Script\x00m_StaticEditorFlags\x00m_Type\x00m_Version\x00Object\x00pair\x00PPtr<Component>\x00PPtr<GameObject>\x00PPtr<Material>\x00PPtr<MonoBehaviour>\x00PPtr<MonoScript>\x00PPtr<Object>\x00PPtr<Prefab>\x00PPtr<Sprite>\x00PPtr<TextAsset>\x00PPtr<Texture>\x00PPtr<Texture2D>\x00PPtr<Transform>\x00Prefab\x00Quaternionf\x00Rectf\x00RectInt\x00RectOffset\x00second\x00set\x00short\x00size\x00SInt16\x00SInt32\x00SInt64\x00SInt8\x00staticvector\x00string\x00TextAsset\x00TextMesh\x00Texture\x00Texture2D\x00Transform\x00TypelessData\x00UInt16\x00UInt32\x00UInt64\x00UInt8\x00unsigned int\x00unsigned long long\x00unsigned short\x00vector\x00Vector2f\x00Vector3f\x00Vector4f\x00m_ScriptingClassIdentifier\x00Gradient\x00Type*\x00int2_storage\x00int3_storage\x00BoundsInt\x00m_CorrespondingSourceObject\x00m_PrefabInstance\x00m_PrefabAsset\x00FileSize\x00Hash128\x00"

var commonTypeTreeStringData = []byte(commonTypeTreeStrings)

type TemplateField struct {
	Name      string
	Type      string
	ValueType AssetValueType
	IsArray   bool
	IsAligned bool
	Children  []*TemplateField
}

func NewTemplateFieldFromTypeTree(tt *TypeTreeType) (*TemplateField, error) {
	if tt == nil || len(tt.Nodes) == 0 {
		return nil, errors.New("type tree has no nodes")
	}
	idx := 0
	return readTemplateField(tt, &idx), nil
}

func readTemplateField(tt *TypeTreeType, idx *int) *TemplateField {
	node := tt.Nodes[*idx]
	field := &TemplateField{
		Name:      node.GetNameString(tt.StringBuffer),
		Type:      node.GetTypeString(tt.StringBuffer),
		ValueType: ValueTypeByTypeName(node.GetTypeString(tt.StringBuffer)),
		IsArray:   node.TypeFlags.Has(TypeTreeNodeFlagArray),
		IsAligned: (node.MetaFlags & 0x4000) != 0,
		Children:  make([]*TemplateField, 0),
	}

	for *idx = *idx + 1; *idx < len(tt.Nodes); *idx++ {
		next := tt.Nodes[*idx]
		if next.Level <= node.Level {
			*idx = *idx - 1
			break
		}
		field.Children = append(field.Children, readTemplateField(tt, idx))
	}

	if field.ValueType == AssetValueTypeString && len(field.Children) > 0 {
		if !field.Children[0].IsArray && field.Children[0].ValueType != AssetValueTypeNone {
			field.Type = "_string"
			field.ValueType = AssetValueTypeNone
		}
	}

	if field.IsArray && len(field.Children) >= 2 {
		if field.Children[1].ValueType == AssetValueTypeUInt8 {
			field.ValueType = AssetValueTypeByteArray
		} else {
			field.ValueType = AssetValueTypeArray
		}
	}

	return field
}

type AssetValueType int

const (
	AssetValueTypeNone AssetValueType = iota
	AssetValueTypeString
	AssetValueTypeInt8
	AssetValueTypeUInt8
	AssetValueTypeInt16
	AssetValueTypeUInt16
	AssetValueTypeInt32
	AssetValueTypeUInt32
	AssetValueTypeInt64
	AssetValueTypeUInt64
	AssetValueTypeFloat32
	AssetValueTypeFloat64
	AssetValueTypeBool
	AssetValueTypeArray
	AssetValueTypeByteArray
)

func ValueTypeByTypeName(typ string) AssetValueType {
	switch strings.ToLower(typ) {
	case "string":
		return AssetValueTypeString
	case "sint8", "sbyte":
		return AssetValueTypeInt8
	case "uint8", "char", "byte":
		return AssetValueTypeUInt8
	case "sint16", "short":
		return AssetValueTypeInt16
	case "uint16", "unsigned short", "ushort":
		return AssetValueTypeUInt16
	case "sint32", "int", "type*":
		return AssetValueTypeInt32
	case "uint32", "unsigned int", "uint":
		return AssetValueTypeUInt32
	case "sint64", "long":
		return AssetValueTypeInt64
	case "uint64", "unsigned long", "ulong", "filesize":
		return AssetValueTypeUInt64
	case "single", "float":
		return AssetValueTypeFloat32
	case "double":
		return AssetValueTypeFloat64
	case "bool":
		return AssetValueTypeBool
	default:
		return AssetValueTypeNone
	}
}
