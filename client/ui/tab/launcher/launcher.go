package launcher

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

type Launcher struct {
	state               *uicommon.State
	launchButton        *widget.Button
	greetingContent     *widget.Label
	createProfileButton *widget.Button
	importProfileButton *widget.Button

	profileList       *widget.List
	profileGrid       *fyne.Container
	profileGridScroll *container.Scroll
	profileViews      *fyne.Container
	toggleViewButton  *widget.Button
	sortSelect        *widget.Select
	profiles          []profile.Profile
	selectedProfileID uuid.UUID
	isGridView        bool
	sortMode          string

	canLaunchListener binding.DataListener
}

var _ uicommon.Tab = (*Launcher)(nil)

const (
	prefLauncherViewMode = "launcher.view_mode"
	prefLauncherSortMode = "launcher.sort_mode"

	viewModeList = "list"
	viewModeGrid = "grid"

	sortModeName     = "name"
	sortModePlaytime = "playtime"
	sortModeRecent   = "recent"
)

func NewLauncherTab(s *uicommon.State) uicommon.Tab {
	var l Launcher
	revision := fyne.CurrentApp().Metadata().Custom["revision"]
	revision = revision[:min(7, len(revision))]
	viewMode := fyne.CurrentApp().Preferences().StringWithFallback(prefLauncherViewMode, viewModeList)
	sortMode := normalizeSortMode(fyne.CurrentApp().Preferences().StringWithFallback(prefLauncherSortMode, sortModeName))
	l = Launcher{
		state:               s,
		launchButton:        widget.NewButtonWithIcon(lang.LocalizeKey("launcher.launch", "Launch"), theme.MediaPlayIcon(), l.runLaunch),
		createProfileButton: widget.NewButtonWithIcon(lang.LocalizeKey("profile.create", "Create Profile"), theme.ContentAddIcon(), l.createProfile),
		importProfileButton: widget.NewButtonWithIcon(lang.LocalizeKey("profile.import_clipboard", "Import from Clipboard"), theme.ContentPasteIcon(), l.showImportDialog),
		greetingContent:     widget.NewLabelWithStyle(fmt.Sprintf("バージョン：%s (%s)", s.Version, revision), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		sortMode:            sortMode,
		isGridView:          viewMode == viewModeGrid,
	}

	l.init()

	return &l
}

func (l *Launcher) init() {
	if l.canLaunchListener == nil {
		l.canLaunchListener = binding.NewDataListener(l.checkLaunchState)
		l.state.CanLaunch.AddListener(l.canLaunchListener)
	}
	l.greetingContent.Wrapping = fyne.TextWrapWord
	l.launchButton.Importance = widget.HighImportance

	l.setupProfileList()
	l.setupProfileGrid()
	l.setupToolbar()
	l.refreshProfiles()
	l.checkLaunchState()
	l.checkSharedURI()

	l.state.OnSharedURIReceived = func(uri string) {
		l.state.SharedURI = uri
		fyne.Do(l.checkSharedURI)
	}
}

func (l *Launcher) shareProfile(prof profile.Profile) {
	uri, err := l.state.Core.ExportProfile(prof)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	fyne.CurrentApp().Clipboard().SetContent(uri)
	dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("profile.shared_clipboard", "Share URI copied to clipboard."), l.state.Window)
}

func (l *Launcher) showImportDialog() {
	entry := widget.NewMultiLineEntry()
	entry.PlaceHolder = "mod-of-us://profile/..."
	entry.SetMinRowsVisible(3)

	dialog.ShowCustomConfirm(lang.LocalizeKey("profile.import_title", "Import Profile"), lang.LocalizeKey("common.add", "Import"), lang.LocalizeKey("common.cancel", "Cancel"), entry, func(confirm bool) {
		if !confirm {
			return
		}
		l.state.SharedURI = strings.TrimSpace(entry.Text)
		l.checkSharedURI()
	}, l.state.Window)
}

func (l *Launcher) checkSharedURI() {
	if l.state.SharedURI == "" {
		return
	}

	prof, err := l.state.Core.HandleSharedProfile(l.state.SharedURI)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	// Reset SharedURI so we don't prompt again on refresh
	l.state.SharedURI = ""

	dialog.ShowConfirm(lang.LocalizeKey("profile.import_title", "Import Profile"), lang.LocalizeKey("profile.import_message", "Do you want to import the shared profile '{{.Name}}'?", map[string]any{"Name": prof.Name}), func(confirm bool) {
		if !confirm {
			return
		}

		if existing, found := l.state.ProfileManager.Get(prof.ID); found {
			if existing.UpdatedAt.After(prof.UpdatedAt) {
				dialog.ShowConfirm(lang.LocalizeKey("profile.overwrite_title", "Overwrite Profile"), lang.LocalizeKey("profile.overwrite_message", "The existing profile is newer than the imported one. Do you want to overwrite it?"), func(confirm bool) {
					if !confirm {
						return
					}
					l.importProfile(prof)
				}, l.state.Window)
				return
			}
		}

		l.importProfile(prof)
	}, l.state.Window)
}

