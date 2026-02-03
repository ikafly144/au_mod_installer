package postgres

import (
	"context"
	"testing"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateMod(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	mod := modmgr.Mod{
		ID:          "new-mod",
		Name:        "New Mod",
		Description: "New Description",
		Author:      "New Author",
		Type:        modmgr.ModTypeMod,
	}

	mock.ExpectExec("INSERT INTO mods").
		WithArgs(mod.ID, mod.Name, mod.Description, mod.Author, string(mod.Type), mod.Thumbnail, mod.Website, mod.LatestVersion).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.CreateMod(ctx, mod)
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_UpdateMod(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	mod := modmgr.Mod{
		ID:          "existing-mod",
		Name:        "Updated Mod",
		Description: "Updated Description",
		Author:      "Updated Author",
		Type:        modmgr.ModTypeMod,
	}

	mock.ExpectExec("UPDATE mods").
		WithArgs(mod.Name, mod.Description, mod.Author, string(mod.Type), mod.Thumbnail, mod.Website, mod.LatestVersion, mod.ID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateMod(ctx, mod)
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_CreateMod_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	mod := modmgr.Mod{ID: "error-mod"}

	mock.ExpectExec("INSERT INTO mods").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(assert.AnError)

	err = repo.CreateMod(ctx, mod)
	assert.Error(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_UpdateMod_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	mod := modmgr.Mod{ID: "error-mod"}

	mock.ExpectExec("UPDATE mods").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(assert.AnError)

	err = repo.UpdateMod(ctx, mod)
	assert.Error(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_DeleteMod(t *testing.T) {

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewRepository(mock)
	ctx := context.Background()

	modID := "test-mod"

	mock.ExpectExec("DELETE FROM mods WHERE id = \\$1").
		WithArgs(modID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.DeleteMod(ctx, modID)
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
