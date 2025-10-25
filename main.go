package main

import (
	"au_mod_installer/ui"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"github.com/sqweek/dialog"
)

func main() {
	a := app.New()
	w := a.NewWindow(lang.LocalizeKey("app.name", "Among Us Mod ローダー"))
	if err := ui.Main(w); err != nil {
		dialog.Message("UIの初期化に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
		os.Exit(1)
	}
}
