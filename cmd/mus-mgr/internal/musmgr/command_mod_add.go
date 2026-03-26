package musmgr

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func (f *commandFactory) newModAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a new mod",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Mod ID (default: uuid)", Value: ""},
			&cli.StringFlag{Name: "name", Usage: "Mod name (required)"},
			&cli.StringFlag{Name: "author", Usage: "Mod author (required)"},
			&cli.StringFlag{Name: "desc", Usage: "Mod description"},
			&cli.StringFlag{Name: "thumbnail-url", Usage: "Mod thumbnail URL"},
		},
		ShellComplete: cli.DefaultCompleteWithFlags,
		Action: wrapAction(func(ctx context.Context, cmd *cli.Command) error {
			if err := requireDB(cmd); err != nil {
				return err
			}
			if cmd.String("name") == "" {
				return fmt.Errorf("name required")
			}
			if cmd.String("author") == "" {
				return fmt.Errorf("author required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			id := cmd.String("id")
			if id == "" {
				id = uuid.New().String()
			}

			mod := &model.ModDetails{
				ID:          id,
				Name:        cmd.String("name"),
				Author:      cmd.String("author"),
				Description: cmd.String("desc"),
			}

			if cmd.IsSet("thumbnail-url") {
				url := cmd.String("thumbnail-url")
				mod.ThumbnailURI = &url
			}

			if _, err := repo.CreateMod(mod); err != nil {
				return err
			}
			fmt.Printf("Created mod: %s\n", mod.ID)
			return nil
		}),
	}
}
