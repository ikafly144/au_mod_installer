package repo

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"log/slog"
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

	"uuid"

	"github.com/ikafly144/modrepo/client/core"
	"github.com/ikafly144/modrepo/client/ui/uicommon"
	"github.com/ikafly144/modrepo/pkg/modmgr"
	"github.com/ikafly144/modrepo/pkg/profile"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const (
	repositoryThumbSize  = float32(114)
	repositoryDetailSize = float32(114)
	searchDebounceDelay  = 300 * time.Millisecond
)

// Sort key constants used by the sort dropdown and SearchMods().
const (
	SortByDownloads = "downloads"
	SortByUpdated   = "updated"
	SortByRating    = "rating"
	SortByName      = "name"
)

type Repository struct {
	state    *uicommon.State
	loadOnce sync.Once

	thumbMu             sync.Mutex
	thumbnailImageCache map[string]image.Image
	thumbnailFetched    map[string]bool
	thumbnailLoading    map[string]bool

	// Filtered mod list (result of SearchMods)
	dataMu   sync.RWMutex
	filteredMods []*modmgr.Mod

	// Search / filter state
	searchQuery string
	sortBy      string
	category    string

	// Debounce
	debounceTimer *time.Timer
	debounceMu    sync.Mutex

	// Containers
	mainContainer *fyne.Container // Stack container for switching views
	listView      *fyne.Container // The list view container
	detailView    *fyne.Container // The detail view container

	// List View Elements
	modList       *widget.List
	searchBar     *widget.Entry
	sortSelect    *widget.Select
	categorySelect *widget.Select
	reloadBtn     *widget.Button
	stateLabel    *widget.Label
}

func NewRepository(state *uicommon.State) *Repository {
	repo := &Repository{
		state:               state,
		thumbnailImageCache: map[string]image.Image{},
		thumbnailFetched:    map[string]bool{},
		thumbnailLoading:    map[string]bool{},
		sortBy:              SortByDownloads,
	}

	// --- Search bar ---
	repo.searchBar = widget.NewEntry()
	repo.searchBar.SetPlaceHolder(lang.LocalizeKey("repository.search_placeholder", "Filter mods by name"))
	repo.searchBar.OnChanged = func(s string) {
		repo.dataMu.Lock()
		repo.searchQuery = s
		repo.dataMu.Unlock()
		repo.scheduleRefresh()
	}

	// --- Sort dropdown ---
	sortOptions := []string{
		lang.LocalizeKey("repository.sort.downloads", "Popular"),
		lang.LocalizeKey("repository.sort.updated", "Recently Updated"),
		lang.LocalizeKey("repository.sort.rating", "Top Rated"),
		lang.LocalizeKey("repository.sort.name", "Name"),
	}
	sortKeys := []string{SortByDownloads, SortByUpdated, SortByRating, SortByName}

	repo.sortSelect = widget.NewSelect(sortOptions, func(selected string) {
		for i, opt := range sortOptions {
			if opt == selected {
				repo.dataMu.Lock()
				repo.sortBy = sortKeys[i]
				repo.dataMu.Unlock()
				repo.scheduleRefresh()
				return
			}
		}
	})
	repo.sortSelect.SetSelectedIndex(0)

	// --- Category dropdown ---
	allCategoriesLabel := lang.LocalizeKey("repository.category.all", "All Categories")
	repo.categorySelect = widget.NewSelect([]string{allCategoriesLabel}, func(selected string) {
		repo.dataMu.Lock()
		if selected == allCategoriesLabel {
			repo.category = ""
		} else {
			repo.category = selected
		}
		repo.dataMu.Unlock()
		repo.scheduleRefresh()
	})
	repo.categorySelect.SetSelectedIndex(0)

	// --- Reload button ---
	repo.reloadBtn = widget.NewButtonWithIcon(lang.LocalizeKey("repository.reload", "Reload"), theme.ViewRefreshIcon(), func() {
		repo.reloadBtn.Disable()
		go repo.reloadMods()
	})

	// --- State label ---
	repo.stateLabel = widget.NewLabel("")
	repo.stateLabel.Hide()
	repo.stateLabel.Wrapping = fyne.TextWrapWord

	// --- Virtualized list ---
	repo.modList = widget.NewList(
		func() int {
			repo.dataMu.RLock()
			defer repo.dataMu.RUnlock()
			return len(repo.filteredMods)
		},
		repo.createListItem,
		repo.updateListItem,
	)

	// Build List View
	toolbar := container.NewVBox(
		container.New(layout.NewBorderLayout(nil, nil, nil, repo.reloadBtn),
			repo.searchBar,
			repo.reloadBtn,
		),
		container.NewGridWithColumns(2, repo.sortSelect, repo.categorySelect),
	)
	bottom := container.NewVBox(
		repo.state.ErrorText,
		repo.stateLabel,
	)
	repo.listView = container.New(layout.NewBorderLayout(toolbar, bottom, nil, nil),
		toolbar,
		bottom,
		repo.modList,
	)

	// Initialize Detail View (empty for now)
	repo.detailView = container.NewStack()

	repo.mainContainer = container.NewStack(repo.listView, repo.detailView)
	repo.detailView.Hide()

	state.ActiveProfile.AddListener(binding.NewDataListener(func() {
		repo.refreshList()
	}))

	return repo
}

