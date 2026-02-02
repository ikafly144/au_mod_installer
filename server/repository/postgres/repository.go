package postgres

import (
	"context"
	"fmt"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgxPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	Close()
}

type Repository struct {
	pool pgxPool
}

var _ repository.ModRepository = (*Repository)(nil)

func NewRepository(pool pgxPool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	query := `SELECT id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, created_at, updated_at FROM mods WHERE id = $1`
	var mod modmgr.Mod
	var modType string
	err := r.pool.QueryRow(ctx, query, modID).Scan(
		&mod.ID, &mod.Name, &mod.Description, &mod.Author, &modType, &mod.Thumbnail, &mod.Website, &mod.LatestVersion, &mod.CreatedAt, &mod.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get mod: %w", err)
	}
	mod.Type = modmgr.ModType(modType)
	return &mod, nil
}

func (r *Repository) GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error) {
	var query string
	var args []any

	// Simple pagination for now
	if after != "" {
		query = `SELECT id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, created_at, updated_at FROM mods WHERE id > $1 ORDER BY id ASC LIMIT $2`
		args = append(args, after, limit)
	} else if before != "" {
		query = `SELECT id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, created_at, updated_at FROM mods WHERE id < $1 ORDER BY id DESC LIMIT $2`
		args = append(args, before, limit)
	} else {
		query = `SELECT id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, created_at, updated_at FROM mods ORDER BY id ASC LIMIT $1`
		args = append(args, limit)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query mods: %w", err)
	}
	defer rows.Close()

	var mods []modmgr.Mod
	for rows.Next() {
		var mod modmgr.Mod
		var modType string
		err := rows.Scan(
			&mod.ID, &mod.Name, &mod.Description, &mod.Author, &modType, &mod.Thumbnail, &mod.Website, &mod.LatestVersion, &mod.CreatedAt, &mod.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mod: %w", err)
		}
		mod.Type = modmgr.ModType(modType)
		mods = append(mods, mod)
	}

	if before != "" {
		// Reverse the list if we fetched using "before"
		for i, j := 0, len(mods)-1; i < j; i, j = i+1, j-1 {
			mods[i], mods[j] = mods[j], mods[i]
		}
	}

	return mods, nil
}

func (r *Repository) GetAllMods(ctx context.Context) ([]modmgr.Mod, error) {
	query := `SELECT id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, created_at, updated_at FROM mods ORDER BY id ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all mods: %w", err)
	}
	defer rows.Close()

	var mods []modmgr.Mod
	for rows.Next() {
		var mod modmgr.Mod
		var modType string
		err := rows.Scan(
			&mod.ID, &mod.Name, &mod.Description, &mod.Author, &modType, &mod.Thumbnail, &mod.Website, &mod.LatestVersion, &mod.CreatedAt, &mod.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mod: %w", err)
		}
		mod.Type = modmgr.ModType(modType)
		mods = append(mods, mod)
	}

	return mods, nil
}

func (r *Repository) SetMod(ctx context.Context, mod modmgr.Mod) error {
	query := `
		INSERT INTO mods (id, name, description, author_name, type, thumbnail_url, website_url, latest_version_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			author_name = EXCLUDED.author_name,
			type = EXCLUDED.type,
			thumbnail_url = EXCLUDED.thumbnail_url,
			website_url = EXCLUDED.website_url,
			latest_version_id = EXCLUDED.latest_version_id,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, mod.ID, mod.Name, mod.Description, mod.Author, mod.Type, mod.Thumbnail, mod.Website, mod.LatestVersion)
	if err != nil {
		return fmt.Errorf("failed to set mod: %w", err)
	}
	return nil
}

