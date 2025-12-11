package repository

import (
	"context"
	"encoding/json"

	"github.com/valkey-io/valkey-go"

	"github.com/ikafly144/au_mod_installer/server-admin/model"
)

const (
	// Key patterns (same as server repository)
	keyModPrefix     = "mod:"
	keyModList       = "mods"
	keyVersionPrefix = "version:"
)

// ValkeyRepository implements Repository using Valkey
type ValkeyRepository struct {
	client valkey.Client
}

// NewValkeyRepository creates a new ValkeyRepository
func NewValkeyRepository(client valkey.Client) *ValkeyRepository {
	return &ValkeyRepository{client: client}
}

// GetMod retrieves a mod by ID
func (r *ValkeyRepository) GetMod(ctx context.Context, modID string) (*model.Mod, error) {
	key := keyModPrefix + modID
	data, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil
		}
		return nil, err
	}

	var mod model.Mod
	if err := json.Unmarshal([]byte(data), &mod); err != nil {
		return nil, err
	}
	return &mod, nil
}

// GetModList retrieves all mods
func (r *ValkeyRepository) GetModList(ctx context.Context) ([]model.Mod, error) {
	modIDs, err := r.client.Do(ctx, r.client.B().Zrange().Key(keyModList).Min("0").Max("-1").Build()).AsStrSlice()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return []model.Mod{}, nil
		}
		return nil, err
	}

	mods := make([]model.Mod, 0, len(modIDs))
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

// SetMod creates or updates a mod
func (r *ValkeyRepository) SetMod(ctx context.Context, mod model.Mod) error {
	data, err := json.Marshal(mod)
	if err != nil {
		return err
	}

	key := keyModPrefix + mod.ID
	if err := r.client.Do(ctx, r.client.B().Set().Key(key).Value(string(data)).Build()).Error(); err != nil {
		return err
	}

	return r.client.Do(ctx, r.client.B().Zadd().Key(keyModList).ScoreMember().ScoreMember(0, mod.ID).Build()).Error()
}

// DeleteMod deletes a mod and all its versions
func (r *ValkeyRepository) DeleteMod(ctx context.Context, modID string) error {
	// Delete mod data
	key := keyModPrefix + modID
	if err := r.client.Do(ctx, r.client.B().Del().Key(key).Build()).Error(); err != nil {
		return err
	}

	// Remove from mod list
	if err := r.client.Do(ctx, r.client.B().Zrem().Key(keyModList).Member(modID).Build()).Error(); err != nil {
		return err
	}

	// Delete all versions
	versionListKey := keyVersionPrefix + modID
	versionIDs, _ := r.client.Do(ctx, r.client.B().Zrange().Key(versionListKey).Min("0").Max("-1").Build()).AsStrSlice()
	for _, vID := range versionIDs {
		versionKey := keyVersionPrefix + modID + ":" + vID
		r.client.Do(ctx, r.client.B().Del().Key(versionKey).Build())
	}
	r.client.Do(ctx, r.client.B().Del().Key(versionListKey).Build())

	return nil
}

// GetVersion retrieves a version by mod ID and version ID
func (r *ValkeyRepository) GetVersion(ctx context.Context, modID, versionID string) (*model.ModVersion, error) {
	key := keyVersionPrefix + modID + ":" + versionID
	data, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil
		}
		return nil, err
	}

	var version model.ModVersion
	if err := json.Unmarshal([]byte(data), &version); err != nil {
		return nil, err
	}
	return &version, nil
}

// GetVersionList retrieves all versions for a mod
func (r *ValkeyRepository) GetVersionList(ctx context.Context, modID string) ([]model.ModVersion, error) {
	listKey := keyVersionPrefix + modID
	versionIDs, err := r.client.Do(ctx, r.client.B().Zrange().Key(listKey).Min("0").Max("-1").Build()).AsStrSlice()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return []model.ModVersion{}, nil
		}
		return nil, err
	}

	versions := make([]model.ModVersion, 0, len(versionIDs))
	for _, id := range versionIDs {
		version, err := r.GetVersion(ctx, modID, id)
		if err != nil {
			return nil, err
		}
		if version != nil {
			versions = append(versions, *version)
		}
	}
	return versions, nil
}

// SetVersion creates or updates a version
func (r *ValkeyRepository) SetVersion(ctx context.Context, modID string, version model.ModVersion) error {
	data, err := json.Marshal(version)
	if err != nil {
		return err
	}

	key := keyVersionPrefix + modID + ":" + version.ID
	if err := r.client.Do(ctx, r.client.B().Set().Key(key).Value(string(data)).Build()).Error(); err != nil {
		return err
	}

	listKey := keyVersionPrefix + modID
	return r.client.Do(ctx, r.client.B().Zadd().Key(listKey).ScoreMember().ScoreMember(0, version.ID).Build()).Error()
}

// DeleteVersion deletes a version
func (r *ValkeyRepository) DeleteVersion(ctx context.Context, modID, versionID string) error {
	// Delete version data
	key := keyVersionPrefix + modID + ":" + versionID
	if err := r.client.Do(ctx, r.client.B().Del().Key(key).Build()).Error(); err != nil {
		return err
	}

	// Remove from version list
	listKey := keyVersionPrefix + modID
	return r.client.Do(ctx, r.client.B().Zrem().Key(listKey).Member(versionID).Build()).Error()
}
