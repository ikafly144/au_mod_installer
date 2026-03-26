package musmgr

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) newVersionInfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Get details of a mod version",
		ArgsUsage: "<mod-id> <version-id>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("mod-id and version-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			modVersion, err := repo.GetModVersionDetails(c.Args().Get(0), c.Args().Get(1))
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(modVersion, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		},
	}
}
