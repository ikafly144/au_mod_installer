package service

import (
	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
)

type ModService struct {
	repo      repository.ModRepository
	shareGame *shareGameManager
}

func NewModService(repo repository.ModRepository) *ModService {
	return &ModService{
		repo:      repo,
		shareGame: newShareGameManager(),
	}
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

func (s *ModService) CreateSharedGame(ip string, req restcommon.ShareGameRequest) (*restcommon.ShareGameResponse, error) {
	return s.shareGame.create(ip, req)
}

func (s *ModService) DeleteSharedGame(sessionID, hostKey string) error {
	return s.shareGame.delete(sessionID, hostKey)
}

func (s *ModService) GetJoinGameDownload(sessionID string) (*restcommon.JoinGameDownloadResponse, error) {
	return s.shareGame.getDownload(sessionID)
}

func (s *ModService) GetJoinGameMeta(sessionID string) (*restcommon.RoomInfo, error) {
	session, err := s.shareGame.getSessionMeta(sessionID)
	if err != nil {
		return nil, err
	}
	room := session.Room
	return &room, nil
}
