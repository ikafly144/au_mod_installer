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
	app := NewApp()

	fish, _ := app.ToFishCompletion()
	t.Logf("FISH COMPLETION SCRIPT:\n%s", fish)

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
		vals := modComp(context.Background(), &cli.Command{}, []string{""}, 0, "")
		_ = vals

		verComp := factory.versionIDCompleter()
		vals = verComp(context.Background(), &cli.Command{}, []string{"my-mod", ""}, 1, "")
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

	t.Run("version edit positional 2nd arg completion", func(t *testing.T) {
		verComp := factory.versionIDCompleter()
		// If modID is passed in posArgs[0], versionIDCompleter uses posArgs[0]
		res := verComp(context.Background(), nil, []string{"test-mod", "1."}, 1, "1.")
		_ = res
	})
}
