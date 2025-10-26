package main

import (
	"au_mod_installer/pkg/modmgr"
	"au_mod_installer/ui"
	"au_mod_installer/ui/common"
	"encoding/json"
	"flag"
	"log/slog"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"github.com/sqweek/dialog"
)

var (
	localMode = flag.String("local", "", "Run the app in local mode for testing")
)

func init() {
	flag.Parse()

	if *localMode != "" {
		common.ModProvider = func() ([]modmgr.Mod, error) {
			var mods []modmgr.Mod
			file, err := os.Open(*localMode)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			if err := json.NewDecoder(file).Decode(&mods); err != nil {
				return nil, err
			}
			return mods, nil
		}
	}
}

func main() {
	a := app.New()
	w := a.NewWindow(lang.LocalizeKey("app.name", "Among Us Mod ランチャー"))
	if err := ui.Main(w, version); err != nil {
		slog.Error("Failed to initialize UI", "error", err)
		dialog.Message("UIの初期化に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
		os.Exit(1)
	}
}
