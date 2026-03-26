package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) newVersionDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a mod version",
		ArgsUsage: "<mod-id> <version-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("mod-id and version-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			if err := repo.DeleteModVersion(c.Args().Get(0), c.Args().Get(1)); err != nil {
				return err
			}
			fmt.Printf("Deleted version %s from mod %s\n", c.Args().Get(1), c.Args().Get(0))
			return nil
		},
	}
}
