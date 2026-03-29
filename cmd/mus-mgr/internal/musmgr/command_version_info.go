package musmgr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newVersionInfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Get details of a mod version",
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

			modVersion, err := repo.GetModVersionDetails(cmd.Args().Get(0), cmd.Args().Get(1))
			if err != nil {
				return err
			}
			b, err := json.Marshal(modVersion)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}),
	}
}
