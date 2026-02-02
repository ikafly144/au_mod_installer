package repository

import (
	"context"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/model"
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

	// GetAllMods retrieves all mods
	GetAllMods(ctx context.Context) ([]modmgr.Mod, error)

	// DeleteMod deletes a mod by ID
	DeleteMod(ctx context.Context, modID string) error

	// GetAllModVersions retrieves all versions of a mod
	GetAllModVersions(ctx context.Context, modID string) ([]modmgr.ModVersion, error)

	// DeleteVersion deletes a specific version of a mod
	DeleteVersion(ctx context.Context, modID, versionID string) error
}

// UserRepository defines the interface for user data storage
type UserRepository interface {
	// GetUser retrieves a user by ID
	GetUser(ctx context.Context, id int) (*model.User, error)

	// GetUserByUsername retrieves a user by username
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// CreateUser stores a new user
	CreateUser(ctx context.Context, user model.User) error

	// UpdateUser updates an existing user
	UpdateUser(ctx context.Context, user model.User) error

	// DeleteUser deletes a user by ID
	DeleteUser(ctx context.Context, id int) error
}
