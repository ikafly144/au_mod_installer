package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) newMigrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Migrate the database schema",
		Action: func(c *cli.Context) error {
			if err := requireDB(c); err != nil {
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
		},
	}
}
