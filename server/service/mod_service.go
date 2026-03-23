package service

import (
	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
)

type ModService struct {
	repo repository.ModRepository
}

func NewModService(repo repository.ModRepository) *ModService {
	return &ModService{repo: repo}
}

func (s *ModService) GetModIds(after string, limit int) ([]string, string, error) {
	switch {
	case limit <= 0:
		limit = 20
	case limit > 100:
		limit = 100
	}
	return s.repo.GetModIds(after, limit)
}

func (s *ModService) GetModDetails(modID string) (*model.ModDetails, error) {
	return s.repo.GetModDetails(modID)
}

func (s *ModService) GetModVersionIds(modID string) ([]string, error) {
	return s.repo.GetModVersionIds(modID)
}

func (s *ModService) GetModVersionDetails(modID, versionID string) (*model.ModVersionDetails, error) {
	return s.repo.GetModVersionDetails(modID, versionID)
}
