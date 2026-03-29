//go:build windows

package settings

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log/slog"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/common/versioning"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
)

type Settings struct {
	state                   *uicommon.State
	BranchSelect            *widget.Select
	DisplayScaleSlider      *widget.Slider
	DisplayScaleSelect      *widget.Select
	ApiServerEntry          *widget.Entry
	SaveConfigButton        *widget.Button
	ImportProfileButton     *widget.Button
	ClearCacheButton        *widget.Button
	DeleteAmongUsDataButton *widget.Button

	uninstallButton      *widget.Button
	installationListener binding.DataListener

	epicAccountLabel *widget.Label
	epicLoginButton  *widget.Button
	epicLogoutButton *widget.Button

	displayScaleValues  map[string]float32
	scaleControlSyncing bool
	currentDisplayScale float32

	thirdPartyLicenses       []thirdPartyLicense
	thirdPartyLicenseLoadErr error
	projectLicense           projectLicense
}

const (
	displayScaleMin  = float32(0.75)
	displayScaleMax  = float32(2.0)
	displayScaleStep = float32(0.05)

	projectLicenseURL = "https://github.com/ikafly144/au_mod_installer/blob/master/LICENSE"
)

var projectModulePath = func() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return buildInfo.Main.Path
}()

//go:embed licenses.json
var thirdPartyLicensesJSON []byte

type licensesDocument struct {
	Project    projectLicense      `json:"project"`
	ThirdParty []thirdPartyLicense `json:"third_party"`
}

type projectLicense struct {
	LicenseURL  string `json:"license_url"`
	LicenseName string `json:"license_name"`
	LicenseText string `json:"license_text"`
}

