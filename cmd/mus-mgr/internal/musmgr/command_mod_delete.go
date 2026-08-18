package musmgr

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newModDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:          "delete",
		Usage:         "Delete a mod",
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

			if err := repo.DeleteMod(cmd.Args().First()); err != nil {
				return err
			}
			fmt.Println("Deleted mod:", cmd.Args().First())
			return nil
		},
	}
}
