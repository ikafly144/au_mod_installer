package repo

import (
	"image/color"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

const ModsPerPage = 3

type Repository struct {
	state *uicommon.State

	lastModID    string
	modsBind     binding.List[modmgr.Mod]
	modContainer *fyne.Container
}

func NewRepository(state *uicommon.State) *Repository {
	var repo Repository
	bind := binding.NewList(func(a, b modmgr.Mod) bool { return a.ID == b.ID })

	repo = Repository{
		state:        state,
		lastModID:    "",
		modsBind:     bind,
		modContainer: container.NewVBox(),
	}
	bind.AddListener(binding.NewDataListener(func() {
		var objs []fyne.CanvasObject
		mods, err := repo.modsBind.Get()
		if err != nil {
			slog.Error("Failed to get mods from binding", "error", err)
			return
		}
		for _, mod := range mods {
			if !mod.Type.IsVisible() {
				continue
			}

			img := canvas.NewSquare(color.Opaque)
			img.SetMinSize(fyne.NewSize(32, 32))
			item := container.NewHBox(
				img,
				widget.NewLabel(mod.Name+" ("+mod.Author+")"),
			)

			objs = append(objs, item, widget.NewSeparator())
		}
		objs = append(objs, widget.NewButton(lang.LocalizeKey("repository.load_next", "さらに読み込む…"), repo.LoadNext))
		repo.modContainer.Objects = objs
		repo.modContainer.Refresh()
	}))
	repo.init()
	return &repo
}

func (r *Repository) init() {
	if err, _ := r.refreshMods(); err != nil {
		slog.Error("Failed to refresh mods in repository tab", "error", err)
	}
}

func (r *Repository) Tab() (*container.TabItem, error) {
	content := container.NewScroll(
		r.modContainer,
	)
	return container.NewTabItem(lang.LocalizeKey("repository.tab_name", "リポジトリ"), content), nil
}

func (r *Repository) LoadNext() {
	mods, _ := r.modsBind.Get()
	slog.Info("Loading next mods in repository tab", "current_mods", mods)
	if err, ok := r.refreshMods(); err != nil {
		slog.Error("Failed to load next mods in repository tab", "error", err)
	} else {
		if !ok {
			slog.Info("No more mods to load in repository tab")
			return
		}
	}
}

func (r *Repository) refreshMods() (error, bool) {
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
				return r.refreshMods()
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
