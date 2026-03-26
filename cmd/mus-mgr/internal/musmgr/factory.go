package musmgr

import (
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

func (f *commandFactory) newRepository() (*gormrepo.GormRepository, error) {
	db, err := gorm.Open(postgres.Open(*f.dbURL))
	if err != nil {
		return nil, err
	}

	return gormrepo.NewGormRepository(db), nil
}