type thirdPartyLicense struct {
	Name        string `json:"name"`
	LicenseURL  string `json:"license_url"`
	LicenseName string `json:"license_name"`
	LicenseText string `json:"license_text"`
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

	currentScale := normalizedDisplayScale(fyne.CurrentApp().Settings().Scale())
	displayScaleValues, displayScaleOptions := availableDisplayScales(currentScale)
	displayScaleSelect := widget.NewSelect(displayScaleOptions, nil)
	displayScaleSelect.PlaceHolder = lang.LocalizeKey("settings.display_scale_hint", "Adjust UI display scale")

	displayScaleSlider := widget.NewSlider(float64(displayScaleMin), float64(displayScaleMax))
	displayScaleSlider.Step = float64(displayScaleStep)
	displayScaleSlider.SetValue(float64(clampDisplayScale(currentScale)))

	apiServerEntry := widget.NewEntry()
	apiServerEntry.PlaceHolder = "https://modofus.sabafly.net/api/v1"
	apiServerEntry.SetText(fyne.CurrentApp().Preferences().String("api_server"))

	thirdPartyLicenses, thirdPartyLicenseLoadErr := loadThirdPartyLicenses()
	if thirdPartyLicenseLoadErr != nil {
		slog.Warn("Failed to load third-party licenses", "error", thirdPartyLicenseLoadErr)
	}
	projectLicense, projectLicenseErr := loadProjectLicense()
	if projectLicenseErr != nil {
		slog.Warn("Failed to load project license", "error", projectLicenseErr)
	}
	if projectLicense.LicenseURL == "" {
		projectLicense.LicenseURL = projectLicenseURL
	}

	s := &Settings{
		state:                    state,
		BranchSelect:             branchSelect,
		DisplayScaleSlider:       displayScaleSlider,
		DisplayScaleSelect:       displayScaleSelect,
		ApiServerEntry:           apiServerEntry,
		uninstallButton:          widget.NewButtonWithIcon(lang.LocalizeKey("installation.uninstall", "Uninstall from Game Folder"), theme.DeleteIcon(), nil), // nil callback initially, set in init
		epicAccountLabel:         widget.NewLabel(""),
		displayScaleValues:       displayScaleValues,
		currentDisplayScale:      clampDisplayScale(currentScale),
		thirdPartyLicenses:       thirdPartyLicenses,
		thirdPartyLicenseLoadErr: thirdPartyLicenseLoadErr,
		projectLicense:           projectLicense,
	}
	s.DisplayScaleSelect.OnChanged = s.onDisplayScaleChanged
	s.DisplayScaleSlider.OnChanged = s.onDisplayScaleSliderChanged
	s.DisplayScaleSlider.OnChangeEnded = s.onDisplayScaleSliderChangeEnded
	s.setDisplayScaleControls(s.currentDisplayScale)

	s.SaveConfigButton = widget.NewButtonWithIcon(lang.LocalizeKey("settings.save", "Save"), theme.DocumentSaveIcon(), s.saveConfig)

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

	s.DeleteAmongUsDataButton = widget.NewButtonWithIcon(lang.LocalizeKey("settings.delete_among_us_data", "Delete Among Us Data"), theme.DeleteIcon(), s.deleteAmongUsData)
	s.DeleteAmongUsDataButton.Importance = widget.DangerImportance

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
	selectedPath := container.NewHScroll(container.New(layout.NewCustomPaddedLayout(0, 10, 0, 0), entry))
	selectedPath.SetMinSize(fyne.NewSize(0, 50))

	basicPage := container.NewVScroll(container.NewVBox(
		widget.NewCard(
			lang.LocalizeKey("installation.select_install_path", "Among Us Installation Path"),
			"",
			container.NewVBox(
				s.state.InstallSelect,
				widget.NewAccordion(
					widget.NewAccordionItem(
						lang.LocalizeKey("installation.selected_install", "Selected Installation Path"),
						selectedPath,
					),
				),
			),
		),
		widget.NewCard(
			lang.LocalizeKey("settings.update_channel", "Update Channel"),
			"",
			settingsEntry(lang.LocalizeKey("settings.select_update_channel", "Select Update Channel"), s.BranchSelect),
		),
		widget.NewCard(
			lang.LocalizeKey("settings.display_scale", "Display Scale"),
			"",
			settingsEntry(
				lang.LocalizeKey("settings.display_scale_hint", "Adjust UI display scale"),
				container.NewBorder(
					nil,
					nil,
					nil,
					container.New(layout.NewGridWrapLayout(fyne.NewSize(110, s.DisplayScaleSelect.MinSize().Height)), s.DisplayScaleSelect),
					s.DisplayScaleSlider,
				),
			),
		),
		widget.NewCard(
			lang.LocalizeKey("settings.cache_management", "Cache Management"),
			"",
			container.NewVBox(s.ClearCacheButton),
		),
	))

	accountPage := container.NewVScroll(container.NewVBox(
		widget.NewCard(
			lang.LocalizeKey("settings.epic_games_account", "Epic Games Account"),
			"",
			container.NewVBox(
				s.epicAccountLabel,
				container.NewHBox(s.epicLoginButton, s.epicLogoutButton),
			),
		),
	))

	warningText := widget.NewRichText(
		&widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  lang.LocalizeKey("settings.page.advanced.warning", "These settings typically do not need to be changed. If you choose to change them, please do so carefully with an understanding of what they do."),
		},
	)
	warningText.Wrapping = fyne.TextWrapBreak

	advancedPage := container.NewVScroll(container.NewVBox(
		warningText,
		widget.NewCard(
			lang.LocalizeKey("settings.legacy_migration", "Legacy Migration"),
			"",
			container.NewVBox(s.ImportProfileButton),
		),
		widget.NewCard(
			lang.LocalizeKey("installation.legacy_installation_status", "Legacy Installation Status"),
			"",
			container.NewVBox(
				s.state.ModInstalledInfo,
				s.uninstallButton,
			),
		),
		widget.NewCard(
			lang.LocalizeKey("settings.advanced_settings", "Advanced Settings"),
			"",
			container.NewVBox(
				settingsEntry(lang.LocalizeKey("settings.server_url", "Server URL"), s.ApiServerEntry),
				s.SaveConfigButton,
			),
		),
		widget.NewCard(
			lang.LocalizeKey("settings.data_management", "Data Management"),
			"",
			container.NewVBox(s.DeleteAmongUsDataButton),
		),
	))

	openSourcePage := s.newOpenSourcePage()

	pageTitles := []string{
		lang.LocalizeKey("settings.page.general", "General"),
		lang.LocalizeKey("settings.page.account", "Account"),
		lang.LocalizeKey("settings.page.advanced", "Advanced"),
		lang.LocalizeKey("settings.page.opensource", "Open Source Licenses"),
	}
	pageContents := []fyne.CanvasObject{
		basicPage,
		accountPage,
		advancedPage,
		openSourcePage,
	}

	pageTitle := widget.NewLabelWithStyle(pageTitles[0], fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	pageTitle.SizeName = theme.SizeNameSubHeadingText
	pageContainer := container.NewStack(pageContents[0])

	navList := widget.NewList(
		func() int { return len(pageTitles) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("page")
			label.Wrapping = fyne.TextWrapWord
			return container.NewPadded(label)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*fyne.Container).Objects[0].(*widget.Label).SetText(pageTitles[id])
		},
	)
	navList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(pageContents) {
			return
		}
		pageTitle.SetText(pageTitles[id])
		pageContainer.Objects = []fyne.CanvasObject{pageContents[id]}
		pageContainer.Refresh()
	}
	navList.Select(0)
	navPanel := widget.NewCard(
		lang.LocalizeKey("settings.page.navigation", "Settings"),
		"",
		navList,
	)
	navPanelMinWidth := canvas.NewRectangle(color.Transparent)
	navPanelMinWidth.SetMinSize(fyne.NewSize(220, 0))
	navPanelContainer := container.NewStack(navPanelMinWidth, navPanel)
	contentPanel := container.NewPadded(container.NewBorder(
		container.NewVBox(pageTitle, widget.NewSeparator()),
		nil,
		nil,
		nil,
		pageContainer,
	))
	pages := container.NewBorder(nil, nil, container.NewPadded(navPanelContainer), nil, contentPanel)

	footer := container.NewVBox(
		widget.NewSeparator(),
		s.state.ErrorText,
	)
	content := container.NewBorder(nil, footer, nil, nil, pages)
	return container.NewTabItem(lang.LocalizeKey("settings.title", "Settings"), content), nil
}