func (l *Launcher) importProfile(shared *profile.SharedProfile) {
	prof := profile.Profile{
		ID:          shared.ID,
		Name:        shared.Name,
		Author:      shared.Author,
		Description: shared.Description,
		UpdatedAt:   time.Now(),
	}

	// Fetch mod version infos
	for modID, versionID := range shared.ModVersions {
		info, err := l.state.Rest.GetModVersion(modID, versionID)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to fetch mod version info for %s:%s: %w", modID, versionID, err), l.state.Window)
			return
		}
		prof.AddModVersion(*info)
	}

	if err := l.state.ProfileManager.Add(prof); err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	l.refreshProfiles()
}

func (l *Launcher) setupProfileList() {
	l.profileList = widget.NewList(
		func() int {
			return len(l.profiles)
		},
		func() fyne.CanvasObject {
			img := canvas.NewImageFromImage(image.NewPaletted(image.Rect(0, 0, 128, 128),
				color.Palette{theme.Color(theme.ColorNameDisabled)},
			))
			img.CornerRadius = 8

			title := widget.NewLabel("Profile Name")
			title.TextStyle = fyne.TextStyle{Bold: true}
			title.SizeName = theme.SizeNameSubHeadingText
			desc := widget.NewLabel("Profile Description")
			desc.Wrapping = fyne.TextWrapWord
			desc.SizeName = theme.SizeNameCaptionText
			label := container.NewVBox(title, desc)
			menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
			menuBtn.Importance = widget.LowImportance

			return container.NewPadded(container.NewBorder(nil, nil, img, menuBtn, img, label, menuBtn))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(l.profiles) {
				return
			}
			prof := l.profiles[id]
			c := item.(*fyne.Container).Objects[0].(*fyne.Container)
			img := c.Objects[0].(*canvas.Image)
			img.SetMinSize(fyne.NewSquareSize(img.Size().Height))
			label := c.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
			desc := c.Objects[1].(*fyne.Container).Objects[1].(*widget.Label)
			desc.SetText(fmt.Sprintf("Last Updated: %s Mods: %d", prof.UpdatedAt.Format("2006-01-02 15:04:05"), len(prof.ModVersions)))
			menuBtn := c.Objects[2].(*widget.Button)
			label.SetText(prof.Name)

			menuBtn.OnTapped = func() {
				l.showProfileMenu(menuBtn, prof)
			}
		},
	)

	l.profileList.OnSelected = func(id widget.ListItemID) {
		if id >= len(l.profiles) {
			return
		}
		l.selectedProfileID = l.profiles[id].ID
		_ = l.state.ActiveProfile.Set(l.selectedProfileID.String())
		l.checkLaunchState()
		l.refreshProfileGrid()
	}
	l.profileList.OnUnselected = func(id widget.ListItemID) {
		l.selectedProfileID = uuid.Nil
		l.checkLaunchState()
		l.refreshProfileGrid()
	}
}

func (l *Launcher) setupProfileGrid() {
	l.profileGrid = container.NewGridWrap(fyne.NewSize(132, 172))
	l.profileGridScroll = container.NewVScroll(l.profileGrid)
}

func (l *Launcher) setupToolbar() {
	l.toggleViewButton = widget.NewButtonWithIcon("", theme.GridIcon(), func() {
		l.isGridView = !l.isGridView
		if l.isGridView {
			fyne.CurrentApp().Preferences().SetString(prefLauncherViewMode, viewModeGrid)
		} else {
			fyne.CurrentApp().Preferences().SetString(prefLauncherViewMode, viewModeList)
		}
		l.updateViewToggleButton()
		l.switchProfileView()
	})
	l.sortSelect = widget.NewSelect([]string{
		lang.LocalizeKey("launcher.sort.name", "Name"),
		lang.LocalizeKey("launcher.sort.playtime", "Play Time"),
		lang.LocalizeKey("launcher.sort.recent", "Recently Launched"),
	}, func(selected string) {
		switch selected {
		case lang.LocalizeKey("launcher.sort.playtime", "Play Time"):
			l.sortMode = sortModePlaytime
		case lang.LocalizeKey("launcher.sort.recent", "Recently Launched"):
			l.sortMode = sortModeRecent
		default:
			l.sortMode = sortModeName
		}
		fyne.CurrentApp().Preferences().SetString(prefLauncherSortMode, l.sortMode)
		l.refreshProfiles()
	})
	l.sortSelect.SetSelected(l.sortModeLabel(l.sortMode))
	l.updateViewToggleButton()
}

