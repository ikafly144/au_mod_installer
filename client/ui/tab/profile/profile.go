package profile

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

type ProfileTab struct {
	state             *uicommon.State
	profileList       *widget.List
	saveProfileButton *widget.Button
	loadProfileButton *widget.Button
	progressBar       *progress.FyneProgress

	profiles          []profile.Profile
	selectedProfileID uuid.UUID
}

var _ uicommon.Tab = (*ProfileTab)(nil)

func NewProfileTab(s *uicommon.State) uicommon.Tab {
	var p ProfileTab
	p = ProfileTab{
		state:       s,
		progressBar: progress.NewFyneProgress(widget.NewProgressBar()),
	}

	p.profileList = widget.NewList(
		func() int {
			return len(p.profiles)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template")
			menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
			menuBtn.Importance = widget.LowImportance

			return container.NewBorder(nil, nil, nil, menuBtn, menuBtn, label)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(p.profiles) {
				return
			}
			prof := p.profiles[id]
			c := item.(*fyne.Container)
			menuBtn := c.Objects[0].(*widget.Button)
			label := c.Objects[1].(*widget.Label)

			label.SetText(prof.Name)
			menuBtn.OnTapped = func() {
				menu := fyne.NewMenu("",
					fyne.NewMenuItem(lang.LocalizeKey("profile.edit", "Edit"), func() {
						p.openProfileEditor(prof)
					}),
					fyne.NewMenuItem(lang.LocalizeKey("profile.duplicate", "Duplicate"), func() {
						p.showDuplicateDialog(prof)
					}),
					fyne.NewMenuItem(lang.LocalizeKey("profile.delete", "Delete"), func() {
						p.deleteProfile(prof.ID)
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, p.state.Window.Canvas(), fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn).Add(fyne.NewPos(0, menuBtn.Size().Height)))
			}
		},
	)

	p.profileList.OnSelected = func(id widget.ListItemID) {
		if id >= len(p.profiles) {
			return
		}
		p.selectedProfileID = p.profiles[id].ID
		p.loadProfileButton.Enable()
	}
	p.profileList.OnUnselected = func(id widget.ListItemID) {
		p.selectedProfileID = uuid.Nil
		p.loadProfileButton.Disable()
	}
	p.profileList.FocusLost()

	p.saveProfileButton = widget.NewButtonWithIcon(lang.LocalizeKey("profile.create", "Create Profile"), theme.DocumentCreateIcon(), p.createProfile)
	p.loadProfileButton = widget.NewButtonWithIcon(lang.LocalizeKey("profile.load", "Load Profile"), theme.DownloadIcon(), func() {
		p.loadProfile(p.selectedProfileID)
	})

	p.loadProfileButton.Disable()

	p.refreshProfiles()

	return &p
}

func (p *ProfileTab) refreshProfiles() {
	p.profiles = p.state.ProfileManager.List()
	p.profileList.Refresh()
}

func (p *ProfileTab) Tab() (*container.TabItem, error) {
	content := container.NewBorder(
		widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("profile.title", "Profiles")),
		container.NewVBox(
			widget.NewSeparator(),
			p.saveProfileButton,
			p.loadProfileButton,
			p.progressBar.Canvas(),
		),
		nil, nil,
		p.profileList,
	)

	return container.NewTabItem(lang.LocalizeKey("profile.tab_name", "Profile"), content), nil
}

