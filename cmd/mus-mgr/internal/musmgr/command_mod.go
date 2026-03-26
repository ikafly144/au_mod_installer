package musmgr

import "github.com/urfave/cli/v2"

func (f *commandFactory) newModCommand() *cli.Command {
	return &cli.Command{
		Name:  "mod",
		Usage: "Manage mods",
		Subcommands: []*cli.Command{
			f.newModAddCommand(),
			f.newModListCommand(),
			f.newModInfoCommand(),
			f.newModEditCommand(),
			f.newModDeleteCommand(),
		},
	}
}
