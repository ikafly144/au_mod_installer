package musmgr

import "github.com/urfave/cli/v3"

func (f *commandFactory) newVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Manage mod versions",
		Commands: []*cli.Command{
			f.newVersionAddCommand(),
			f.newVersionListCommand(),
			f.newVersionInfoCommand(),
			f.newVersionEditCommand(),
			f.newVersionDeleteCommand(),
		},
	}
}
