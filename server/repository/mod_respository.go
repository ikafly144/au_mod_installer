package repository

import (
	"github.com/ikafly144/au_mod_installer/server/model"
)

type ModRepository interface {
	CreateMod(details *model.ModDetails) (string, error)
	CreateModVersion(modID string, details *model.ModVersionDetails) (string, error)

	GetModIds(next string, limit int) (ids []string, nextID string, err error)
	GetModDetails(modID string) (*model.ModDetails, error)
	GetModVersionIds(modID string) ([]string, error)
	GetModVersionDetails(modID, versionID string) (*model.ModVersionDetails, error)

	UpdateMod(modID string, details *model.ModDetails) error
	UpdateModVersion(modID, versionID string, details *model.ModVersionDetails) error

	DeleteMod(modID string) error
	DeleteModVersion(modID, versionID string) error
}
