package valkey

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/valkey-io/valkey-go"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

const (
	// Key patterns
	keyModPrefix     = "mod:"
	keyModList       = "mods"
	keyVersionPrefix = "version:"
)

// Repository provides Valkey storage operations for mods
type Repository struct {
	client valkey.Client
}

// NewRepository creates a new Valkey repository
func NewRepository(client valkey.Client) *Repository {
	return &Repository{
		client: client,
	}
}

// Close closes the Valkey client connection
func (r *Repository) Close() {
	r.client.Close()
}

// SetMod stores a mod in Valkey
func (r *Repository) SetMod(ctx context.Context, mod modmgr.Mod) error {
	data, err := json.Marshal(mod)
	if err != nil {
		return fmt.Errorf("failed to marshal mod: %w", err)
	}

	// Store mod data
	key := keyModPrefix + mod.ID
	err = r.client.Do(ctx, r.client.B().Set().Key(key).Value(string(data)).Build()).Error()
	if err != nil {
		return fmt.Errorf("failed to set mod: %w", err)
	}

	// Add to mod list (sorted set for ordering)
	err = r.client.Do(ctx, r.client.B().Zadd().Key(keyModList).ScoreMember().ScoreMember(0, mod.ID).Build()).Error()
	if err != nil {
		return fmt.Errorf("failed to add mod to list: %w", err)
	}

	slog.Debug("stored mod", "id", mod.ID)
	return nil
}

// GetMod retrieves a mod by ID from Valkey
func (r *Repository) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	key := keyModPrefix + modID
	data, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get mod: %w", err)
	}

	var mod modmgr.Mod
	if err := json.Unmarshal([]byte(data), &mod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mod: %w", err)
	}

	return &mod, nil
}

// GetModList retrieves a list of mods with pagination
func (r *Repository) GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error) {
	var modIDs []string
	var err error

	// Get all mod IDs from sorted set
	if after != "" {
		// Get mods after the specified ID
		rank, err := r.client.Do(ctx, r.client.B().Zrank().Key(keyModList).Member(after).Build()).AsInt64()
		if err != nil && !valkey.IsValkeyNil(err) {
			return nil, fmt.Errorf("failed to get rank: %w", err)
		}
		start := rank + 1
		end := int64(-1)
		if limit > 0 {
			end = start + int64(limit) - 1
		}
		modIDs, err = r.client.Do(ctx, r.client.B().Zrange().Key(keyModList).Min(fmt.Sprint(start)).Max(fmt.Sprint(end)).Build()).AsStrSlice()
		if err != nil {
			return nil, fmt.Errorf("failed to get mod list: %w", err)
		}
	} else if before != "" {
		// Get mods before the specified ID
		rank, err := r.client.Do(ctx, r.client.B().Zrank().Key(keyModList).Member(before).Build()).AsInt64()
		if err != nil && !valkey.IsValkeyNil(err) {
			return nil, fmt.Errorf("failed to get rank: %w", err)
		}
		start := int64(0)
		end := rank - 1
		if limit > 0 && end-start+1 > int64(limit) {
			start = end - int64(limit) + 1
		}
		modIDs, err = r.client.Do(ctx, r.client.B().Zrange().Key(keyModList).Min(fmt.Sprint(start)).Max(fmt.Sprint(end)).Build()).AsStrSlice()
		if err != nil {
			return nil, fmt.Errorf("failed to get mod list: %w", err)
		}
	} else {
		// Get all mods
		end := int64(-1)
		if limit > 0 {
			end = int64(limit) - 1
		}
		modIDs, err = r.client.Do(ctx, r.client.B().Zrange().Key(keyModList).Min("0").Max(fmt.Sprint(end)).Build()).AsStrSlice()
		if err != nil {
			return nil, fmt.Errorf("failed to get mod list: %w", err)
		}
	}

	if len(modIDs) == 0 {
		return []modmgr.Mod{}, nil
	}

	// Get mod data for each ID
	mods := make([]modmgr.Mod, 0, len(modIDs))
	for _, id := range modIDs {
		mod, err := r.GetMod(ctx, id)
		if err != nil {
			return nil, err
		}
		if mod != nil {
			mods = append(mods, *mod)
		}
	}

	return mods, nil
}

// SetModVersion stores a mod version in Valkey
func (r *Repository) SetModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	data, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	// Store version data
	key := keyVersionPrefix + modID + ":" + version.ID
	err = r.client.Do(ctx, r.client.B().Set().Key(key).Value(string(data)).Build()).Error()
	if err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}

	// Add to version list for this mod (sorted set)
	listKey := keyVersionPrefix + modID
	err = r.client.Do(ctx, r.client.B().Zadd().Key(listKey).ScoreMember().ScoreMember(0, version.ID).Build()).Error()
	if err != nil {
		return fmt.Errorf("failed to add version to list: %w", err)
	}

	slog.Debug("stored mod version", "modID", modID, "versionID", version.ID)
	return nil
}

// GetModVersion retrieves a specific version of a mod
func (r *Repository) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	key := keyVersionPrefix + modID + ":" + versionID
	data, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	var version modmgr.ModVersion
	if err := json.Unmarshal([]byte(data), &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	return &version, nil
}

// GetModVersions retrieves all versions of a mod with pagination
func (r *Repository) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	listKey := keyVersionPrefix + modID

	var versionIDs []string
	var err error

	if after != "" {
		// Get versions after the specified ID
		rank, err := r.client.Do(ctx, r.client.B().Zrank().Key(listKey).Member(after).Build()).AsInt64()
		if err != nil && !valkey.IsValkeyNil(err) {
			return nil, fmt.Errorf("failed to get rank: %w", err)
		}
		start := rank + 1
		end := int64(-1)
		if limit > 0 {
			end = start + int64(limit) - 1
		}
		versionIDs, err = r.client.Do(ctx, r.client.B().Zrange().Key(listKey).Min(fmt.Sprint(start)).Max(fmt.Sprint(end)).Build()).AsStrSlice()
		if err != nil {
			return nil, fmt.Errorf("failed to get version list: %w", err)
		}
	} else {
		// Get all versions
		end := int64(-1)
		if limit > 0 {
			end = int64(limit) - 1
		}
		versionIDs, err = r.client.Do(ctx, r.client.B().Zrange().Key(listKey).Min("0").Max(fmt.Sprint(end)).Build()).AsStrSlice()
		if err != nil {
			return nil, fmt.Errorf("failed to get version list: %w", err)
		}
	}

	if len(versionIDs) == 0 {
		return []modmgr.ModVersion{}, nil
	}

	// Get version data for each ID
	versions := make([]modmgr.ModVersion, 0, len(versionIDs))
	for _, id := range versionIDs {
		version, err := r.GetModVersion(ctx, modID, id)
		if err != nil {
			return nil, err
		}
		if version != nil {
			versions = append(versions, *version)
		}
	}

	return versions, nil
}
