package musmgr

import "github.com/urfave/cli/v3"

func (f *commandFactory) newModCommand() *cli.Command {
	return &cli.Command{
		Name:  "mod",
		Usage: "Manage mods",
		Commands: []*cli.Command{
			f.newModAddCommand(),
			f.newModListCommand(),
			f.newModInfoCommand(),
			f.newModEditCommand(),
			f.newModDeleteCommand(),
		},
	}
}
