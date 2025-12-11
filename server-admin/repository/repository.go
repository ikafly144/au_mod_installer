package repository

import (
	"context"

	"github.com/ikafly144/au_mod_installer/server-admin/model"
)

// Repository defines the interface for mod data storage
type Repository interface {
	// Mod operations
	GetMod(ctx context.Context, modID string) (*model.Mod, error)
	GetModList(ctx context.Context) ([]model.Mod, error)
	SetMod(ctx context.Context, mod model.Mod) error
	DeleteMod(ctx context.Context, modID string) error

	// Version operations
	GetVersion(ctx context.Context, modID, versionID string) (*model.ModVersion, error)
	GetVersionList(ctx context.Context, modID string) ([]model.ModVersion, error)
	SetVersion(ctx context.Context, modID string, version model.ModVersion) error
	DeleteVersion(ctx context.Context, modID, versionID string) error
}