func (r *Repository) EnsureLoaded() {
	r.loadOnce.Do(func() {
		go r.refreshFilteredMods()
	})
}

func (r *Repository) Tab() (*container.TabItem, error) {
	r.EnsureLoaded()
	return container.NewTabItem(lang.LocalizeKey("repository.tab_name", "Repository"), r.mainContainer), nil
}

// scheduleRefresh debounces search/filter changes.
func (r *Repository) scheduleRefresh() {
	r.debounceMu.Lock()
	defer r.debounceMu.Unlock()
	if r.debounceTimer != nil {
		r.debounceTimer.Stop()
	}
	r.debounceTimer = time.AfterFunc(searchDebounceDelay, func() {
		r.refreshFilteredMods()
	})
}

// refreshFilteredMods queries SearchMods with current filters and updates the list.
func (r *Repository) refreshFilteredMods() {
	r.dataMu.RLock()
	query := r.searchQuery
	cat := r.category
	sort := r.sortBy
	r.dataMu.RUnlock()

	mods, err := r.state.Rest.SearchMods(query, cat, sort)
	if err != nil {
		slog.Error("Failed to search mods", "error", err)
		r.state.SetError(fmt.Errorf("%s", lang.LocalizeKey("repository.failed_to_load", "Failed to load mods: {{.Error}}", map[string]any{"Error": err.Error()})))
		return
	}

	r.dataMu.Lock()
	r.filteredMods = mods
	r.dataMu.Unlock()

	r.refreshList()
}

// refreshList tells the virtualized list to re-render visible items.
func (r *Repository) refreshList() {
	fyne.Do(func() {
		r.modList.Refresh()
	})
}

// createListItem creates a template list item widget (called once per visible slot).
func (r *Repository) createListItem() fyne.CanvasObject {
	thumb := canvas.NewImageFromImage(placeholderModThumbnail(int(repositoryThumbSize)))
	thumb.FillMode = canvas.ImageFillContain
	thumb.CornerRadius = 3
	thumb.SetMinSize(fyne.NewSquareSize(repositoryThumbSize))
	thumbBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	thumbBg.CornerRadius = 6
	thumbArea := container.NewStack(thumbBg, container.NewCenter(thumb))

	titleLabel := widget.NewLabelWithStyle("Mod Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLabel.Wrapping = fyne.TextWrapOff
	titleLabel.Truncation = fyne.TextTruncateEllipsis

	updateBadge := widget.NewLabel("")
	updateBadge.Hide()

	authorLabel := widget.NewLabel("Author")
	authorLabel.Wrapping = fyne.TextWrapOff
	authorLabel.Truncation = fyne.TextTruncateEllipsis
	descriptionLabel := widget.NewLabel("Description")
	descriptionLabel.Wrapping = fyne.TextWrapOff
	descriptionLabel.Truncation = fyne.TextTruncateEllipsis

	titleRow := container.NewBorder(nil, nil, nil, updateBadge, titleLabel)
	textContainer := container.NewVBox(
		titleRow,
		authorLabel,
		descriptionLabel,
	)

	content := container.New(&modListItemLayout{
		minThumbSize: repositoryThumbSize,
		spacing:      theme.Padding(),
	}, thumbArea, container.NewPadded(textContainer))

	tappable := uicommon.NewTappableContainer(content, nil)

	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameButton)
	bg.StrokeWidth = 1
	bg.CornerRadius = theme.InputRadiusSize()

	return container.NewStack(bg, container.NewPadded(tappable))
}

