package repo

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

const ModsPerPage = 10

type Repository struct {
	state *uicommon.State
	mu    sync.Mutex

	lastModID    string
	modsBind     binding.List[modmgr.Mod]
	modContainer *fyne.Container
	modScroll    *container.Scroll
	progressBar  *progress.FyneProgress
	searchBar    *widget.Entry
	reloadBtn    *widget.Button
	stateLabel   *widget.Label

	versionSelects []*versionSelectMenu
	installBtns    []*widget.Button
}

func NewRepository(state *uicommon.State) *Repository {
	var repo Repository
	bind := binding.NewList(func(a, b modmgr.Mod) bool { return a.ID == b.ID })

	repo = Repository{
		state:     state,
		lastModID: "",
		searchBar: widget.NewEntry(),
		reloadBtn: widget.NewButtonWithIcon(lang.LocalizeKey("repository.reload", "リロード"), theme.ViewRefreshIcon(), func() {
			repo.reloadBtn.Disable()
			go func() {
				repo.reloadMods()
			}()
		}),
		modsBind:     bind,
		modContainer: container.NewVBox(),
		progressBar:  progress.NewFyneProgress(widget.NewProgressBar()),
		stateLabel:   widget.NewLabel(""),
	}
	repo.modScroll = container.NewVScroll(repo.modContainer)

	repo.stateLabel.Hide()
	repo.stateLabel.Wrapping = fyne.TextWrapWord

	repo.searchBar.SetPlaceHolder(lang.LocalizeKey("repository.search_placeholder", "Modを名前で絞り込む"))
	repo.searchBar.OnChanged = func(s string) {
		go repo.updateModList(s)
	}

	repo.init()
	return &repo
}

func (r *Repository) init() {
	go func() {
		if err, _ := r.fetchMods(); err != nil {
			slog.Error("Failed to refresh mods in repository tab", "error", err)
		}
	}()

	r.state.CanInstall.AddListener(binding.NewDataListener(func() {
		r.updateInstallState(true)
	}))
}

func (r *Repository) updateModList(filter string) {
	defer r.updateInstallState(true)
	defer fyne.Do(r.reloadBtn.Enable)
	var objs []fyne.CanvasObject
	mods, err := r.modsBind.Get()
	if err != nil {
		slog.Error("Failed to get mods from binding", "error", err)
		return
	}
	if len(mods) == 0 {
		return
	}
	var newInstallBtns []*widget.Button
	var newVersionSelects []*versionSelectMenu
	wg := sync.WaitGroup{}
	searchText := strings.ToLower(filter)
	for _, mod := range mods {
		if !mod.Type.IsVisible() {
			continue
		}
		if searchText != "" && !strings.Contains(strings.ToLower(mod.Name), searchText) {
			continue
		}

		versionSelect := newVersionSelectMenu(nil)
		wg.Add(1)
		go func(mod modmgr.Mod) {
			defer wg.Done()
			versionSelect.SupplyMods(func() ([]modmgr.ModVersion, error) {
				versions, err := r.state.Rest.GetModVersions(mod.ID, 10, "")
				if err != nil {
					slog.Error("Failed to get mod versions", "modId", mod.ID, "error", err)
					return nil, err
				}
				return versions, nil
			})
			versionSelect.SetSelected(mod.LatestVersion)
		}(mod)
		newVersionSelects = append(newVersionSelects, versionSelect)

		img := canvas.NewSquare(theme.Color(theme.ColorNameDisabled))
		img.SetMinSize(fyne.NewSize(64, 64))
		installBtn := widget.NewButton("インストール", func() {
			r.stateLabel.Hide()
			version := mod.LatestVersion
			if v, err := versionSelect.GetSelected(); err == nil && v != "" {
				version = v
			}

			if version == "" {
				slog.Error("No version selected for installation", "modId", mod.ID)
				r.state.SetError(fmt.Errorf("インストールするバージョンが選択されていません: %s", mod.Name))
				return
			}

			r.state.ClearError()
			_ = r.state.CanInstall.Set(false)
			_ = r.state.CanLaunch.Set(false)
			go func() {
				versionData, err := r.state.Rest.GetModVersion(mod.ID, version)
				if err != nil {
					slog.Error("Failed to get mod version for installation", "modId", mod.ID, "versionId", version, "error", err)
					r.state.SetError(err)
					return
				}
				if err := r.state.InstallMods(mod.ID, *versionData, r.progressBar); err != nil {
					slog.Error("Mod installation failed", "error", err)
					r.state.SetError(err)
				} else {
					slog.Info("Mod installation succeeded", "modId", mod.ID, "versionId", version)
					r.state.ClearError()
					_ = r.state.CanLaunch.Set(true)
					fyne.DoAndWait(func() {
						r.stateLabel.SetText("インストールが完了しました: " + mod.Name + " (" + version + ")")
						r.stateLabel.Show()
					})
				}
				_ = r.state.CanInstall.Set(true)
				r.state.CheckInstalled()
			}()
		})

		newInstallBtns = append(newInstallBtns, installBtn)

		bottom := container.NewHBox(
			versionSelect.Canvas(), installBtn,
		)
		item := container.New(layout.NewBorderLayout(nil, nil, img, nil),
			container.New(layout.NewBorderLayout(nil, bottom, nil, nil),
				container.NewHBox(
					widget.NewLabel(mod.Name+" ("+mod.Author+")"),
				),
				bottom,
			),
			img,
		)

		objs = append(objs, item, widget.NewSeparator())
	}
	objs = append(objs, widget.NewButton(lang.LocalizeKey("repository.load_next", "さらに読み込む…"), r.LoadNext))

	wg.Wait()
	r.mu.Lock()
	r.installBtns = newInstallBtns
	r.versionSelects = newVersionSelects
	r.mu.Unlock()

	fyne.Do(func() {
		r.modContainer.Objects = objs
		r.modContainer.Refresh()
	})
}

