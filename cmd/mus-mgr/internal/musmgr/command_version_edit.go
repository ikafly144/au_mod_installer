package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func (f *commandFactory) newVersionEditCommand() *cli.Command {
	return &cli.Command{
		Name:      "edit",
		Usage:     "Edit an existing mod version",
		ArgsUsage: "<mod-id> <version-id>",
		BashComplete: func(c *cli.Context) {
			if c.NArg() <= 1 {
				f.printModIDCompletions(c)
				return
			}
			if c.NArg() <= 2 {
				f.printVersionIDCompletions(c, c.Args().Get(0))
			}
		},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "dependency", Usage: "Replace dependencies. Format: mod_id:version_id:type"},
			&cli.BoolFlag{Name: "set-latest", Usage: "Set this version as latest on the mod"},
			&cli.BoolFlag{Name: "clear-latest-version", Usage: "Clear latest version on the mod"},
		},
		Action: func(c *cli.Context) error {
			if err := requireDB(c); err != nil {
				return err
			}
			if c.NArg() < 2 {
				return fmt.Errorf("mod-id and version-id required")
			}
			if c.Bool("set-latest") && c.Bool("clear-latest-version") {
				return fmt.Errorf("set-latest and clear-latest-version cannot be used together")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			modID := c.Args().Get(0)
			versionID := c.Args().Get(1)
			changed := false

			if c.IsSet("dependency") {
				updates := map[string]any{
					"dependencies": model.DependencyArray(parseDependencies(c.StringSlice("dependency"))),
				}
				if err := repo.UpdateModVersionFields(modID, versionID, updates); err != nil {
					return err
				}
				changed = true
			}

			if c.Bool("set-latest") {
				if err := repo.UpdateModFields(modID, map[string]any{"latest_version_id": versionID}); err != nil {
					return fmt.Errorf("failed to update latest version: %w", err)
				}
				changed = true
			} else if c.Bool("clear-latest-version") {
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