func (l *Launcher) updateViewToggleButton() {
	if l.isGridView {
		l.toggleViewButton.SetIcon(theme.ListIcon())
		// l.toggleViewButton.SetText(lang.LocalizeKey("launcher.view.list", "List"))
		return
	}
	l.toggleViewButton.SetIcon(theme.GridIcon())
	// l.toggleViewButton.SetText(lang.LocalizeKey("launcher.view.grid", "Grid"))
}

func (l *Launcher) switchProfileView() {
	if l.profileViews == nil {
		return
	}
	if l.isGridView {
		l.profileGridScroll.Show()
		l.profileList.Hide()
		return
	}
	l.profileList.Show()
	l.profileGridScroll.Hide()
}

func (l *Launcher) showProfileMenu(anchor fyne.CanvasObject, prof profile.Profile) {
	menu := fyne.NewMenu("",
		fyne.NewMenuItem(lang.LocalizeKey("profile.edit", "Edit"), func() {
			l.openProfileEditor(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.sync", "Sync (Clear & Re-download)"), func() {
			l.syncProfile(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.share", "Share"), func() {
			l.shareProfile(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.duplicate", "Duplicate"), func() {
			l.showDuplicateDialog(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.delete", "Delete"), func() {
			l.deleteProfile(prof.ID)
		}),
	)
	widget.ShowPopUpMenuAtPosition(menu, l.state.Window.Canvas(), fyne.CurrentApp().Driver().AbsolutePositionForObject(anchor).Add(fyne.NewPos(0, anchor.Size().Height)))
}

func (l *Launcher) refreshProfileGrid() {
	if l.profileGrid == nil {
		return
	}
	var items []fyne.CanvasObject
	for i, prof := range l.profiles {
		index := i
		p := prof

		img := canvas.NewImageFromImage(image.NewPaletted(image.Rect(0, 0, 120, 120), color.Palette{theme.Color(theme.ColorNameDisabled)}))
		img.CornerRadius = 3
		img.SetMinSize(fyne.NewSquareSize(116))
		img.FillMode = canvas.ImageFillContain

		text := canvas.NewText(prof.Name, theme.Color(theme.ColorNameForeground))
		text.TextStyle = fyne.TextStyle{Bold: true}
		desc := canvas.NewText(l.profileMetaText(p), theme.Color(theme.ColorNameDisabled))
		desc.TextSize = theme.TextSize() * 0.76

		menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
		menuBtn.Importance = widget.LowImportance
		menuBtn.Resize(fyne.NewSize(22, 22))
		menuBtn.Move(fyne.NewPos(94, 4))
		menuBtn.OnTapped = func() {
			l.showProfileMenu(menuBtn, p)
		}

		iconAreaBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		iconAreaBg.CornerRadius = 8
		iconArea := container.NewStack(
			iconAreaBg,
			container.NewCenter(img),
			container.NewWithoutLayout(menuBtn),
		)
		iconAreaSized := container.NewPadded(iconArea)

		cardContent := container.NewBorder(
			nil,
			desc,
			nil,
			nil,
			container.NewVBox(
				container.NewCenter(iconAreaSized),
				container.NewCenter(text),
			),
		)
		tappable := uicommon.NewTappableContainerWithSecondary(cardContent, func() {
			l.profileList.Select(index)
		}, func(ev *fyne.PointEvent) {
			l.showProfileMenuAt(ev.AbsolutePosition, p)
		})

		bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
		bg.StrokeColor = theme.Color(theme.ColorNameButton)
		bg.StrokeWidth = 1
		bg.CornerRadius = theme.InputRadiusSize()
		if p.ID == l.selectedProfileID {
			bg.StrokeColor = theme.Color(theme.ColorNamePrimary)
			bg.StrokeWidth = 2
		}

		items = append(items, container.NewStack(bg, container.NewPadded(tappable)))
	}
	if len(items) == 0 {
		items = append(items, container.NewCenter(widget.NewLabel(lang.LocalizeKey("launcher.no_profiles", "No profiles found."))))
	}
	fyne.Do(func() {
		l.profileGrid.Objects = items
		l.profileGrid.Refresh()
	})
}

func (l *Launcher) Tab() (*container.TabItem, error) {
	// sortLabel := widget.NewLabelWithStyle(lang.LocalizeKey("launcher.sort.title", "Sort"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	sortIcon := canvas.NewImageFromResource(theme.MoveUpIcon())
	sortIcon.SetMinSize(fyne.NewSquareSize(32))
	header := container.NewVBox(
		widget.NewCard(lang.LocalizeKey("launcher.card_title", "Mod of Us"), lang.LocalizeKey("launcher.card_subtitle", "Among Us Mod Manager"), l.greetingContent),
		container.NewPadded(container.NewBorder(
			nil,
			nil,
			container.NewHBox(sortIcon, l.sortSelect),
			l.importProfileButton,
			container.NewBorder(
				nil,
				nil,
				l.toggleViewButton,
				nil,
				l.createProfileButton,
			),
		)),
		// widget.NewRichTextFromMarkdown("### "+lang.LocalizeKey("installation.installation_status", "Installation Status")), l.state.ModInstalledInfo,
		// widget.NewSeparator(),
	)

	footer := container.NewVBox(
		l.launchButton,
		l.state.ErrorText,
	)
	l.profileViews = container.NewStack(l.profileList, l.profileGridScroll)
	l.switchProfileView()

	content := container.NewBorder(
		header,
		footer,
		nil, nil,
		l.profileViews,
	)
	return container.NewTabItem(lang.LocalizeKey("launcher.tab_name", "Launcher"), content), nil
}

func (l *Launcher) runLaunch() {
	l.state.ClearError()
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.error.no_path", "Game path is not specified.")))
		return
	}

	if l.selectedProfileID == uuid.Nil {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.error.no_profile", "Please select a profile to launch.")))
		return
	}

	binaryType, err := aumgr.GetBinaryType(path)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	// Find selected profile
	var targetProfile profile.Profile
	for _, prof := range l.profiles {
		if prof.ID == l.selectedProfileID {
			targetProfile = prof
			break
		}
	}

	launchDialog, launchProgress := l.newLaunchProgressDialog()
	l.launchButton.Disable()
	if err := l.state.CanInstall.Set(false); err != nil {
		slog.Warn("Failed to set canInstall", "error", err)
	}
	if err := l.state.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set canLaunch", "error", err)
	} // Disable launch while downloading
	fyne.Do(launchDialog.Show)

	go func() {
		var launchErr error
		defer func() {
			fyne.DoAndWait(func() {
				launchDialog.Hide()
				l.checkLaunchState() // Re-enable button logic
			})
			if launchErr != nil {
				l.state.SetError(launchErr)
			}
		}()

		// Resolve dependencies
		resolvedVersions, err := l.state.Core.ResolveDependencies(targetProfile.Versions())
		if err != nil {
			launchErr = fmt.Errorf("failed to resolve dependencies: %w", err)
			return
		}

		// Download mods to cache
		configDir, err := os.UserConfigDir()
		if err != nil {
			launchErr = err
			return
		}
		cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")

		if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, launchProgress, false); err != nil {
			launchErr = err
			return
		}

		// Set active profile
		if err := l.state.ActiveProfile.Set(targetProfile.ID.String()); err != nil {
			launchErr = err
			return
		}

		l.state.ClearError()

		// Proceed to Launch
		l.state.Launch(path)
		fyne.Do(l.refreshProfiles)
	}()
}

func (l *Launcher) newLaunchProgressDialog() (*dialog.CustomDialog, *progress.FyneProgress) {
	return l.newProgressDialog(
		"launcher.launch.title",
		"Launching",
		"launcher.launch.in_progress",
		"Preparing launch. Please wait...",
	)
}

func (l *Launcher) checkLaunchState() {
	// Enable launch if profile selected and game path exists
	// We might also check if game is running (handled in state.Launch but button state is good to have)

	// Check Game Path
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		l.launchButton.Disable()
		return
	}
	if _, err := os.Stat(filepath.Join(path, "Among Us.exe")); os.IsNotExist(err) {
		l.launchButton.Disable()
		return
	}

	// Check Profile Selected
	if l.selectedProfileID == uuid.Nil {
		l.launchButton.Disable()
		return
	}

	// Check "CanLaunch" from state (e.g. game running)
	// state.CanLaunch is updated by RefreshModInstallation which checks "InstalledMod" status.
	// We want to ignore "InstalledMod" status for Profile launch, but respect "Game Running".
	// s.checkPlayingProcess() updates CanInstall/CanLaunch if running.
	// We can trust s.CanInstall or s.CanLaunch for "Not Running" status?
	// s.CanLaunch is false if "Not Installed".
	// Let's rely on s.CanInstall as a proxy for "Not Running"?
	// Or better, just check if running?
	// But `state.Launch` has a lock.

	// For now, let's enable it if profile is selected and game exists.
	// The `state.Launch` will show error if already running.
	l.launchButton.Enable()
}

