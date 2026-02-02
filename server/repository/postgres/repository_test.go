package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepository(t *testing.T) {
	repo := NewRepository(nil)
	assert.NotNil(t, repo)
}

func TestRepository_GetUserByUsername(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	username := "testuser"
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "username", "password_hash", "display_name", "is_admin", "created_at", "updated_at"}).
		AddRow(1, username, "hash", "Test User", false, now, now)

	mock.ExpectQuery("SELECT id, username, password_hash, display_name, is_admin, created_at, updated_at FROM users WHERE username = \\$1").
		WithArgs(username).
		WillReturnRows(rows)

	user, err := repo.GetUserByUsername(ctx, username)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, "Test User", user.DisplayName)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_GetModList(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "name", "description", "author_name", "type", "thumbnail_url", "website_url", "latest_version_id", "created_at", "updated_at"}).
		AddRow("mod-1", "Mod 1", "Desc 1", "Author 1", "mod", "thumb 1", "site 1", "v1", now, now).
		AddRow("mod-2", "Mod 2", "Desc 2", "Author 2", "mod", "thumb 2", "site 2", "v2", now, now)

	mock.ExpectQuery("SELECT id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, created_at, updated_at FROM mods ORDER BY id ASC LIMIT \\$1").
		WithArgs(10).
		WillReturnRows(rows)

	mods, err := repo.GetModList(ctx, 10, "", "")
	require.NoError(t, err)
	assert.Len(t, mods, 2)
	assert.Equal(t, "mod-1", mods[0].ID)
	assert.Equal(t, "mod-2", mods[1].ID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_SetModVersion(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	modID := "test-mod"
	version := modmgr.ModVersion{
		ID:        "v1.0.0",
		ModID:     modID,
		CreatedAt: time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO mod_versions").
		WithArgs(modID, version.ID, version.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("DELETE FROM mod_files").WithArgs(modID, version.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("DELETE FROM mod_dependencies").WithArgs(modID, version.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("DELETE FROM mod_version_game_versions").WithArgs(modID, version.ID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectCommit()
	mock.ExpectRollback() // pgx.BeginFunc calls rollback in defer

	err = repo.SetModVersion(ctx, modID, version)

	require.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_GetMod(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	modID := "test-mod"
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "name", "description", "author_name", "type", "thumbnail_url", "website_url", "latest_version_id", "created_at", "updated_at"}).
		AddRow(modID, "Test Mod", "Description", "Author", "mod", "thumb", "site", "v1", now, now)

	mock.ExpectQuery("SELECT id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, created_at, updated_at FROM mods WHERE id = \\$1").
		WithArgs(modID).
		WillReturnRows(rows)

	mod, err := repo.GetMod(ctx, modID)
	require.NoError(t, err)
	assert.NotNil(t, mod)
	assert.Equal(t, modID, mod.ID)
	assert.Equal(t, "Test Mod", mod.Name)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
