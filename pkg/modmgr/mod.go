package modmgr

import (
	"iter"
	"slices"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

type Mod struct {
	model.ModDetails
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
	model.ModVersionDetails
}

// Deprecated: use model.ModVersionDependency instead
type ModDependency struct {
	ID      string            `json:"id"`
	Version string            `json:"version,omitempty"`
	Type    ModDependencyType `json:"type"`
}

type ModDependencyType = model.DependencyType

const (
	ModDependencyTypeRequired ModDependencyType = model.DependencyTypeRequired
	ModDependencyTypeOptional ModDependencyType = model.DependencyTypeOptional
	ModDependencyTypeConflict ModDependencyType = model.DependencyTypeConflict
	ModDependencyTypeEmbedded ModDependencyType = model.DependencyTypeEmbedded
)

// Deprecated: use profile.Profile to represent mod packs now
type ModPack struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

// Deprecated: use model.ModVersionFile instead
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
	if m.CompatibleFilesCount(binaryType) == 0 && len(m.Files) > 0 {
		return false
	}
	// Check game version compatibility
	supported := slices.Contains(m.GameVersions, gameVersion)
	return supported || len(m.GameVersions) == 0
}

func (m ModVersion) CompatibleFilesCount(binaryType aumgr.BinaryType) int {
	var count int
	for _, file := range m.Files {
		if binaryType.IsCompatibleWith(file.TargetPlatform) {
			count++
		}
	}
	return count
}

func (m ModVersion) Downloads(binaryType aumgr.BinaryType) iter.Seq[model.ModVersionFile] {
	return func(yield func(model.ModVersionFile) bool) {
		for _, file := range m.Files {
			if binaryType.IsCompatibleWith(file.TargetPlatform) {
				if !yield(file) {
					return
				}
				break
			}
		}
	}
}
