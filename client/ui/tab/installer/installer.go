package installer

import (
	"log/slog"
	"os"

	"fyne.io/fyne/v2"
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

type Installer struct {
	state                *uicommon.State
	uninstallButton      *widget.Button
	progressBar          *progress.FyneProgress
	installationListener binding.DataListener
}

var _ uicommon.Tab = (*Installer)(nil)

func NewInstallerTab(s *uicommon.State) uicommon.Tab {
	var i Installer
	i = Installer{
		state:           s,
		uninstallButton: widget.NewButtonWithIcon(lang.LocalizeKey("installation.uninstall", "アンインストール"), theme.DeleteIcon(), i.runUninstall),
		progressBar:     progress.NewFyneProgress(widget.NewProgressBar()),
	}

	i.init()

	return &i
}

func (i *Installer) init() {
	i.uninstallButton.Importance = widget.DangerImportance
	i.uninstallButton.Disable()
	if i.installationListener == nil {
		i.installationListener = binding.NewDataListener(func() {
			if ok, err := i.state.ModInstalled.Get(); ok && err == nil {
				i.uninstallButton.Enable()
			} else if err == nil {
				i.uninstallButton.Disable()
			} else {
				slog.Warn("Failed to get modInstalled", "error", err)
			}
		})
		i.state.ModInstalled.AddListener(i.installationListener)
		i.state.SelectedGamePath.AddListener(i.installationListener)
		i.state.RefreshModInstallation()
	}
}

func (i *Installer) Tab() (*container.TabItem, error) {
	bottom := container.NewVBox(
		i.state.ErrorText,
	)
	content := container.New(
		layout.NewBorderLayout(nil, bottom, nil, nil),
		container.NewVScroll(container.NewVBox(
			widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("installation.select_install_path", "Among Usのインストール先を選択")),
			i.state.InstallSelect,
			widget.NewSeparator(),
			widget.NewAccordion(
				widget.NewAccordionItem(lang.LocalizeKey("installation.selected_install", "選択されたインストールパス"), widget.NewLabelWithData(i.state.SelectedGamePath)),
			),
			widget.NewSeparator(),
			widget.NewRichTextFromMarkdown("### "+lang.LocalizeKey("installation.installation_status", "インストール状況")), i.state.ModInstalledInfo,
			i.uninstallButton,
			widget.NewSeparator(),
		)),
		bottom,
	)
	return container.NewTabItem(lang.LocalizeKey("installation.tab_name", "インストール"), content), nil
}

// func (i *Installer) runInstall() {
// 	i.state.ErrorText.Hide()
// 	path, err := i.state.SelectedGamePath.Get()
// 	if err != nil || path == "" {
// 		i.state.ErrorText.Segments = []widget.RichTextSegment{
// 			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.no_path", "インストールパスが指定されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 		}
// 		i.state.ErrorText.Refresh()
// 		i.state.ErrorText.Show()
// 		return
// 	}
// 	selectedModIndex := i.modsSelect.SelectedIndex()
// 	if selectedModIndex < 0 || selectedModIndex >= len(i.modIndexMap) {
// 		i.state.ErrorText.Segments = []widget.RichTextSegment{
// 			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.no_mod_selected", "インストールするModが選択されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 		}
// 		i.state.ErrorText.Refresh()
// 		i.state.ErrorText.Show()
// 		return
// 	}
// 	selected, err := i.state.Mods.GetValue(i.modIndexMap[selectedModIndex])
// 	if err != nil {
// 		i.state.ErrorText.Segments = []widget.RichTextSegment{
// 			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.mod_not_found", "選択されたModが見つかりません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 		}
// 		i.state.ErrorText.Refresh()
// 		i.state.ErrorText.Show()
// 		return
// 	}
// 	var mods []modmgr.Mod
// 	if selected.Type == modmgr.ModTypeModPack {
// 		allMods, err := i.state.Mods.Get()
// 		if err != nil {
// 			i.state.ErrorText.Segments = []widget.RichTextSegment{
// 				&widget.TextSegment{Text: lang.LocalizeKey("installation.error.failed_to_get_mods", "Modの取得に失敗しました。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 			}
// 			i.state.ErrorText.Refresh()
// 			i.state.ErrorText.Show()
// 			return
// 		}
// 		modsMap := make(map[string]modmgr.Mod)
// 		for _, mod := range allMods {
// 			modsMap[mod.ID] = mod
// 		}
// 		for _, modID := range selected.Mods {
// 			mod, ok := modsMap[modID]
// 			if !ok {
// 				i.state.ErrorText.Segments = []widget.RichTextSegment{
// 					&widget.TextSegment{Text: lang.LocalizeKey("installation.error.mod_not_found_in_pack", "Modパック内のModが見つかりません: ") + modID, Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 				}
// 				i.state.ErrorText.Refresh()
// 				i.state.ErrorText.Show()
// 				return
// 			}
// 			mods = append(mods, mod)
// 		}
// 	} else {
// 		mods = []modmgr.Mod{selected}
// 	}
// 	// Resolve dependencies
// 	resolved := make(map[string]modmgr.Mod)
// 	unresolved := make(map[string]struct{})
// 	conflict := make(map[string][]string)
// 	for _, mod := range mods {
// 		if err := i.resolveDependencies(mod, resolved, unresolved, conflict); err != nil {
// 			i.state.ErrorText.Segments = []widget.RichTextSegment{
// 				&widget.TextSegment{Text: lang.LocalizeKey("installation.error.dependency_resolution_failed", "Modの依存関係の解決に失敗しました: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 			}
// 			i.state.ErrorText.Refresh()
// 			i.state.ErrorText.Show()
// 			return
// 		}
// 	}
// 	if conflictMods := make([]string, 0); len(conflict) > 0 {
// 		for modID, conflicts := range conflict {
// 			for _, c := range conflicts {
// 				if _, ok := resolved[c]; ok {
// 					conflictMods = append(conflictMods, fmt.Sprintf("%s ↔ %s", modID, c))
// 				}
// 			}
// 		}
// 		if len(conflictMods) > 0 {
// 			i.state.ErrorText.Segments = []widget.RichTextSegment{
// 				&widget.TextSegment{Text: lang.LocalizeKey("installation.error.dependency_conflict", "Modの依存関係に競合があります: ") + strings.Join(conflictMods, ", "), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 			}
// 			i.state.ErrorText.Refresh()
// 			i.state.ErrorText.Show()
// 			return
// 		}
// 	}
// 	mods = make([]modmgr.Mod, 0, len(resolved))
// 	for _, mod := range resolved {
// 		if slices.ContainsFunc(mods, func(e modmgr.Mod) bool {
// 			return e.ID == mod.ID
// 		}) {
// 			continue
// 		}
// 		mods = append(mods, mod)
// 	}

