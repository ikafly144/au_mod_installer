package uicommon

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/mod/semver"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/common/versioning"
)

func FindBranchVersion(info *restcommon.VersionInfo, branch string) string {
	return versioning.FindBranchVersion(info, branch)
}

func (s *State) CheckForUpdates(ctx context.Context, interactive bool) {
	if s.Rest == nil {
		if interactive {
			s.ShowErrorDialog(errors.New(lang.LocalizeKey("update.error.offline", "Cannot check for updates in offline mode.")))
		}
		return
	}

	branchName := "stable"
	if app := fyne.CurrentApp(); app != nil && app.Preferences() != nil {
		branchName = app.Preferences().StringWithFallback("core.update_branch", "stable")
	}
	branch := versioning.BranchFromString(branchName)

	info, err := s.Rest.GetVersionInfo()
	if err != nil {
		slog.Error("Failed to check for updates via server", "error", err)
		if interactive {
			s.ShowErrorDialog(errors.New(lang.LocalizeKey("update.check_failed", "Failed to check for updates: {{.Error}}", map[string]any{"Error": err.Error()})))
		}
		return
	}

	tag := FindBranchVersion(info, branch.String())
	stable := FindBranchVersion(info, versioning.BranchStable.String())
	if tag != "" && semver.Compare(tag, s.Version) <= 0 {
		tag = ""
	}

	if tag != "" {
		slog.Info("Update available", "version", tag, "current", s.Version)
		isMandatory := s.Version != "(devel)" && semver.Prerelease(tag) == "" && stable != "" && semver.Compare(stable, s.Version) > 0
		s.ShowUpdateDialog(tag, isMandatory)
	} else {
		slog.Info("No updates available", "current", s.Version)
		if interactive {
			s.ShowInfoDialog(
				lang.LocalizeKey("update.title", "Update"),
				lang.LocalizeKey("update.latest", "You are using the latest version ({{.Version}}).", map[string]any{"Version": s.Version}),
			)
		}
	}
}

func (s *State) ShowUpdateDialog(tag string, isMandatory bool) {
	if s.Window == nil {
		return
	}

	fyne.Do(func() {
		s.dialogLock.Lock()
		if s.activeDialog != nil {
			s.dialogLock.Unlock()
			return
		}
		s.dialogLock.Unlock()

		confirmMsg := lang.LocalizeKey("update.available", "New version \"{{.Version}}\" is available. Do you want to update now?", map[string]any{"Version": tag})

		confirmDialog := dialog.NewConfirm(
			lang.LocalizeKey("update.title", "Update Available"),
			confirmMsg,
			func(yes bool) {
				if yes {
					s.PerformUpdate(tag)
				} else if isMandatory {
					s.showMandatoryUpdateRequiredDialog()
				}
			},
			s.Window,
		)
		confirmDialog.SetDismissText(lang.LocalizeKey("update.later", "Later"))
		confirmDialog.SetConfirmText(lang.LocalizeKey("update.now", "Update Now"))

		var d dialog.Dialog = confirmDialog

		s.dialogLock.Lock()
		s.activeDialog = d
		s.dialogLock.Unlock()

		d.SetOnClosed(func() {
			s.dialogLock.Lock()
			if s.activeDialog == d {
				s.activeDialog = nil
			}
			s.dialogLock.Unlock()
		})

		d.Show()
	})
}

func (s *State) PerformUpdate(tag string) {
	if s.Window == nil {
		return
	}

	statusLabel := widget.NewLabel(lang.LocalizeKey("update.downloading", "Downloading and applying update..."))
	progressBar := widget.NewProgressBarInfinite()
	content := container.NewVBox(
		statusLabel,
		progressBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons(
		lang.LocalizeKey("update.title", "Update"),
		content,
		s.Window,
	)
	progressDialog.Show()

	go func() {
		installerLaunched, err := versioning.Update(context.Background(), tag)
		if err != nil {
			slog.Error("Failed to update", "error", err)
			fyne.Do(func() {
				progressDialog.Hide()
				s.ShowErrorDialog(errors.New(lang.LocalizeKey("update.failed", "Update failed: {{.Error}}", map[string]any{"Error": err.Error()})))
			})
			return
		}

		fyne.Do(func() {
			progressDialog.Hide()
			if installerLaunched {
				slog.Info("Installer launched, exiting to allow update")
				if app := fyne.CurrentApp(); app != nil {
					app.Quit()
				}
				return
			}
			execCmd := exec.Command(os.Args[0], os.Args[1:]...)
			execCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if err := execCmd.Start(); err != nil {
				slog.Error("Failed to restart application", "error", err)
			}
			if app := fyne.CurrentApp(); app != nil {
				app.Quit()
			}
		})
	}()
}

func (s *State) showMandatoryUpdateRequiredDialog() {
	if s.Window == nil {
		if app := fyne.CurrentApp(); app != nil {
			app.Quit()
		}
		return
	}
	errDialog := dialog.NewError(
		errors.New(lang.LocalizeKey("update.required", "Update is required to continue. Please update to the latest version and restart the application.")),
		s.Window,
	)
	errDialog.SetOnClosed(func() {
		if app := fyne.CurrentApp(); app != nil {
			app.Quit()
		}
	})
	errDialog.Show()
}

func (s *State) StartPeriodicUpdateChecker(ctx context.Context) {
	go func() {
		// Initial check after 3 seconds (skip if updater already performed initial check)
		if !s.IsInitial {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				s.CheckForUpdates(ctx, false)
			}
		}

		// Periodic check every 1 hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.Core != nil && s.Core.IsAnyProfileBusy() {
					continue
				}
				s.CheckForUpdates(ctx, false)
			}
		}
	}()
}
