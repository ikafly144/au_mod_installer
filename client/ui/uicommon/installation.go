package uicommon

import (
	"log/slog"
	"path/filepath"

	"github.com/ikafly144/au_mod_installer/client/core"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

func (s *State) CheckInstalled() bool {
	if err := s.ModInstalled.Set(false); err != nil {
		slog.Warn("Failed to set modInstalled", "error", err)
	}
	path, err := s.SelectedGamePath.Get()
	if err != nil || path == "" {
		return false
	}

	status := s.Core.GetInstallationStatus(path, false)
	isInstalled := status.Status != core.StatusNotInstalled

	if err := s.ModInstalled.Set(isInstalled); err != nil {
		slog.Warn("Failed to set modInstalled", "error", err)
	}

	return isInstalled
}

func (i *State) selectLauncher(s string) {
	i.ErrorText.Hide()
	if aumgr.LauncherFromString(s) != aumgr.LauncherUnknown {
		_ = i.SelectedGamePath.Set(i.DetectedGamePath)
	} else {
		beforePath, err := i.SelectedGamePath.Get()
		if err != nil {
			slog.Warn("Failed to get selected game path", "error", err)
		}
		beforeType := i.Core.DetectLauncherType(beforePath)
		path, err := i.ExplorerOpenFile("Among Us", "Among Us.exe")
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
	i.CheckInstalled()
}
