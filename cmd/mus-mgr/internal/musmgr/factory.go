package musmgr

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/urfave/cli/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	gormrepo "github.com/ikafly144/au_mod_installer/server/repository/gorm"
)

type commandFactory struct {
	dbURL *string
}

func newCommandFactory(dbURL *string) *commandFactory {
	return &commandFactory{dbURL: dbURL}
}

func wrapAction(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		if slices.Contains(os.Args, "--generate-shell-completion") {
			for _, f := range cmd.VisibleFlags() {
				for _, name := range f.Names() {
					fmt.Println("--" + name)
				}
			}
			return nil
		}
		return action(ctx, cmd)
	}
}

func (f *commandFactory) newRepository() (*gormrepo.GormRepository, error) {
	db, err := gorm.Open(postgres.Open(*f.dbURL))
	if err != nil {
		return nil, err
	}

	return gormrepo.NewGormRepository(db), nil
}
