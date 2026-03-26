package musmgr

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func (f *commandFactory) newModAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a new mod",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Mod ID (default: uuid)", Value: ""},
			&cli.StringFlag{Name: "name", Required: true},
			&cli.StringFlag{Name: "author", Required: true},
			&cli.StringFlag{Name: "desc", Required: false},
		},
		Action: func(c *cli.Context) error {
			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			id := c.String("id")
			if id == "" {
				id = uuid.New().String()
			}

			mod := &model.ModDetails{
				ID:          id,
				Name:        c.String("name"),
				Author:      c.String("author"),
				Description: c.String("desc"),
			}

			if _, err := repo.CreateMod(mod); err != nil {
				return err
			}
			fmt.Printf("Created mod: %s\n", mod.ID)
			return nil
		},
	}
}
