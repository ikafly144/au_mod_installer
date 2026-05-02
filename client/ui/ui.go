//go:build windows

package ui

import (
	"log/slog"
	"runtime"

	"github.com/ikafly144/au_mod_installer/client/ui/tab/launcher"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/repo"
	servertab "github.com/ikafly144/au_mod_installer/client/ui/tab/server"
	"github.com/ikafly144/au_mod_installer/client/ui/tab/settings"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type Config struct {
	stateOptions []uicommon.Option
	stateInits   []func(*uicommon.State)
}

func WithStateOptions(options ...uicommon.Option) func(*Config) {
	return func(cfg *Config) {
		cfg.stateOptions = options
	}
}

func WithStateInit(init func(*uicommon.State)) func(*Config) {
	return func(cfg *Config) {
		cfg.stateInits = append(cfg.stateInits, init)
	}
}

func Main(w fyne.Window, version string, sharedURI string, sharedArchive string, cfg ...func(*Config)) error {
	var config Config

	for _, c := range cfg {
		c(&config)
	}

	state, err := uicommon.NewState(w, version, config.stateOptions...)
	if err != nil {
		return err
	}
	state.SharedURI = sharedURI
	state.SharedArchive = sharedArchive

	for _, init := range config.stateInits {
		init(state)
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

	s := settings.NewSettings(state)
	settingsTab, err := s.Tab()
	if err != nil {
		return err
	}

	st := servertab.NewServerTab(state)
	serverTab, err := st.Tab()
	if err != nil {
		return err
	}

	canvas := container.NewAppTabs(
		launcherTab,
		repoTab,
		serverTab,
		settingsTab,
	)
	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		if state.OnDroppedURIs != nil {
			state.OnDroppedURIs(uris)
		}
	})
	w.SetContent(canvas)
	w.SetFixedSize(false)
	w.Show()
	if _, err := state.EnableNativeCustomWindowFrame(); err != nil {
		slog.Warn("Failed to enable native custom window frame", "error", err)
	}
	onClosed := func() {
		uicommon.SaveMainWindowSize(w)
	}
	if cleanup, err := state.EnableNativeTextDrop(); err != nil {
		slog.Warn("Failed to enable native OLE text drop", "error", err)
	} else {
		onClosed = func() {
			uicommon.SaveMainWindowSize(w)
			cleanup()
		}
	}
	w.SetOnClosed(onClosed)
	fyne.Do(func() {
		if uicommon.RestoreMainWindowSize(w) {
			w.CenterOnScreen()
		}
	})
	runtime.LockOSThread()
	fyne.CurrentApp().Run()
	return nil
}
