package musmgr

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newMigrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Migrate the database schema",
		Action: wrapAction(func(ctx context.Context, cmd *cli.Command) error {
			if err := requireDB(cmd); err != nil {
				return err
			}
			repo, err := f.newRepository()
			if err != nil {
				return err
			}
			if err := repo.Migrate(); err != nil {
				return err
			}
			fmt.Println("Migration successful.")
			return nil
		}),
	}
}
