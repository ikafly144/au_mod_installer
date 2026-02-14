package gorm

import (
	"context"
	"errors"
	"fmt"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/model" // alias to avoid conflict if needed, but package name is model
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Repository implements ModRepository and UserRepository using GORM
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new GORM repository
func NewRepository(databaseURL string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate schema
	if err := db.AutoMigrate(
		&model.User{},
		&model.Mod{},
		&model.ModVersion{},
		&model.ModFile{},
		&model.ModDependency{},
		&model.ModVersionGameVersion{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Close() {
	sqlDB, err := r.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}

// --- ModRepository Implementation ---

func (r *Repository) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	var m model.Mod
	if err := r.db.WithContext(ctx).First(&m, "id = ?", modID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found, consistent with old repo
		}
		return nil, err
	}
	return toModMgrMod(m), nil
}

func (r *Repository) GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error) {
	var mods []model.Mod
	query := r.db.WithContext(ctx).Model(&model.Mod{})
	if after != "" {
		query = query.Where("id > ?", after)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&mods).Error; err != nil {
		return nil, err
	}

	result := make([]modmgr.Mod, len(mods))
	for i, m := range mods {
		result[i] = *toModMgrMod(m)
	}
	return result, nil
}

func (r *Repository) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	var v model.ModVersion
	if err := r.db.WithContext(ctx).
		Preload("Files").
		Preload("Dependencies").
		Preload("GameVersions").
		Where("mod_id = ? AND version_id = ?", modID, versionID).
		First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toModMgrVersion(v), nil
}

func (r *Repository) CreateMod(ctx context.Context, mod modmgr.Mod) error {
	m := toGormMod(mod)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *Repository) UpdateMod(ctx context.Context, mod modmgr.Mod) error {
	m := toGormMod(mod)
	return r.db.WithContext(ctx).Save(&m).Error
}

func (r *Repository) CreateModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	v := toGormVersion(modID, version)
	return r.db.WithContext(ctx).Create(&v).Error
}

func (r *Repository) UpdateModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	// GORM's Save or Updates works, but for nested associations (files), it's trickier.
	// Best approach with GORM association mode: replace files.

	v := toGormVersion(modID, version)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update Version fields
		if err := tx.Model(&model.ModVersion{}).
			Where("mod_id = ? AND version_id = ?", modID, version.ID).
			Updates(v).Error; err != nil {
			return err
		}

		// Replace associations
		var existingV model.ModVersion
		if err := tx.Where("mod_id = ? AND version_id = ?", modID, version.ID).First(&existingV).Error; err != nil {
			return err
		}

		if err := tx.Model(&existingV).Association("Files").Replace(v.Files); err != nil {
			return err
		}
		if err := tx.Model(&existingV).Association("Dependencies").Replace(v.Dependencies); err != nil {
			return err
		}
		if err := tx.Model(&existingV).Association("GameVersions").Replace(v.GameVersions); err != nil {
			return err
		}

		return nil
	})
}

