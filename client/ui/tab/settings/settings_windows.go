//go:build windows

package settings

import (
	"context"
	_ "embed"
	"encoding/json/v2"
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
	"sync"

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
	"github.com/ikafly144/modrepo/client/ui/uicommon"
	"github.com/ikafly144/modrepo/common/versioning"
)

type Settings struct {
	state                 *uicommon.State
	BranchEntry           *widget.Entry
	BranchHintLabel       *widget.Label
	BranchStatusLabel     *widget.RichText
	TrayResidentCheck     *widget.Check
	StartSilentCheck      *widget.Check
	AutoStartCheck        *widget.Check
	DisplayScaleSlider    *widget.Slider
	DisplayScaleSelect    *widget.Select
	ClearCacheButton      *widget.Button
	CheckForUpdatesButton *widget.Button

	displayScaleValues  map[string]float32
	scaleControlSyncing bool
	currentDisplayScale float32

	loadLicensesOnce         sync.Once
	thirdPartyLicenses       []thirdPartyLicense
	thirdPartyLicenseLoadErr error
	projectLicense           projectLicense
}

const (
	displayScaleMin  = float32(0.5)
	displayScaleMax  = float32(2.0)
	displayScaleStep = float32(0.1)

	projectLicenseURL = "https://github.com/ikafly144/modrepo/blob/master/LICENSE"
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
	branchEntry := widget.NewEntry()
	branchEntry.PlaceHolder = lang.LocalizeKey("settings.update_branch_placeholder", "Enter update branch (advanced)")
	branchHintLabel := widget.NewLabelWithStyle(lang.LocalizeKey("settings.update_branch_hint", "Advanced: enter branch name manually. Leave blank for stable."), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	branchHintLabel.Wrapping = fyne.TextWrapWord
	branchStatusLabel := widget.NewRichText()
	branchStatusLabel.Hide()
	currentBranch := strings.TrimSpace(fyne.CurrentApp().Preferences().StringWithFallback("core.update_branch", "stable"))
	normalizedBranch := strings.ToLower(currentBranch)
	if versioning.BranchFromString(normalizedBranch).String() != normalizedBranch {
		normalizedBranch = versioning.BranchStable.String()
	}
	branchEntry.SetText(normalizedBranch)
	branchEntry.OnChanged = func(s string) {
		normalized := strings.ToLower(strings.TrimSpace(s))
		if normalized == "" {
			fyne.CurrentApp().Preferences().SetString("core.update_branch", versioning.BranchStable.String())
			branchStatusLabel.Hide()
			return
		}
		if versioning.BranchFromString(normalized).String() != normalized {
			fyne.CurrentApp().Preferences().SetString("core.update_branch", versioning.BranchStable.String())
			branchStatusLabel.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text: lang.LocalizeKey("settings.update_branch_invalid", "Invalid branch name. Using stable."),
					Style: widget.RichTextStyle{
						ColorName: theme.ColorNameError,
					},
				},
			}
			branchStatusLabel.Wrapping = fyne.TextWrapWord
			branchStatusLabel.Show()
			branchStatusLabel.Refresh()
			return
		}
		fyne.CurrentApp().Preferences().SetString("core.update_branch", normalized)
		branchStatusLabel.Hide()
	}

	trayResidentCheck := widget.NewCheck(lang.LocalizeKey("settings.tray_resident_label", "Stay in System Tray"), func(checked bool) {
		fyne.CurrentApp().Preferences().SetBool("tray_resident", checked)
	})
	trayResidentCheck.Checked = fyne.CurrentApp().Preferences().BoolWithFallback("tray_resident", true)

	startSilentCheck := widget.NewCheck(lang.LocalizeKey("settings.start_silent_label", "Start Minimized to Tray on OS Startup"), func(checked bool) {
		fyne.CurrentApp().Preferences().SetBool("start_silent", checked)
		uicommon.SyncAutoStart(
			fyne.CurrentApp().Preferences().BoolWithFallback("launch_on_startup", true),
			checked,
		)
	})
	startSilentCheck.Checked = fyne.CurrentApp().Preferences().BoolWithFallback("start_silent", true)

	var updatingAutoStart bool
	autoStartCheck := widget.NewCheck(lang.LocalizeKey("settings.autostart_label", "Launch on OS Startup"), nil)
	autoStartCheck.OnChanged = func(checked bool) {
		if updatingAutoStart {
			return
		}
		startSilent := fyne.CurrentApp().Preferences().BoolWithFallback("start_silent", true)
		if err := uicommon.SetAutoStartEnabled(checked, startSilent); err != nil {
			slog.Error("Failed to update auto start setting", "error", err)
			dialog.ShowError(err, state.Window)
			updatingAutoStart = true
			autoStartCheck.SetChecked(!checked)
			updatingAutoStart = false
			return
		}
		fyne.CurrentApp().Preferences().SetBool("launch_on_startup", checked)
	}
	autoStartCheck.Checked = fyne.CurrentApp().Preferences().BoolWithFallback("launch_on_startup", true)

	currentScale := normalizedDisplayScale(fyne.CurrentApp().Settings().Scale())
	displayScaleValues, displayScaleOptions := availableDisplayScales(currentScale)
	displayScaleSelect := widget.NewSelect(displayScaleOptions, nil)
	displayScaleSelect.PlaceHolder = lang.LocalizeKey("settings.display_scale_hint", "Adjust UI display scale")

	displayScaleSlider := widget.NewSlider(float64(displayScaleMin), float64(displayScaleMax))
	displayScaleSlider.Step = float64(displayScaleStep)
	displayScaleSlider.SetValue(float64(clampDisplayScale(currentScale)))

	s := &Settings{
		state:               state,
		BranchEntry:         branchEntry,
		BranchHintLabel:     branchHintLabel,
		BranchStatusLabel:   branchStatusLabel,
		TrayResidentCheck:   trayResidentCheck,
		StartSilentCheck:    startSilentCheck,
		AutoStartCheck:      autoStartCheck,
		DisplayScaleSlider:  displayScaleSlider,
		DisplayScaleSelect:  displayScaleSelect,
		displayScaleValues:  displayScaleValues,
		currentDisplayScale: clampDisplayScale(currentScale),
	}
	s.DisplayScaleSelect.OnChanged = s.onDisplayScaleChanged
	s.DisplayScaleSlider.OnChanged = s.onDisplayScaleSliderChanged
	s.DisplayScaleSlider.OnChangeEnded = s.onDisplayScaleSliderChangeEnded
	s.setDisplayScaleControls(s.currentDisplayScale)

	s.ClearCacheButton = widget.NewButtonWithIcon(lang.LocalizeKey("settings.clear_cache", "Clear Mod Cache"), theme.DeleteIcon(), s.clearCache)

	s.CheckForUpdatesButton = widget.NewButtonWithIcon(lang.LocalizeKey("settings.check_for_updates", "Check for Updates"), theme.ViewRefreshIcon(), func() {
		s.CheckForUpdatesButton.Disable()
		s.CheckForUpdatesButton.SetText(lang.LocalizeKey("update.checking", "Checking for updates..."))
		go func() {
			defer fyne.Do(func() {
				s.CheckForUpdatesButton.Enable()
				s.CheckForUpdatesButton.SetText(lang.LocalizeKey("settings.check_for_updates", "Check for Updates"))
			})
			s.state.CheckForUpdates(context.Background(), true)
		}()
	})

	return s
}

