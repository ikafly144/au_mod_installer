package musmgr

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) newModListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List mods",
		Action: wrapAction(func(ctx context.Context, cmd *cli.Command) error {
			if err := requireDB(cmd); err != nil {
				return err
			}
			repo, err := f.newRepository()
			if err != nil {
				return err
			}

			ids, _, err := repo.GetModIds("", 100)
			if err != nil {
				return err
			}
			for _, id := range ids {
				mod, err := repo.GetModDetails(id)
				if err != nil {
					continue
				}
				fmt.Printf("%s\t%s\t%s\n", mod.ID, mod.Name, mod.Author)
			}
			return nil
		}),
	}
}
