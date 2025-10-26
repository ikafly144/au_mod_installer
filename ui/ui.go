package ui

import (
	"au_mod_installer/ui/common"
	"au_mod_installer/ui/tab/installer"
	"au_mod_installer/ui/tab/launcher"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func Main(w fyne.Window, version string) error {
	state, err := common.NewState(w, version)
	if err != nil {
		return err
	}

	if err := state.FetchMods(); err != nil {
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

	canvas := container.NewAppTabs(
		launcherTab,
		installerTab,
	)
	w.SetContent(canvas)
	w.CenterOnScreen()
	w.Resize(fyne.NewSize(400, 600))
	w.SetFixedSize(true)
	w.ShowAndRun()
	return nil
}
