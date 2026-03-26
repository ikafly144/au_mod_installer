package gorm

import (
	"gorm.io/gorm"

	"github.com/ikafly144/au_mod_installer/server/model"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.ModDetails{}, &model.ModVersionFile{}, &model.ModVersionDetails{}); err != nil {
		return err
	}
	return nil
}

func (r *GormRepository) CreateMod(details *model.ModDetails) (string, error) {
	result := r.db.Create(details)
	if result.Error != nil {
		return "", result.Error
	}
	return details.ID, nil
}

func (r *GormRepository) GetModIds(after string, limit int) ([]string, string, error) {
	var ids []string
	var next string
	result := r.db.Model(&model.ModDetails{}).
		Scopes(func(db *gorm.DB) *gorm.DB {
			db.Order("created_at DESC")
			if after != "" {
				db = db.Where("id > ?", after)
			}
			return db.Limit(limit)
		}).
		Pluck("ID", &ids)
	if result.Error != nil {
		return nil, "", result.Error
	}

	if len(ids) > 0 {
		next = ids[len(ids)-1]
	}

	return ids, next, nil
}

func (r *GormRepository) GetModDetails(modID string) (*model.ModDetails, error) {
	var mod model.ModDetails
	result := r.db.First(&mod, "id = ?", modID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &mod, nil
}

func (r *GormRepository) CreateModVersion(modID string, details *model.ModVersionDetails) (string, error) {
	details.ModID = &modID
	result := r.db.Create(details)
	if result.Error != nil {
		return "", result.Error
	}
	return details.ID, nil
}

func (r *GormRepository) GetModVersionIds(modID string) ([]string, error) {
	var ids []string
	result := r.db.Model(&model.ModVersionDetails{}).Where("mod_id = ?", modID).Pluck("ID", &ids)
	if result.Error != nil {
		return nil, result.Error
	}
	return ids, nil
}

func (r *GormRepository) GetModVersionDetails(modID, versionID string) (*model.ModVersionDetails, error) {
	var version model.ModVersionDetails
	result := r.db.Preload("Files").First(&version, "mod_id = ? AND id = ?", modID, versionID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &version, nil
}

func (r *GormRepository) UpdateMod(modID string, details *model.ModDetails) error {
	result := r.db.Model(&model.ModDetails{}).Where("id = ?", modID).Updates(details)
	return result.Error
}

func (r *GormRepository) UpdateModFields(modID string, updates map[string]any) error {
	result := r.db.Model(&model.ModDetails{}).Where("id = ?", modID).Updates(updates)
	return result.Error
}

func (r *GormRepository) UpdateModVersion(modID, versionID string, details *model.ModVersionDetails) error {
	result := r.db.Model(&model.ModVersionDetails{}).Where("mod_id = ? AND id = ?", modID, versionID).Updates(details)
	return result.Error
}

func (r *GormRepository) UpdateModVersionFields(modID, versionID string, updates map[string]any) error {
	result := r.db.Model(&model.ModVersionDetails{}).Where("mod_id = ? AND id = ?", modID, versionID).Updates(updates)
	return result.Error
}

func (r *GormRepository) DeleteMod(modID string) error {
	mod, err := r.GetModDetails(modID)
	if err != nil {
		return err
	}
	result := r.db.Select("LatestVersion").Delete(&mod)
	return result.Error
}

func (r *GormRepository) DeleteModVersion(modID, versionID string) error {
	version, err := r.GetModVersionDetails(modID, versionID)
	if err != nil {
		return err
	}
	result := r.db.Select("Files").Delete(&version)
	return result.Error
}
