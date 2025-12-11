package model

import "time"

// Mod represents a mod entity
type Mod struct {
	ID              string `json:"id"`
	Type            string `json:"type,omitempty"`
	Name            string `json:"name"`
	Author          string `json:"author"`
	LatestVersionID string `json:"latest_version_id,omitempty"`
	Description     string `json:"description,omitempty"`
}

// ModVersion represents a mod version entity
type ModVersion struct {
	ID            string            `json:"id"`
	CreatedAt     time.Time         `json:"created_at"`
	Dependencies  []ModDependency   `json:"dependencies,omitempty"`
	Mods          []string          `json:"mods,omitempty"`
	Files         []ModFile         `json:"files,omitempty"`
	TargetVersion map[string]string `json:"target_version,omitempty"`
}

// ModDependency represents a mod dependency
type ModDependency struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ModFile represents a mod file
type ModFile struct {
	Compatible []string `json:"compatible"`
	FileType   string   `json:"file_type"`
	Path       string   `json:"path,omitempty"`
	URL        string   `json:"url"`
}

// ModWithVersions represents a mod with its versions (for import/export)
type ModWithVersions struct {
	Mod
	Versions []ModVersion `json:"versions"`
}
