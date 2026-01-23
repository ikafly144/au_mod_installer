package launcher

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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

	profileList       *widget.List
	progressBar       *progress.FyneProgress
	profiles          []profile.Profile
	selectedProfileID uuid.UUID

	canLaunchListener binding.DataListener
}

var _ uicommon.Tab = (*Launcher)(nil)

func NewLauncherTab(s *uicommon.State) uicommon.Tab {
	var l Launcher
	revision := fyne.CurrentApp().Metadata().Custom["revision"]
	revision = revision[:min(7, len(revision))]
	l = Launcher{
		state:               s,
		progressBar:         progress.NewFyneProgress(widget.NewProgressBar()),
		launchButton:        widget.NewButtonWithIcon(lang.LocalizeKey("launcher.launch", "Launch"), theme.MediaPlayIcon(), l.runLaunch),
		createProfileButton: widget.NewButtonWithIcon(lang.LocalizeKey("profile.create", "Create Profile"), theme.ContentAddIcon(), l.createProfile),
		greetingContent:     widget.NewLabelWithStyle(fmt.Sprintf("バージョン：%s (%s)", s.Version, revision), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
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
					l.importProfile(*prof)
				}, l.state.Window)
				return
			}
		}

		l.importProfile(*prof)
	}, l.state.Window)
}

func (l *Launcher) importProfile(prof profile.Profile) {
	prof.UpdatedAt = time.Now()
	// prof.UpdatedAt is preserved from import

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
				widget.ShowPopUpMenuAtPosition(menu, l.state.Window.Canvas(), fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn).Add(fyne.NewPos(0, menuBtn.Size().Height)))
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
	}
	l.profileList.OnUnselected = func(id widget.ListItemID) {
		l.selectedProfileID = uuid.Nil
		l.checkLaunchState()
	}
}

func (l *Launcher) Tab() (*container.TabItem, error) {
	header := container.NewVBox(
		widget.NewCard(lang.LocalizeKey("launcher.card_title", "Mod of Us"), lang.LocalizeKey("launcher.card_subtitle", "Among Us Mod Manager"), l.greetingContent),
		// widget.NewRichTextFromMarkdown("### "+lang.LocalizeKey("installation.installation_status", "Installation Status")), l.state.ModInstalledInfo,
		// widget.NewSeparator(),
	)

	footer := container.NewVBox(
		widget.NewSeparator(),
		container.NewGridWithColumns(2, l.createProfileButton, l.launchButton),
		l.progressBar.Canvas(),
		l.state.ErrorText,
	)

	content := container.NewBorder(
		header,
		footer,
		nil, nil,
		l.profileList,
	)
	return container.NewTabItem(lang.LocalizeKey("launcher.tab_name", "Launcher"), content), nil
}

func (l *Launcher) runLaunch() {
	l.state.ErrorText.Hide()
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		l.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("launcher.error.no_path", "Game path is not specified."), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		l.state.ErrorText.Refresh()
		l.state.ErrorText.Show()
		return
	}

	if l.selectedProfileID == uuid.Nil {
		l.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("launcher.error.no_profile", "Please select a profile to launch."), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		l.state.ErrorText.Refresh()
		l.state.ErrorText.Show()
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

	l.progressBar.Canvas().Show()
	l.launchButton.Disable()
	if err := l.state.CanInstall.Set(false); err != nil {
		slog.Warn("Failed to set canInstall", "error", err)
	}
	if err := l.state.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set canLaunch", "error", err)
	} // Disable launch while downloading

	go func() {
		defer fyne.Do(func() {
			l.progressBar.Canvas().Hide()
			l.checkLaunchState() // Re-enable button logic
			// l.state.CanInstall.Set(true)
		})

		// Resolve dependencies
		resolvedVersions, err := l.state.Core.ResolveDependencies(targetProfile.Versions())
		if err != nil {
			l.state.SetError(fmt.Errorf("failed to resolve dependencies: %w", err))
			return
		}

		// Download mods to cache
		configDir, err := os.UserConfigDir()
		if err != nil {
			l.state.SetError(err)
			return
		}
		cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")

		if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, l.progressBar, false); err != nil {
			l.state.SetError(err)
			return
		}

		// Set active profile
		if err := l.state.ActiveProfile.Set(targetProfile.ID.String()); err != nil {
			l.state.SetError(err)
			return
		}

		l.state.ClearError()

		// Proceed to Launch
		l.state.Launch(path)
	}()
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

	l.progressBar.Canvas().Show()
	l.launchButton.Disable()
	if err := l.state.CanInstall.Set(false); err != nil {
		slog.Warn("Failed to set canInstall", "error", err)
	}
	if err := l.state.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set canLaunch", "error", err)
	}

	go func() {
		defer fyne.Do(func() {
			l.progressBar.Canvas().Hide()
			l.checkLaunchState()
		})

		// Resolve dependencies
		resolvedVersions, err := l.state.Core.ResolveDependencies(prof.Versions())
		if err != nil {
			l.state.SetError(fmt.Errorf("failed to resolve dependencies: %w", err))
			return
		}

		// Download mods to cache with force=false (don't clear global cache)
		configDir, err := os.UserConfigDir()
		if err != nil {
			l.state.SetError(err)
			return
		}
		cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")

		if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, l.progressBar, false); err != nil {
			l.state.SetError(err)
			return
		}

		// Sync profile directory
		if err := l.state.Core.SyncProfile(prof.ID, binaryType, gameVersion); err != nil {
			l.state.SetError(err)
			return
		}

		l.state.ClearError()
		fyne.Do(func() {
			dialog.ShowInformation("Sync Complete", "Profile has been re-synced and mods re-downloaded.", l.state.Window)
		})
	}()
}

