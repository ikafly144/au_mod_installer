package musmgr

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newModEditCommand() *cli.Command {
	return &cli.Command{
		Name:      "edit",
		Usage:     "Edit an existing mod",
		ArgsUsage: "<mod-id>",
		ShellComplete: func(ctx context.Context, cmd *cli.Command) {
			if cmd.NArg() <= 1 {
				f.printModIDCompletions(cmd)
			}
			cli.DefaultCompleteWithFlags(ctx, cmd)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Updated mod name"},
			&cli.StringFlag{Name: "author", Usage: "Updated mod author"},
			&cli.StringFlag{Name: "desc", Usage: "Updated mod description"},
			&cli.StringFlag{Name: "thumbnail-url", Usage: "Updated mod thumbnail URL"},
			&cli.BoolFlag{Name: "clear-thumbnail", Usage: "Clear thumbnail URL"},
			&cli.StringFlag{Name: "latest-version-id", Usage: "Updated latest version ID"},
			&cli.BoolFlag{Name: "clear-latest-version", Usage: "Clear latest version ID"},
		},
		DisableSliceFlagSeparator: true,
		Action: wrapAction(func(ctx context.Context, cmd *cli.Command) error {
			if err := requireDB(cmd); err != nil {
				return err
			}
			if cmd.NArg() < 1 {
				return fmt.Errorf("mod-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			modID := cmd.Args().First()
			updates := make(map[string]any)

			if cmd.IsSet("name") {
				updates["name"] = cmd.String("name")
			}
			if cmd.IsSet("author") {
				updates["author"] = cmd.String("author")
			}
			if cmd.IsSet("desc") {
				updates["description"] = cmd.String("desc")
			}

			if cmd.Bool("clear-thumbnail") {
				updates["thumbnail_uri"] = nil
			} else if cmd.IsSet("thumbnail-url") {
				updates["thumbnail_uri"] = cmd.String("thumbnail-url")
			}

			if cmd.Bool("clear-latest-version") {
				updates["latest_version_id"] = nil
			} else if cmd.IsSet("latest-version-id") {
				updates["latest_version_id"] = cmd.String("latest-version-id")
			}

			if len(updates) == 0 {
				return fmt.Errorf("no update fields provided")
			}

			if err := repo.UpdateModFields(modID, updates); err != nil {
				return err
			}
			fmt.Println("Updated mod:", modID)
			return nil
		}),
	}
}
