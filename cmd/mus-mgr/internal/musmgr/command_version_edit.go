package musmgr

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func (f *commandFactory) newVersionEditCommand() *cli.Command {
	return &cli.Command{
		Name:      "edit",
		Usage:     "Edit an existing mod version",
		ArgsUsage: "<mod-id> <version-id>",
		ShellComplete: func(ctx context.Context, cmd *cli.Command) {
			if cmd.NArg() <= 1 {
				f.printModIDCompletions(cmd)
				return
			}
			if cmd.NArg() <= 2 {
				f.printVersionIDCompletions(cmd, cmd.Args().Get(0))
			}
			cli.DefaultCompleteWithFlags(ctx, cmd)
		},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "dependency", Usage: "Replace dependencies. Format: mod_id:version_id:type"},
			&cli.StringSliceFlag{Name: "feature", Usage: "Replace features. Format: name=true|false"},
			&cli.BoolFlag{Name: "set-latest", Usage: "Set this version as latest on the mod"},
			&cli.BoolFlag{Name: "clear-latest-version", Usage: "Clear latest version on the mod"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := requireDB(cmd); err != nil {
				return err
			}
			if cmd.NArg() < 2 {
				return fmt.Errorf("mod-id and version-id required")
			}
			if cmd.Bool("set-latest") && cmd.Bool("clear-latest-version") {
				return fmt.Errorf("set-latest and clear-latest-version cannot be used together")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			modID := cmd.Args().Get(0)
			versionID := cmd.Args().Get(1)
			changed := false

			if cmd.IsSet("dependency") {
				updates := map[string]any{
					"dependencies": model.DependencyArray(parseDependencies(cmd.StringSlice("dependency"))),
				}
				if err := repo.UpdateModVersionFields(modID, versionID, updates); err != nil {
					return err
				}
				changed = true
			}
			if cmd.IsSet("feature") {
				updates := map[string]any{
					"features": model.Features(parseFeatures(cmd.StringSlice("feature"))),
				}
				if err := repo.UpdateModVersionFields(modID, versionID, updates); err != nil {
					return err
				}
				changed = true
			}

			if cmd.Bool("set-latest") {
				if err := repo.UpdateModFields(modID, map[string]any{"latest_version_id": versionID}); err != nil {
					return fmt.Errorf("failed to update latest version: %w", err)
				}
				changed = true
			} else if cmd.Bool("clear-latest-version") {
				if err := repo.UpdateModFields(modID, map[string]any{"latest_version_id": nil}); err != nil {
					return fmt.Errorf("failed to clear latest version: %w", err)
				}
				changed = true
			}

			if !changed {
				return fmt.Errorf("no update fields provided")
			}

			fmt.Printf("Updated version %s of mod %s\n", versionID, modID)
			return nil
		},
	}
}
