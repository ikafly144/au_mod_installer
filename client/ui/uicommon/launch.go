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

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
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

	// Apply mods if a profile is selected
	activeProfileIDStr, _ := s.ActiveProfile.Get()
	activeProfileID, err := uuid.Parse(activeProfileIDStr)
	if err != nil {
		slog.Warn("Failed to parse active profile ID", "error", err)
		activeProfileID = uuid.Nil
	}
	var restoreInfo *modmgr.RestoreInfo
	if activeProfileID != uuid.Nil {
		profile, found := s.ProfileManager.Get(activeProfileID)
		if found {
			configDir, err := os.UserConfigDir()
			if err != nil {
				s.SetError(err)
				return
			}
			cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")
			binaryType, err := aumgr.GetBinaryType(path)
			if err != nil {
				s.SetError(err)
				return
			}

			fyne.Do(func() {
				s.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("launch.applying_mods", "Applying mods...")},
				}
				s.ErrorText.Refresh()
				s.ErrorText.Show()
			})

			restoreInfo, err = modmgr.ApplyMods(path, cacheDir, profile.Versions(), binaryType)
			if err != nil {
				s.SetError(err)
				return
			}
			defer func() {
				fyne.Do(func() {
					s.ErrorText.Segments = []widget.RichTextSegment{
						&widget.TextSegment{Text: lang.LocalizeKey("launch.restoring", "Restoring game files...")},
					}
					s.ErrorText.Refresh()
					s.ErrorText.Show()
				})
				if err := modmgr.RestoreGame(path, restoreInfo); err != nil {
					slog.Error("Failed to restore game", "error", err)
					s.SetError(err)
				} else {
					fyne.Do(func() {
						s.ErrorText.Hide()
					})
				}
			}()
		}
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