func (p *ProfileTab) newModDetailsDialog(mod modmgr.Mod, onSelect func(modmgr.ModVersion)) *dialog.CustomDialog {
	// Fetch versions
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

	d = dialog.NewCustom(mod.Name, lang.LocalizeKey("common.cancel", "Cancel"), content, p.state.Window)
	d.Resize(fyne.NewSize(400, 300))

	go func() {
		defer fyne.Do(loading.Hide)
		v, err := p.state.Rest.GetModVersions(mod.ID, 100, "")
		if err != nil {
			d.Hide()
			dialog.ShowError(err, p.state.Window)
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

func (p *ProfileTab) deleteProfile(id uuid.UUID) {
	if id == uuid.Nil {
		return
	}

	dialog.ShowConfirm(lang.LocalizeKey("profile.delete_confirm_title", "Delete Profile"), lang.LocalizeKey("profile.delete_confirm_message", "Are you sure you want to delete this profile?"), func(confirm bool) {
		if !confirm {
			return
		}

		if err := p.state.ProfileManager.Remove(id); err != nil {
			dialog.ShowError(err, p.state.Window)
			return
		}
		p.refreshProfiles()
		p.profileList.UnselectAll()
	}, p.state.Window)
}

func (p *ProfileTab) loadProfile(id uuid.UUID) {
	if id == uuid.Nil {
		return
	}

	profiles := p.state.ProfileManager.List()
	var targetProfile profile.Profile
	found := false
	for _, prof := range profiles {
		if prof.ID == id {
			targetProfile = prof
			found = true
			break
		}
	}
	if !found {
		return
	}

	path := p.state.ModInstallDir()
	if path == "" {
		dialog.ShowError(os.ErrNotExist, p.state.Window)
		return
	}

	binaryType, err := aumgr.GetBinaryType(path)
	if err != nil {
		dialog.ShowError(err, p.state.Window)
		return
	}

	p.progressBar.Canvas().Show()
	p.state.CanInstall.Set(false)
	p.state.CanLaunch.Set(false)

	go func() {
		defer fyne.Do(func() {
			p.progressBar.Canvas().Hide()
			p.state.CanInstall.Set(true)
			p.state.RefreshModInstallation()
		})
		// New flow: Download mods to cache
		configDir, err := os.UserConfigDir()
		if err != nil {
			p.state.SetError(err)
			return
		}
		cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")

		if err := modmgr.DownloadMods(cacheDir, targetProfile.Versions(), binaryType, p.progressBar); err != nil {
			p.state.SetError(err)
			return
		}

		// Set active profile
		if err := p.state.ActiveProfile.Set(targetProfile.ID.String()); err != nil {
			p.state.SetError(err)
			return
		}

		p.state.ClearError()
		// Show success message?
		// Or maybe just update UI to show "Ready to launch with profile X"
	}()
}

func (p *ProfileTab) createProfile() {
	// Generate unique name
	baseName := "New Profile" // FIXME: localization
	name := baseName
	counter := 1
	existing := p.state.ProfileManager.List()
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
		LastUpdated: time.Now(),
	}

	if err := p.state.ProfileManager.Add(prof); err != nil {
		dialog.ShowError(err, p.state.Window)
		return
	}
	p.refreshProfiles()

	// Select the new profile
	for i, pr := range p.profiles {
		if pr.ID == prof.ID {
			p.profileList.Select(i)
			break
		}
	}

	p.openProfileEditor(prof)
}

func (p *ProfileTab) openProfileEditor(prof profile.Profile) {
	// Working copy
	currentProfile := prof

	nameEntry := widget.NewEntry()
	nameEntry.SetText(currentProfile.Name)
	nameEntry.Validator = func(s string) error {
		if s == "" {
			return errors.New(lang.LocalizeKey("profile.error_name_empty", "Profile name cannot be empty"))
		}
		return nil
	}

	// Icon placeholder
	iconPlaceholder := canvas.NewImageFromImage(image.NewPaletted(image.Rectangle{
		Max: image.Point{64, 64},
	}, color.Palette{theme.Color(theme.ColorNameDisabled)}))
	iconPlaceholder.SetMinSize(fyne.NewSize(128, 128))

	// Mod List
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

			label.SetText(v.ModID + " (" + v.ID + ")") // Display ModID and VersionID
			delBtn.OnTapped = func() {
				// Remove mod
				currentProfile.RemoveModVersion(v.ModID)
				// Refresh list
				item.(*fyne.Container).Refresh() // This might not be enough, need to refresh list
				// Hacky refresh
				// Ideally we should use binding, but for now let's just refresh the widget
				// But widget.List refresh doesn't handle length change well if not using binding or careful
				// Let's just refresh the dialog content or the list
				// Actually, widget.List handles length change if we call Refresh()
			}
		},
	)
	// We need to handle the delete button callback carefully because 'id' is captured.
	// But widget.List reuses items, so we can't rely on closure capture of 'id' in OnTapped if the item is reused for another index.
	// Wait, OnTapped is set in UpdateItem (the 3rd func). That is called when data changes.
	// So capturing 'id' there is correct for that moment.
	// BUT if we delete an item, the list length changes.
	// Let's use a more robust delete approach.
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
			// Remove at index id
			if id < len(currentProfile.Versions()) {
				currentProfile.RemoveModVersion(v.ModID)
				modList.Refresh()
			}
		}
	}

	addModBtn := widget.NewButtonWithIcon(lang.LocalizeKey("profile.add_mod", "Add Mod"), theme.ContentAddIcon(), func() {
		p.showAddModDialog(func(addedMods []modmgr.ModVersion) {
			for _, m := range addedMods {
				currentProfile.AddModVersion(m)
			}
			modList.Refresh()
		})
	})

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(iconPlaceholder, layout.NewSpacer()), // Icon area
			widget.NewForm(widget.NewFormItem(lang.LocalizeKey("profile.name", "Profile Name"), nameEntry)),
			widget.NewSeparator(),
			widget.NewLabel(lang.LocalizeKey("profile.mods", "Mods")),
		),
		addModBtn, nil, nil,
		modList,
	)

	d := dialog.NewCustomConfirm(
		lang.LocalizeKey("profile.edit_title", "Edit Profile"),
		lang.LocalizeKey("common.save", "Save"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		content,
		func(confirm bool) {
			if !confirm {
				return
			}
			newName := nameEntry.Text
			if err := nameEntry.Validate(); err != nil {
				dialog.ShowError(err, p.state.Window)
				return
			}

			// Update profile
			// If name changed, we might need to handle ID change if ID=Name
			// For now assuming ID=Name
			oldID := prof.ID
			currentProfile.Name = newName
			currentProfile.LastUpdated = time.Now()

			if err := p.state.ProfileManager.Add(currentProfile); err != nil {
				dialog.ShowError(err, p.state.Window)
				return
			}

			if oldID != currentProfile.ID {
				p.state.ProfileManager.Remove(oldID)
			}

			p.refreshProfiles()
			// Reselect if needed
			for i, pr := range p.profiles {
				if pr.ID == currentProfile.ID {
					p.profileList.Select(i)
					break
				}
			}
		},
		p.state.Window,
	)
	d.Resize(fyne.NewSize(500, 600))
	d.Show()
}

