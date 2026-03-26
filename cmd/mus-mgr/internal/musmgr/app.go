package musmgr

import "github.com/urfave/cli/v2"

func NewApp() *cli.App {
	var dbURL string
	factory := newCommandFactory(&dbURL)

	return &cli.App{
		Name:                 "mus-mgr",
		Usage:                "Manage the au_mod_installer server database",
		EnableBashCompletion: true,
		Suggest:              true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "db",
				Usage:       "PostgreSQL connection string",
				EnvVars:     []string{"DATABASE_URL"},
				Destination: &dbURL,
			},
		},
		Commands: []*cli.Command{
			factory.newMigrateCommand(),
			factory.newModCommand(),
			factory.newVersionCommand(),
		},
	}
}
