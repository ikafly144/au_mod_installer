package settings

import (
	"log/slog"
	"net/url"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/common/versioning"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

type Settings struct {
	state               *uicommon.State
	BranchSelect        *widget.Select
	ImportProfileButton *widget.Button
	ClearCacheButton    *widget.Button

	uninstallButton      *widget.Button
	progressBar          *progress.FyneProgress
	installationListener binding.DataListener

	epicAccountLabel *widget.Label
	epicLoginButton  *widget.Button
	epicLogoutButton *widget.Button
}

func NewSettings(state *uicommon.State) *Settings {
	branchOptions := []string{
		versioning.BranchStable.String(),
		versioning.BranchPreview.String(),
		versioning.BranchBeta.String(),
		versioning.BranchCanary.String(),
		versioning.BranchDev.String(),
	}
	branchSelect := widget.NewSelect(branchOptions, nil)
	branchSelect.PlaceHolder = lang.LocalizeKey("settings.select_update_channel", "Select Update Channel")
	branchSelect.OnChanged = func(s string) {
		fyne.CurrentApp().Preferences().SetString("core.update_branch", s)
	}
	currentBranch := fyne.CurrentApp().Preferences().StringWithFallback("core.update_branch", "stable")
	branchSelect.SetSelected(currentBranch)

	s := &Settings{
		state:            state,
		BranchSelect:     branchSelect,
		uninstallButton:  widget.NewButtonWithIcon(lang.LocalizeKey("installation.uninstall", "Uninstall from Game Folder"), theme.DeleteIcon(), nil), // nil callback initially, set in init
		progressBar:      progress.NewFyneProgress(widget.NewProgressBar()),
		epicAccountLabel: widget.NewLabel(""),
	}

	s.epicLoginButton = widget.NewButton(lang.LocalizeKey("settings.epic_login", "Login"), s.showEpicLoginDialog)
	s.epicLogoutButton = widget.NewButton(lang.LocalizeKey("settings.epic_logout", "Logout"), s.epicLogout)

	s.refreshEpicAccountInfo()

	s.uninstallButton.OnTapped = s.runUninstall
	s.uninstallButton.Importance = widget.DangerImportance
	s.uninstallButton.Disable()

	if s.installationListener == nil {
		s.installationListener = binding.NewDataListener(s.checkUninstallState)
		s.state.ModInstalled.AddListener(s.installationListener)
		s.state.SelectedGamePath.AddListener(s.installationListener)
		s.state.CanInstall.AddListener(s.installationListener)
		s.state.RefreshModInstallation()
	}

	s.ImportProfileButton = widget.NewButtonWithIcon(lang.LocalizeKey("settings.import_profile", "Import Profile from Current Installation"), theme.DocumentSaveIcon(), s.importProfile)
	s.ClearCacheButton = widget.NewButtonWithIcon(lang.LocalizeKey("settings.clear_cache", "Clear Mod Cache"), theme.DeleteIcon(), s.clearCache)

	return s
}

func (s *Settings) clearCache() {
	dialog.ShowConfirm(lang.LocalizeKey("settings.clear_cache_confirm_title", "Clear Mod Cache"), lang.LocalizeKey("settings.clear_cache_confirm_message", "Are you sure you want to clear the mod cache? This will force re-downloading mods next time."), func(confirm bool) {
		if !confirm {
			return
		}
		if err := s.state.Core.ClearModCache(); err != nil {
			dialog.ShowError(err, s.state.Window)
		} else {
			dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("settings.cache_cleared", "Mod cache cleared successfully."), s.state.Window)
		}
	}, s.state.Window)
}

func (s *Settings) checkUninstallState() {
	if ok, err := s.state.CanInstall.Get(); !ok || err != nil {
		s.uninstallButton.Disable()
		return
	}
	if ok, err := s.state.ModInstalled.Get(); ok && err == nil {
		s.uninstallButton.Enable()
	} else if err == nil {
		s.uninstallButton.Disable()
	} else {
		slog.Warn("Failed to get modInstalled", "error", err)
	}
}

func (s *Settings) Tab() (*container.TabItem, error) {
	entry := widget.NewLabelWithData(s.state.SelectedGamePath)
	installPathSection := container.NewVBox(
		widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("installation.select_install_path", "Among Us Installation Path")),
		s.state.InstallSelect,
		widget.NewAccordion(
			widget.NewAccordionItem(lang.LocalizeKey("installation.selected_install", "Selected Installation Path"), container.NewHScroll(container.New(layout.NewCustomPaddedLayout(0, 10, 0, 0), entry))),
		),
	)

	uninstallSection := container.NewVBox(
		widget.NewRichTextFromMarkdown("### "+lang.LocalizeKey("installation.installation_status", "Installation Status (Direct Install)")),
		s.state.ModInstalledInfo,
		s.uninstallButton,
		s.progressBar.Canvas(), // Added progress bar
	)

	epicAccountSection := container.NewVBox(
		widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("settings.epic_games_account", "Epic Games Account")),
		s.epicAccountLabel,
		container.NewHBox(s.epicLoginButton, s.epicLogoutButton),
	)

	list := container.NewVScroll(container.NewVBox(
		installPathSection,
		widget.NewSeparator(),
		settingsEntry(lang.LocalizeKey("settings.update_channel", "Update Channel"), s.BranchSelect),
		widget.NewSeparator(),
		epicAccountSection,
		widget.NewSeparator(),
		settingsEntry(lang.LocalizeKey("settings.cache_management", "Cache Management"), s.ClearCacheButton),
		widget.NewSeparator(),
		settingsEntry(lang.LocalizeKey("settings.legacy_migration", "Legacy Migration"), s.ImportProfileButton),
		widget.NewSeparator(),
		uninstallSection,
		s.state.ErrorText,
	))
	return container.NewTabItem(lang.LocalizeKey("settings.title", "Settings"), list), nil
}

