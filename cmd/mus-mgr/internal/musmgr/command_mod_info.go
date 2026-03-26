package musmgr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newModInfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Get details of a mod",
		ArgsUsage: "<mod-id>",
		ShellComplete: func(ctx context.Context, cmd *cli.Command) {
			if cmd.NArg() <= 1 {
				f.printModIDCompletions(cmd)
			}
			cli.DefaultCompleteWithFlags(ctx, cmd)
		},
		Action: wrapAction(func(ctx context.Context, cmd *cli.Command) error {
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

			mod, err := repo.GetModDetails(cmd.Args().First())
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(mod, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}),
	}
}
