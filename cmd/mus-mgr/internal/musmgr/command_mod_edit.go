package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) newModEditCommand() *cli.Command {
	return &cli.Command{
		Name:      "edit",
		Usage:     "Edit an existing mod",
		ArgsUsage: "<mod-id>",
		BashComplete: func(c *cli.Context) {
			if c.NArg() <= 1 {
				f.printModIDCompletions(c)
			}
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Updated mod name"},
			&cli.StringFlag{Name: "author", Usage: "Updated mod author"},
			&cli.StringFlag{Name: "desc", Usage: "Updated mod description"},
			&cli.StringFlag{Name: "latest-version-id", Usage: "Updated latest version ID"},
			&cli.BoolFlag{Name: "clear-latest-version", Usage: "Clear latest version ID"},
		},
		Action: func(c *cli.Context) error {
			if err := requireDB(c); err != nil {
				return err
			}
			if c.NArg() < 1 {
				return fmt.Errorf("mod-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			modID := c.Args().First()
			updates := make(map[string]any)

			if c.IsSet("name") {
				updates["name"] = c.String("name")
			}
			if c.IsSet("author") {
				updates["author"] = c.String("author")
			}
			if c.IsSet("desc") {
				updates["description"] = c.String("desc")
			}

			if c.Bool("clear-latest-version") {
				updates["latest_version_id"] = nil
			} else if c.IsSet("latest-version-id") {
				updates["latest_version_id"] = c.String("latest-version-id")
			}

			if len(updates) == 0 {
				return fmt.Errorf("no update fields provided")
			}

			if err := repo.UpdateModFields(modID, updates); err != nil {
				return err
			}
			fmt.Println("Updated mod:", modID)
			return nil
		},
	}
}
