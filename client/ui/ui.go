package ui

import (
	"fmt"
	"runtime/debug"
	"slices"

	"github.com/ikafly144/au_mod_installer/client/ui/tab/installer"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/launcher"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/repo"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type Config struct {
	stateOptions []uicommon.Option
}

func WithStateOptions(options ...uicommon.Option) func(*Config) {
	return func(cfg *Config) {
		cfg.stateOptions = options
	}
}

func Main(w fyne.Window, version string, cfg ...func(*Config)) error {
	var config Config

	for _, c := range cfg {
		c(&config)
	}

	info, ok := debug.ReadBuildInfo()
	if ok {
		fmt.Println(info)
	}
	vscIdx := slices.IndexFunc(info.Settings, func(b debug.BuildSetting) bool {
		return b.Key == "vcs.revision"
	})
	if vscIdx != -1 {
		version += " (" + info.Settings[vscIdx].Value[:min(len(info.Settings[vscIdx].Value), 7)] + ")"
	} else {
		version = "(devel)"
	}

	state, err := uicommon.NewState(w, version, config.stateOptions...)
	if err != nil {
		return err
	}

	state.CheckInstalled()

	i := installer.NewInstallerTab(state)
	installerTab, err := i.Tab()
	if err != nil {
		return err
	}

	l := launcher.NewLauncherTab(state)
	launcherTab, err := l.Tab()
	if err != nil {
		return err
	}

	r := repo.NewRepository(state)
	repoTab, err := r.Tab()
	if err != nil {
		return err
	}

	canvas := container.NewAppTabs(
		launcherTab,
		installerTab,
		repoTab,
	)
	w.SetContent(canvas)
	w.CenterOnScreen()
	w.Resize(fyne.NewSize(440, 720))
	w.SetFixedSize(true)
	w.ShowAndRun()
	return nil
}
