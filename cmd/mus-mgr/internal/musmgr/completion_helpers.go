package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func (f *commandFactory) printModIDCompletions(c *cli.Context) {
	repo, err := f.newRepository()
	if err != nil {
		return
	}

	ids, _, err := repo.GetModIds("", 500)
	if err != nil {
		return
	}
	for _, id := range ids {
		fmt.Fprintln(c.App.Writer, id)
	}
}

func (f *commandFactory) printVersionIDCompletions(c *cli.Context, modID string) {
	repo, err := f.newRepository()
	if err != nil {
		return
	}

	ids, err := repo.GetModVersionIds(modID)
	if err != nil {
		return
	}
	for _, id := range ids {
		fmt.Fprintln(c.App.Writer, id)
	}
}