// Helpers
func toModMgrMod(m model.Mod) *modmgr.Mod {
	return &modmgr.Mod{
		ID:            m.ID,
		Name:          m.Name,
		Description:   m.Description,
		Author:        m.AuthorName, // Simplified mapping
		Type:          modmgr.ModType(m.Type),
		Thumbnail:     m.ThumbnailURL,
		Website:       m.WebsiteURL,
		GitHubRepo:    m.GitHubRepo,
		LatestVersion: m.LatestVersionID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func toGormMod(m modmgr.Mod) model.Mod {
	return model.Mod{
		ID:              m.ID,
		Name:            m.Name,
		Description:     m.Description,
		AuthorName:      m.Author,
		Type:            string(m.Type),
		ThumbnailURL:    m.Thumbnail,
		WebsiteURL:      m.Website,
		GitHubRepo:      m.GitHubRepo,
		LatestVersionID: m.LatestVersion,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func toModMgrVersion(v model.ModVersion) *modmgr.ModVersion {
	files := make([]modmgr.ModFile, len(v.Files))
	for i, f := range v.Files {
		compat := make([]aumgr.BinaryType, len(f.CompatibleBinaryTypes))
		for j, c := range f.CompatibleBinaryTypes {
			compat[j] = aumgr.BinaryType(c)
		}
		files[i] = modmgr.ModFile{
			FileType:   modmgr.FileType(f.FileType),
			Path:       f.Path,
			URL:        f.URL,
			Compatible: compat,
		}
	}

	deps := make([]modmgr.ModDependency, len(v.Dependencies))
	for i, d := range v.Dependencies {
		deps[i] = modmgr.ModDependency{
			ID:      d.DependencyID,
			Version: d.DependencyVersion,
			Type:    modmgr.ModDependencyType(d.DependencyType),
		}
	}

	gvs := make([]string, len(v.GameVersions))
	for i, gv := range v.GameVersions {
		gvs[i] = gv.GameVersion
	}

	return &modmgr.ModVersion{
		ID:           v.VersionID,
		ModID:        v.ModID,
		CreatedAt:    v.CreatedAt,
		Files:        files,
		Dependencies: deps,
		GameVersions: gvs,
	}
}

func toGormVersion(modID string, v modmgr.ModVersion) model.ModVersion {
	files := make([]model.ModFile, len(v.Files))
	for i, f := range v.Files {
		compat := make([]string, len(f.Compatible))
		for j, c := range f.Compatible {
			compat[j] = string(c)
		}
		files[i] = model.ModFile{
			ModID:                 modID,
			VersionID:             v.ID,
			FileType:              string(f.FileType),
			Path:                  f.Path,
			URL:                   f.URL,
			CompatibleBinaryTypes: compat,
		}
	}

	deps := make([]model.ModDependency, len(v.Dependencies))
	for i, d := range v.Dependencies {
		deps[i] = model.ModDependency{
			ModID:             modID,
			VersionID:         v.ID,
			DependencyID:      d.ID,
			DependencyVersion: d.Version,
			DependencyType:    string(d.Type),
		}
	}

	gvs := make([]model.ModVersionGameVersion, len(v.GameVersions))
	for i, gv := range v.GameVersions {
		gvs[i] = model.ModVersionGameVersion{
			ModID:       modID,
			VersionID:   v.ID,
			GameVersion: gv,
		}
	}

	return model.ModVersion{
		ModID:        modID,
		VersionID:    v.ID,
		CreatedAt:    v.CreatedAt,
		Files:        files,
		Dependencies: deps,
		GameVersions: gvs,
	}
}

// Additional methods to satisfy interface

func (r *Repository) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	var versions []model.ModVersion
	query := r.db.WithContext(ctx).
		Preload("Files").
		Preload("Dependencies").
		Preload("GameVersions").
		Where("mod_id = ?", modID)

	if after != "" {
		query = query.Where("version_id > ?", after)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&versions).Error; err != nil {
		return nil, err
	}

	result := make([]modmgr.ModVersion, len(versions))
	for i, v := range versions {
		result[i] = *toModMgrVersion(v)
	}
	return result, nil
}

func (r *Repository) SetMod(ctx context.Context, mod modmgr.Mod) error {
	return r.CreateMod(ctx, mod) // Or use Save/FirstOrCreate
}

func (r *Repository) SetModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	return r.CreateModVersion(ctx, modID, version)
}

func (r *Repository) GetAllMods(ctx context.Context) ([]modmgr.Mod, error) {
	return r.GetModList(ctx, 0, "", "")
}

func (r *Repository) GetAllModVersions(ctx context.Context, modID string) ([]modmgr.ModVersion, error) {
	return r.GetModVersions(ctx, modID, 0, "")
}

func (r *Repository) DeleteMod(ctx context.Context, modID string) error {
	return r.db.WithContext(ctx).Delete(&model.Mod{}, "id = ?", modID).Error
}

func (r *Repository) DeleteModVersion(ctx context.Context, modID, versionID string) error {
	return r.db.WithContext(ctx).Delete(&model.ModVersion{}, "mod_id = ? AND version_id = ?", modID, versionID).Error
}

func (r *Repository) DeleteVersion(ctx context.Context, modID, versionID string) error {
	return r.DeleteModVersion(ctx, modID, versionID)
}

// --- UserRepository Implementation ---

func (r *Repository) GetUser(ctx context.Context, id int) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByDiscordID(ctx context.Context, discordID string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("discord_id = ?", discordID).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(ctx context.Context, user model.User) error {
	return r.db.WithContext(ctx).Create(&user).Error
}

func (r *Repository) UpdateUser(ctx context.Context, user model.User) error {
	return r.db.WithContext(ctx).Save(&user).Error
}

func (r *Repository) DeleteUser(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}