func (r *Repository) DeleteMod(ctx context.Context, modID string) error {
	query := `DELETE FROM mods WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, modID)
	if err != nil {
		return fmt.Errorf("failed to delete mod: %w", err)
	}
	return nil
}

func (r *Repository) SetModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Insert version
		_, err := tx.Exec(ctx, `
			INSERT INTO mod_versions (mod_id, version_id, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (mod_id, version_id) DO NOTHING
		`, modID, version.ID, version.CreatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert mod version: %w", err)
		}

		// Delete existing files, dependencies, game versions
		_, _ = tx.Exec(ctx, `DELETE FROM mod_files WHERE mod_id = $1 AND version_id = $2`, modID, version.ID)
		_, _ = tx.Exec(ctx, `DELETE FROM mod_dependencies WHERE mod_id = $1 AND version_id = $2`, modID, version.ID)
		_, _ = tx.Exec(ctx, `DELETE FROM mod_version_game_versions WHERE mod_id = $1 AND version_id = $2`, modID, version.ID)

		// Insert files
		for _, file := range version.Files {
			compat := make([]string, len(file.Compatible))
			for i, c := range file.Compatible {
				compat[i] = string(c)
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO mod_files (mod_id, version_id, file_type, path, url, compatible_binary_types)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, modID, version.ID, file.FileType, file.Path, file.URL, compat)
			if err != nil {
				return fmt.Errorf("failed to insert mod file: %w", err)
			}
		}

		// Insert dependencies
		for _, dep := range version.Dependencies {
			_, err = tx.Exec(ctx, `
				INSERT INTO mod_dependencies (mod_id, version_id, dependency_id, dependency_version, dependency_type)
				VALUES ($1, $2, $3, $4, $5)
			`, modID, version.ID, dep.ID, dep.Version, dep.Type)
			if err != nil {
				return fmt.Errorf("failed to insert mod dependency: %w", err)
			}
		}

		// Insert game versions
		for _, gv := range version.GameVersions {
			_, err = tx.Exec(ctx, `
				INSERT INTO mod_version_game_versions (mod_id, version_id, game_version)
				VALUES ($1, $2, $3)
			`, modID, version.ID, gv)
			if err != nil {
				return fmt.Errorf("failed to insert mod game version: %w", err)
			}
		}

		return nil
	})
}

func (r *Repository) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	var version modmgr.ModVersion
	err := r.pool.QueryRow(ctx, `SELECT mod_id, version_id, created_at FROM mod_versions WHERE mod_id = $1 AND version_id = $2`, modID, versionID).Scan(
		&version.ModID, &version.ID, &version.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get mod version: %w", err)
	}

	// Fetch files
	rows, err := r.pool.Query(ctx, `SELECT file_type, path, url, compatible_binary_types FROM mod_files WHERE mod_id = $1 AND version_id = $2`, modID, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mod files: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var file modmgr.ModFile
		var compat []string
		if err := rows.Scan(&file.FileType, &file.Path, &file.URL, &compat); err != nil {
			return nil, fmt.Errorf("failed to scan mod file: %w", err)
		}
		// Convert strings to BinaryType
		for _, c := range compat {
			file.Compatible = append(file.Compatible, aumgr.BinaryType(c))
		}
		version.Files = append(version.Files, file)

	}

	// Fetch dependencies
	depRows, err := r.pool.Query(ctx, `SELECT dependency_id, dependency_version, dependency_type FROM mod_dependencies WHERE mod_id = $1 AND version_id = $2`, modID, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mod dependencies: %w", err)
	}
	defer depRows.Close()
	for depRows.Next() {
		var dep modmgr.ModDependency
		if err := depRows.Scan(&dep.ID, &dep.Version, &dep.Type); err != nil {
			return nil, fmt.Errorf("failed to scan mod dependency: %w", err)
		}
		version.Dependencies = append(version.Dependencies, dep)
	}

	// Fetch game versions
	gvRows, err := r.pool.Query(ctx, `SELECT game_version FROM mod_version_game_versions WHERE mod_id = $1 AND version_id = $2`, modID, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mod game versions: %w", err)
	}
	defer gvRows.Close()
	for gvRows.Next() {
		var gv string
		if err := gvRows.Scan(&gv); err != nil {
			return nil, fmt.Errorf("failed to scan mod game version: %w", err)
		}
		version.GameVersions = append(version.GameVersions, gv)
	}

	return &version, nil
}

func (r *Repository) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	// Simple implementation: get all version IDs and then fetch details
	// This is not very efficient for many versions but works for now.
	query := `SELECT version_id FROM mod_versions WHERE mod_id = $1`
	var args []any
	args = append(args, modID)
	if after != "" {
		query += ` AND version_id > $2`
		args = append(args, after)
	}
	query += ` ORDER BY version_id ASC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []modmgr.ModVersion
	for rows.Next() {
		var vid string
		if err := rows.Scan(&vid); err != nil {
			return nil, err
		}
		v, err := r.GetModVersion(ctx, modID, vid)
		if err != nil {
			return nil, err
		}
		if v != nil {
			versions = append(versions, *v)
		}
	}
	return versions, nil
}

func (r *Repository) GetAllModVersions(ctx context.Context, modID string) ([]modmgr.ModVersion, error) {
	return r.GetModVersions(ctx, modID, 0, "")
}

func (r *Repository) DeleteVersion(ctx context.Context, modID, versionID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM mod_versions WHERE mod_id = $1 AND version_id = $2`, modID, versionID)
	return err
}

// UserRepository implementation

func (r *Repository) GetUser(ctx context.Context, id int) (*model.User, error) {
	query := `SELECT id, username, password_hash, display_name, is_admin, created_at, updated_at FROM users WHERE id = $1`
	var u model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT id, username, password_hash, display_name, is_admin, created_at, updated_at FROM users WHERE username = $1`
	var u model.User
	err := r.pool.QueryRow(ctx, query, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &u, nil
}

func (r *Repository) CreateUser(ctx context.Context, u model.User) error {
	query := `INSERT INTO users (username, password_hash, display_name, is_admin, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())`
	_, err := r.pool.Exec(ctx, query, u.Username, u.PasswordHash, u.DisplayName, u.IsAdmin)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *Repository) UpdateUser(ctx context.Context, u model.User) error {
	query := `UPDATE users SET username = $1, password_hash = $2, display_name = $3, is_admin = $4, updated_at = NOW() WHERE id = $5`
	_, err := r.pool.Exec(ctx, query, u.Username, u.PasswordHash, u.DisplayName, u.IsAdmin, u.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *Repository) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