func (s *Settings) ensureLicensesLoaded() {
	s.loadLicensesOnce.Do(func() {
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
		s.thirdPartyLicenses = thirdPartyLicenses
		s.thirdPartyLicenseLoadErr = thirdPartyLicenseLoadErr
		s.projectLicense = projectLicense
	})
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

func (s *Settings) Tab() (*container.TabItem, error) {
	entry := widget.NewLabelWithData(s.state.SelectedGamePath)
	entry.Selectable = true
	pathBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	pathBg.CornerRadius = theme.InputRadiusSize()
	selectedPath := container.NewHScroll(
		container.NewStack(
			pathBg,
			entry,
		),
	)

	openInExplorerButton := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		path, err := s.state.SelectedGamePath.Get()
		if err != nil || path == "" {
			s.state.ShowErrorDialog(errors.New(lang.LocalizeKey("installation.error.no_path", "Installation path is not specified.")))
			return
		}
		slog.Info("Opening installation folder in explorer", "path", path)
		if err := s.state.ExplorerOpenFolder(path); err != nil {
			s.state.ShowErrorDialog(errors.New(lang.LocalizeKey("installation.error.open_explorer_failed", "Failed to open file explorer: ") + err.Error()))
			return
		}
	})

	gameVersionLabel := widget.NewLabel("")
	s.state.SelectedGamePath.AddListener(binding.NewDataListener(func() {
		path, err := s.state.SelectedGamePath.Get()
		if err != nil {
			slog.Warn("Failed to get selected game path", "error", err)
			gameVersionLabel.SetText(lang.LocalizeKey("installation.error.get_version_failed", "Failed to get game version information"))
			return
		}
		gameVersion, err := s.state.Core.GetGameVersion(path)
		if err != nil {
			slog.Warn("Failed to get installation status for game version label", "error", err)
			gameVersionLabel.SetText(lang.LocalizeKey("installation.error.get_version_failed", "Failed to get game version information"))
			return
		}
		gameVersionLabel.SetText(lang.LocalizeKey("installation.game_version", "Game Version: {{.Version}}", map[string]any{
			"Version": gameVersion,
		}))
	}))

	revision := fyne.CurrentApp().Metadata().Custom["revision"]
	revision = revision[:min(7, len(revision))]
	versionContent := widget.NewLabelWithStyle(lang.LocalizeKey("settings.app.version", "version: {{.Version}} ({{.Revision}})", map[string]any{"Version": s.state.Version, "Revision": revision}), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	versionContent.Wrapping = fyne.TextWrapWord
	basicPage := container.NewVScroll(container.NewVBox(
		widget.NewCard(
			lang.LocalizeKey("settings.app.title", "MODREPO"),
			lang.LocalizeKey("settings.app.subtitle",
				"R.E.P.O. Mod Manager"),
			container.NewVBox(
				versionContent,
				container.NewCenter(s.CheckForUpdatesButton),
			),
		),
		widget.NewCard(
			lang.LocalizeKey("installation.select_install_info", "R.E.P.O. Installation Information"),
			"",
			container.NewVBox(
				s.state.InstallSelect,
				widget.NewAccordion(
					widget.NewAccordionItem(
						lang.LocalizeKey("installation.selected_install", "Selected Installation"),
						container.NewBorder(
							container.NewBorder(
								nil,
								nil,
								nil,
								openInExplorerButton,
								selectedPath,
							),
							nil, nil, nil,
							gameVersionLabel,
						),
					),
				),
			),
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
			lang.LocalizeKey("settings.tray_resident", "System Tray"),
			"",
			container.NewVBox(
				s.TrayResidentCheck,
				newHintLabel(lang.LocalizeKey("settings.tray_resident_hint", "Keep running in the background and stay in the system tray when the window is closed.")),
			),
		),
		widget.NewCard(
			lang.LocalizeKey("settings.autostart", "Run on Startup"),
			"",
			container.NewVBox(
				s.AutoStartCheck,
				newHintLabel(lang.LocalizeKey("settings.autostart_hint", "Automatically start the application when Windows starts.")),
				s.StartSilentCheck,
				newHintLabel(lang.LocalizeKey("settings.start_silent_hint", "Start the application minimized in the system tray when launching on OS startup.")),
			),
		),
		widget.NewCard(
			lang.LocalizeKey("settings.cache_management", "Cache Management"),
			"",
			container.NewVBox(s.ClearCacheButton),
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
			lang.LocalizeKey("settings.update_channel", "Update Channel"),
			"",
			settingsEntry(lang.LocalizeKey("settings.update_branch_input", "Update Branch"), container.NewVBox(
				s.BranchEntry,
				s.BranchHintLabel,
				s.BranchStatusLabel,
			)),
		),
	))

	openSourcePage := s.newOpenSourcePage()

	pageTitles := []string{
		lang.LocalizeKey("settings.page.general", "General"),
		lang.LocalizeKey("settings.page.advanced", "Advanced"),
		lang.LocalizeKey("settings.page.opensource", "Open Source Licenses"),
	}
	pageContents := []fyne.CanvasObject{
		basicPage,
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
		s.ensureLicensesLoaded()
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
				openBtn := newTruncatedButton(
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
				)
				licenseList.Add(openBtn)
			}
		}

		projLicenseBtn := newTruncatedButton(
			lang.LocalizeKey("settings.opensource.project_license", "Project License"),
			func() {
				showDetailPage(
					lang.LocalizeKey("settings.opensource.project_license", "Project License"),
					s.projectLicense.LicenseText,
					s.projectLicense.LicenseURL,
				)
			},
		)

		listPage := container.NewVScroll(container.NewVBox(
			projLicenseBtn,
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

type truncatedButton struct {
	widget.BaseWidget
	Text     string
	OnTapped func()
}

func newTruncatedButton(text string, tapped func()) *truncatedButton {
	b := &truncatedButton{Text: text, OnTapped: tapped}
	b.ExtendBaseWidget(b)
	return b
}

type truncatedButtonRenderer struct {
	b     *truncatedButton
	btn   *widget.Button
	label *widget.Label
}

func (r *truncatedButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.btn, r.label}
}

func (r *truncatedButtonRenderer) Destroy() {}

func (r *truncatedButtonRenderer) MinSize() fyne.Size {
	return r.label.MinSize()
}

func (r *truncatedButtonRenderer) Refresh() {
	r.label.SetText(r.b.Text)
	r.btn.OnTapped = r.b.OnTapped
}

func (r *truncatedButtonRenderer) Layout(size fyne.Size) {
	r.btn.Resize(size)
	r.label.Resize(size)
}

func (b *truncatedButton) CreateRenderer() fyne.WidgetRenderer {
	label := widget.NewLabelWithStyle(b.Text, fyne.TextAlignCenter, fyne.TextStyle{})
	label.Truncation = fyne.TextTruncateEllipsis
	btn := widget.NewButton("", b.OnTapped)
	return &truncatedButtonRenderer{
		b:     b,
		btn:   btn,
		label: label,
	}
}

func newHintLabel(text string) *widget.Label {
	lbl := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	lbl.Wrapping = fyne.TextWrapWord
	return lbl
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
	if err := json.UnmarshalRead(file, schema); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