func (s *Settings) refreshEpicAccountInfo() {
	session := s.state.Core.EpicSessionManager.GetSession()
	if session == nil {
		s.epicAccountLabel.SetText(lang.LocalizeKey("settings.epic_logged_out", "Not Logged In")) // Reuse if appropriate or use new key
		s.epicLoginButton.Show()
		s.epicLogoutButton.Hide()
	} else {
		s.epicAccountLabel.SetText(lang.LocalizeKey("settings.epic_logged_in", "Logged in Epic Games Account"))
		s.epicLoginButton.Hide()
		s.epicLogoutButton.Show()
	}
}

func (s *Settings) showEpicLoginDialog() {
	s.state.ShowEpicLoginWindow(func() {
		s.refreshEpicAccountInfo()
		dialog.ShowInformation(lang.LocalizeKey("settings.login_success", "Login Successful"), lang.LocalizeKey("settings.login_success_message", "You have been logged in successfully."), s.state.Window)
	}, nil)
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
	s.state.ClearError()
	path, err := s.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		s.state.ShowErrorDialog(errors.New(lang.LocalizeKey("installation.error.no_path", "Installation path is not specified.")))
		return
	}
	slog.Info("Uninstalling mod", "path", path)

	go func() {
		if err := s.state.Core.UninstallMod(path, nil); err != nil {
			s.state.ShowErrorDialog(errors.New(lang.LocalizeKey("installation.error.failed_to_uninstall", "Failed to uninstall mod: ") + err.Error()))
			slog.Warn("Failed to uninstall mod", "error", err)
			return
		}
		slog.Info("Mod uninstalled successfully", "path", path)
		s.state.ShowInfoDialog(
			lang.LocalizeKey("common.success", "Success"),
			lang.LocalizeKey("installation.success.uninstalled", "Mod uninstalled successfully."),
		)
		fyne.Do(s.state.RefreshModInstallation)
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
			mods[m.ID] = m.ModVersion
		}

		prof := profile.Profile{
			ID:          uuid.New(),
			Name:        name,
			ModVersions: mods,
			UpdatedAt:   time.Now(),
		}

		if err := s.state.ProfileManager.Add(prof); err != nil {
			dialog.ShowError(err, s.state.Window)
			return
		}
		dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("settings.profile_imported", "Profile imported successfully."), s.state.Window)
	}, s.state.Window)
}

func (s *Settings) deleteAmongUsData() {
	dialog.ShowConfirm(lang.LocalizeKey("settings.delete_among_us_data_confirm_title", "Delete Among Us Data"), lang.LocalizeKey("settings.delete_among_us_data_confirm_message", "Are you sure you want to delete all Among Us data? This will reset all your Among Us settings and save data. This action cannot be undone."), func(confirm bool) {
		if !confirm {
			return
		}
		appDataDir, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppDataLow, 0)
		if err != nil {
			slog.Error("Failed to get LocalAppDataLow folder path", "error", err)
			dialog.ShowError(err, s.state.Window)
			return
		}
		auDataDir := filepath.Join(appDataDir, "Innersloth", "Among Us")
		if err := os.RemoveAll(auDataDir); err != nil {
			dialog.ShowError(err, s.state.Window)
		} else {
			dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("settings.among_us_data_deleted", "Among Us data deleted successfully."), s.state.Window)
			s.state.RefreshModInstallation()
		}
	}, s.state.Window)
}

