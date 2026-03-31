package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type StringMap map[string]string

func (m StringMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

func (m *StringMap) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		// PostgreSQL might return string
		s, ok := value.(string)
		if !ok {
			return errors.New("type assertion to []byte or string failed")
		}
		b = []byte(s)
	}
	return json.Unmarshal(b, m)
}

type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *StringArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		s, ok := value.(string)
		if !ok {
			return errors.New("type assertion to []byte or string failed")
		}
		b = []byte(s)
	}

	// Postgres native array or invalid old JSON might appear as {} instead of []
	if len(b) > 0 && b[0] == '{' {
		*a = StringArray{}
		return nil
	}

	return json.Unmarshal(b, a)
}

type ModDetails struct {
	ID           string  `gorm:"primaryKey" json:"id"`
	Name         string  `gorm:"not null" json:"name"`
	Description  string  `gorm:"not null" json:"description"`
	Author       string  `gorm:"not null" json:"author"`
	ThumbnailURI *string `gorm:"default:null" json:"-"`

	LatestVersionID *string `gorm:"index;default:null;" json:"latest_version"`

	Versions []ModVersionDetails `gorm:"foreignKey:ModID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Files    []ModVersionFile    `gorm:"foreignKey:ModID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ModVersionDetails struct {
	ID    string `gorm:"primaryKey" json:"id"`
	ModID string `gorm:"primaryKey" json:"mod_id"`

	Files        []ModVersionFile `gorm:"foreignKey:VersionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"files,omitempty"`
	Dependencies DependencyArray  `gorm:"type:json" json:"dependencies,omitempty"`
	Features     Features         `gorm:"type:json" json:"features,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ModVersionFile struct {
	ID        string  `gorm:"primaryKey" json:"id"`
	ModID     *string `gorm:"index:idx_mod_version_file;not null" json:"-"`
	VersionID *string `gorm:"index:idx_mod_version_file;not null" json:"-"`

	Filename    string   `gorm:"not null" json:"filename"`
	ContentType FileType `gorm:"not null" json:"content_type"`
	Size        int64    `gorm:"not null" json:"size"`

	ExtractPath    *string        `gorm:"default:null" json:"extract_path,omitempty"`
	TargetPlatform TargetPlatform `gorm:"not null;default:'any'" json:"target_platform"`

	Hashes    StringMap   `gorm:"type:json" json:"hashes"`
	Downloads StringArray `gorm:"type:json" json:"downloads"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type FileType string

const (
	FileTypeArchive   FileType = "archive"
	FileTypePluginDll FileType = "plugin_dll"
	FileTypeBinary    FileType = "binary"
)

type TargetPlatform string

const (
	TargetPlatformAny     TargetPlatform = "any"
	TargetPlatformX64     TargetPlatform = "x64"
	TargetPlatformX86     TargetPlatform = "x86"
	TargetPlatformAArch64 TargetPlatform = "aarch64"
)

type DependencyArray []ModVersionDependency

func (a DependencyArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *DependencyArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		s, ok := value.(string)
		if !ok {
			return errors.New("type assertion to []byte or string failed")
		}
		b = []byte(s)
	}

	// Postgres or old data might have stored an empty object "{}" instead of "[]"
	if len(b) > 0 && b[0] == '{' {
		*a = DependencyArray{}
		return nil
	}

	return json.Unmarshal(b, a)
}

type ModVersionDependency struct {
	ModID          string         `json:"mod_id"`
	VersionID      string         `json:"version_id"`
	DependencyType DependencyType `json:"dependency_type"` // "required" or "optional"
}

type DependencyType string

const (
	DependencyTypeRequired DependencyType = "required"
	DependencyTypeOptional DependencyType = "optional"
	DependencyTypeConflict DependencyType = "conflict"
	DependencyTypeEmbedded DependencyType = "embedded"
)

type Features map[string]any

func (f Features) Value() (driver.Value, error) {
	if f == nil {
		return "{}", nil
	}
	return json.Marshal(f)
}

func (f *Features) Scan(value any) error {
	if value == nil {
		*f = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		s, ok := value.(string)
		if !ok {
			return errors.New("type assertion to []byte or string failed")
		}
		b = []byte(s)
	}
	return json.Unmarshal(b, f)
}