func (r *Repository) updateInstallState(update bool) {
	fyne.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		ok, err := r.state.CanInstall.Get()
		if err != nil {
			slog.Warn("Failed to get install state", "error", err)
			return
		}
		for _, btn := range r.installBtns {
			if ok {
				btn.Enable()
			} else {
				btn.Disable()
			}
			btn.Refresh()
		}
		for _, selectWidget := range r.versionSelects {
			if ok {
				selectWidget.Enable()
			} else {
				selectWidget.Disable()
			}
			if update {
				selectWidget.Refresh()
			}
		}
	})
}

func (r *Repository) Tab() (*container.TabItem, error) {
	top := container.New(layout.NewBorderLayout(nil, nil, nil, r.reloadBtn),
		r.searchBar,
		r.reloadBtn,
	)
	bottom := container.NewVBox(
		r.state.ErrorText,
		r.stateLabel,
		r.progressBar.Canvas(),
	)
	content := container.New(layout.NewBorderLayout(top, bottom, nil, nil),
		r.modScroll,
		top,
		bottom,
	)
	return container.NewTabItem(lang.LocalizeKey("repository.tab_name", "リポジトリ"), content), nil
}

func (r *Repository) LoadNext() {
	go func() {
		mods, _ := r.modsBind.Get()
		slog.Info("Loading next mods in repository tab", "current_mods", mods)
		if err, ok := r.fetchMods(); err != nil {
			slog.Error("Failed to load next mods in repository tab", "error", err)
		} else {
			if !ok {
				slog.Info("No more mods to load in repository tab")
				return
			}
		}
	}()
}

func (r *Repository) fetchMods() (error, bool) {
	defer r.updateInstallState(true)
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
			r.mu.Unlock()

			if !slices.ContainsFunc(mods, func(m modmgr.Mod) bool {
				return m.Type.IsVisible()
			}) {
				slog.Info("No visible mods in loaded mods, loading next page")
				return r.fetchMods()
			}

			return nil, true
		} else {
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
	r.mu.Unlock()
	fyne.DoAndWait(r.modScroll.ScrollToTop)
	_ = r.modsBind.Set([]modmgr.Mod{})
	if err, _ := r.fetchMods(); err != nil {
		slog.Error("Failed to reload mods in repository tab", "error", err)
	}
}
