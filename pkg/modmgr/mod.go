package modmgr

import (
	"iter"
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

type Mod struct {
	ID     string  `json:"id"`
	Type   ModType `json:"type,omitempty"`
	Name   string  `json:"name"`
	Author string  `json:"author"`

	LatestVersion string `json:"latest_version,omitempty"`
	Description   string `json:"description,omitempty"`
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
	ID            string                        `json:"id"`
	CreatedAt     time.Time                     `json:"created_at"`
	Dependencies  []ModDependency               `json:"dependencies,omitempty"`
	Mods          []ModPack                     `json:"mods,omitempty"`
	Files         []ModFile                     `json:"files,omitempty"`
	TargetVersion map[aumgr.LauncherType]string `json:"target_version,omitempty"`
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

type ModPack struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type ModFile struct {
	Compatible []aumgr.BinaryType `json:"compatible"`
	FileType   FileType           `json:"file_type"`
	Path       string             `json:"path,omitempty"`
	URL        string             `json:"url"`
}

type FileType string

const (
	FileTypeZip    FileType = "zip"
	FileTypeNormal FileType = "normal"
)

func (m ModVersion) IsCompatible(launcherType aumgr.LauncherType, binaryType aumgr.BinaryType, gameVersion string) bool {
	if version, ok := m.TargetVersion[launcherType]; ok && version != "" && version != gameVersion {
		return false
	}
	return m.CompatibleFilesCount(binaryType) > 0 || (len(m.Dependencies) > 0 && len(m.Files) == 0)
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