func (l *Launcher) syncProfile(prof profile.Profile) {
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		dialog.ShowError(errors.New(lang.LocalizeKey("launcher.error.no_path", "Game path is not specified.")), l.state.Window)
		return
	}

	binaryType, err := aumgr.GetBinaryType(path)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	gameVersion, err := aumgr.GetVersion(path)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	syncDialog, syncProgress := l.newSyncProgressDialog()
	l.launchButton.Disable()
	if err := l.state.CanInstall.Set(false); err != nil {
		slog.Warn("Failed to set canInstall", "error", err)
	}
	if err := l.state.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set canLaunch", "error", err)
	}
	fyne.Do(syncDialog.Show)

	go func() {
		var syncErr error
		defer func() {
			fyne.DoAndWait(func() {
				syncDialog.Hide()
				l.checkLaunchState()
			})
			if syncErr != nil {
				l.state.SetError(syncErr)
				return
			}
			l.state.ClearError()
			l.state.ShowInfoDialog(
				lang.LocalizeKey("common.success", "Success"),
				lang.LocalizeKey("launcher.sync.success", "Profile has been re-synced and mods re-downloaded."),
			)
		}()

		// Resolve dependencies
		resolvedVersions, err := l.state.Core.ResolveDependencies(prof.Versions())
		if err != nil {
			syncErr = fmt.Errorf("failed to resolve dependencies: %w", err)
			return
		}

		// Download mods to cache with force=false (don't clear global cache)
		configDir, err := os.UserConfigDir()
		if err != nil {
			syncErr = err
			return
		}
		cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")

		if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, syncProgress, false); err != nil {
			syncErr = err
			return
		}

		// Sync profile directory
		if err := l.state.Core.SyncProfile(prof.ID, binaryType, gameVersion); err != nil {
			syncErr = err
			return
		}
	}()
}

