package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) newModDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a mod",
		ArgsUsage: "<mod-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("mod-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			if err := repo.DeleteMod(c.Args().First()); err != nil {
				return err
			}
			fmt.Println("Deleted mod:", c.Args().First())
			return nil
		},
	}
}
