package musmgr

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newVersionDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a mod version",
		ArgsUsage: "<mod-id> <version-id>",
		ShellComplete: func(ctx context.Context, cmd *cli.Command) {
			if cmd.NArg() <= 1 {
				f.printModIDCompletions(cmd)
			}
			if cmd.NArg() <= 2 {
				f.printVersionIDCompletions(cmd, cmd.Args().Get(0))
			}
			cli.DefaultCompleteWithFlags(ctx, cmd)
		},
		Action: wrapAction(func(ctx context.Context, cmd *cli.Command) error {
			if err := requireDB(cmd); err != nil {
				return err
			}
			if cmd.NArg() < 2 {
				return fmt.Errorf("mod-id and version-id required")
			}

			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			if err := repo.DeleteModVersion(cmd.Args().Get(0), cmd.Args().Get(1)); err != nil {
				return err
			}
			fmt.Printf("Deleted version %s from mod %s\n", cmd.Args().Get(1), cmd.Args().Get(0))
			return nil
		}),
	}
}
