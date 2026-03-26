package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) newVersionListCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List versions for a mod",
		ArgsUsage: "<mod-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("mod-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			ids, err := repo.GetModVersionIds(c.Args().First())
			if err != nil {
				return err
			}
			for _, id := range ids {
				fmt.Println(id)
			}
			return nil
		},
	}
}