func (l *Launcher) newSyncProgressDialog() (*dialog.CustomDialog, *progress.FyneProgress) {
	return l.newProgressDialog(
		"launcher.sync.title",
		"Syncing Profile",
		"launcher.sync.in_progress",
		"Syncing profile. Please wait...",
	)
}

func (l *Launcher) newProgressDialog(titleKey, titleDefault, messageKey, messageDefault string) (*dialog.CustomDialog, *progress.FyneProgress) {
	bar := widget.NewProgressBar()
	bar.SetValue(0)
	progressBar := progress.NewFyneProgress(bar)
	content := container.NewVBox(
		widget.NewLabel(lang.LocalizeKey(messageKey, messageDefault)),
		bar,
	)
	d := dialog.NewCustomWithoutButtons(
		lang.LocalizeKey(titleKey, titleDefault),
		content,
		l.state.Window,
	)
	d.Resize(fyne.NewSize(420, 140))
	return d, progressBar
}

func (l *Launcher) refreshProfiles() {
	l.profiles = l.state.ProfileManager.List()
	l.sortProfiles()
	l.profileList.Refresh()
	l.refreshProfileGrid()

	// Select active profile
	activeIDStr, _ := l.state.ActiveProfile.Get()
	if activeIDStr != "" {
		activeID, _ := uuid.Parse(activeIDStr)
		for i, p := range l.profiles {
			if p.ID == activeID {
				l.profileList.Select(i)
				break
			}
		}
	} else {
		l.profileList.UnselectAll()
		l.selectedProfileID = uuid.Nil
		l.checkLaunchState()
		l.refreshProfileGrid()
	}
}

