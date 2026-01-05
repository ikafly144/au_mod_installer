package repository

import (
	"context"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type ModWithVersions struct {
	modmgr.Mod
	Versions []modmgr.ModVersion
}

// ModRepository defines the interface for mod data storage
type ModRepository interface {
	// GetMod retrieves a mod by ID
	GetMod(ctx context.Context, modID string) (*modmgr.Mod, error)

	// GetModList retrieves a list of mods with pagination
	GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error)

	// GetModVersion retrieves a specific version of a mod
	GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error)

	// GetModVersions retrieves all versions of a mod with pagination
	GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error)

	// SetMod stores a mod
	SetMod(ctx context.Context, mod modmgr.Mod) error

	// SetModVersion stores a mod version
	SetModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error

	// Close closes the repository connection
	Close()

	// Admin operations can be added here

	// Mod operations

	// GetAllMods retrieves all mods
	GetAllMods(ctx context.Context) ([]modmgr.Mod, error)

	// DeleteMod deletes a mod by ID
	DeleteMod(ctx context.Context, modID string) error

	// Version operations

	// GetAllModVersions retrieves all versions of a mod
	GetAllModVersions(ctx context.Context, modID string) ([]modmgr.ModVersion, error)

	// DeleteVersion deletes a specific version of a mod
	DeleteVersion(ctx context.Context, modID, versionID string) error
}