func (s *Settings) saveConfig() {
	server := s.ApiServerEntry.Text
	if server == "" {
		fyne.CurrentApp().Preferences().RemoveValue("api_server")
	} else {
		fyne.CurrentApp().Preferences().SetString("api_server", server)
	}
	dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("settings.saved", "Settings saved successfully. Please restart the application."), s.state.Window)
}

func (s *Settings) newOpenSourcePage() fyne.CanvasObject {
	pageStack := container.NewStack()

	showListPage := func() {}

	showDetailPage := func(title, licenseText, licenseURL string) {
		licenseText = strings.TrimSpace(licenseText)
		if licenseText == "" {
			licenseText = lang.LocalizeKey("settings.opensource.load_failed", "Couldn't load license information. Please restart the app and try again.")
		}
		titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		titleLabel.Wrapping = fyne.TextWrapWord

		backButton := widget.NewButtonWithIcon(
			lang.LocalizeKey("common.back", "Back"),
			theme.NavigateBackIcon(),
			showListPage,
		)
		header := container.NewVBox(
			container.NewHBox(backButton),
			titleLabel,
		)
		if u, err := url.Parse(licenseURL); err == nil {
			header.Add(widget.NewHyperlink(
				lang.LocalizeKey("settings.opensource.open_dependency_license", "Open official license page"),
				u,
			))
		}

		licenseBody := widget.NewRichText(
			&widget.TextSegment{Text: strings.TrimSpace(licenseText)},
		)
		licenseBody.Wrapping = fyne.TextWrapWord
		licenseCard := widget.NewCard("", "", licenseBody)

		detail := container.NewBorder(
			header,
			nil,
			nil,
			nil,
			container.NewVScroll(licenseCard),
		)
		pageStack.Objects = []fyne.CanvasObject{detail}
		pageStack.Refresh()
	}

	showListPage = func() {
		licenseList := container.NewVBox()

		switch {
		case s.thirdPartyLicenseLoadErr != nil:
			licenseList.Add(widget.NewLabel(lang.LocalizeKey("settings.opensource.load_failed", "Couldn't load license information. Please restart the app and try again.")))
			errLabel := widget.NewLabel(s.thirdPartyLicenseLoadErr.Error())
			errLabel.Wrapping = fyne.TextWrapWord
			licenseList.Add(errLabel)
		case len(s.thirdPartyLicenses) == 0:
			licenseList.Add(widget.NewLabel(lang.LocalizeKey("settings.opensource.no_license_data", "License information is not generated yet.")))
		default:
			for _, dependency := range s.thirdPartyLicenses {
				dep := dependency
				licenseList.Add(widget.NewButton(
					lang.LocalizeKey("settings.opensource.package_button", "{{.Name}} ({{.License}})", map[string]any{
						"Name":    dep.Name,
						"License": dep.LicenseName,
					}),
					func() {
						showDetailPage(
							lang.LocalizeKey("settings.opensource.package_title", "{{.Name}}", map[string]any{"Name": dep.Name}),
							dep.LicenseText,
							dep.LicenseURL,
						)
					},
				))
			}
		}

		listPage := container.NewVScroll(container.NewVBox(
			widget.NewButton(
				lang.LocalizeKey("settings.opensource.project_license", "Project License"),
				func() {
					showDetailPage(
						lang.LocalizeKey("settings.opensource.project_license", "Project License"),
						s.projectLicense.LicenseText,
						s.projectLicense.LicenseURL,
					)
				},
			),
			widget.NewCard(
				lang.LocalizeKey("settings.opensource.dependencies", "Third-party dependencies"),
				"",
				container.NewVBox(
					licenseList,
				),
			),
		))
		pageStack.Objects = []fyne.CanvasObject{listPage}
		pageStack.Refresh()
	}

	showListPage()
	return pageStack
}

