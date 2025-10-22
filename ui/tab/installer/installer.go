package installer

import (
	"au_mod_installer/pkg/aumgr"
	"au_mod_installer/pkg/modmgr"
	"au_mod_installer/pkg/progress"
	"au_mod_installer/ui/common"
	"log/slog"
	"os"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Installer struct {
	state            *common.State
	uninstallButton  *widget.Button
	installButton    *widget.Button
	modInstalledInfo *widget.Label
	modsSelect       *widget.Select
	progressBar      *progress.FyneProgress
	modRefreshButton *widget.Button

	installationListener    binding.DataListener
	modListener             binding.DataListener
	modInstallStateListener binding.DataListener
}

var _ common.Tab = (*Installer)(nil)

func NewInstallerTab(s *common.State) common.Tab {
	var i Installer
	i = Installer{
		state:            s,
		uninstallButton:  widget.NewButtonWithIcon(lang.LocalizeKey("installer.uninstall", "アンインストール"), theme.DeleteIcon(), i.runUninstall),
		installButton:    widget.NewButtonWithIcon(lang.LocalizeKey("installer.install", "インストール"), theme.DownloadIcon(), i.runInstall),
		modInstalledInfo: widget.NewLabel(lang.LocalizeKey("installer.select_install_path", "インストール先を選択してください。")),
		modsSelect:       widget.NewSelect([]string{}, nil),
		progressBar:      progress.NewFyneProgress(widget.NewProgressBar()),
		modRefreshButton: widget.NewButtonWithIcon(lang.LocalizeKey("installer.refresh_mods", "Modを再取得"), theme.ViewRefreshIcon(), i.refetchMods),
	}

	i.init()

	return &i
}

func (i *Installer) init() {
	i.modInstalledInfo.Wrapping = fyne.TextWrapWord
	i.modInstalledInfo.TextStyle.Symbol = true
	i.installButton.Importance = widget.HighImportance
	i.uninstallButton.Importance = widget.DangerImportance
	i.uninstallButton.Disable()

	i.modsSelect.PlaceHolder = lang.LocalizeKey("installer.select_mod", "（Modを選択）")
	if i.modListener == nil {
		i.modListener = binding.NewDataListener(i.refreshModList)
		i.refreshModList()
	}

	if i.installationListener == nil {
		i.installationListener = binding.NewDataListener(i.refreshModInstallation)
		i.state.ModInstalled.AddListener(i.installationListener)
		i.state.SelectedGamePath.AddListener(i.installationListener)
		i.refreshModInstallation()
	}

	if i.modInstallStateListener == nil {
		i.modInstallStateListener = binding.NewDataListener(i.installStateUpdate)
		i.state.CanInstall.AddListener(i.modInstallStateListener)
		_ = i.state.CanInstall.Set(true)
		i.installStateUpdate()
	}
}

func (i *Installer) Tab() (*container.TabItem, error) {
	bottom := container.NewVBox(
		i.progressBar.Canvas(),
	)
	content := container.New(
		layout.NewBorderLayout(nil, bottom, nil, nil),
		container.NewVScroll(container.NewVBox(
			widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("installer.select_install_path", "Among Usのインストール先を選択")),
			i.state.InstallSelect,
			widget.NewSeparator(),
			widget.NewAccordion(
				widget.NewAccordionItem(lang.LocalizeKey("installer.selected_install", "選択されたインストールパス"), widget.NewLabelWithData(i.state.SelectedGamePath)),
			),
			widget.NewSeparator(),
			widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("installer.select_mod", "インストールするModを選択")),
			container.New(layout.NewBorderLayout(nil, nil, nil, i.modRefreshButton), i.modsSelect, i.modRefreshButton),
			i.installButton,
			widget.NewSeparator(),
			widget.NewRichTextFromMarkdown("### "+lang.LocalizeKey("installer.installation_status", "インストール状況")), i.modInstalledInfo,
			i.uninstallButton,
			widget.NewSeparator(),
			i.state.ErrorText,
		)),
		bottom,
	)
	return container.NewTabItem(lang.LocalizeKey("installer.tab_name", "インストーラー"), content), nil
}