func (l *Launcher) sortProfiles() {
	switch l.sortMode {
	case sortModePlaytime:
		sort.SliceStable(l.profiles, func(i, j int) bool {
			if l.profiles[i].PlayDurationNS == l.profiles[j].PlayDurationNS {
				return strings.Compare(strings.ToLower(l.profiles[i].Name), strings.ToLower(l.profiles[j].Name)) < 0
			}
			return l.profiles[i].PlayDurationNS > l.profiles[j].PlayDurationNS
		})
	case sortModeRecent:
		sort.SliceStable(l.profiles, func(i, j int) bool {
			li := l.profiles[i].LastLaunchedAt
			lj := l.profiles[j].LastLaunchedAt
			if li.Equal(lj) {
				return strings.Compare(strings.ToLower(l.profiles[i].Name), strings.ToLower(l.profiles[j].Name)) < 0
			}
			return li.After(lj)
		})
	default:
		slices.SortFunc(l.profiles, func(a, b profile.Profile) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
	}
}

func (l *Launcher) profileMetaText(p profile.Profile) string {
	if p.LastLaunchedAt.IsZero() {
		return lang.LocalizeKey("launcher.meta.never_launched", "Never launched")
	}
	return lang.LocalizeKey("launcher.meta.last_launched", "Last: {{.Date}}", map[string]any{
		"Date": p.LastLaunchedAt.Format("2006-01-02"),
	})
}

func normalizeSortMode(mode string) string {
	switch mode {
	case sortModePlaytime, sortModeRecent, sortModeName:
		return mode
	default:
		return sortModeName
	}
}

func (l *Launcher) sortModeLabel(mode string) string {
	switch mode {
	case sortModePlaytime:
		return lang.LocalizeKey("launcher.sort.playtime", "Play Time")
	case sortModeRecent:
		return lang.LocalizeKey("launcher.sort.recent", "Recently Launched")
	default:
		return lang.LocalizeKey("launcher.sort.name", "Name")
	}
}

func formatPlayDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int64(d.Round(time.Second).Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func (l *Launcher) showProfileMenuAt(pos fyne.Position, prof profile.Profile) {
	menu := fyne.NewMenu("",
		fyne.NewMenuItem(lang.LocalizeKey("profile.edit", "Edit"), func() {
			l.openProfileEditor(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.sync", "Sync (Clear & Re-download)"), func() {
			l.syncProfile(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.share", "Share"), func() {
			l.shareProfile(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.duplicate", "Duplicate"), func() {
			l.showDuplicateDialog(prof)
		}),
		fyne.NewMenuItem(lang.LocalizeKey("profile.delete", "Delete"), func() {
			l.deleteProfile(prof.ID)
		}),
	)
	widget.ShowPopUpMenuAtPosition(menu, l.state.Window.Canvas(), pos)
}

// -- Profile Management Methods --

func (l *Launcher) createProfile() {
	baseName := "New Profile"
	name := baseName
	counter := 1
	existing := l.state.ProfileManager.List()
	for {
		found := false
		for _, prof := range existing {
			if prof.Name == name {
				found = true
				break
			}
		}
		if !found {
			break
		}
		counter++
		name = fmt.Sprintf("%s (%d)", baseName, counter)
	}

	prof := profile.Profile{
		ID:          uuid.New(),
		Name:        name,
		ModVersions: map[string]modmgr.ModVersion{},
		UpdatedAt:   time.Now(),
	}

	if err := l.state.ProfileManager.Add(prof); err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	l.refreshProfiles()

	// Select the new profile
	for i, pr := range l.profiles {
		if pr.ID == prof.ID {
			l.profileList.Select(i)
			break
		}
	}

	l.openProfileEditor(prof)
}

func (l *Launcher) deleteProfile(id uuid.UUID) {
	if id == uuid.Nil {
		return
	}

	dialog.ShowConfirm(lang.LocalizeKey("profile.delete_confirm_title", "Delete Profile"), lang.LocalizeKey("profile.delete_confirm_message", "Are you sure you want to delete this profile?"), func(confirm bool) {
		if !confirm {
			return
		}

		if err := l.state.ProfileManager.Remove(id); err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		l.refreshProfiles()
		l.profileList.UnselectAll()
	}, l.state.Window)
}

func (l *Launcher) showDuplicateDialog(prof profile.Profile) {
	entry := widget.NewEntry()
	entry.SetText(prof.Name + " - Copy")
	entry.Validator = func(s string) error {
		if s == "" {
			return os.ErrInvalid
		}
		return nil
	}

	d := dialog.NewForm(lang.LocalizeKey("profile.duplicate_title", "Duplicate Profile"), lang.LocalizeKey("common.save", "Save"), lang.LocalizeKey("common.cancel", "Cancel"), []*widget.FormItem{
		widget.NewFormItem(lang.LocalizeKey("profile.name", "Profile Name"), entry),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		newName := entry.Text

		newProf := prof
		newProf.ID = uuid.New()
		newProf.Name = newName
		newProf.UpdatedAt = time.Now()

		if err := l.state.ProfileManager.Add(newProf); err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		l.refreshProfiles()
	}, l.state.Window)
	d.Resize(fyne.NewSize(400, 200))
	d.Show()
}

func (l *Launcher) openProfileEditor(prof profile.Profile) {
	currentProfile := prof

	var saveBtn *widget.Button
	var d *dialog.CustomDialog
	nameEntry := widget.NewEntry()
	nameEntry.SetText(currentProfile.Name)
	nameEntry.OnChanged = func(s string) {
		if nameEntry.Validate() != nil {
			saveBtn.Disable()
		} else {
			saveBtn.Enable()
		}
	}
	nameEntry.Validator = func(s string) (err error) {
		if s == "" {
			return errors.New(lang.LocalizeKey("profile.error_name_empty", "Profile name cannot be empty"))
		}
		return nil
	}
	nameForm := widget.NewForm(widget.NewFormItem(lang.LocalizeKey("profile.name", "Profile Name"), nameEntry))

	lastLaunchedText := lang.LocalizeKey("profile.stats.never_launched", "Last Launch: Never")
	if !currentProfile.LastLaunchedAt.IsZero() {
		lastLaunchedText = lang.LocalizeKey("profile.stats.last_launched", "Last Launch: {{.Date}}", map[string]any{
			"Date": currentProfile.LastLaunchedAt.Format("2006-01-02 15:04:05"),
		})
	}
	playDurationText := lang.LocalizeKey("profile.stats.play_time", "Play Time: {{.Duration}}", map[string]any{
		"Duration": formatPlayDuration(currentProfile.PlayDuration()),
	})
	statsContent := container.NewVBox(
		widget.NewLabel(lastLaunchedText),
		widget.NewLabel(playDurationText),
	)
	statsCard := widget.NewCard(lang.LocalizeKey("profile.stats.title", "Play Stats"), "", statsContent)

	iconPlaceholder := canvas.NewImageFromImage(image.NewPaletted(image.Rect(0, 0, 128, 128),
		color.Palette{theme.Color(theme.ColorNameDisabled)},
	))
	iconPlaceholder.CornerRadius = 8
	iconPlaceholder.SetMinSize(fyne.NewSize(128, 128))

	modList := widget.NewList(
		func() int { return len(currentProfile.Versions()) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("Mod Name")
			badge := widget.NewLabel("")
			badge.Hide()
			return container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.DeleteIcon(), nil), container.NewHBox(label, badge))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// This will be overridden by UpdateItem below
		},
	)

	updatesAvailable := make(map[string]string) // ModID -> LatestVersionID
	go func() {
		installed := make(map[string]string)
		for _, v := range currentProfile.Versions() {
			installed[v.ModID] = v.ID
		}
		updates, err := l.state.Rest.CheckForUpdates(installed)
		if err == nil {
			for modID, latest := range updates {
				updatesAvailable[modID] = latest.ID
			}
			fyne.Do(func() {
				modList.Refresh()
			})
		}
	}()

	// Hook up update item to ensure closure correctness
	modList.UpdateItem = func(id widget.ListItemID, item fyne.CanvasObject) {
		if id >= len(currentProfile.Versions()) {
			return
		}
		v := currentProfile.Versions()[id]
		c := item.(*fyne.Container)
		hbox := c.Objects[0].(*fyne.Container)
		label := hbox.Objects[0].(*widget.Label)
		badge := hbox.Objects[1].(*widget.Label)
		delBtn := c.Objects[1].(*widget.Button)

		fyne.DoAndWait(func() {
			label.SetText(v.ModID + " (" + v.ID + ")")
		})

		if latestID, ok := updatesAvailable[v.ModID]; ok {
			badge.SetText(lang.LocalizeKey("repository.update_available", "Update Available") + " (" + latestID + ")")
			badge.Importance = widget.WarningImportance
			badge.Show()
		} else {
			badge.Hide()
		}

		delBtn.OnTapped = func() {
			currentProfile.RemoveModVersion(v.ModID)
			modList.Refresh()
		}
	}

	addModBtn := widget.NewButtonWithIcon(lang.LocalizeKey("profile.add_mod", "Add Mod"), theme.ContentAddIcon(), func() {
		l.showAddModDialog(func(addedMods []modmgr.ModVersion) {
			for _, m := range addedMods {
				currentProfile.AddModVersion(m)
			}
			modList.Refresh()
		})
	})

	saveBtn = widget.NewButtonWithIcon(lang.LocalizeKey("common.save", "Save"), theme.DocumentSaveIcon(),
		func() {
			if err := nameForm.Validate(); err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}
			newName := nameEntry.Text
			if err := nameEntry.Validate(); err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}

			oldID := prof.ID
			currentProfile.Name = newName
			currentProfile.UpdatedAt = time.Now()

			if err := l.state.ProfileManager.Add(currentProfile); err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}

			if oldID != currentProfile.ID {
				if err := l.state.ProfileManager.Remove(oldID); err != nil {
					slog.Warn("Failed to remove old profile", "error", err)
				}
			}

			l.refreshProfiles()
			for i, pr := range l.profiles {
				if pr.ID == currentProfile.ID {
					l.profileList.Select(i)
					break
				}
			}
			d.Dismiss()
		})

	content := container.NewBorder(
		container.NewVBox(
			container.NewBorder(
				nil,
				nil,
				iconPlaceholder,
				nil,
				container.NewBorder(
					nil,
					statsCard,
					nil,
					nil,
					nameForm,
				),
			),
			widget.NewSeparator(),
			widget.NewLabel(lang.LocalizeKey("profile.mods", "Mods")),
		),
		addModBtn, nil, nil,
		modList,
	)

	d = dialog.NewCustomWithoutButtons(
		lang.LocalizeKey("profile.edit_title", "Edit Profile"),
		container.NewVScroll(content),
		l.state.Window,
	)
	d.SetButtons([]fyne.CanvasObject{
		widget.NewButtonWithIcon(lang.LocalizeKey("common.cancel", "Cancel"), theme.CancelIcon(), func() {
			d.Dismiss()
		}),
		saveBtn,
	})
	d.Resize(fyne.NewSize(500, 600))
	d.Show()
}

