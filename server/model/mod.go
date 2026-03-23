package model

import "time"

type ModDetails struct {
	ID          string `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Description string `gorm:"not null" json:"description"`
	Author      string `gorm:"not null" json:"author"`

	LatestVersionID string             `gorm:"index" json:"latest_version"`
	LatestVersion   *ModVersionDetails `gorm:"foreignKey:LatestVersionID" json:"latest_version_details"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ModVersionDetails struct {
	ID    string `gorm:"primaryKey" json:"id"`
	ModID string `gorm:"index" json:"mod_id"`

	Files []ModVersionFile `gorm:"foreignKey:VersionID" json:"files"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ModVersionFile struct {
	ID        string `gorm:"primaryKey" json:"id"`
	VersionID string `gorm:"index" json:"version_id"`

	Filename    string   `gorm:"not null" json:"filename"`
	ContentType FileType `gorm:"not null" json:"content_type"`
	Size        int64    `gorm:"not null" json:"size"`

	// Hashes is a map of hash algorithm to hash value, e.g. "sha256" -> "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	Hashes    map[string]string `gorm:"type:json" json:"hashes"`
	Downloads []string          `gorm:"type:json" json:"downloads"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type FileType string

const (
	FileTypeArchive   FileType = "archive"
	FileTypePluginDll FileType = "plugin_dll"
	FileTypeBinary    FileType = "binary"
)
