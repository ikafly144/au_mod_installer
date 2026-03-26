package musmgr

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func requireDB(c *cli.Context) error {
	if c.String("db") == "" {
		return fmt.Errorf("db required (set --db or DATABASE_URL)")
	}
	return nil
}
