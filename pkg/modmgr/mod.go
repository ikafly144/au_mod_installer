package modmgr

import (
	"au_mod_installer/pkg/aumgr"
	"iter"
)

type Mod struct {
	Hidden        bool                          `json:"hidden,omitempty"`
	Name          string                        `json:"name"`
	Version       string                        `json:"version"`
	Author        string                        `json:"author"`
	Dependencies  []ModDependency               `json:"dependencies,omitempty"`
	Files         []ModFile                     `json:"files"`
	TargetVersion map[aumgr.LauncherType]string `json:"target_version,omitempty"`
}

type ModDependency struct {
	Name string            `json:"name"`
	Type ModDependencyType `json:"type"`
}

type ModDependencyType string

const (
	ModDependencyTypeRequired ModDependencyType = "required"
	ModDependencyTypeOptional ModDependencyType = "optional"
	ModDependencyTypeConflict ModDependencyType = "conflict"
)

type ModFile struct {
	Compatible []aumgr.LauncherType `json:"compatible"`
	FileType   FileType             `json:"file_type"`
	Path       string               `json:"path,omitempty"`
	URL        string               `json:"url"`
}

type FileType string

const (
	FileTypeZip    FileType = "zip"
	FileTypeNormal FileType = "normal"
)

func (m Mod) IsCompatible(launcherType aumgr.LauncherType, gameVersion string) bool {
	if version, ok := m.TargetVersion[launcherType]; !ok || version != gameVersion {
		return false
	}
	return m.CompatibleFilesCount(launcherType) > 0 || (len(m.Dependencies) > 0 && len(m.Files) == 0)
}

func (m Mod) CompatibleFilesCount(launcherType aumgr.LauncherType) int {
	var count int
	for _, file := range m.Files {
		for _, t := range file.Compatible {
			if t == launcherType {
				count++
				break
			}
		}
	}
	return count
}

func (m Mod) Downloads(launcherType aumgr.LauncherType) iter.Seq[ModFile] {
	return func(yield func(ModFile) bool) {
		for _, file := range m.Files {
			for _, t := range file.Compatible {
				if t == launcherType {
					if !yield(file) {
						return
					}
					break
				}
			}
		}
	}
}
