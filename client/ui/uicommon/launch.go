package uicommon

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

func (s *State) Launch(path string) {
	if !s.launchLock.TryLock() {
		s.SetError(errors.New(lang.LocalizeKey("error.game_already_running", "Already running.")))
		return
	}
	defer s.launchLock.Unlock()
	fyne.Do(func() {
		s.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("launch.running", "Among Us is currently running...")},
		}
		s.ErrorText.Refresh()
		s.ErrorText.Show()
	})
	if _, err := os.Stat(filepath.Join(path, "Among Us.exe")); os.IsNotExist(err) {
		fyne.Do(func() {
			s.ErrorText.Segments = []widget.RichTextSegment{
				&widget.TextSegment{Text: lang.LocalizeKey("launch.error.executable_not_found", "Among Us executable not found: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
				&widget.TextSegment{Text: lang.LocalizeKey("launch.error.reinstall_instruction", "Please uninstall the mod and reinstall Among Us.")},
			}
			s.ErrorText.Refresh()
		})
		slog.Warn("Among Us executable not found", "error", err)
		return
	}
	if err := aumgr.LaunchAmongUs(aumgr.DetectLauncherType(path), path, s.ModInstallDir()); err != nil {
		fyne.Do(func() {
			s.ErrorText.Segments = []widget.RichTextSegment{
				&widget.TextSegment{Text: lang.LocalizeKey("launch.error.launch_failed", "Failed to launch Among Us: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
			}
			s.ErrorText.Refresh()
		})
		slog.Warn("Failed to launch Among Us", "error", err)
		return
	} else {
		fyne.Do(func() {
			s.ErrorText.Hide()
		})
	}
	_ = s.CanLaunch.Set(true)
	_ = s.CanInstall.Set(true)
}
