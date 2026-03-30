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

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/client/core"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const (
	ModsPerPage          = 10
	repositoryThumbSize  = float32(114)
	repositoryDetailSize = float32(114)
)

type Repository struct {
	state *uicommon.State
	mu    sync.Mutex

	thumbMu             sync.Mutex
	thumbnailImageCache map[string]image.Image
	thumbnailFetched    map[string]bool
	thumbnailLoading    map[string]bool

	lastModID  string
	noMoreMods bool
	loading    bool
	modsBind   binding.List[*modmgr.Mod]

	// Containers
	mainContainer *fyne.Container // Stack container for switching views
	listView      *fyne.Container // The list view container
	detailView    *fyne.Container // The detail view container

	// List View Elements
	modListContainer *fyne.Container
	modScroll        *container.Scroll
	searchBar        *widget.Entry
	reloadBtn        *widget.Button
	stateLabel       *widget.Label
}

func NewRepository(state *uicommon.State) *Repository {
	bind := binding.NewList(func(a, b *modmgr.Mod) bool { return a.ID == b.ID })

	repo := &Repository{
		state:               state,
		lastModID:           "",
		modsBind:            bind,
		modListContainer:    container.NewVBox(),
		stateLabel:          widget.NewLabel(""),
		thumbnailImageCache: map[string]image.Image{},
		thumbnailFetched:    map[string]bool{},
		thumbnailLoading:    map[string]bool{},
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
		threshold := float32(4)
		bottomY := repo.modListContainer.Size().Height - repo.modScroll.Size().Height
		reachedBottom := bottomY > threshold && pos.Y >= bottomY-threshold
		if reachedBottom {
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
	r.LoadNext()
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
		if searchText != "" && !strings.Contains(strings.ToLower(mod.Name), searchText) {
			continue
		}

		thumb := r.newModThumbnailCanvas(mod.ID, repositoryThumbSize, 3)
		r.ensureThumbnailLoaded(mod.ID)
		thumbBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		thumbBg.CornerRadius = 6
		thumbArea := container.NewStack(thumbBg, container.NewCenter(thumb))

		titleLabel := widget.NewLabelWithStyle(mod.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		titleLabel.Wrapping = fyne.TextWrapOff
		titleLabel.Truncation = fyne.TextTruncateEllipsis

		updateBadge := widget.NewLabel("")
		updateBadge.Hide()

		activeProfileIDStr, _ := r.state.ActiveProfile.Get()
		if activeProfileIDStr != "" {
			if activeID, err := uuid.Parse(activeProfileIDStr); err == nil {
				if activeProfile, ok := r.state.ProfileManager.Get(activeID); ok {
					if installedVersion, ok := activeProfile.ModVersions[mod.ID]; ok {
						if installedVersion.ID != mod.LatestVersionID {
							updateBadge.SetText(lang.LocalizeKey("repository.update_available", "Update Available"))
							updateBadge.Importance = widget.WarningImportance
							updateBadge.Show()
						}
					}
				}
			}
		}

		authorLabel := widget.NewLabel(mod.Author)
		authorLabel.Wrapping = fyne.TextWrapOff
		authorLabel.Truncation = fyne.TextTruncateEllipsis
		descriptionLabel := widget.NewLabel(repositoryListSummary(mod.Description, 120))
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
	} else if len(mods) > 0 {
		endLabel := widget.NewLabel(lang.LocalizeKey("common.scroll_end_reached", "Reached the bottom."))
		endLabel.Alignment = fyne.TextAlignCenter
		endLabel.Importance = widget.LowImportance
		objs = append(objs, container.NewCenter(endLabel))
	}

	fyne.Do(func() {
		r.modListContainer.Objects = objs
		r.modListContainer.Refresh()
		r.modScroll.Refresh()
	})
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
	r.ensureThumbnailLoaded(mod.ID)
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

	// if mod.Website != "" {
	// 	if u, err := url.Parse(mod.Website); err == nil {
	// 		headerText.Add(widget.NewHyperlink(lang.LocalizeKey("repository.website", "Website"), u))
	// 	} else {
	// 		slog.Warn("Failed to parse mod website URL", "url", mod.Website, "error", err)
	// 	}
	// }

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
				versionsList.Add(widget.NewLabel("Failed to load versions: " + err.Error()))
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
	if r.noMoreMods || r.loading {
		r.mu.Unlock()
		return
	}
	r.loading = true
	r.mu.Unlock()

	go func() {
		defer func() {
			r.mu.Lock()
			r.loading = false
			r.mu.Unlock()
		}()

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
		r.updateModList(r.currentSearchText())
	}()
	if r.state.Rest != nil {
		mods, err := r.modsBind.Get()
		if err != nil {
			return err, false
		}
		var afterId string
		r.mu.Lock()
		if r.lastModID != "" && len(mods) > 0 {
			afterId = r.lastModID
		}
		r.mu.Unlock()

		slog.Info("Refreshing mods", "afterId", afterId)

		if modIDs, err := r.state.Rest.GetModIDs(ModsPerPage, afterId, ""); err != nil {
			return err, false
		} else if len(modIDs) > 0 {
			startIndex := len(mods)
			for _, modID := range modIDs {
				loadingMod := &modmgr.Mod{}
				loadingMod.ID = modID
				loadingMod.Name = lang.LocalizeKey("repository.loading_mod", "Loading mod '{{.ID}}'...", map[string]any{"ID": modID})
				loadingMod.Description = lang.LocalizeKey("repository.loading_mod_details", "Fetching mod details...")
				if err := r.modsBind.Append(loadingMod); err != nil {
					return err, false
				}
			}

			for i, modID := range modIDs {
				go r.loadModDetailsAsync(modID, startIndex+i)
			}
			r.mu.Lock()
			r.lastModID = modIDs[len(modIDs)-1]
			if len(modIDs) < ModsPerPage {
				r.noMoreMods = true
			}
			r.mu.Unlock()

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

func (r *Repository) currentSearchText() string {
	var text string
	fyne.DoAndWait(func() {
		text = r.searchBar.Text
	})
	return text
}

func (r *Repository) loadModDetailsAsync(modID string, listIndex int) {
	modData, err := r.state.Rest.GetMod(modID)
	if err != nil {
		slog.Warn("Failed to fetch mod details while refreshing mods", "modID", modID, "error", err)
		modData = &modmgr.Mod{}
		modData.ID = modID
		modData.Name = lang.LocalizeKey("repository.failed_to_load_mod", "Failed to load mod '{{.ID}}'", map[string]any{"ID": modID})
		modData.Description = lang.LocalizeKey("repository.failed_to_load_mod_description", "Could not fetch mod details: {{.Error}}", map[string]any{"Error": err.Error()})
	} else if modData == nil {
		modData = &modmgr.Mod{}
		modData.ID = modID
		modData.Name = lang.LocalizeKey("repository.mod_not_found", "Mod '{{.ID}}' not found", map[string]any{"ID": modID})
		modData.Description = lang.LocalizeKey("repository.mod_not_found_description", "The mod details are unavailable.")
	}

	currentValue, err := r.modsBind.GetValue(listIndex)
	if err != nil || currentValue == nil {
		return
	}
	if currentValue.ID != modID {
		return
	}
	if err := r.modsBind.SetValue(listIndex, modData); err != nil {
		slog.Warn("Failed to update mod details in list", "modID", modID, "index", listIndex, "error", err)
		return
	}

	r.updateModList(r.currentSearchText())
}

func (r *Repository) reloadMods() {
	slog.Info("Reloading repository mods")
	r.mu.Lock()
	r.lastModID = ""
	r.noMoreMods = false
	r.loading = false
	r.mu.Unlock()
	r.thumbMu.Lock()
	r.thumbnailImageCache = map[string]image.Image{}
	r.thumbnailFetched = map[string]bool{}
	r.thumbnailLoading = map[string]bool{}
	r.thumbMu.Unlock()
	fyne.DoAndWait(r.modScroll.ScrollToTop)
	_ = r.modsBind.Set([]*modmgr.Mod{})
	r.LoadNext()
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

func (r *Repository) ensureThumbnailLoaded(modID string) {
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

		r.updateModList(r.currentSearchText())
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
