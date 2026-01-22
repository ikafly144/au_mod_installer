package uicommon

import (
	"errors"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"
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

	activeProfileIDStr, _ := s.ActiveProfile.Get()
	activeProfileID, err := uuid.Parse(activeProfileIDStr)
	if err != nil {
		slog.Warn("Failed to parse active profile ID", "error", err)
		activeProfileID = uuid.Nil
	}

	if activeProfileID != uuid.Nil {
		fyne.Do(func() {
			s.ErrorText.Segments = []widget.RichTextSegment{
				&widget.TextSegment{Text: lang.LocalizeKey("launch.applying_mods", "Applying mods...")},
			}
			s.ErrorText.Refresh()
			s.ErrorText.Show()
		})
	}

	cleanup, err := s.Core.PrepareLaunch(path, activeProfileID)
	if err != nil {
		slog.Error("Failed to prepare launch", "error", err)
		s.SetError(err)
		return
	}

	defer func() {
		if activeProfileID != uuid.Nil {
			fyne.Do(func() {
				s.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("launch.restoring", "Restoring game files...")},
				}
				s.ErrorText.Refresh()
				s.ErrorText.Show()
			})
		}
		if err := cleanup(); err != nil {
			slog.Error("Failed to restore game", "error", err)
			s.SetError(err)
		} else {
			fyne.Do(func() {
				s.ErrorText.Hide()
			})
		}
	}()

	// Re-set "running" message if we changed it to "applying mods"
	if activeProfileID != uuid.Nil {
		fyne.Do(func() {
			s.ErrorText.Segments = []widget.RichTextSegment{
				&widget.TextSegment{Text: lang.LocalizeKey("launch.running", "Among Us is currently running...")},
			}
			s.ErrorText.Refresh()
			s.ErrorText.Show()
		})
	}

	if err := s.Core.ExecuteLaunch(path); err != nil {
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