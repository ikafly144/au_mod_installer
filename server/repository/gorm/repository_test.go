package gorm_test

import (
	"context"
	"testing"
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/model"
	. "github.com/ikafly144/au_mod_installer/server/repository/gorm" // Import with dot to use NewRepository
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB initializes an in-memory SQLite database for testing
func setupTestDB(t *testing.T) (*Repository, func()) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Suppress SQL logs for cleaner test output
	})
	assert.NoError(t, err)

	// Auto-migrate schema
	err = db.AutoMigrate(
		&model.User{},
		&model.Mod{},
		&model.ModVersion{},
		&model.ModFile{},
		&model.ModDependency{},
		&model.ModVersionGameVersion{},
	)
	assert.NoError(t, err)

	repo := &Repository{Db: db}

	return repo, func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
}

func TestRepository_DeleteModHardDelete(t *testing.T) {
	repo, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()
	modID := "test-mod-soft-delete"
	mod := modmgr.Mod{
		ID:        modID,
		Name:      "Test Mod",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a mod
	err := repo.CreateMod(ctx, mod)
	assert.NoError(t, err)

	// Verify mod exists
	fetchedMod, err := repo.GetMod(ctx, modID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedMod)
	assert.Equal(t, modID, fetchedMod.ID)

	// Delete the mod (soft delete should happen)
	err = repo.DeleteMod(ctx, modID)
	assert.NoError(t, err)

	// Attempt to retrieve mod using GetMod - should return nil because soft deleted
	fetchedModAfterDelete, err := repo.GetMod(ctx, modID)
	assert.NoError(t, err)
	assert.Nil(t, fetchedModAfterDelete, "Mod should be soft-deleted and not retrievable by GetMod")

	// Verify mod is permanently deleted
	var deletedMod model.Mod
	err = repo.Db.WithContext(ctx).Unscoped().First(&deletedMod, "id = ?", modID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Mod should be permanently deleted")

	// Attempt to create a new mod with the same ID - should fail if soft delete prevents it
	newModWithSameID := modmgr.Mod{
		ID:        modID,
		Name:      "New Test Mod",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = repo.CreateMod(ctx, newModWithSameID)
	assert.NoError(t, err, "Creating a new mod with the same ID after hard-delete should succeed")
}

func TestRepository_DeleteModVersionHardDelete(t *testing.T) {
	repo, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()
	modID := "test-mod-version-soft-delete"
	versionID := "v1.0.0"
	mod := modmgr.Mod{
		ID:        modID,
		Name:      "Test Mod",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	version := modmgr.ModVersion{
		ID:        versionID,
		ModID:     modID,
		CreatedAt: time.Now(),
		Files: []modmgr.ModFile{{
			URL:        "http://example.com/file.zip",
			FileType:   modmgr.FileTypeZip,
			Compatible: []aumgr.BinaryType{aumgr.BinaryType64Bit},
		}},
	}

	// Create mod and version
	err := repo.CreateMod(ctx, mod)
	assert.NoError(t, err)
	err = repo.CreateModVersion(ctx, modID, version)
	assert.NoError(t, err)

	// Verify version exists
	fetchedVersion, err := repo.GetModVersion(ctx, modID, versionID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedVersion)
	assert.Equal(t, versionID, fetchedVersion.ID)

	// Delete the version (soft delete should happen)
	err = repo.DeleteModVersion(ctx, modID, versionID)
	assert.NoError(t, err)

	// Attempt to retrieve version using GetModVersion - should return nil because soft deleted
	fetchedVersionAfterDelete, err := repo.GetModVersion(ctx, modID, versionID)
	assert.NoError(t, err)
	assert.Nil(t, fetchedVersionAfterDelete, "Mod version should be soft-deleted and not retrievable by GetModVersion")

	// Verify version is permanently deleted
	var deletedVersion model.ModVersion
	err = repo.Db.WithContext(ctx).Unscoped().First(&deletedVersion, "mod_id = ? AND version_id = ?", modID, versionID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Mod version should be permanently deleted")

	// Attempt to create a new version with the same ID - should fail if soft delete prevents it
	newVersionWithSameID := modmgr.ModVersion{
		ID:        versionID,
		ModID:     modID,
		CreatedAt: time.Now(),
		Files: []modmgr.ModFile{{
			URL:        "http://example.com/newfile.zip",
			FileType:   modmgr.FileTypeZip,
			Compatible: []aumgr.BinaryType{aumgr.BinaryType64Bit},
		}},
	}
	err = repo.CreateModVersion(ctx, modID, newVersionWithSameID)
	assert.NoError(t, err, "Creating a new version with the same ID after hard-delete should succeed")
}
