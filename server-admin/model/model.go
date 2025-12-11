package model

import (
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

// Mod represents a mod entity
type Mod = modmgr.Mod

// ModVersion represents a mod version entity
type ModVersion = modmgr.ModVersion

// ModPack represents a mod pack entity
type ModPack = modmgr.ModPack

// ModDependency represents a mod dependency
type ModDependency = modmgr.ModDependency

// ModFile represents a mod file
type ModFile = modmgr.ModFile

// ModWithVersions represents a mod with its versions (for import/export)
type ModWithVersions struct {
	Mod
	Versions []ModVersion `json:"versions"`
}
