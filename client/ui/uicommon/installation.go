package uicommon

import (
	"log/slog"
	"path/filepath"

	"github.com/ikafly144/modrepo/pkg/repomgr"
)

func (i *State) selectLauncher(s string) {
	i.ErrorText.Hide()
	if repomgr.LauncherFromString(s) != repomgr.LauncherUnknown {
		_ = i.SelectedGamePath.Set(i.DetectedGamePath)
	} else {
		beforePath, err := i.SelectedGamePath.Get()
		if err != nil {
			slog.Warn("Failed to get selected game path", "error", err)
		}
		beforeType := i.Core.DetectLauncherType(beforePath)
		path, err := i.ExplorerOpenFile("R.E.P.O.", repomgr.ExecutableName)
		if err != nil {
			slog.Info("File selection cancelled or failed", "error", err)
			i.InstallSelect.Selected = beforeType.String()
			return
		}
		slog.Info("User selected game path", "path", path)
		l := i.Core.DetectLauncherType(path)
		_ = i.SelectedGamePath.Set(filepath.Dir(path))
		i.InstallSelect.Selected = l.String()
	}
}
