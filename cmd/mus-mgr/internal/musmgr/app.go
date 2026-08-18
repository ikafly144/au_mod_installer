package musmgr

import "github.com/urfave/cli/v3"

func NewApp() *cli.Command {
	var dbURL string
	factory := newCommandFactory(&dbURL)

	return &cli.Command{
		Name:                  "mus-mgr",
		Usage:                 "Manage the au_mod_installer server database",
		EnableShellCompletion: true,
		Suggest:               true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "db",
				Usage:       "PostgreSQL connection string",
				Sources:     cli.EnvVars("DATABASE_URL"),
				Destination: &dbURL,
			},
		},
		ShellComplete: factory.makeShellComplete(),
		Commands: []*cli.Command{
			factory.newMigrateCommand(),
			factory.newModCommand(),
			factory.newVersionCommand(),
		},
	}
}
