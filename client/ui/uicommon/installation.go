package uicommon

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func (s *State) CheckInstalled() bool {
	if err := s.ModInstalled.Set(false); err != nil {
		slog.Warn("Failed to set modInstalled", "error", err)
	}
	path, err := s.SelectedGamePath.Get()
	if err != nil || path == "" {
		return false
	}
	modInstallLocation, err := os.OpenRoot(s.ModInstallDir())
	if err != nil {
		slog.Warn("Failed to open game root", "error", err)
		return false
	}
	if _, err := modInstallLocation.Stat(modmgr.InstallationInfoFileName); err == nil || !os.IsNotExist(err) {
		if err := s.ModInstalled.Set(true); err != nil {
			slog.Warn("Failed to set modInstalled", "error", err)
		}
	} else {
		if err := s.ModInstalled.Set(false); err != nil {
			slog.Warn("Failed to set modInstalled", "error", err)
		}
	}
	ok, err := s.ModInstalled.Get()
	if err != nil {
		slog.Warn("Failed to get modInstalled", "error", err)
		return false
	}
	return ok
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
		beforeType := aumgr.DetectLauncherType(beforePath)
		path, err := i.ExplorerOpenFile("Among Us", "Among Us.exe")
		if err != nil {
			slog.Info("File selection cancelled or failed", "error", err)
			i.InstallSelect.Selected = beforeType.String()
			return
		}
		slog.Info("User selected game path", "path", path)
		l := aumgr.DetectLauncherType(path)
		_ = i.SelectedGamePath.Set(filepath.Dir(path))
		i.InstallSelect.Selected = l.String()
	}
	i.CheckInstalled()
}
