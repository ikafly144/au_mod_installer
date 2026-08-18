package musmgr

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ikafly144/au_mod_installer/server/model"
)

// PositionalCompleter provides completion candidates for a positional argument index.
type PositionalCompleter func(ctx context.Context, cmd *cli.Command, argIndex int, prefix string) []string

func (f *commandFactory) makeShellComplete(completers ...PositionalCompleter) cli.ShellCompleteFunc {
	return func(ctx context.Context, cmd *cli.Command) {
		if cmd == nil {
			return
		}
		var args []string
		if cmd.Args() != nil {
			args = cmd.Args().Slice()
		}
		var lastArg string
		if len(args) > 0 {
			lastArg = args[len(args)-1]
		}

		// 1. Check if previous argument was a flag expecting a value
		if len(args) >= 2 {
			prevArg := args[len(args)-2]
			if strings.HasPrefix(prevArg, "-") && isFlagWithValue(cmd, prevArg) {
				if values := f.completeFlagValue(ctx, cmd, prevArg, lastArg); len(values) > 0 {
					for _, val := range values {
						_, _ = fmt.Fprintln(cmd.Writer, val)
					}
					return
				}
			}
		}

		// 2. Check for --flag=value inline completion
		if strings.HasPrefix(lastArg, "-") && strings.Contains(lastArg, "=") {
			parts := strings.SplitN(lastArg, "=", 2)
			flagName := parts[0]
			valPrefix := parts[1]
			if values := f.completeFlagValue(ctx, cmd, flagName, valPrefix); len(values) > 0 {
				for _, val := range values {
					_, _ = fmt.Fprintln(cmd.Writer, flagName+"="+val)
				}
				return
			}
		}

		// 3. If typing a flag name (starts with "-")
		if strings.HasPrefix(lastArg, "-") {
			for _, flag := range cmd.VisibleFlags() {
				for _, name := range flag.Names() {
					candidate := "--" + name
					if len(name) == 1 {
						candidate = "-" + name
					}
					if strings.HasPrefix(candidate, lastArg) {
						_, _ = fmt.Fprintln(cmd.Writer, candidate)
					}
				}
			}
			return
		}

		// 4. Positional argument completion
		var posArgs []string
		for i := 0; i < len(args); i++ {
			a := args[i]
			if strings.HasPrefix(a, "-") {
				if isFlagWithValue(cmd, a) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
				}
				continue
			}
			posArgs = append(posArgs, a)
		}

		posIndex := max(len(posArgs)-1, 0)

		prefix := ""
		if len(posArgs) > 0 {
			prefix = posArgs[len(posArgs)-1]
		}

		if posIndex < len(completers) && completers[posIndex] != nil {
			candidates := completers[posIndex](ctx, cmd, posIndex, prefix)
			for _, c := range candidates {
				if prefix == "" || strings.HasPrefix(c, prefix) {
					_, _ = fmt.Fprintln(cmd.Writer, c)
				}
			}
			return
		}

		// If no positional completer matched and command has subcommands, complete subcommands
		if len(cmd.Commands) > 0 {
			for _, sub := range cmd.VisibleCommands() {
				if sub.Name == "help" || sub.Name == "h" {
					continue
				}
				if prefix == "" || strings.HasPrefix(sub.Name, prefix) {
					_, _ = fmt.Fprintln(cmd.Writer, sub.Name)
				}
			}
		}
	}
}

func isFlagWithValue(cmd *cli.Command, flagArg string) bool {
	cleanName := strings.TrimLeft(strings.SplitN(flagArg, "=", 2)[0], "-")
	for _, fl := range cmd.VisibleFlags() {
		for _, name := range fl.Names() {
			if name == cleanName {
				if _, ok := fl.(*cli.BoolFlag); ok {
					return false
				}
				return true
			}
		}
	}
	return false
}

func (f *commandFactory) completeFlagValue(ctx context.Context, cmd *cli.Command, flagName, valPrefix string) []string {
	cleanName := strings.TrimLeft(strings.SplitN(flagName, "=", 2)[0], "-")
	switch cleanName {
	case "mod-id", "mod":
		return f.getModIDs(valPrefix)
	case "latest-version-id", "version-id", "version":
		modID := ""
		if cmd != nil {
			modID = cmd.String("mod-id")
			if modID == "" && cmd.Args() != nil && cmd.NArg() > 0 {
				modID = cmd.Args().Get(0)
			}
		}
		if modID != "" {
			return f.getVersionIDs(modID, valPrefix)
		}
		return f.getModIDs(valPrefix)
	case "target_platform", "target-platform":
		platforms := []string{
			string(model.TargetPlatformAny),
			string(model.TargetPlatformX64),
			string(model.TargetPlatformX86),
			string(model.TargetPlatformAArch64),
		}
		var matches []string
		for _, p := range platforms {
			if valPrefix == "" || strings.HasPrefix(p, valPrefix) {
				matches = append(matches, p)
			}
		}
		return matches
	case "type":
		types := []string{
			string(model.FileTypeArchive),
			string(model.FileTypePluginDll),
			string(model.FileTypeBinary),
		}
		var matches []string
		for _, t := range types {
			if valPrefix == "" || strings.HasPrefix(t, valPrefix) {
				matches = append(matches, t)
			}
		}
		return matches
	case "feature":
		features := []string{
			"direct_join=true",
			"direct_join=false",
		}
		var matches []string
		for _, feat := range features {
			if valPrefix == "" || strings.HasPrefix(feat, valPrefix) {
				matches = append(matches, feat)
			}
		}
		return matches
	case "dependency":
		modIDs := f.getModIDs(valPrefix)
		var matches []string
		for _, id := range modIDs {
			matches = append(matches, id+":")
		}
		return matches
	}
	return nil
}

func (f *commandFactory) modIDCompleter() PositionalCompleter {
	return func(ctx context.Context, cmd *cli.Command, argIndex int, prefix string) []string {
		return f.getModIDs(prefix)
	}
}

func (f *commandFactory) versionIDCompleter() PositionalCompleter {
	return func(ctx context.Context, cmd *cli.Command, argIndex int, prefix string) []string {
		if cmd == nil || cmd.Args() == nil {
			return nil
		}
		modID := cmd.Args().Get(0)
		if modID == "" {
			return nil
		}
		return f.getVersionIDs(modID, prefix)
	}
}

func (f *commandFactory) getModIDs(prefix string) []string {
	repo, err := f.newRepository()
	if err != nil {
		return nil
	}
	ids, _, err := repo.GetModIds("", 500)
	if err != nil {
		return nil
	}
	var res []string
	for _, id := range ids {
		if prefix == "" || strings.HasPrefix(id, prefix) {
			res = append(res, id)
		}
	}
	return res
}

func (f *commandFactory) getVersionIDs(modID, prefix string) []string {
	repo, err := f.newRepository()
	if err != nil {
		return nil
	}
	ids, err := repo.GetModVersionIds(modID)
	if err != nil {
		return nil
	}
	var res []string
	for _, id := range ids {
		if prefix == "" || strings.HasPrefix(id, prefix) {
			res = append(res, id)
		}
	}
	return res
}