// updateListItem binds data to a list item at the given index.
func (r *Repository) updateListItem(id widget.ListItemID, item fyne.CanvasObject) {
	r.dataMu.RLock()
	if id >= len(r.filteredMods) {
		r.dataMu.RUnlock()
		return
	}
	mod := r.filteredMods[id]
	r.dataMu.RUnlock()

	// Navigate the widget tree:
	// item = Stack[bg, Padded[tappable]]
	stackObjs := item.(*fyne.Container).Objects
	padded := stackObjs[1].(*fyne.Container)        // Padded
	tappable := padded.Objects[0].(*uicommon.TappableContainer) // TappableContainer
	contentLayout := tappable.Content.(*fyne.Container) // modListItemLayout container

	thumbArea := contentLayout.Objects[0].(*fyne.Container) // Stack[thumbBg, Center[thumb]]
	centerContainer := thumbArea.Objects[1].(*fyne.Container)
	thumb := centerContainer.Objects[0].(*canvas.Image)

	paddedText := contentLayout.Objects[1].(*fyne.Container) // Padded[textContainer]
	textContainer := paddedText.Objects[0].(*fyne.Container) // VBox[titleRow, author, desc]

	titleRow := textContainer.Objects[0].(*fyne.Container) // Border[titleLabel, updateBadge]
	titleLabel := titleRow.Objects[0].(*widget.Label)
	updateBadge := titleRow.Objects[1].(*widget.Label)

	authorLabel := textContainer.Objects[1].(*widget.Label)
	descriptionLabel := textContainer.Objects[2].(*widget.Label)

	// Update content
	titleLabel.SetText(mod.Name)
	authorLabel.SetText(mod.Author)
	descriptionLabel.SetText(repositoryListSummary(mod.Description, 120))

	// Update thumbnail
	thumb.Image = r.modThumbnailImage(mod.ID, int(repositoryThumbSize))
	thumb.Refresh()
	r.ensureThumbnailLoaded(mod.ID, id)

	// Update badge
	updateBadge.Hide()
	activeProfileIDStr, _ := r.state.ActiveProfile.Get()
	if activeProfileIDStr != "" {
		if activeID, err := uuid.Parse(activeProfileIDStr); err == nil {
			if activeProfile, ok := r.state.ProfileManager.Get(activeID); ok {
				if installedVersion, ok := activeProfile.ModVersions[mod.ID]; ok {
					if installedVersion.VersionID != mod.LatestVersionID {
						updateBadge.SetText(lang.LocalizeKey("repository.update_available", "Update Available"))
						updateBadge.Importance = widget.WarningImportance
						updateBadge.Show()
					}
				}
			}
		}
	}

	// Update tap handler
	modCopy := mod
	tappable.OnTapped = func() {
		r.showModDetails(modCopy)
	}
}