// 	detectedLauncher := aumgr.DetectLauncherType(path)
// 	if detectedLauncher == aumgr.LauncherUnknown {
// 		i.state.ErrorText.Segments = []widget.RichTextSegment{
// 			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.launcher_not_found", "指定されたパスからAmong Usの実行ファイルが見つかりません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 		}
// 		i.state.ErrorText.Refresh()
// 		i.state.ErrorText.Show()
// 		return
// 	}

// 	detectedBinaryType, err := aumgr.DetectBinaryType(path)
// 	if err != nil {
// 		i.state.ErrorText.Segments = []widget.RichTextSegment{
// 			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.binary_not_found", "指定されたパスからAmong Usの実行ファイルが見つかりません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 		}
// 		i.state.ErrorText.Refresh()
// 		i.state.ErrorText.Show()
// 		return
// 	}

// 	slog.Info("Installing mod", "mod", selected.ID, "path", path, "launcher", detectedLauncher.String())
// 	manifest, err := aumgr.GetManifest(detectedLauncher, path)
// 	if err != nil {
// 		i.state.ErrorText.Segments = []widget.RichTextSegment{
// 			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.failed_to_get_version", "ゲームのバージョン情報の取得に失敗しました。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 		}
// 		i.state.ErrorText.Refresh()
// 		i.state.ErrorText.Show()
// 		return
// 	}
// 	go func() {
// 		defer i.state.CheckInstalled()
// 		defer func() { _ = i.state.CanInstall.Set(true) }()
// 		fyne.DoAndWait(func() {
// 			_ = i.state.ModInstalled.Set(false)
// 			_ = i.state.CanLaunch.Set(false)
// 			_ = i.state.CanInstall.Set(false)
// 		})

// 		modInstallLocation, err := os.OpenRoot(i.state.ModInstallDir())
// 		if err != nil {
// 			slog.Warn("Failed to open game root", "error", err)
// 			return
// 		}

// 		_, err = modmgr.InstallMod(modInstallLocation, manifest, detectedLauncher, detectedBinaryType, mods, i.progressBar)
// 		if err != nil {
// 			fyne.DoAndWait(func() {
// 				i.state.ErrorText.Segments = []widget.RichTextSegment{
// 					&widget.TextSegment{Text: lang.LocalizeKey("installation.error.failed_to_install", "Modのインストールに失敗しました: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
// 				}
// 				i.state.ErrorText.Refresh()
// 				i.state.ErrorText.Show()
// 				slog.Warn("Failed to install mod", "error", err)
// 				i.state.CheckInstalled()
// 			})
// 			return
// 		}
// 		fyne.DoAndWait(func() {
// 			i.state.ErrorText.ParseMarkdown(lang.LocalizeKey("installation.success.installed", "Modのインストールに成功しました。"))
// 			i.state.ErrorText.Refresh()
// 			i.state.ErrorText.Show()
// 			slog.Info("Mod installed successfully", "mod", selected.ID, "path", path)
// 			i.state.CheckInstalled()
// 		})
// 	}()
// }

func (i *Installer) runUninstall() {
	defer i.state.RefreshModInstallation()
	i.state.ErrorText.Hide()
	path, err := i.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.no_path", "インストールパスが指定されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	slog.Info("Uninstalling mod", "path", path)

	modInstallLocation, err := os.OpenRoot(i.state.ModInstallDir())
	if err != nil {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.failed_to_open_path", "指定されたパスのオープンに失敗しました。: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		slog.Warn("Failed to open game root", "error", err)
		return
	}

	if _, err := modInstallLocation.Stat(modmgr.InstallationInfoFileName); os.IsNotExist(err) {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.mod_not_installed", "このパスにはModがインストールされていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	go func() {
		if err := modmgr.UninstallMod(modInstallLocation, i.progressBar, nil); err != nil {
			fyne.Do(func() {
				i.state.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("installation.error.failed_to_uninstall", "Modのアンインストールに失敗しました: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
				}
				i.state.ErrorText.Refresh()
				i.state.ErrorText.Show()
				slog.Warn("Failed to uninstall mod", "error", err)
			})
			return
		}
		fyne.Do(func() {
			i.state.ErrorText.ParseMarkdown(lang.LocalizeKey("installation.success.uninstalled", "Modのアンインストールに成功しました。"))
			i.state.ErrorText.Refresh()
			i.state.ErrorText.Show()
			slog.Info("Mod uninstalled successfully", "path", path)
			i.state.CheckInstalled()
			i.state.RefreshModInstallation()
		})
	}()
}
