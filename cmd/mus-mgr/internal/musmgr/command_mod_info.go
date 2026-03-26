package musmgr

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) newModInfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Get details of a mod",
		ArgsUsage: "<mod-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("mod-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			mod, err := repo.GetModDetails(c.Args().First())
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(mod, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		},
	}
}