func (l *Launcher) refreshProfiles() {
	l.profiles = l.state.ProfileManager.List()
	l.profileList.Refresh()

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
	}
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

	iconPlaceholder := canvas.NewImageFromImage(image.NewPaletted(image.Rect(0, 0, 128, 128),
		color.Palette{theme.Color(theme.ColorNameDisabled)},
	))
	iconPlaceholder.CornerRadius = 8
	iconPlaceholder.SetMinSize(fyne.NewSize(128, 128))

	modList := widget.NewList(
		func() int { return len(currentProfile.Versions()) },
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.DeleteIcon(), nil), widget.NewLabel("Mod Name"))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(currentProfile.Versions()) {
				return
			}
			v := currentProfile.Versions()[id]
			c := item.(*fyne.Container)
			label := c.Objects[0].(*widget.Label)
			delBtn := c.Objects[1].(*widget.Button)

			label.SetText(v.ModID + " (" + v.ID + ")")
			delBtn.OnTapped = func() {
				// We can't safely use 'id' or 'v' in closure here if list changes?
				// But we will refresh the list, so it's fine for one action.
				currentProfile.RemoveModVersion(v.ModID)
				// Force refresh of the dialog content or list
				// Since we are inside the list callback, we need to be careful.
				// But OnTapped is called later.
				// We will trigger a refresh of the list widget from outside.
			}
		},
	)

	// Hook up update item to ensure closure correctness
	modList.UpdateItem = func(id widget.ListItemID, item fyne.CanvasObject) {
		if id >= len(currentProfile.Versions()) {
			return
		}
		v := currentProfile.Versions()[id]
		c := item.(*fyne.Container)
		label := c.Objects[0].(*widget.Label)
		delBtn := c.Objects[1].(*widget.Button)

		label.SetText(v.ModID + " (" + v.ID + ")")
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
			container.NewBorder(nil, nil, iconPlaceholder, nil,
				iconPlaceholder,
				nameForm,
			),
			widget.NewSeparator(),
			widget.NewLabel(lang.LocalizeKey("profile.mods", "Mods")),
		),
		addModBtn, nil, nil,
		modList,
	)

	d = dialog.NewCustomWithoutButtons(
		lang.LocalizeKey("profile.edit_title", "Edit Profile"),
		content,
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
	l.progressBar.Canvas().Show()
	defer l.progressBar.Canvas().Hide()

	contentBox := container.NewVBox()
	scroll := container.NewVScroll(contentBox)

	// Create dialog first
	var d *dialog.CustomDialog

	// Refresh function
	refreshList := func(mods []modmgr.Mod) {
		contentBox.Objects = nil
		for _, mod := range mods {
			mod := mod // capture loop var

			// Create Item UI
			imgRect := canvas.NewRectangle(theme.Color(theme.ColorNameDisabled))
			imgRect.SetMinSize(fyne.NewSquareSize(80))
			img := container.NewCenter(imgRect)

			textContainer := container.NewVBox(
				widget.NewLabelWithStyle(mod.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(mod.Author),
			)

			itemContent := container.New(layout.NewBorderLayout(nil, nil, img, nil),
				img,
				container.NewPadded(textContainer),
			)

			// Make clickable
			card := uicommon.NewTappableContainer(itemContent, func() {
				detailsDialog := l.newModDetailsDialog(mod, func(v modmgr.ModVersion) {
					onAdd([]modmgr.ModVersion{v})
					d.Dismiss()
				})
				detailsDialog.Show()
			})

			// Styling
			bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
			bg.StrokeColor = theme.Color(theme.ColorNameButton)
			bg.StrokeWidth = 1
			bg.CornerRadius = theme.InputRadiusSize()

			item := container.NewStack(bg, container.NewPadded(card))
			contentBox.Add(item)
		}
		contentBox.Refresh()
	}

	go func() {
		m, err := l.state.Rest.GetModList(100, "", "")
		if err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		fyne.Do(func() {
			refreshList(m)
		})
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

func (l *Launcher) newModDetailsDialog(mod modmgr.Mod, onSelect func(modmgr.ModVersion)) *dialog.CustomDialog {
	loading := widget.NewProgressBarInfinite()
	loading.Start()

	var versions []modmgr.ModVersion
	var d *dialog.CustomDialog

	versionList := widget.NewList(
		func() int { return len(versions) },
		func() fyne.CanvasObject {
			return widget.NewButton("ver", nil)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(versions) {
				return
			}
			v := versions[id]
			btn := item.(*widget.Button)
			btn.SetText(v.ID)
			btn.OnTapped = func() {
				d.Dismiss()
				onSelect(v)
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
		defer fyne.Do(loading.Hide)
		v, err := l.state.Rest.GetModVersions(mod.ID, 100, "")
		if err != nil {
			d.Hide()
			dialog.ShowError(err, l.state.Window)
			return
		}
		sort.SliceStable(v, func(i, j int) bool {
			return v[i].CreatedAt.After(v[j].CreatedAt)
		})
		versions = v
		fyne.Do(func() {
			versionList.Refresh()
		})
	}()
	return d
}
