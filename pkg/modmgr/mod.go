package modmgr

import (
	"iter"
	"slices"

	"github.com/ikafly144/modrepo/common/rest/model"
	"github.com/ikafly144/modrepo/pkg/repomgr"
)

type Mod struct {
	model.ModDetails
}

type ModType string

const (
	ModTypeMod     ModType = "mod"
	ModTypeLibrary ModType = "library"
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

func (m ModVersion) IsCompatible(launcherType repomgr.LauncherType, binaryType repomgr.BinaryType, gameVersion string) bool {
	if len(m.GameVersions) == 0 {
		return true
	}
	return slices.Contains(m.GameVersions, gameVersion)
}

func (m ModVersion) CompatibleFilesCount(binaryType repomgr.BinaryType) int {
	if len(m.Files) == 0 {
		return 1
	}
	return len(m.Files)
}

func (m ModVersion) Downloads(binaryType repomgr.BinaryType) iter.Seq[model.ModVersionFile] {
	return func(yield func(model.ModVersionFile) bool) {
		for _, file := range m.Files {
			if !yield(file) {
				return
			}
		}
	}
}

func (m ModVersion) HasFeature(feature string) bool {
	if m.Features == nil {
		return false
	}
	value, ok := m.Features[feature]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false" && v != "0"
	default:
		return true
	}
}
