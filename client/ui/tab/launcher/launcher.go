package launcher

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
)

type Launcher struct {
	state           *uicommon.State
	launchButton    *widget.Button
	greetingContent *widget.Label

	canLaunchListener binding.DataListener
}

var _ uicommon.Tab = (*Launcher)(nil)

func NewLauncherTab(s *uicommon.State) uicommon.Tab {
	var l Launcher
	l = Launcher{
		state:           s,
		launchButton:    widget.NewButtonWithIcon(lang.LocalizeKey("launcher.launch", "起動"), theme.MediaPlayIcon(), l.runLaunch),
		greetingContent: widget.NewLabelWithStyle(fmt.Sprintf("バージョン：%s", s.Version), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	}

	l.init()

	return &l
}

func (l *Launcher) init() {
	if l.canLaunchListener == nil {
		l.canLaunchListener = binding.NewDataListener(l.checkLaunchState)
		l.state.CanLaunch.AddListener(l.canLaunchListener)
		l.checkLaunchState()
	}
	l.greetingContent.Wrapping = fyne.TextWrapWord

	l.launchButton.Importance = widget.HighImportance
}

func (l *Launcher) Tab() (*container.TabItem, error) {
	content := container.NewPadded(container.NewVBox(
		widget.NewCard("Mod of Us", "Among UsのModマネージャー", l.greetingContent),
		widget.NewRichTextFromMarkdown("### "+lang.LocalizeKey("installation.installation_status", "インストール状況")), l.state.ModInstalledInfo,
		widget.NewSeparator(),
		l.launchButton,
		l.state.ErrorText,
	))
	return container.NewTabItem(lang.LocalizeKey("launcher.tab_name", "ランチャー"), content), nil
}

func (l *Launcher) runLaunch() {
	l.state.ErrorText.Hide()
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		l.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("launcher.error.no_path", "ゲームパスが指定されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		l.state.ErrorText.Refresh()
		l.state.ErrorText.Show()
		return
	}

	// // Copy winhttp.dll to Among Us directory
	// if err := copyFile(filepath.Join(l.state.ModInstallDir(), "winhttp.dll"), filepath.Join(path, "winhttp.dll")); err != nil {
	// 	l.state.ErrorText.Segments = []widget.RichTextSegment{
	// 		&widget.TextSegment{Text: "winhttp.dllのコピーに失敗しました: " + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
	// 	}
	// 	l.state.ErrorText.Refresh()
	// 	l.state.ErrorText.Show()
	// 	slog.Warn("Failed to copy winhttp.dll", "error", err)
	// 	return
	// }

	// Generate and write doorstop_config.ini to Among Us directory
	// doorstopConfig := generateDoorstopConfig(l.state.ModInstallDir())
	// if err := os.WriteFile(filepath.Join(path, "doorstop_config.ini"), []byte(doorstopConfig), 0644); err != nil {
	// 	l.state.ErrorText.Segments = []widget.RichTextSegment{
	// 		&widget.TextSegment{Text: "doorstop_config.iniの書き込みに失敗しました: " + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
	// 	}
	// 	l.state.ErrorText.Refresh()
	// 	l.state.ErrorText.Show()
	// 	slog.Warn("Failed to write doorstop_config.ini", "error", err)
	// 	return
	// }

	// args := []string{
	// 	"--doorstop-enabled true",
	// 	"--doorstop-target-assembly", fmt.Sprintf("\"%s\"", filepath.Join(l.state.ModInstallDir(), "BepInEx", "core", "BepInEx.Unity.IL2CPP.dll")),
	// 	"--doorstop-clr-corlib-dir", fmt.Sprintf("\"%s\"", filepath.Join(l.state.ModInstallDir(), "dotnet")),
	// 	"--doorstop-clr-runtime-coreclr-path", fmt.Sprintf("\"%s\"", filepath.Join(l.state.ModInstallDir(), "dotnet", "coreclr.dll")),
	// }
	_ = l.state.CanLaunch.Set(false)
	_ = l.state.CanInstall.Set(false)
	go l.state.Launch(path)
}

func (l *Launcher) checkLaunchState() {
	canLaunch, err := l.state.CanLaunch.Get()
	if err != nil {
		slog.Error("Failed to get CanLaunch state", "error", err)
		return
	}
	if canLaunch {
		l.launchButton.Enable()
	} else {
		l.launchButton.Disable()
	}
}

// // copyFile copies a file from src to dst
// func copyFile(src, dst string) error {
// 	srcFile, err := os.Open(src)
// 	if err != nil {
// 		return fmt.Errorf("failed to open source file: %w", err)
// 	}
// 	defer srcFile.Close()

// 	dstFile, err := os.Create(dst)
// 	if err != nil {
// 		return fmt.Errorf("failed to create destination file: %w", err)
// 	}
// 	defer dstFile.Close()

// 	if _, err := io.Copy(dstFile, srcFile); err != nil {
// 		return fmt.Errorf("failed to copy file: %w", err)
// 	}

// 	return nil
// }

// // generateDoorstopConfig generates the doorstop_config.ini content with the correct paths
// func generateDoorstopConfig(modPath string) string {
// 	return fmt.Sprintf(`# General options for Unity Doorstop
// [General]

// # Enable Doorstop?
// enabled = true

// # Path to the assembly to load and execute
// # NOTE: The entrypoint must be of format `+"`static void Doorstop.Entrypoint.Start()`"+`
// target_assembly = %s

// # If true, Unity's output log is redirected to <current folder>\output_log.txt
// redirect_output_log = false

// # Overrides the default boot.config file path
// boot_config_override =

// # If enabled, DOORSTOP_DISABLE env var value is ignored
// # USE THIS ONLY WHEN ASKED TO OR YOU KNOW WHAT THIS MEANS
// ignore_disable_switch = false

// # Options specific to running under Unity Mono runtime
// [UnityMono]

// # Overrides default Mono DLL search path
// # Sometimes it is needed to instruct Mono to seek its assemblies from a different path
// # (e.g. mscorlib is stripped in original game)
// # This option causes Mono to seek mscorlib and core libraries from a different folder before Managed
// # Original Managed folder is added as a secondary folder in the search path
// dll_search_path_override =

// # If true, Mono debugger server will be enabled
// debug_enabled = false

// # When debug_enabled is true, this option specifies whether Doorstop should initialize the debugger server
// # If you experience crashes when starting the debugger on debug UnityPlayer builds, try setting this to false
// debug_start_server = true

// # When debug_enabled is true, specifies the address to use for the debugger server
// debug_address = 127.0.0.1:10000

// # If true and debug_enabled is true, Mono debugger server will suspend the game execution until a debugger is attached
// debug_suspend = false

// # Options sepcific to running under Il2Cpp runtime
// [Il2Cpp]

// # Path to coreclr.dll that contains the CoreCLR runtime
// coreclr_path = %s

// # Path to the directory containing the managed core libraries for CoreCLR (mscorlib, System, etc.)
// corlib_dir = %s
// `,
// 		filepath.Join(modPath, "BepInEx", "core", "BepInEx.Unity.IL2CPP.dll"),
// 		filepath.Join(modPath, "dotnet", "coreclr.dll"),
// 		filepath.Join(modPath, "dotnet"),
// 	)
// }
