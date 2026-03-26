package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

func requireDB(cmd *cli.Command) error {
	if cmd.String("db") == "" {
		return fmt.Errorf("db required (set --db or DATABASE_URL)")
	}
	return nil
}
