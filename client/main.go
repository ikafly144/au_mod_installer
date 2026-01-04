//go:build windows

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/client/ui"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/common/versioning"
	"github.com/nightlyone/lockfile"
	"github.com/sqweek/dialog"
)

var DefaultServer = "https://modofus.sabafly.net/v1"

func main() {
	lock, err := lockfile.New(filepath.Join(os.Getenv("PROGRAMDATA"), "au_mod_installer.lock"))
	if err != nil {
		slog.Error("Failed to create lockfile", "error", err)
		os.Exit(1)
	}
	err = lock.TryLock()
	if err != nil {
		slog.Error("Another instance is already running", "error", err)
		//ignore:printf
		(&dialog.MsgBuilder{Msg: lang.LocalizeKey("app.already_running", "Another instance of Among Us Mod Installer is already running.")}).Title(lang.LocalizeKey("app.error", "Error")).Error()
		_ = lock.Unlock()
		os.Exit(1)
	}
	defer func() {
		if p := recover(); p != nil {
			slog.Error("Application panicked", "panic", p)
			_ = lock.Unlock()
			os.Exit(1)
		}
	}()
	mainErr := realMain()
	if err := lock.Unlock(); err != nil {
		slog.Error("Failed to unlock lockfile", "error", err)
	}
	if mainErr != nil {
		os.Exit(1)
	}
}

func realMain() error {
	var (
		localMode string
		server    string
	)

	flag.StringVar(&localMode, "local", "", "Path to local mods.json file for local mode")
	flag.StringVar(&server, "server", DefaultServer, "URL of the mod server")
	flag.Parse()

	a := app.New()

	branch := versioning.BranchFromString(a.Preferences().StringWithFallback("core.update_branch", "stable"))

	tag, err := versioning.CheckForUpdates(context.Background(), branch, version)
	if err != nil {
		slog.Error("Failed to check for updates", "error", err)
	} else if tag != "" {
		slog.Info("Update available", "version", tag)
		yes := (&dialog.MsgBuilder{Msg: lang.LocalizeKey("update.available", "新しいバージョン\"{{.Version}}\"が利用可能です。「はい」をクリックすると更新します。", map[string]any{"Version": tag})}).Title(lang.LocalizeKey("update.title", "Update Available")).YesNo()
		if yes {
			slog.Info("Updating to new version", "version", tag)
			if err := versioning.Update(context.Background(), tag); err != nil {
				slog.Error("Failed to update", "error", err)
				(&dialog.MsgBuilder{Msg: lang.LocalizeKey("update.failed", "更新に失敗しました: %s", map[string]any{"Error": err.Error()})}).Title(lang.LocalizeKey("app.error", "Error")).Error()
				return err
			}
			execCmd := exec.Command(os.Args[0], os.Args[1:]...)
			return execCmd.Start()
		}
	} else {
		slog.Info("No updates available")
	}

	w := a.NewWindow(lang.LocalizeKey("app.name", "Mod of Us"))

	var client rest.Client
	if localMode != "" {
		slog.Info("Running in local mode", "path", localMode)
		f, err := rest.NewFileClient(localMode)
		if err != nil {
			slog.Error("Failed to create local file client", "error", err)
			dialog.Message("ローカルファイルクライアントの作成に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
			return err
		}
		if err := f.LoadData(); err != nil {
			slog.Error("Failed to load data from local file", "error", err)
			dialog.Message("ローカルファイルからのデータの読み込みに失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
			return err
		}
		client = f
	} else {
		slog.Info("Running in server mode", "server", server)
		client = rest.NewClient(server)
	}

	// TODO: オフラインモードの実装（アンインストール・インストール済みモッドの管理のみ可能）
	if _, err := client.GetHealthStatus(); err != nil {
		slog.Error("Failed to connect to server", "error", err)
		dialog.Message("サーバーへの接続に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
		return err
	}

	if err := ui.Main(w, version,
		ui.WithStateOptions(
			uicommon.WithRestClient(client),
		),
	); err != nil {
		slog.Error("Failed to initialize UI", "error", err)
		dialog.Message("UIの初期化に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
		return err
	}
	return nil
}