func (r *Repository) showModDetails(mod *modmgr.Mod) {
	// Navigation Bar (Topmost)
	backBtn := widget.NewButtonWithIcon(lang.LocalizeKey("common.back", "Back"), theme.NavigateBackIcon(), func() {
		r.detailView.Hide()
		r.listView.Show()
	})
	topBar := container.NewHBox(backBtn)

	// Header Info
	img := r.newModThumbnailCanvas(mod.ID, repositoryDetailSize, 10)
	r.ensureThumbnailLoaded(mod.ID, -1)
	imgBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	imgBg.CornerRadius = 10
	imgArea := container.NewStack(imgBg, container.NewCenter(img))

	titleLabel := widget.NewLabelWithStyle(mod.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLabel.Wrapping = fyne.TextWrapOff
	titleLabel.Truncation = fyne.TextTruncateEllipsis
	authorLabel := widget.NewLabel(lang.LocalizeKey("repository.author", "Author: {{.Author}}", map[string]any{"Author": mod.Author}))
	authorLabel.Wrapping = fyne.TextWrapOff
	authorLabel.Truncation = fyne.TextTruncateEllipsis
	headerText := container.NewVBox(titleLabel, authorLabel)

	headerText.Add(widget.NewButton(lang.LocalizeKey("repository.install_latest", "Install Latest"), func() {
		r.installModVersion(mod, mod.LatestVersionID)
	}))

	header := container.New(layout.NewBorderLayout(nil, nil, imgArea, nil),
		imgArea,
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
		versions, err := r.state.Rest.GetModVersionIDs(mod.ID, 100, "")
		fyne.Do(func() {
			versionsList.Objects = nil
			if err != nil {
				versionsList.Add(widget.NewLabel(lang.LocalizeKey("repository.error.failed_to_load_versions", "Failed to load versions: {{.Error}}", map[string]any{"Error": err.Error()})))
				return
			}
			for _, v := range versions {
				verLabel := widget.NewLabel(v)
				verLabel.Wrapping = fyne.TextWrapOff
				verLabel.Truncation = fyne.TextTruncateEllipsis
				addBtn := widget.NewButton(lang.LocalizeKey("repository.add_to_profile", "Add to Profile"), func() {
					r.installModVersion(mod, v)
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

func (r *Repository) installModVersion(mod *modmgr.Mod, versionID string) {
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
						r.state.SetError(errors.New(lang.LocalizeKey("repository.error.profile_not_found", "Profile not found.")))
					})
					return
				}

				versionData, err := r.state.Rest.GetModVersion(mod.ID, versionID)
				if err != nil {
					slog.Error("Failed to get mod version for installation", "modId", mod.ID, "versionId", versionID, "error", err)
					r.state.SetError(err)
					return
				}

				profileLock, err := r.state.Core.AcquireProfileLaunchLock(targetProfile.ID)
				if err != nil {
					if errors.Is(err, core.ErrProfileLaunchBusy) {
						r.state.SetError(errors.New(lang.LocalizeKey("error.game_already_running", "Already running.")))
						return
					}
					slog.Error("Failed to acquire profile lock before adding mod", "profile", targetProfile.Name, "error", err)
					r.state.SetError(err)
					return
				}
				defer func() {
					if err := profileLock.Release(); err != nil {
						slog.Error("Failed to release profile lock after adding mod", "profile", targetProfile.Name, "error", err)
						r.state.SetError(err)
					}
				}()

				targetProfile.AddModVersion(*versionData)
				targetProfile.UpdatedAt = time.Now()

				if err := r.state.ProfileManager.Add(targetProfile); err != nil {
					slog.Error("Failed to add mod to profile", "error", err)
					r.state.SetError(err)
				} else {
					slog.Info("Mod added to profile", "modId", mod.ID, "versionId", versionID, "profile", targetProfile.Name)
					r.state.ClearError()
					fyne.Do(func() {
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

// reloadMods refreshes packages from Thunderstore and rebuilds the mod list.
func (r *Repository) reloadMods() {
	defer fyne.Do(r.reloadBtn.Enable)
	slog.Info("Reloading repository mods")

	r.thumbMu.Lock()
	r.thumbnailImageCache = map[string]image.Image{}
	r.thumbnailFetched = map[string]bool{}
	r.thumbnailLoading = map[string]bool{}
	r.thumbMu.Unlock()

	r.refreshFilteredMods()

	// Refresh category list after reload
	r.refreshCategoryList()
}

// refreshCategoryList updates the category dropdown options from the loaded package data.
func (r *Repository) refreshCategoryList() {
	cats := r.state.Rest.GetCategories()
	allLabel := lang.LocalizeKey("repository.category.all", "All Categories")
	options := make([]string, 0, len(cats)+1)
	options = append(options, allLabel)
	options = append(options, cats...)
	fyne.Do(func() {
		r.categorySelect.Options = options
		r.categorySelect.Refresh()
	})
}

func repositoryListSummary(text string, maxRunes int) string {
	line := strings.Join(strings.Fields(strings.ReplaceAll(text, "\n", " ")), " ")
	if maxRunes <= 0 {
		return line
	}
	runes := []rune(line)
	if len(runes) <= maxRunes {
		return line
	}
	return string(runes[:maxRunes-1]) + "…"
}

func placeholderModThumbnail(size int) image.Image {
	return image.NewPaletted(image.Rect(0, 0, max(size, 1), max(size, 1)), color.Palette{theme.Color(theme.ColorNameDisabled)})
}

func centerCropSquareThumbnail(src image.Image) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return placeholderModThumbnail(1)
	}
	side := min(width, height)
	startX := bounds.Min.X + (width-side)/2
	startY := bounds.Min.Y + (height-side)/2
	dstRect := image.Rect(0, 0, side, side)
	dst := image.NewRGBA(dstRect)
	imagedraw.Draw(dst, dstRect, src, image.Point{X: startX, Y: startY}, imagedraw.Src)
	return dst
}

func (r *Repository) modThumbnailImage(modID string, fallbackSize int) image.Image {
	r.thumbMu.Lock()
	img := r.thumbnailImageCache[modID]
	r.thumbMu.Unlock()
	if img == nil {
		return placeholderModThumbnail(fallbackSize)
	}
	return img
}

func (r *Repository) newModThumbnailCanvas(modID string, size float32, cornerRadius float32) *canvas.Image {
	img := canvas.NewImageFromImage(r.modThumbnailImage(modID, int(size)))
	img.FillMode = canvas.ImageFillContain
	img.CornerRadius = cornerRadius
	img.SetMinSize(fyne.NewSquareSize(size))
	return img
}

// ensureThumbnailLoaded starts an async thumbnail download if not already cached.
// listIndex is the widget.List item ID to refresh when the thumbnail is ready (-1 to skip).
func (r *Repository) ensureThumbnailLoaded(modID string, listIndex int) {
	if modID == "" || r.state.Rest == nil {
		return
	}
	r.thumbMu.Lock()
	if r.thumbnailFetched[modID] || r.thumbnailLoading[modID] {
		r.thumbMu.Unlock()
		return
	}
	r.thumbnailLoading[modID] = true
	r.thumbMu.Unlock()

	go func(targetModID string) {
		thumbBytes, err := r.state.Rest.GetModThumbnail(targetModID)
		var decoded image.Image
		if err == nil && len(thumbBytes) > 0 {
			decoded, _, err = image.Decode(bytes.NewReader(thumbBytes))
			if err == nil {
				decoded = centerCropSquareThumbnail(decoded)
			}
		}
		if err != nil {
			slog.Debug("Failed to load mod thumbnail", "modID", targetModID, "error", err)
		}

		r.thumbMu.Lock()
		delete(r.thumbnailLoading, targetModID)
		r.thumbnailFetched[targetModID] = true
		if decoded != nil {
			r.thumbnailImageCache[targetModID] = decoded
		}
		r.thumbMu.Unlock()

		// Only refresh the specific list item instead of rebuilding the entire list
		if listIndex >= 0 {
			fyne.Do(func() {
				r.modList.RefreshItem(listIndex)
			})
		}
	}(modID)
}

type modListItemLayout struct {
	minThumbSize float32
	spacing      float32
}

func (l *modListItemLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	thumb := objects[0]
	body := objects[1]

	thumbSide := size.Height
	if thumbSide < l.minThumbSize {
		thumbSide = l.minThumbSize
	}
	if thumbSide > size.Width {
		thumbSide = size.Width
	}
	thumb.Resize(fyne.NewSize(thumbSide, thumbSide))
	thumb.Move(fyne.NewPos(0, (size.Height-thumbSide)/2))

	bodyX := thumbSide + l.spacing
	bodyWidth := size.Width - bodyX
	if bodyWidth < 0 {
		bodyWidth = 0
	}
	body.Resize(fyne.NewSize(bodyWidth, size.Height))
	body.Move(fyne.NewPos(bodyX, 0))
}

func (l *modListItemLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}
	thumbMin := objects[0].MinSize()
	bodyMin := objects[1].MinSize()
	height := max(thumbMin.Height, max(bodyMin.Height, l.minThumbSize))
	return fyne.NewSize(height+l.spacing, height)
}
