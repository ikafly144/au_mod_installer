package musmgr

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func (f *commandFactory) newVersionAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a new version for a mod",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "mod-id", Usage: "Target mod ID (required)"},
			&cli.StringFlag{Name: "version-id", Usage: "Version ID (default: auto-incremented SemVer)"},
			&cli.StringSliceFlag{Name: "file", Usage: "Files to add. Multiple flags supported. Format: path=...,type=...,url=...,extract_path=...,target_platform=... or direct URL/Path"},
			&cli.StringSliceFlag{Name: "dependency", Usage: "Dependencies to add. Multiple flags supported. Format: mod_id:version_id:type (type is optional, default: required)"},
			&cli.StringSliceFlag{Name: "feature", Usage: "Features to set. Format: name=true|false (e.g. direct_join=true)"},
			&cli.BoolFlag{Name: "set-latest", Usage: "Set this version as the latest version for the mod"},
		},
		ShellComplete: cli.DefaultCompleteWithFlags,
		Action: wrapAction(func(ctx context.Context, cmd *cli.Command) error {
			if err := requireDB(cmd); err != nil {
				return err
			}
			if cmd.String("mod-id") == "" {
				return fmt.Errorf("mod-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			modID := cmd.String("mod-id")
			versionID := cmd.String("version-id")
			if versionID == "" {
				existingIDs, err := repo.GetModVersionIds(modID)
				if err != nil {
					return fmt.Errorf("failed to fetch existing versions for auto-increment: %w", err)
				}
				versionID = nextVersionID(existingIDs)
				fmt.Printf("Auto-generated version ID: %s\n", versionID)
			}

			ver := &model.ModVersionDetails{
				ID:           versionID,
				ModID:        &modID,
				Dependencies: parseDependencies(cmd.StringSlice("dependency")),
				Features:     parseFeatures(cmd.StringSlice("feature")),
			}

			for _, fileFlag := range cmd.StringSlice("file") {
				pf := parseFileFlag(fileFlag)

				filename, size, hashStr, err := fileMetadataFromParsedFile(pf)
				if err != nil {
					return err
				}

				verFile := model.ModVersionFile{
					ID:             uuid.New().String(),
					ModID:          &modID,
					VersionID:      &ver.ID,
					Filename:       filename,
					ContentType:    model.FileType(pf.Type),
					Size:           size,
					ExtractPath:    pf.ExtractPath,
					TargetPlatform: model.TargetPlatform(pf.TargetPlatform),
					Hashes:         map[string]string{"sha256": hashStr},
					Downloads:      pf.URLs,
				}
				ver.Files = append(ver.Files, verFile)
			}

			if _, err := repo.CreateModVersion(modID, ver); err != nil {
				return err
			}
			fmt.Printf("Created version: %s\n", ver.ID)

			if cmd.Bool("set-latest") {
				update := &model.ModDetails{LatestVersionID: &ver.ID}
				if err := repo.UpdateMod(modID, update); err != nil {
					return fmt.Errorf("failed to update latest version: %w", err)
				}
				fmt.Println("Set as latest version.")
			}

			return nil
		}),
	}
}