func (s *Settings) refreshEpicAccountInfo() {
	session := s.state.Core.EpicSessionManager.GetSession()
	if session == nil {
		s.epicAccountLabel.SetText(lang.LocalizeKey("settings.epic_logged_out", "Not Logged In")) // Reuse if appropriate or use new key
		s.epicLoginButton.Show()
		s.epicLogoutButton.Hide()
	} else {
		s.epicAccountLabel.SetText(lang.LocalizeKey("settings.epic_logged_in_as", "Logged in as: {{.DisplayName}}", map[string]any{"DisplayName": session.DisplayName}))
		s.epicLoginButton.Hide()
		s.epicLogoutButton.Show()
	}
}

func (s *Settings) showEpicLoginDialog() {
	authUrl := s.state.Core.EpicApi.GetAuthUrl()

	instruction := widget.NewLabel(lang.LocalizeKey("settings.epic_login_instruction", "Please login with Epic Games and enter the code below."))
	instruction.Wrapping = fyne.TextWrapWord

	openButton := widget.NewButton(lang.LocalizeKey("settings.epic_login_url_button", "Open Login Page"), func() {
		u, _ := url.Parse(authUrl)
		_ = fyne.CurrentApp().OpenURL(u)
	})

	entry := widget.NewEntry()
	entry.SetPlaceHolder(lang.LocalizeKey("settings.epic_login_code_label", "Authorization Code"))

	content := container.NewVBox(
		instruction,
		openButton,
		entry,
	)

	dialog.ShowCustomConfirm(
		lang.LocalizeKey("settings.epic_login", "Login"),
		lang.LocalizeKey("common.save", "Login"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		content,
		func(confirm bool) {
			if !confirm || entry.Text == "" {
				return
			}

			code := entry.Text
			session, err := s.state.Core.EpicApi.LoginWithAuthCode(code)
			if err != nil {
				dialog.ShowError(err, s.state.Window)
				return
			}

			if err := s.state.Core.EpicSessionManager.Save(session); err != nil {
				dialog.ShowError(err, s.state.Window)
				return
			}

			s.refreshEpicAccountInfo()
			dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("common.success", "Logged in successfully."), s.state.Window)
		},
		s.state.Window,
	)
}

func (s *Settings) epicLogout() {
	if err := s.state.Core.EpicSessionManager.Clear(); err != nil {
		dialog.ShowError(err, s.state.Window)
		return
	}
	s.refreshEpicAccountInfo()
}

func (s *Settings) runUninstall() {
	defer s.state.RefreshModInstallation()
	s.state.ErrorText.Hide()
	path, err := s.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		s.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.no_path", "Installation path is not specified."), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		s.state.ErrorText.Refresh()
		s.state.ErrorText.Show()
		return
	}
	slog.Info("Uninstalling mod", "path", path)

	go func() {
		if err := s.state.Core.UninstallMod(path, s.progressBar); err != nil {
			fyne.Do(func() {
				s.state.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("installation.error.failed_to_uninstall", "Failed to uninstall mod: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
				}
				s.state.ErrorText.Refresh()
				s.state.ErrorText.Show()
				slog.Warn("Failed to uninstall mod", "error", err)
			})
			return
		}
		fyne.Do(func() {
			s.state.ErrorText.ParseMarkdown(lang.LocalizeKey("installation.success.uninstalled", "Mod uninstalled successfully."))
			s.state.ErrorText.Refresh()
			s.state.ErrorText.Show()
			slog.Info("Mod uninstalled successfully", "path", path)
			s.state.RefreshModInstallation()
		})
	}()
}

func (s *Settings) importProfile() {
	path := s.state.ModInstallDir()
	if path == "" {
		dialog.ShowError(os.ErrNotExist, s.state.Window)
		return
	}

	modInstallLocation, err := os.OpenRoot(path)
	if err != nil {
		dialog.ShowError(err, s.state.Window)
		return
	}
	defer modInstallLocation.Close()

	installationInfo, err := modmgr.LoadInstallationInfo(modInstallLocation)
	if err != nil {
		dialog.ShowError(err, s.state.Window)
		return
	}

	entry := widget.NewEntry()
	entry.Validator = func(str string) error {
		if str == "" {
			return os.ErrInvalid
		}
		return nil
	}

	dialog.ShowForm(lang.LocalizeKey("profile.save_title", "Create Profile"), lang.LocalizeKey("common.save", "Save"), lang.LocalizeKey("common.cancel", "Cancel"), []*widget.FormItem{
		widget.NewFormItem(lang.LocalizeKey("profile.name", "Profile Name"), entry),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		name := entry.Text
		mods := make(map[string]modmgr.ModVersion)
		for _, m := range installationInfo.InstalledMods {
			mods[m.ModVersion.ID] = m.ModVersion
		}

		prof := profile.Profile{
			ID:          uuid.New(),
			Name:        name,
			ModVersions: mods,
			LastUpdated: time.Now(),
		}

		if err := s.state.ProfileManager.Add(prof); err != nil {
			dialog.ShowError(err, s.state.Window)
			return
		}
		dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("settings.profile_imported", "Profile imported successfully."), s.state.Window)
	}, s.state.Window)
}

func settingsEntry(title string, content fyne.CanvasObject) fyne.CanvasObject {
	label := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.New(layout.NewBorderLayout(nil, nil, label, nil), content, label)
}
