package modmgr

import (
	"iter"
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

type Mod struct {
	ID        string    `json:"id"`                  // Mod unique ID
	Type      ModType   `json:"type,omitempty"`      // Mod type
	Name      string    `json:"name"`                // Mod name
	Author    string    `json:"author"`              // Author name
	Thumbnail string    `json:"thumbnail,omitempty"` // URL to thumbnail image (optional)
	Website   string    `json:"website,omitempty"`   // Mod website URL (optional)
	CreatedAt time.Time `json:"created_at"`          // Creation timestamp
	UpdatedAt time.Time `json:"updated_at"`          // Last update timestamp

	LatestVersion string `json:"latest_version,omitempty"` // Latest version ID (optional)
	Description   string `json:"description,omitempty"`    // Mod description
}

type ModType string

const (
	ModTypeMod     ModType = "mod"
	ModTypeLibrary ModType = "library"
	// Deprecated: should not be used anymore
	// ModPack will be represented as profile.Profile now
	ModTypeModPack ModType = "modpack"
)

func (mt ModType) IsVisible() bool {
	switch mt {
	case ModTypeMod, ModTypeModPack:
		return true
	default:
		return false
	}
}

type ModVersion struct {
	ID           string          `json:"id"`
	ModID        string          `json:"mod_id"`
	CreatedAt    time.Time       `json:"created_at"`
	Dependencies []ModDependency `json:"dependencies,omitempty"`
	Files        []ModFile       `json:"files,omitempty"`
	GameVersions []string        `json:"game_versions,omitempty"`

	// Deprecated: use profile.Profile to represent mod packs now
	Mods []ModPack `json:"mods,omitempty"`
}

type ModDependency struct {
	ID      string            `json:"id"`
	Version string            `json:"version,omitempty"`
	Type    ModDependencyType `json:"type"`
}

type ModDependencyType string

const (
	ModDependencyTypeRequired ModDependencyType = "required"
	ModDependencyTypeOptional ModDependencyType = "optional"
	ModDependencyTypeConflict ModDependencyType = "conflict"
	ModDependencyTypeEmbedded ModDependencyType = "embedded"
)

// Deprecated: use profile.Profile to represent mod packs now
type ModPack struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type ModFile struct {
	Compatible []aumgr.BinaryType `json:"compatible"`
	FileType   FileType           `json:"file_type"`
	// When FileType is Normal or Plugin, Path is used.
	Path string `json:"path,omitempty"`
	URL  string `json:"url"`
}

type FileType string

const (
	FileTypeZip    FileType = "zip"
	FileTypeNormal FileType = "normal"
	FileTypePlugin FileType = "plugin"
)

func (m ModVersion) IsCompatible(launcherType aumgr.LauncherType, binaryType aumgr.BinaryType, gameVersion string) bool {
	if m.CompatibleFilesCount(binaryType) == 0 && (len(m.Mods) == 0 && len(m.Files) > 0) {
		return false
	}
	// Check game version compatibility
	supported := false
	// Check deprecated TargetVersion first for backward compatibility
	for _, v := range m.GameVersions {
		if v == gameVersion {
			supported = true
			break
		}
	}
	return supported || len(m.GameVersions) == 0
}

func (m ModVersion) CompatibleFilesCount(binaryType aumgr.BinaryType) int {
	var count int
	for _, file := range m.Files {
		for _, t := range file.Compatible {
			if t == binaryType {
				count++
				break
			}
		}
	}
	return count
}

func (m ModVersion) Downloads(binaryType aumgr.BinaryType) iter.Seq[ModFile] {
	return func(yield func(ModFile) bool) {
		for _, file := range m.Files {
			for _, t := range file.Compatible {
				if t == binaryType {
					if !yield(file) {
						return
					}
					break
				}
			}
		}
	}
}
