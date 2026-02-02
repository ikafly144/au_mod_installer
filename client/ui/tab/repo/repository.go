package repo

import (
	"fmt"
	"log/slog"
	"net/url"
	"slices"
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
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

const ModsPerPage = 10

type Repository struct {
	state *uicommon.State
	mu    sync.Mutex

	lastModID  string
	noMoreMods bool
	modsBind   binding.List[modmgr.Mod]

	// Containers
	mainContainer *fyne.Container // Stack container for switching views
	listView      *fyne.Container // The list view container
	detailView    *fyne.Container // The detail view container

	// List View Elements
	modListContainer *fyne.Container
	modScroll        *container.Scroll
	progressBar      *progress.FyneProgress
	searchBar        *widget.Entry
	reloadBtn        *widget.Button
	stateLabel       *widget.Label
}

func NewRepository(state *uicommon.State) *Repository {
	bind := binding.NewList(func(a, b modmgr.Mod) bool { return a.ID == b.ID })

	repo := &Repository{
		state:            state,
		lastModID:        "",
		modsBind:         bind,
		modListContainer: container.NewVBox(),
		progressBar:      progress.NewFyneProgress(widget.NewProgressBar()),
		stateLabel:       widget.NewLabel(""),
	}

	// Initialize UI components
	repo.searchBar = widget.NewEntry()
	repo.searchBar.SetPlaceHolder(lang.LocalizeKey("repository.search_placeholder", "Filter mods by name"))
	repo.searchBar.OnChanged = func(s string) {
		go repo.updateModList(s)
	}

	repo.reloadBtn = widget.NewButtonWithIcon(lang.LocalizeKey("repository.reload", "Reload"), theme.ViewRefreshIcon(), func() {
		repo.reloadBtn.Disable()
		go func() {
			repo.reloadMods()
		}()
	})

	repo.stateLabel.Hide()
	repo.stateLabel.Wrapping = fyne.TextWrapWord

	repo.modScroll = container.NewVScroll(repo.modListContainer)
	repo.modScroll.OnScrolled = func(pos fyne.Position) {
		if pos.Y >= repo.modListContainer.Size().Height-repo.modScroll.Size().Height {
			repo.LoadNext()
		}
	}

	// Build List View
	top := container.New(layout.NewBorderLayout(nil, nil, nil, repo.reloadBtn),
		repo.searchBar,
		repo.reloadBtn,
	)
	bottom := container.NewVBox(
		repo.state.ErrorText,
		repo.stateLabel,
		repo.progressBar.Canvas(),
	)
	repo.listView = container.New(layout.NewBorderLayout(top, bottom, nil, nil),
		top,
		bottom,
		repo.modScroll,
	)

	// Initialize Detail View (empty for now)
	repo.detailView = container.NewStack()

	repo.mainContainer = container.NewStack(repo.listView, repo.detailView)
	repo.detailView.Hide()

	state.ActiveProfile.AddListener(binding.NewDataListener(func() {
		repo.updateModList(repo.searchBar.Text)
	}))

	repo.init()
	return repo
}

func (r *Repository) init() {
	go func() {
		if err, _ := r.fetchMods(); err != nil {
			slog.Error("Failed to refresh mods in repository tab", "error", err)
			r.state.SetError(fmt.Errorf("%s", lang.LocalizeKey("repository.failed_to_load", "Failed to load mods: {{.Error}}", map[string]any{"Error": err.Error()})))
		}
	}()
}

func (r *Repository) Tab() (*container.TabItem, error) {
	return container.NewTabItem(lang.LocalizeKey("repository.tab_name", "Repository"), r.mainContainer), nil
}

func (r *Repository) updateModList(filter string) {
	defer fyne.Do(r.reloadBtn.Enable)
	var objs []fyne.CanvasObject
	mods, err := r.modsBind.Get()
	if err != nil {
		slog.Error("Failed to get mods from binding", "error", err)
		return
	}

	searchText := strings.ToLower(filter)
	for _, mod := range mods {
		if !mod.Type.IsVisible() {
			continue
		}
		if searchText != "" && !strings.Contains(strings.ToLower(mod.Name), searchText) {
			continue
		}

		// Create List Item
		imgRect := canvas.NewRectangle(theme.Color(theme.ColorNameDisabled))
		imgRect.SetMinSize(fyne.NewSquareSize(80))
		// Use a container that centers the image to maintain its aspect ratio
		// while the BorderLayout stretches the container itself.
		img := container.NewCenter(imgRect)

		titleLabel := widget.NewLabelWithStyle(mod.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

		updateBadge := widget.NewLabel("")
		updateBadge.Hide()

		activeProfileIDStr, _ := r.state.ActiveProfile.Get()
		if activeProfileIDStr != "" {
			if activeID, err := uuid.Parse(activeProfileIDStr); err == nil {
				if activeProfile, ok := r.state.ProfileManager.Get(activeID); ok {
					if installedVersion, ok := activeProfile.ModVersions[mod.ID]; ok {
						if installedVersion.ID != mod.LatestVersion {
							updateBadge.SetText(lang.LocalizeKey("repository.update_available", "Update Available"))
							updateBadge.Importance = widget.WarningImportance
							updateBadge.Show()
						}
					}
				}
			}
		}

		textContainer := container.NewVBox(
			container.NewHBox(titleLabel, updateBadge),
			widget.NewLabel(mod.Author),
			widget.NewLabel(mod.Description),
		)
		textContainer.Objects[2].(*widget.Label).Wrapping = fyne.TextWrapWord

		// Border layout with centered img on the left
		content := container.New(layout.NewBorderLayout(nil, nil, img, nil),
			img,
			container.NewPadded(textContainer),
		)

		// Make it clickable
		card := uicommon.NewTappableContainer(content, func() {
			r.showModDetails(mod)
		})

		// Add some padding/background similar to a Card
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
		bg.StrokeColor = theme.Color(theme.ColorNameButton)
		bg.StrokeWidth = 1
		bg.CornerRadius = theme.InputRadiusSize()

		item := container.NewStack(bg, container.NewPadded(card))

		objs = append(objs, item)
	}

	if len(objs) == 0 {
		objs = append(objs, container.NewCenter(widget.NewLabel(lang.LocalizeKey("repository.no_mods_found", "No mods found."))))
	}

	r.mu.Lock()
	noMore := r.noMoreMods
	r.mu.Unlock()

	if !noMore && len(mods) > 0 {
		objs = append(objs, widget.NewButton(lang.LocalizeKey("repository.load_next", "Load more..."), r.LoadNext))
	}

	fyne.Do(func() {
		r.modListContainer.Objects = objs
		r.modListContainer.Refresh()
		r.modScroll.Refresh()
	})
}

func (r *Repository) showModDetails(mod modmgr.Mod) {
	// Navigation Bar (Topmost)
	backBtn := widget.NewButtonWithIcon(lang.LocalizeKey("common.back", "Back"), theme.NavigateBackIcon(), func() {
		r.detailView.Hide()
		r.listView.Show()
	})
	topBar := container.NewHBox(backBtn)

	// Header Info
	img := canvas.NewSquare(theme.Color(theme.ColorNameDisabled))
	img.SetMinSize(fyne.NewSize(128, 128))

	headerText := container.NewVBox(
		widget.NewLabelWithStyle(mod.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(lang.LocalizeKey("repository.author", "Author: {{.Author}}", map[string]any{"Author": mod.Author})),
	)

	if mod.Website != "" {
		if u, err := url.Parse(mod.Website); err == nil {
			headerText.Add(widget.NewHyperlink(lang.LocalizeKey("repository.website", "Website"), u))
		} else {
			slog.Warn("Failed to parse mod website URL", "url", mod.Website, "error", err)
		}
	}

	headerText.Add(widget.NewButton(lang.LocalizeKey("repository.install_latest", "Install Latest"), func() {
		r.installModVersion(mod, mod.LatestVersion)
	}))

	header := container.New(layout.NewBorderLayout(nil, nil, img, nil),
		img,
		headerText,
	)

	// Tabs
	detailsTab := container.NewTabItem(lang.LocalizeKey("repository.tab.details", "Details"),
		container.NewVScroll(widget.NewLabel(mod.Description)),
	)

	versionsList := container.NewVBox()
	versionsTab := container.NewTabItem(lang.LocalizeKey("repository.tab.versions", "Versions"),
		container.NewVScroll(versionsList),
	)

	// Loading versions
	versionsList.Add(widget.NewProgressBarInfinite())
	go func() {
		versions, err := r.state.Rest.GetModVersions(mod.ID, 100, "")
		fyne.Do(func() {
			versionsList.Objects = nil
			if err != nil {
				versionsList.Add(widget.NewLabel("Failed to load versions: " + err.Error()))
				return
			}
			for _, v := range versions {
				verLabel := widget.NewLabel(v.ID)
				addBtn := widget.NewButton(lang.LocalizeKey("repository.add_to_profile", "Add to Profile"), func() {
					r.installModVersion(mod, v.ID)
				})
				row := container.New(layout.NewBorderLayout(nil, nil, nil, addBtn),
					addBtn,
					verLabel,
				)
				versionsList.Add(row)
				versionsList.Add(widget.NewSeparator())
			}
			versionsTab.Content.Refresh()
		})
	}()

	tabs := container.NewAppTabs(detailsTab, versionsTab)

	// Assemble Detail View
	detailContent := container.New(layout.NewBorderLayout(header, nil, nil, nil),
		header,
		tabs,
	)

	finalContent := container.New(layout.NewBorderLayout(topBar, nil, nil, nil),
		topBar,
		detailContent,
	)

	r.detailView.Objects = []fyne.CanvasObject{finalContent}
	r.detailView.Refresh()

	r.listView.Hide()
	r.detailView.Show()
}

func (r *Repository) installModVersion(mod modmgr.Mod, versionID string) {
	r.stateLabel.Hide()

	profiles := r.state.ProfileManager.List()
	if len(profiles) == 0 {
		r.state.SetError(fmt.Errorf("%s", lang.LocalizeKey("repository.error.no_profiles", "No profiles found. Please create one in the Launcher tab.")))
		return
	}

	var selectedProfile *profile.Profile
	profileNames := make([]string, len(profiles))
	for i, p := range profiles {
		profileNames[i] = p.Name
	}

	selectWidget := widget.NewSelect(profileNames, func(s string) {
		for _, p := range profiles {
			if p.Name == s {
				pCopy := p
				selectedProfile = &pCopy
				break
			}
		}
	})

	// Pre-select active profile
	activeIDStr, _ := r.state.ActiveProfile.Get()
	if activeIDStr != "" {
		activeID, _ := uuid.Parse(activeIDStr)
		for _, p := range profiles {
			if p.ID == activeID {
				selectWidget.SetSelected(p.Name)
				break
			}
		}
	}
	if selectedProfile == nil && len(profiles) > 0 {
		selectWidget.SetSelectedIndex(0)
	}

	d := dialog.NewCustomConfirm(
		lang.LocalizeKey("repository.select_profile_title", "Select Profile"),
		lang.LocalizeKey("common.add", "Add"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		container.NewVBox(
			widget.NewLabel(lang.LocalizeKey("repository.select_profile_msg", "Select a profile to add this mod to:")),
			selectWidget,
		),
		func(confirm bool) {
			if !confirm || selectedProfile == nil {
				return
			}

			r.state.ClearError()
			targetID := selectedProfile.ID

			go func() {
				targetProfile, found := r.state.ProfileManager.Get(targetID)
				if !found {
					fyne.Do(func() {
						r.state.SetError(fmt.Errorf("profile not found"))
					})
					return
				}

				versionData, err := r.state.Rest.GetModVersion(mod.ID, versionID)
				if err != nil {
					slog.Error("Failed to get mod version for installation", "modId", mod.ID, "versionId", versionID, "error", err)
					r.state.SetError(err)
					return
				}

				targetProfile.AddModVersion(*versionData)
				targetProfile.UpdatedAt = time.Now()

				if err := r.state.ProfileManager.Add(targetProfile); err != nil {
					slog.Error("Failed to add mod to profile", "error", err)
					r.state.SetError(err)
				} else {
					slog.Info("Mod added to profile", "modId", mod.ID, "versionId", versionID, "profile", targetProfile.Name)
					r.state.ClearError()
					fyne.DoAndWait(func() {
						r.stateLabel.SetText(lang.LocalizeKey("repository.added_to_profile", "Added to profile '{{.Profile}}': {{.ModName}} ({{.Version}})", map[string]any{"Profile": targetProfile.Name, "ModName": mod.Name, "Version": versionID}))
						r.stateLabel.Show()
					})
				}
			}()
		},
		r.state.Window,
	)
	d.Show()
}

func (r *Repository) LoadNext() {
	r.mu.Lock()
	if r.noMoreMods {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	go func() {
		mods, _ := r.modsBind.Get()
		slog.Info("Loading next mods in repository tab", "current_mods", mods)
		if err, ok := r.fetchMods(); err != nil {
			slog.Error("Failed to load next mods in repository tab", "error", err)
			r.state.SetError(fmt.Errorf("%s", lang.LocalizeKey("repository.failed_to_load", "Failed to load mods: {{.Error}}", map[string]any{"Error": err.Error()})))
		} else {
			if !ok {
				slog.Info("No more mods to load in repository tab")
				return
			}
		}
	}()
}

func (r *Repository) fetchMods() (error, bool) {
	defer func() {
		var text string
		fyne.DoAndWait(func() {
			text = r.searchBar.Text
		})
		r.updateModList(text)
	}()
	if r.state.Rest != nil {
		mods, err := r.modsBind.Get()
		if err != nil {
			return err, false
		}
		var afterId, lastId string
		r.mu.Lock()
		if r.lastModID != "" && len(mods) > 0 {
			afterId = r.lastModID
		}
		r.mu.Unlock()

		slog.Info("Refreshing mods", "firstId", afterId, "lastId", lastId)

		if mods, err := r.state.Rest.GetModList(ModsPerPage, afterId, lastId); err != nil {
			return err, false
		} else if len(mods) > 0 {
			for _, m := range mods {
				if err := r.modsBind.Append(m); err != nil {
					return err, false
				}
			}
			r.mu.Lock()
			r.lastModID = mods[len(mods)-1].ID
			if len(mods) < ModsPerPage {
				r.noMoreMods = true
			}
			r.mu.Unlock()

			if !slices.ContainsFunc(mods, func(m modmgr.Mod) bool {
				return m.Type.IsVisible()
			}) {
				slog.Info("No visible mods in loaded mods, loading next page")
				return r.fetchMods()
			}

			return nil, true
		} else {
			r.mu.Lock()
			r.noMoreMods = true
			r.mu.Unlock()
			slog.Info("No more mods to load")
			return nil, false
		}
	}
	slog.Error("rest client is nil, cannot refresh mods")
	return nil, false
}

func (r *Repository) reloadMods() {
	slog.Info("Reloading repository mods")
	r.mu.Lock()
	r.lastModID = ""
	r.noMoreMods = false
	r.mu.Unlock()
	fyne.DoAndWait(r.modScroll.ScrollToTop)
	_ = r.modsBind.Set([]modmgr.Mod{})
	if err, _ := r.fetchMods(); err != nil {
		slog.Error("Failed to reload mods in repository tab", "error", err)
		r.state.SetError(fmt.Errorf("%s", lang.LocalizeKey("repository.failed_to_load", "Failed to load mods: {{.Error}}", map[string]any{"Error": err.Error()})))
	}
}
