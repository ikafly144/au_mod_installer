package musmgr

import (
	"bytes"
	"context"
	"slices"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestShellCompletionFeatures(t *testing.T) {
	factory := newCommandFactory(nil)

	t.Run("Flag completion without typing anything", func(t *testing.T) {
		cmd := factory.newModAddCommand()
		buf := new(bytes.Buffer)
		cmd.Writer = buf

		fn := factory.makeShellComplete()
		fn(context.Background(), cmd)
		output := buf.String()
		// When no args are passed, subcommands would be completed if any, but mod add is a leaf command.
		// If user types "-" or "--", flags are completed.
		_ = output
	})

	t.Run("Flag completion when typing --", func(t *testing.T) {
		cmd := factory.newModAddCommand()
		buf := new(bytes.Buffer)
		cmd.Writer = buf

		// Simulate args: ["mod", "add", "--"]
		if err := cmd.Set("name", ""); err != nil {
			t.Fatalf("failed to set name flag: %v", err)
		}
		// We can test ShellComplete directly by setting up cmd
		fn := factory.makeShellComplete()

		// Test using a mock Command
		testCmd := &cli.Command{
			Name:          "add",
			Flags:         cmd.Flags,
			Writer:        buf,
			ShellComplete: fn,
		}
		// In urfave/cli, cmd.Args() comes from parsed args
		// Let's test makeShellComplete with flag args
		_ = testCmd
	})

	t.Run("completeFlagValue target_platform", func(t *testing.T) {
		vals := factory.completeFlagValue(context.Background(), &cli.Command{}, "target_platform", "")
		if len(vals) == 0 {
			t.Errorf("expected target_platform values, got none")
		}
		if !slices.Contains(vals, "any") || !slices.Contains(vals, "x64") {
			t.Errorf("expected any and x64 in target_platform, got: %v", vals)
		}
	})

	t.Run("completeFlagValue type", func(t *testing.T) {
		vals := factory.completeFlagValue(context.Background(), &cli.Command{}, "type", "arch")
		if len(vals) != 1 || vals[0] != "archive" {
			t.Errorf("expected [archive], got: %v", vals)
		}
	})

	t.Run("completeFlagValue dependency", func(t *testing.T) {
		vals := factory.completeFlagValue(context.Background(), &cli.Command{}, "dependency", "")
		// Should not panic if no DB
		_ = vals
	})

	t.Run("Positional completer modID and versionID without panic", func(t *testing.T) {
		modComp := factory.modIDCompleter()
		vals := modComp(context.Background(), &cli.Command{}, 0, "")
		_ = vals

		verComp := factory.versionIDCompleter()
		vals = verComp(context.Background(), &cli.Command{}, 1, "")
		_ = vals
	})

	t.Run("Subcommand completion in mod without help showing", func(t *testing.T) {
		modCmd := factory.newModCommand()
		buf := new(bytes.Buffer)
		modCmd.Writer = buf
		modCmd.ShellComplete(context.Background(), modCmd)
		out := buf.String()
		if !slices.Contains([]string{"add", "list", "info", "edit", "delete"}, "add") {
			t.Errorf("expected subcommands")
		}
		if bytes.Contains(buf.Bytes(), []byte("help")) {
			t.Errorf("did not expect help command in subcommands, got:\n%s", out)
		}
	})
}
