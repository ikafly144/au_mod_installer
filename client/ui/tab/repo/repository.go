package repo

import (
	"log/slog"
	"slices"

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

	lastModID    string
	modsBind     binding.List[modmgr.Mod]
	modContainer *fyne.Container
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
				fyne.DoAndWait(func() {
					repo.reloadBtn.Enable()
				})
			}()
		}),
		modsBind:     bind,
		modContainer: container.NewVBox(),
		progressBar:  progress.NewFyneProgress(widget.NewProgressBar()),
		stateLabel:   widget.NewLabel(""),
	}
	repo.stateLabel.Hide()
	repo.stateLabel.Wrapping = fyne.TextWrapWord

	repo.searchBar.SetPlaceHolder(lang.LocalizeKey("repository.search_placeholder", "Modを名前で絞り込む（未実装）"))
	repo.searchBar.Disable()

	bind.AddListener(binding.NewDataListener(func() {
		var objs []fyne.CanvasObject
		mods, err := repo.modsBind.Get()
		if err != nil {
			slog.Error("Failed to get mods from binding", "error", err)
			return
		}
		repo.installBtns = nil
		repo.versionSelects = nil
		for _, mod := range mods {
			if !mod.Type.IsVisible() {
				continue
			}

			versionSelect := newVersionSelectMenu(nil)
			go func() {
				versionSelect.SupplyMods(func() ([]modmgr.ModVersion, error) {
					versions, err := repo.state.Rest.GetModVersions(mod.ID, 10, "")
					if err != nil {
						slog.Error("Failed to get mod versions", "modId", mod.ID, "error", err)
						return nil, err
					}
					return versions, nil
				})
				versionSelect.SetSelected(mod.LatestVersion)
			}()
			repo.versionSelects = append(repo.versionSelects, versionSelect)

			img := canvas.NewSquare(theme.Color(theme.ColorNameDisabled))
			img.SetMinSize(fyne.NewSize(64, 64))
			installBtn := widget.NewButton("インストール", func() {
				repo.stateLabel.Hide()
				version := mod.LatestVersion
				if v, err := versionSelect.GetSelected(); err == nil && v != "" {
					version = v
				}

				repo.state.ClearError()
				_ = repo.state.CanInstall.Set(false)
				_ = repo.state.CanLaunch.Set(false)
				go func() {
					versionData, err := repo.state.Rest.GetModVersion(mod.ID, version)
					if err != nil {
						slog.Error("Failed to get mod version for installation", "modId", mod.ID, "versionId", version, "error", err)
						repo.state.SetError(err)
						return
					}
					if err := repo.state.InstallMods(mod.ID, *versionData, repo.progressBar); err != nil {
						slog.Error("Mod installation failed", "error", err)
						repo.state.SetError(err)
					} else {
						slog.Info("Mod installation succeeded", "modId", mod.ID, "versionId", version)
						repo.state.ClearError()
						_ = repo.state.CanLaunch.Set(true)
						fyne.DoAndWait(func() {
							repo.stateLabel.SetText("インストールが完了しました: " + mod.Name + " (" + version + ")")
							repo.stateLabel.Show()
						})
					}
					_ = repo.state.CanInstall.Set(true)
					repo.state.CheckInstalled()
				}()
			})

			repo.installBtns = append(repo.installBtns, installBtn)

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
		objs = append(objs, widget.NewButton(lang.LocalizeKey("repository.load_next", "さらに読み込む…"), repo.LoadNext))
		fyne.Do(func() {
			repo.modContainer.Objects = objs
			repo.modContainer.Refresh()
		})
	}))
	repo.init()
	return &repo
}

func (r *Repository) init() {
	if err, _ := r.fetchMods(); err != nil {
		slog.Error("Failed to refresh mods in repository tab", "error", err)
	}

	r.state.CanInstall.AddListener(binding.NewDataListener(func() {
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
			selectWidget.Refresh()
		}
	}))
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
		container.NewScroll(
			r.modContainer,
		),
		top,
		bottom,
	)
	return container.NewTabItem(lang.LocalizeKey("repository.tab_name", "リポジトリ"), content), nil
}

func (r *Repository) LoadNext() {
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
}

func (r *Repository) fetchMods() (error, bool) {
	if r.state.Rest != nil {
		mods, err := r.modsBind.Get()
		if err != nil {
			return err, false
		}
		var afterId, lastId string
		if r.lastModID != "" && len(mods) > 0 {
			afterId = r.lastModID
		}

		slog.Info("Refreshing mods", "firstId", afterId, "lastId", lastId)

		if mods, err := r.state.Rest.GetModList(ModsPerPage, afterId, lastId); err != nil {
			return err, false
		} else if len(mods) > 0 {
			for _, m := range mods {
				if err := r.modsBind.Append(m); err != nil {
					return err, false
				}
			}
			r.lastModID = mods[len(mods)-1].ID

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
	r.lastModID = ""
	_ = r.modsBind.Set([]modmgr.Mod{})
	if err, _ := r.fetchMods(); err != nil {
		slog.Error("Failed to reload mods in repository tab", "error", err)
	}
}
