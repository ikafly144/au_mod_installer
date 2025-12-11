package main

import (
	"flag"
	"log/slog"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/client/ui"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/sqweek/dialog"
)

func main() {
	var (
		localMode string
		server    string
	)

	flag.StringVar(&localMode, "local", "", "Path to local mods.json file for local mode")
	flag.StringVar(&server, "server", "http://localhost:8080", "URL of the mod server")
	flag.Parse()

	a := app.New()
	w := a.NewWindow(lang.LocalizeKey("app.name", "Among Us Mod ランチャー"))

	var client rest.Client
	if localMode != "" {
		slog.Info("Running in local mode", "path", localMode)
		f, err := rest.NewFileClient(localMode)
		if err != nil {
			slog.Error("Failed to create local file client", "error", err)
			dialog.Message("ローカルファイルクライアントの作成に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
			os.Exit(1)
		}
		if err := f.LoadData(); err != nil {
			slog.Error("Failed to load data from local file", "error", err)
			dialog.Message("ローカルファイルからのデータの読み込みに失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
			os.Exit(1)
		}
		client = f
	} else {
		slog.Info("Running in server mode", "server", server)
		client = rest.NewClient(server)
	}
	if err := ui.Main(w, a.Metadata().Version,
		ui.WithStateOptions(
			uicommon.WithRestClient(client),
		),
	); err != nil {
		slog.Error("Failed to initialize UI", "error", err)
		dialog.Message("UIの初期化に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
		os.Exit(1)
	}
}