func (p *ProfileTab) showAddModDialog(onAdd func([]modmgr.ModVersion)) {
	// Fetch mods first
	p.progressBar.Canvas().Show()
	defer p.progressBar.Canvas().Hide()
	var mods []modmgr.Mod
	// Mod selection state

	var modList *widget.List

	go func() {
		m, err := p.state.Rest.GetModList(100, "", "") // FIXME: pagination?
		if err != nil {
			dialog.ShowError(err, p.state.Window)
			return
		}
		mods = m
		fyne.Do(modList.Refresh)
	}()

	// Mod List
	modList = widget.NewList(
		func() int { return len(mods) },
		func() fyne.CanvasObject {
			modIcon := canvas.NewSquare(theme.Color(theme.ColorNameDisabled)) // TODO: Load actual mod icon if available
			modIcon.SetMinSize(fyne.NewSize(96, 96))
			modName := widget.NewLabel("Mod Name")
			modDescription := widget.NewRichText()
			versionLabel := widget.NewLabel("(None)")
			versionLabel.SizeName = theme.SizeNameCaptionText
			return container.NewBorder(nil, nil, modIcon, nil,
				modIcon,
				container.NewVBox(
					modName,
					modDescription,
					versionLabel,
				),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(mods) {
				return
			}
			mod := mods[id]
			c := item.(*fyne.Container)
			// modIcon := c.Objects[0].(*canvas.Rectangle)
			modName := c.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
			modDescription := c.Objects[1].(*fyne.Container).Objects[1].(*widget.RichText)
			versionLabel := c.Objects[1].(*fyne.Container).Objects[2].(*widget.Label)

			// Update mod info
			modName.SetText(mod.Name)

			modDescription.ParseMarkdown(mod.Description)

			// Update Version Label
			versionLabel.SetText(mod.LatestVersion)
		},
	)

	var d *dialog.CustomDialog
	modList.OnSelected = func(id widget.ListItemID) {
		// Open details
		if id >= len(mods) {
			return
		}
		mod := mods[id]
		d := p.newModDetailsDialog(mod, func(v modmgr.ModVersion) {
			// Add to profile
			onAdd([]modmgr.ModVersion{v})
			d.Dismiss()
			modList.Unselect(id)
		})
		d.SetOnClosed(modList.UnselectAll)
		d.Show()
	}

	d = dialog.NewCustom(
		lang.LocalizeKey("profile.add_mod_title", "Add Mods"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		modList,
		p.state.Window,
	)
	d.Resize(fyne.NewSize(600, 600))
	d.Show()
}

func (p *ProfileTab) showDuplicateDialog(prof profile.Profile) {
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
		newProf.LastUpdated = time.Now()

		if err := p.state.ProfileManager.Add(newProf); err != nil {
			dialog.ShowError(err, p.state.Window)
			return
		}
		p.refreshProfiles()
	}, p.state.Window)
	d.Resize(fyne.NewSize(400, 200))
	d.Show()
}
