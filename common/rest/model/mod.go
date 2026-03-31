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

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ModVersionFile struct {
	ID string `json:"id"`

	Filename    string      `json:"filename"`
	ContentType ContentType `json:"content_type"`
	Size        int64       `json:"size"`

	// ContentTypeが `binary` か `plugin_dll` の場合、ExtractPathの位置に配置される。nullの場合は、`plugin_dll`は BepInEx/plugins に、`binary`はゲームのルートに配置される。
	// `archive` の場合は、アーカイブを展開した後のファイルの配置に影響する。ExtractPathがnullの場合、アーカイブ内のファイルはすべてゲームのルートに配置される。
	ExtractPath    string         `json:"extract_path,omitempty"`
	TargetPlatform TargetPlatform `json:"target_platform"`

	// Hashes is a map of hash algorithm to hash value, e.g. "sha256" -> "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	Hashes    map[string]string `json:"hashes"`
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