func (i *Installer) runInstall() {
	i.state.ErrorText.Hide()
	path, err := i.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installer.error.no_path", "インストールパスが指定されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	selectedModIndex := i.modsSelect.SelectedIndex()
	if selectedModIndex < 0 || selectedModIndex >= i.state.Mods.Length() {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installer.error.no_mod_selected", "インストールするModが選択されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	selected, err := i.state.Mods.GetValue(selectedModIndex)
	if err != nil {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installer.error.mod_not_found", "選択されたModが見つかりません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}

	detectedLauncher := aumgr.DetectLauncherType(path)
	if detectedLauncher == aumgr.LauncherUnknown {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installer.error.launcher_not_found", "指定されたパスからAmong Usの実行ファイルが見つかりません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}

	slog.Info("Installing mod", "mod", selected.Name, "path", path, "launcher", detectedLauncher.String())
	manifest, err := aumgr.GetManifest(detectedLauncher, path)
	if err != nil {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installer.error.failed_to_get_version", "ゲームのバージョン情報の取得に失敗しました。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	go func() {
		defer i.state.CheckInstalled()
		defer func() { _ = i.state.CanInstall.Set(true) }()
		fyne.DoAndWait(func() {
			_ = i.state.ModInstalled.Set(false)
			_ = i.state.CanLaunch.Set(false)
			_ = i.state.CanInstall.Set(false)
		})
		_, err = modmgr.InstallMod(path, manifest, detectedLauncher, selected, i.progressBar)
		if err != nil {
			fyne.DoAndWait(func() {
				i.state.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("installer.error.failed_to_install", "Modのインストールに失敗しました: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
				}
				i.state.ErrorText.Refresh()
				i.state.ErrorText.Show()
				slog.Warn("Failed to install mod", "error", err)
				i.state.CheckInstalled()
			})
			return
		}
		fyne.DoAndWait(func() {
			i.state.ErrorText.ParseMarkdown(lang.LocalizeKey("installer.success.installed", "Modのインストールに成功しました。"))
			i.state.ErrorText.Refresh()
			i.state.ErrorText.Show()
			slog.Info("Mod installed successfully", "mod", selected.Name, "path", path)
			i.state.CheckInstalled()
		})
	}()
}

func (i *Installer) runUninstall() {
	defer i.refreshModInstallation()
	i.state.ErrorText.Hide()
	path, err := i.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installer.error.no_path", "インストールパスが指定されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	slog.Info("Uninstalling mod", "path", path)
	installationInfoFilePath := modmgr.GetInstallationInfoFilePath(path)
	if _, err := os.Stat(installationInfoFilePath); os.IsNotExist(err) {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installer.error.mod_not_installed", "このパスにはModがインストールされていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	go func() {
		if err := modmgr.UninstallMod(path, installationInfoFilePath, i.progressBar); err != nil {
			fyne.Do(func() {
				i.state.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("installer.error.failed_to_uninstall", "Modのアンインストールに失敗しました: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
				}
				i.state.ErrorText.Refresh()
				i.state.ErrorText.Show()
				slog.Warn("Failed to uninstall mod", "error", err)
			})
			return
		}
		fyne.Do(func() {
			i.state.ErrorText.ParseMarkdown(lang.LocalizeKey("installer.success.uninstalled", "Modのアンインストールに成功しました。"))
			i.state.ErrorText.Refresh()
			i.state.ErrorText.Show()
			slog.Info("Mod uninstalled successfully", "path", path)
			i.state.CheckInstalled()
			i.refreshModInstallation()
		})
	}()
}

func (i *Installer) refetchMods() {
	i.state.ErrorText.Hide()
	go func() {
		if err := i.state.FetchMods(); err != nil {
			fyne.Do(func() {
				i.state.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("installer.error.failed_to_fetch_mods", "Modの取得に失敗しました: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
				}
				i.state.ErrorText.Refresh()
				i.state.ErrorText.Show()
				slog.Warn("Failed to fetch mods", "error", err)
			})
			return
		}
		fyne.Do(func() {
			i.state.ErrorText.ParseMarkdown(lang.LocalizeKey("installer.success_fetched_mods", "{{.mod_count}}個のModを取得しました", map[string]interface{}{"mod_count": i.state.Mods.Length()}))
			i.state.ErrorText.Refresh()
			i.state.ErrorText.Show()
			slog.Info("Mods fetched successfully")
			i.refreshModList()
		})
	}()
}

func (i *Installer) refreshModInstallation() {
	if err := i.state.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set launchable", "error", err)
	}
	path, err := i.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		defer i.modInstalledInfo.Refresh()
		i.modInstalledInfo.SetText(lang.LocalizeKey("installer.info.select_path", "インストール先を選択してください。"))
		return
	}
	if ok, err := i.state.ModInstalled.Get(); ok && err == nil {
		i.uninstallButton.Enable()
		defer i.modInstalledInfo.Refresh()
		detectedLauncher := aumgr.DetectLauncherType(path)
		manifest, err := aumgr.GetManifest(detectedLauncher, path)
		if err != nil {
			slog.Warn("Failed to get game manifest", "error", err)
			i.modInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_get_version", "Modがインストールされていますが、ゲームのバージョン情報の取得に失敗しました。"))
			return
		}
		installationInfoFilePath := modmgr.GetInstallationInfoFilePath(path)
		installationInfo, err := modmgr.LoadInstallationInfo(installationInfoFilePath)
		if err != nil {
			slog.Warn("Failed to load installation info", "error", err)
			i.modInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_get_installation_info", "Modがインストールされていますが、インストール情報の取得に失敗しました。"))
			return
		}
		if installationInfo.Status == modmgr.InstallStatusBroken {
			i.modInstalledInfo.SetText(lang.LocalizeKey("installer.error.broken_installation", "Modのインストールが壊れています。Modアンインストールしてから再インストールしてください。"))
			return
		}
		canLaunch := false
		info := lang.LocalizeKey("installer.info.mod_installed", "Modがインストールされています。") + "\n"
		if manifest.GetVersion() == installationInfo.InstalledGameVersion {
			info += lang.LocalizeKey("installer.info.game_version", "ゲームバージョン: ") + manifest.GetVersion() + "\n"
			mods, err := i.state.Mods.Get()
			if err != nil {
				slog.Warn("Failed to get mods", "error", err)
			}
			if i := slices.IndexFunc(mods, func(m modmgr.Mod) bool {
				return installationInfo.InstalledMod.Name == m.Name && installationInfo.InstalledMod.Version != m.Version
			}); i != -1 {
				info += lang.LocalizeKey("installer.info.mod_version_outdated", "Modのバージョンが最新のものと異なります。Modを更新してください。") + "\n"
			} else {
				canLaunch = true
			}
		} else {
			info += lang.LocalizeKey("installer.info.game_version", "ゲームバージョン: ") + manifest.GetVersion() + " (Modインストール時: " + installationInfo.InstalledGameVersion + ")\n"
			info += lang.LocalizeKey("installer.info.mod_incompatible", "Modは現在のゲームバージョンと互換性がありません。") + "\n"
			installationInfo.Status = modmgr.InstallStatusIncompatible
			if err := modmgr.SaveInstallationInfo(installationInfoFilePath, installationInfo); err != nil {
				slog.Warn("Failed to save installation info", "error", err)
			}
		}
		info += lang.LocalizeKey("installer.info.mod_name", "Mod: ") + installationInfo.InstalledMod.Name + "\n"
		info += lang.LocalizeKey("installer.info.mod_version", "Modバージョン: ") + installationInfo.InstalledMod.Version + "\n"
		i.modInstalledInfo.SetText(strings.TrimSpace(info))
		if err := i.state.CanLaunch.Set(canLaunch); err != nil {
			slog.Warn("Failed to set launchable", "error", err)
		}
	} else if err == nil {
		i.uninstallButton.Disable()
		defer i.modInstalledInfo.Refresh()
		i.modInstalledInfo.SetText(lang.LocalizeKey("installer.info.mod_not_installed", "Modはインストールされていません。"))
	} else {
		slog.Warn("Failed to get mod installed", "error", err)
	}
}

func (i *Installer) refreshModList() {
	mods, err := i.state.Mods.Get()
	if err != nil {
		slog.Warn("Failed to get mods", "error", err)
		return
	}
	modNames := make([]string, len(mods))
	for idx, mod := range mods {
		modNames[idx] = mod.Name + " (" + mod.Version + ")"
	}
	i.modsSelect.Options = modNames
	i.modsSelect.ClearSelected()
	i.modsSelect.Refresh()
}

func (i *Installer) installStateUpdate() {
	ok, err := i.state.CanInstall.Get()
	if err != nil {
		slog.Warn("Failed to get install state", "error", err)
		return
	}
	if ok {
		i.installButton.Enable()
	} else {
		i.installButton.Disable()
	}
}
