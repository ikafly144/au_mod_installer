package valkey

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

// LoadModsFromFile loads mod data from a JSON file into Valkey
func LoadModsFromFile(ctx context.Context, repo *Repository, filePath string) error {
	source, err := os.Open(filePath)
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

	for _, m := range fileMods {
		// Store mod
		if err := repo.SetMod(ctx, m.Mod); err != nil {
			return err
		}

		// Store versions
		for _, v := range m.Versions {
			if err := repo.SetModVersion(ctx, m.ID, v); err != nil {
				return err
			}
		}
	}

	slog.Info("mods loaded from file into Valkey", "file", filePath, "count", len(fileMods))
	return nil
}
