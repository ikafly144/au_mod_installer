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
	return json.Unmarshal(b, a)
}

type ModDetails struct {
	ID          string `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Description string `gorm:"not null" json:"description"`
	Author      string `gorm:"not null" json:"author"`

	LatestVersionID string             `gorm:"index;default:null" json:"latest_version"`
	LatestVersion   *ModVersionDetails `gorm:"foreignKey:LatestVersionID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ModVersionDetails struct {
	ID    string `gorm:"primaryKey" json:"id"`
	ModID string `gorm:"index" json:"mod_id"`

	Files        []ModVersionFile       `gorm:"foreignKey:VersionID" json:"files,omitempty"`
	Dependencies []ModVersionDependency `gorm:"type:json" json:"dependencies,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ModVersionFile struct {
	ID        string `gorm:"primaryKey" json:"id"`
	VersionID string `gorm:"index" json:"-"`

	Filename    string   `gorm:"not null" json:"filename"`
	ContentType FileType `gorm:"not null" json:"content_type"`
	Size        int64    `gorm:"not null" json:"size"`

	// ContentTypeが `binary` か `plugin_dll` の場合、ExtractPathの位置に配置される。nullの場合は、`plugin_dll`は BepInEx/plugins に、`binary`はゲームのルートに配置される。
	// `archive` の場合は、アーカイブを展開した後のファイルの配置に影響する。ExtractPathがnullの場合、アーカイブ内のファイルはすべてゲームのルートに配置される。
	ExtractPath    *string        `gorm:"default:null" json:"extract_path,omitempty"`
	TargetPlatform TargetPlatform `gorm:"not null;default:'any'" json:"target_platform"`

	// Hashes is a map of hash algorithm to hash value, e.g. "sha256" -> "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
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
	// Any platform (default)
	TargetPlatformAny TargetPlatform = "any"
	// Epic/MSStore
	TargetPlatformX64 TargetPlatform = "x64"
	// Steam/Itch
	TargetPlatformX86 TargetPlatform = "x86"
	// Android
	TargetPlatformAArch64 TargetPlatform = "aarch64"
)

type ModVersionDependency struct {
	ModID string `json:"mod_id"`
	// if `any` is specified, it means any version of the mod is acceptable
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
