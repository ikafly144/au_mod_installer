package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

func (f *commandFactory) printModIDCompletions(cmd *cli.Command) {
	repo, err := f.newRepository()
	if err != nil {
		return
	}

	ids, _, err := repo.GetModIds("", 500)
	if err != nil {
		return
	}
	for _, id := range ids {
		_, _ = fmt.Fprintln(cmd.Writer, id)
	}
}

func (f *commandFactory) printVersionIDCompletions(cmd *cli.Command, modID string) {
	repo, err := f.newRepository()
	if err != nil {
		return
	}

	ids, err := repo.GetModVersionIds(modID)
	if err != nil {
		return
	}
	for _, id := range ids {
		_, _ = fmt.Fprintln(cmd.Writer, id)
	}
}
