package main

import (
	"au_mod_installer/pkg/aumgr"
	"au_mod_installer/ui"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/sqweek/dialog"
)

var (
	detectedGamePath string             = "" // Example: "C:\\Program Files (x86)\\Steam\\steamapps\\common\\Among Us\\"
	detectedLauncher aumgr.LauncherType = aumgr.LauncherUnknown
	launchButton     *widget.Button

	selectedModIndex int = 0

	initErr error = nil
)

func init() {
	gamePath, err := aumgr.GetAmongUsDir()
	if err == nil {
		detectedGamePath = gamePath
	}
	detectedLauncher = aumgr.DetectLauncherType(detectedGamePath)
	if detectedLauncher == aumgr.LauncherUnknown {
		detectedGamePath = ""
		return
	}
}

func main() {
	a := app.New()
	w := a.NewWindow(lang.LocalizeKey("app.name", "Among Us Mod Loader"))
	if err := ui.Main(w); err != nil {
		dialog.Message("UIの初期化に失敗しました: %s", err.Error()).Title("エラーが発生しました").Error()
		os.Exit(1)
	}
}
