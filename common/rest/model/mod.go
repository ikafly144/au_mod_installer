package model

import (
	"time"
)

type ModListResult struct {
	IDs    []string `json:"ids"`
	NextID string   `json:"next_id,omitempty"`
}

type ModDetails struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`

	LatestVersionID string `json:"latest_version,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	IconURL        string   `json:"icon_url,omitempty"`
	PackageURL     string   `json:"package_url,omitempty"`
	TotalDownloads int      `json:"total_downloads,omitempty"`
	RatingScore    int      `json:"rating_score,omitempty"`
	Categories     []string `json:"categories,omitempty"`
	IsPinned       bool     `json:"is_pinned,omitempty"`
	IsDeprecated   bool     `json:"is_deprecated,omitempty"`
}

type ModVersionListResult struct {
	IDs []string `json:"ids"`
}

type ModVersionDetails struct {
	VersionID string `json:"version_id"`
	ModID     string `json:"mod_id"`

	GameVersions []string `json:"game_versions,omitempty"`

	Files        []ModVersionFile       `json:"files,omitempty"`
	Dependencies []ModVersionDependency `json:"dependencies,omitempty"`
	Features     map[string]any         `json:"features,omitempty"`

	DownloadURL string `json:"download_url,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
	IconURL     string `json:"icon_url,omitempty"`
	Description string `json:"description,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ModVersionFile struct {
	ID string `json:"id"`

	Filename    string      `json:"filename"`
	ContentType ContentType `json:"content_type"`
	Size        int64       `json:"size"`

	ExtractPath    string         `json:"extract_path,omitempty"`
	TargetPlatform TargetPlatform `json:"target_platform"`

	Hashes    map[string]string `json:"hashes,omitempty"`
	Downloads []string          `json:"downloads"`

	CreatedAt time.Time `json:"created_at"`
}

type ContentType string

const (
	ContentTypeArchive   ContentType = "archive"
	ContentTypePluginDll ContentType = "plugin_dll"
	ContentTypeBinary    ContentType = "binary"
)

type TargetPlatform string

const (
	TargetPlatformAny TargetPlatform = "any"
	TargetPlatformX64 TargetPlatform = "x64"
	TargetPlatformX86 TargetPlatform = "x86"
)

type ModVersionDependency struct {
	ModID          string         `json:"mod_id"`
	VersionID      string         `json:"version_id"`
	DependencyType DependencyType `json:"dependency_type"`
}

type DependencyType string

const (
	DependencyTypeRequired DependencyType = "required"
	DependencyTypeOptional DependencyType = "optional"
	DependencyTypeConflict DependencyType = "conflict"
	DependencyTypeEmbedded DependencyType = "embedded"
)
