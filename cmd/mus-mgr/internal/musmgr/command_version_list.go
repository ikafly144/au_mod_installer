package musmgr

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newVersionListCommand() *cli.Command {
	return &cli.Command{
		Name:          "list",
		Usage:         "List versions for a mod",
		ArgsUsage:     "<mod-id>",
		ShellComplete: f.makeShellComplete(f.modIDCompleter()),
		Action: func(ctx context.Context, cmd *cli.Command) error {
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

			ids, err := repo.GetModVersionIds(cmd.Args().First())
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