func loadThirdPartyLicenses() ([]thirdPartyLicense, error) {
	var doc licensesDocument
	if err := json.Unmarshal(thirdPartyLicensesJSON, &doc); err != nil {
		return nil, err
	}
	licenses := doc.ThirdParty
	filtered := make([]thirdPartyLicense, 0, len(licenses))
	for _, license := range licenses {
		license.Name = strings.TrimSpace(license.Name)
		license.LicenseURL = strings.TrimSpace(strings.ReplaceAll(license.LicenseURL, "\\", "/"))
		license.LicenseName = strings.TrimSpace(license.LicenseName)
		license.LicenseText = strings.TrimSpace(license.LicenseText)
		if license.Name == "" || license.LicenseURL == "" || license.LicenseName == "" || license.LicenseText == "" {
			continue
		}
		if license.Name == projectModulePath || strings.HasPrefix(license.Name, projectModulePath+"/") {
			continue
		}
		filtered = append(filtered, license)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	return filtered, nil
}

func loadProjectLicense() (projectLicense, error) {
	var doc licensesDocument
	if err := json.Unmarshal(thirdPartyLicensesJSON, &doc); err != nil {
		return projectLicense{}, err
	}
	project := doc.Project
	project.LicenseText = strings.TrimSpace(project.LicenseText)
	project.LicenseURL = strings.TrimSpace(strings.ReplaceAll(project.LicenseURL, "\\", "/"))
	project.LicenseName = strings.TrimSpace(project.LicenseName)
	return project, nil
}

func settingsEntry(title string, content fyne.CanvasObject) fyne.CanvasObject {
	label := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.New(layout.NewBorderLayout(nil, nil, label, nil), content, label)
}

func (s *Settings) onDisplayScaleChanged(selected string) {
	if s.scaleControlSyncing {
		return
	}
	scale, ok := s.displayScaleValues[selected]
	if !ok {
		return
	}
	s.applyDisplayScale(scale)
}

func (s *Settings) onDisplayScaleSliderChanged(value float64) {
	if s.scaleControlSyncing {
		return
	}
	scale := clampDisplayScale(float32(value))
	s.scaleControlSyncing = true
	s.DisplayScaleSelect.SetSelected(nearestDisplayScaleLabel(scale, s.displayScaleValues))
	s.scaleControlSyncing = false
}

func (s *Settings) onDisplayScaleSliderChangeEnded(value float64) {
	if s.scaleControlSyncing {
		return
	}
	s.applyDisplayScale(float32(value))
}

func (s *Settings) applyDisplayScale(scale float32) {
	scale = clampDisplayScale(scale)
	currentScale := s.currentDisplayScale
	if almostEqualScale(scale, currentScale) {
		return
	}

	if err := saveDisplayScale(scale); err != nil {
		dialog.ShowError(err, s.state.Window)
		s.setDisplayScaleControls(currentScale)
		return
	}

	s.currentDisplayScale = scale
	s.setDisplayScaleControls(scale)
}

func (s *Settings) setDisplayScaleControls(scale float32) {
	scale = clampDisplayScale(scale)
	s.scaleControlSyncing = true
	defer func() {
		s.scaleControlSyncing = false
	}()
	s.DisplayScaleSlider.SetValue(float64(scale))
	s.DisplayScaleSelect.SetSelected(nearestDisplayScaleLabel(scale, s.displayScaleValues))
}

func availableDisplayScales(currentScale float32) (map[string]float32, []string) {
	presets := []float32{}
	for v := displayScaleMin; v <= displayScaleMax+displayScaleStep/2; v += displayScaleStep {
		presets = append(presets, clampDisplayScale(v))
	}
	hasCurrent := false
	for _, preset := range presets {
		if almostEqualScale(preset, currentScale) {
			hasCurrent = true
			break
		}
	}
	if !hasCurrent {
		presets = append(presets, currentScale)
	}
	slices.Sort(presets)

	values := map[string]float32{}
	options := make([]string, 0, len(presets))
	for _, preset := range presets {
		label := displayScaleLabel(preset)
		if _, exists := values[label]; exists {
			continue
		}
		values[label] = preset
		options = append(options, label)
	}
	return values, options
}

func displayScaleLabel(scale float32) string {
	return fmt.Sprintf("%.0f%%", scale*100)
}

func nearestDisplayScaleLabel(scale float32, options map[string]float32) string {
	nearest := ""
	nearestDiff := float32(math.MaxFloat32)
	for label, option := range options {
		diff := float32(math.Abs(float64(option - scale)))
		if diff < nearestDiff {
			nearestDiff = diff
			nearest = label
		}
	}
	return nearest
}

func normalizedDisplayScale(scale float32) float32 {
	if scale <= 0 {
		return 1
	}
	return scale
}

func clampDisplayScale(scale float32) float32 {
	if scale < displayScaleMin {
		scale = displayScaleMin
	}
	if scale > displayScaleMax {
		scale = displayScaleMax
	}
	return float32(math.Round(float64(scale)/float64(displayScaleStep)) * float64(displayScaleStep))
}

func almostEqualScale(a, b float32) bool {
	return math.Abs(float64(a-b)) < 0.0001
}

func saveDisplayScale(scale float32) error {
	var schema fyneapp.SettingsSchema
	path := schema.StoragePath()
	if err := loadDisplaySettings(path, &schema); err != nil {
		return err
	}
	schema.Scale = scale
	data, err := json.Marshal(&schema)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	return nil
}

func loadDisplaySettings(path string, schema *fyneapp.SettingsSchema) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(schema); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
