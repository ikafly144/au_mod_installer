package gorm

import (
	"errors"

	"gorm.io/gorm"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func (r *GormRepository) CreateModSubmission(sub *model.ModSubmission) (string, error) {
	result := r.db.Create(sub)
	if result.Error != nil {
		return "", result.Error
	}
	return sub.ID, nil
}

func (r *GormRepository) GetModSubmission(id string) (*model.ModSubmission, error) {
	var sub model.ModSubmission
	result := r.db.First(&sub, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &sub, nil
}

func (r *GormRepository) GetModSubmissionsByModID(modID string) ([]model.ModSubmission, error) {
	var list []model.ModSubmission
	result := r.db.Where("mod_id = ?", modID).Order("created_at DESC").Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	return list, nil
}

func (r *GormRepository) GetPendingModSubmissions() ([]model.ModSubmission, error) {
	var list []model.ModSubmission
	result := r.db.Where("status = ?", model.SubmissionStatusPendingReview).Order("created_at ASC").Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	return list, nil
}

func (r *GormRepository) GetUserModSubmissions(submitterID string) ([]model.ModSubmission, error) {
	var list []model.ModSubmission
	result := r.db.Where("submitter_id = ?", submitterID).Order("created_at DESC").Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	return list, nil
}

func (r *GormRepository) UpdateModSubmission(sub *model.ModSubmission) error {
	result := r.db.Save(sub)
	return result.Error
}

func (r *GormRepository) DeleteModSubmission(id string) error {
	result := r.db.Delete(&model.ModSubmission{}, "id = ?", id)
	return result.Error
}

func (r *GormRepository) CreateVersionSubmission(sub *model.VersionSubmission) (string, error) {
	result := r.db.Create(sub)
	if result.Error != nil {
		return "", result.Error
	}
	return sub.ID, nil
}

func (r *GormRepository) GetVersionSubmission(id string) (*model.VersionSubmission, error) {
	var sub model.VersionSubmission
	result := r.db.First(&sub, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &sub, nil
}

func (r *GormRepository) GetPendingVersionSubmissions(modID string) ([]model.VersionSubmission, error) {
	var list []model.VersionSubmission
	query := r.db.Where("status = ?", model.SubmissionStatusPendingReview)
	if modID != "" {
		query = query.Where("mod_id = ?", modID)
	}
	result := query.Order("created_at ASC").Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	return list, nil
}

func (r *GormRepository) GetUserVersionSubmissions(submitterID string) ([]model.VersionSubmission, error) {
	var list []model.VersionSubmission
	result := r.db.Where("submitter_id = ?", submitterID).Order("created_at DESC").Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	return list, nil
}

func (r *GormRepository) UpdateVersionSubmission(sub *model.VersionSubmission) error {
	result := r.db.Save(sub)
	return result.Error
}

func (r *GormRepository) DeleteVersionSubmission(id string) error {
	result := r.db.Delete(&model.VersionSubmission{}, "id = ?", id)
	return result.Error
}

// GetModsByOwner returns all mods where the user is owner
func (r *GormRepository) GetModsByOwner(ownerDiscordID string) ([]model.ModDetails, error) {
	var mods []model.ModDetails
	result := r.db.Where("owner_discord_id = ?", ownerDiscordID).Order("created_at DESC").Find(&mods)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return []model.ModDetails{}, nil
		}
		return nil, result.Error
	}
	return mods, nil
}
