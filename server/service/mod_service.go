package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"os"
	"slices"
	"sort"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/repository"
)

var (
	ErrNotFound = errors.New("not found")
)

// ModService provides business logic for mod operations
type ModService struct {
	repo repository.ModRepository
}

// NewModServiceWithRepo creates a new ModService with a repository
func NewModServiceWithRepo(repo repository.ModRepository) *ModService {
	return &ModService{
		repo: repo,
	}
}

// GetModList retrieves a list of mods with pagination
func (s *ModService) GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error) {
	mods, err := s.repo.GetModList(ctx, limit, after, before)
	if err != nil {
		return nil, err
	}
	if len(mods) == 0 {
		return []modmgr.Mod{}, nil
	}
	return mods, nil
}

// GetMod retrieves a mod by ID
func (s *ModService) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	return s.repo.GetMod(ctx, modID)
}

// GetModVersions retrieves all versions of a mod with pagination
func (s *ModService) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	return s.repo.GetModVersions(ctx, modID, limit, after)
}

// GetModVersion retrieves a specific version of a mod
func (s *ModService) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	return s.repo.GetModVersion(ctx, modID, versionID)
}

// FileModService is a file-based implementation for backward compatibility
type FileModService struct {
	file         string
	modStore     map[string]modmgr.Mod
	versionStore map[string]map[string]modmgr.ModVersion
	modOrder     []string // Maintain insertion order for pagination
}

// NewModService creates a new file-based ModService (for backward compatibility)
func NewModService(file string) (*FileModService, error) {
	s := &FileModService{
		file: file,
	}
	if err := s.loadData(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileModService) loadData() error {
	source, err := os.Open(s.file)
	if err != nil {
		return err
	}
	defer source.Close()

	type fileMod struct {
		modmgr.Mod
		Versions []modmgr.ModVersion `json:"versions"`
	}
	var fileMods []fileMod
	if err = json.NewDecoder(source).Decode(&fileMods); err != nil {
		return err
	}

	s.modStore = make(map[string]modmgr.Mod)
	s.versionStore = make(map[string]map[string]modmgr.ModVersion)
	s.modOrder = make([]string, 0, len(fileMods))

	for _, m := range fileMods {
		s.modStore[m.ID] = m.Mod
		s.modOrder = append(s.modOrder, m.ID)

		if _, ok := s.versionStore[m.ID]; !ok {
			s.versionStore[m.ID] = make(map[string]modmgr.ModVersion)
		}
		for _, v := range m.Versions {
			s.versionStore[m.ID][v.ID] = v
		}
	}

	slog.Info("mods loaded from file", "file", s.file, "mods", s.modStore, "versions", s.versionStore)

	return nil
}

func (s *FileModService) GetModList(limit int, after string, before string) ([]modmgr.Mod, error) {
	var mods []modmgr.Mod

	startIndex := 0
	endIndex := len(s.modOrder)

	if after != "" {
		for i, id := range s.modOrder {
			if id == after {
				startIndex = i + 1
				break
			}
		}
	}

	if before != "" {
		for i, id := range s.modOrder {
			if id == before {
				endIndex = i
				break
			}
		}
	}

	for i := startIndex; i < endIndex; i++ {
		if limit > 0 && len(mods) >= limit {
			break
		}
		m := s.modStore[s.modOrder[i]]
		mods = append(mods, m)
	}

	if mods == nil {
		return nil, ErrNotFound
	}

	return mods, nil
}

func (s *FileModService) GetMod(modID string) (*modmgr.Mod, error) {
	m, ok := s.modStore[modID]
	if !ok {
		return nil, ErrNotFound
	}
	return &m, nil
}

func (s *FileModService) GetModVersions(modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	versionsMap, ok := s.versionStore[modID]
	if !ok {
		return []modmgr.ModVersion{}, ErrNotFound
	}

	allVersions := slices.Collect(maps.Values(versionsMap))

	// Sort by version ID (newest first, assuming semantic versioning or similar)
	sortedVersions := make([]modmgr.ModVersion, len(allVersions))
	copy(sortedVersions, allVersions)
	sort.Slice(sortedVersions, func(i, j int) bool {
		return sortedVersions[i].ID > sortedVersions[j].ID
	})

	startIndex := 0
	if after != "" {
		for i, v := range sortedVersions {
			if v.ID == after {
				startIndex = i + 1
				break
			}
		}
	}

	var result []modmgr.ModVersion
	for i := startIndex; i < len(sortedVersions); i++ {
		if limit > 0 && len(result) >= limit {
			break
		}
		result = append(result, sortedVersions[i])
	}

	if result == nil {
		result = []modmgr.ModVersion{}
	}

	return result, nil
}

func (s *FileModService) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	versionsMap, ok := s.versionStore[modID]
	if !ok {
		return nil, ErrNotFound
	}
	versions, ok := versionsMap[versionID]
	if !ok {
		return nil, ErrNotFound
	}
	return &versions, nil
}

// ReloadData reloads the mod data from the file
func (s *FileModService) ReloadData() error {
	return s.loadData()
}