func (l *Launcher) showAddModDialog(onAdd func([]modmgr.ModVersion)) {
	contentBox := container.NewVBox()
	scroll := container.NewVScroll(contentBox)

	// Create dialog first
	var d *dialog.CustomDialog

	buildItem := func(title, subtitle string, onTap func()) fyne.CanvasObject {
		imgRect := canvas.NewRectangle(theme.Color(theme.ColorNameDisabled))
		imgRect.SetMinSize(fyne.NewSquareSize(80))
		img := container.NewCenter(imgRect)

		textContainer := container.NewVBox(
			widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(subtitle),
		)

		itemContent := container.New(layout.NewBorderLayout(nil, nil, img, nil),
			img,
			container.NewPadded(textContainer),
		)

		card := uicommon.NewTappableContainer(itemContent, onTap)
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
		bg.StrokeColor = theme.Color(theme.ColorNameButton)
		bg.StrokeWidth = 1
		bg.CornerRadius = theme.InputRadiusSize()
		return container.NewStack(bg, container.NewPadded(card))
	}

	go func() {
		modIDs, err := l.state.Rest.GetModIDs(100, "", "")
		if err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		fyne.Do(func() {
			contentBox.Objects = nil
			for range modIDs {
				contentBox.Add(buildItem(lang.LocalizeKey("profile.loading_mod", "Loading mod details..."), "", nil))
			}
			contentBox.Refresh()
		})

		for i, modID := range modIDs {
			go func(index int, id string) {
				mod, fetchErr := l.state.Rest.GetMod(id)
				fyne.Do(func() {
					if index >= len(contentBox.Objects) {
						return
					}
					if fetchErr != nil || mod == nil {
						if fetchErr != nil {
							slog.Warn("Failed to fetch mod details", "modID", id, "error", fetchErr)
						}
						title := lang.LocalizeKey("profile.failed_mod", "Failed to load mod '{{.ID}}'", map[string]any{"ID": id})
						subtitle := lang.LocalizeKey("profile.failed_mod_description", "Reopen this dialog to retry")
						contentBox.Objects[index] = buildItem(title, subtitle, nil)
						contentBox.Refresh()
						return
					}

					contentBox.Objects[index] = buildItem(mod.Name, mod.Author, func() {
						detailsDialog := l.newModDetailsDialog(mod, func(v modmgr.ModVersion) {
							onAdd([]modmgr.ModVersion{v})
							d.Dismiss()
						})
						detailsDialog.Show()
					})
					contentBox.Refresh()
				})
			}(i, modID)
		}
	}()

	d = dialog.NewCustom(
		lang.LocalizeKey("profile.add_mod_title", "Add Mods"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		scroll,
		l.state.Window,
	)
	d.Resize(fyne.NewSize(600, 600))
	d.Show()
}

func (l *Launcher) newModDetailsDialog(mod *modmgr.Mod, onSelect func(modmgr.ModVersion)) *dialog.CustomDialog {
	loading := widget.NewProgressBarInfinite()
	loading.Start()

	type versionRow struct {
		versionID string
		version   *modmgr.ModVersion
		err       error
		loading   bool
	}

	var rows []versionRow
	var d *dialog.CustomDialog

	versionList := widget.NewList(
		func() int { return len(rows) },
		func() fyne.CanvasObject {
			return widget.NewButton("ver", nil)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(rows) {
				return
			}
			row := rows[id]
			btn := item.(*widget.Button)

			btn.OnTapped = nil
			btn.Disable()

			switch {
			case row.loading:
				btn.SetText(lang.LocalizeKey("profile.loading_version", "Loading version '{{.ID}}'...", map[string]any{"ID": row.versionID}))
			case row.err != nil:
				btn.SetText(lang.LocalizeKey("profile.failed_version", "Failed to load version '{{.ID}}'", map[string]any{"ID": row.versionID}))
			case row.version != nil:
				btn.SetText(row.version.ID)
				btn.Enable()
				version := *row.version
				btn.OnTapped = func() {
					d.Dismiss()
					onSelect(version)
				}
			default:
				btn.SetText(lang.LocalizeKey("profile.unavailable_version", "Version '{{.ID}}' unavailable", map[string]any{"ID": row.versionID}))
			}
		},
	)

	description := widget.NewRichTextFromMarkdown(mod.Description)
	content := container.NewBorder(description,
		loading, nil, nil,
		description,
		loading,
		versionList,
	)

	d = dialog.NewCustom(mod.Name, lang.LocalizeKey("common.cancel", "Cancel"), content, l.state.Window)
	d.Resize(fyne.NewSize(400, 300))

	go func() {
		v, err := l.state.Rest.GetModVersionIDs(mod.ID, 100, "")
		if err != nil {
			d.Hide()
			dialog.ShowError(err, l.state.Window)
			return
		}
		fyne.Do(func() {
			rows = make([]versionRow, len(v))
			for i, versionID := range v {
				rows[i] = versionRow{
					versionID: versionID,
					loading:   true,
				}
			}
			versionList.Refresh()
		})

		var wg sync.WaitGroup
		for i, id := range v {
			wg.Add(1)
			go func(index int, versionID string) {
				defer wg.Done()
				version, fetchErr := l.state.Rest.GetModVersion(mod.ID, versionID)
				fyne.Do(func() {
					if index >= len(rows) {
						return
					}
					rows[index].loading = false
					rows[index].err = fetchErr
					if fetchErr == nil && version != nil {
						rows[index].version = version
					}
					versionList.RefreshItem(index)
				})
			}(i, id)
		}
		wg.Wait()
		fyne.Do(loading.Hide)
	}()
	return d
}
