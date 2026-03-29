package assetstools

import (
	"fmt"
)

type ValueField struct {
	Template *TemplateField
	Children []*ValueField
	Value    any
}

func (f *TemplateField) MakeValue(reader *AssetsFileReader) (*ValueField, error) {
	if f == nil {
		return nil, fmt.Errorf("template field is nil")
	}
	field := &ValueField{Template: f}
	return readValueField(reader, field)
}

func (f *TemplateField) MakeValueAt(reader *AssetsFileReader, pos int64) (*ValueField, error) {
	if err := reader.SeekAt(pos); err != nil {
		return nil, err
	}
	return f.MakeValue(reader)
}

func readValueField(reader *AssetsFileReader, field *ValueField) (*ValueField, error) {
	template := field.Template

	if template.IsArray {
		if len(template.Children) != 2 {
			return nil, fmt.Errorf("array field %q should have 2 children but has %d", template.Name, len(template.Children))
		}
		size, err := reader.ReadInt32()
		if err != nil {
			return nil, err
		}
		if size < 0 {
			return nil, fmt.Errorf("negative array size %d in field %q", size, template.Name)
		}

		if template.ValueType == AssetValueTypeByteArray {
			data, err := reader.ReadBytes(int(size))
			if err != nil {
				return nil, err
			}
			field.Children = []*ValueField{}
			field.Value = data
			if template.IsAligned {
				if err := reader.Align(); err != nil {
					return nil, err
				}
			}
			return field, nil
		}

		field.Children = make([]*ValueField, size)
		itemTemplate := template.Children[1]
		for i := 0; i < int(size); i++ {
			itemField := &ValueField{Template: itemTemplate}
			value, err := readValueField(reader, itemField)
			if err != nil {
				return nil, err
			}
			field.Children[i] = value
		}
		field.Value = int(size)
		if template.IsAligned {
			if err := reader.Align(); err != nil {
				return nil, err
			}
		}
		return field, nil
	}

	if template.ValueType == AssetValueTypeNone {
		field.Children = make([]*ValueField, len(template.Children))
		for i, childTemplate := range template.Children {
			child := &ValueField{Template: childTemplate}
			value, err := readValueField(reader, child)
			if err != nil {
				return nil, err
			}
			field.Children[i] = value
		}
		if template.IsAligned {
			if err := reader.Align(); err != nil {
				return nil, err
			}
		}
		return field, nil
	}

	if len(template.Children) > 0 {
		return nil, fmt.Errorf("primitive field %q unexpectedly has %d children", template.Name, len(template.Children))
	}
	field.Children = []*ValueField{}

	switch template.ValueType {
	case AssetValueTypeString:
		length, err := reader.ReadInt32()
		if err != nil {
			return nil, err
		}
		if length < 0 {
			return nil, fmt.Errorf("negative string length %d in field %q", length, template.Name)
		}
		data, err := reader.ReadBytes(int(length))
		if err != nil {
			return nil, err
		}
		field.Value = string(data)
		if err := reader.Align(); err != nil {
			return nil, err
		}
	case AssetValueTypeInt8:
		v, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		field.Value = int8(v)
	case AssetValueTypeUInt8:
		v, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeBool:
		v, err := reader.ReadBoolean()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeInt16:
		v, err := reader.ReadInt16()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeUInt16:
		v, err := reader.ReadUint16()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeInt32:
		v, err := reader.ReadInt32()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeUInt32:
		v, err := reader.ReadUint32()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeInt64:
		v, err := reader.ReadInt64()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeUInt64:
		v, err := reader.ReadUint64()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeFloat32:
		v, err := reader.ReadFloat32()
		if err != nil {
			return nil, err
		}
		field.Value = v
	case AssetValueTypeFloat64:
		v, err := reader.ReadFloat64()
		if err != nil {
			return nil, err
		}
		field.Value = v
	default:
		return nil, fmt.Errorf("unsupported value type %v for field %q", template.ValueType, template.Name)
	}

	if template.IsAligned {
		if err := reader.Align(); err != nil {
			return nil, err
		}
	}
	return field, nil
}

func (f *ValueField) Child(name string) *ValueField {
	if f == nil {
		return nil
	}
	for _, child := range f.Children {
		if child.Template != nil && child.Template.Name == name {
			return child
		}
	}
	return nil
}

func (f *ValueField) Find(path ...string) *ValueField {
	cur := f
	for _, part := range path {
		if cur == nil {
			return nil
		}
		cur = cur.Child(part)
	}
	return cur
}

func (f *ValueField) AsString() (string, bool) {
	if f == nil {
		return "", false
	}
	s, ok := f.Value.(string)
	return s, ok
}
