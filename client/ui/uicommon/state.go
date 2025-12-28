package uicommon

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type Option func(*Config)

type Config struct {
	rest rest.Client
}

func WithRestClient(c rest.Client) func(*Config) {
	return func(cfg *Config) {
		cfg.rest = c
	}
}

func NewState(w fyne.Window, version string, options ...Option) (*State, error) {
	detectedPath, err := aumgr.GetAmongUsDir()
	if err != nil {
		return nil, err
	}
	if aumgr.DetectLauncherType(detectedPath) == aumgr.LauncherUnknown {
		return nil, errors.New("Among Us detected but launcher type is unknown")
	}

	// execPath, err := os.Executable()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get executable path: %w", err)
	// }

	// modPath := filepath.Join(filepath.Dir(execPath), "mods")

	// if err := os.MkdirAll(modPath, 0755); err != nil {
	// 	return nil, fmt.Errorf("failed to create mods directory: %w", err)
	// }

	var cfg Config
	for _, option := range options {
		option(&cfg)
	}

	var s State
	s = State{
		Version: version,
		Window:  w,
		// ModPath:          modPath,
		SelectedGamePath: binding.NewString(),
		DetectedGamePath: detectedPath,
		ModInstalled:     binding.NewBool(),
		CanLaunch:        binding.NewBool(),
		CanInstall:       binding.NewBool(),
		InstallSelect:    widget.NewSelect([]string{}, s.selectLauncher),
		ErrorText:        widget.NewRichTextFromMarkdown(""),

		ModInstalledInfo: widget.NewLabel(lang.LocalizeKey("installer.select_install_path", "インストール先を選択してください。")),
		Rest:             cfg.rest,
	}

	if err := s.CanInstall.Set(true); err != nil {
		return nil, err
	}

	listener := binding.NewDataListener(s.RefreshModInstallation)
	s.ModInstalled.AddListener(listener)
	s.SelectedGamePath.AddListener(listener)
	s.ModInstalledInfo.Wrapping = fyne.TextWrapWord
	s.ModInstalledInfo.TextStyle.Symbol = true
	s.ErrorText.Wrapping = fyne.TextWrapWord
	s.ErrorText.Hide()
	s.InstallSelect.PlaceHolder = lang.LocalizeKey("installer.select_install", "（Among Usを選択）")
	detectedLauncher := aumgr.DetectLauncherType(detectedPath)
	s.InstallSelect.Options = []string{detectedLauncher.String(), lang.LocalizeKey("installer.manual_select", "手動選択")}
	s.InstallSelect.Selected = detectedLauncher.String()
	if err := s.SelectedGamePath.Set(detectedPath); err != nil {
		return nil, err
	}

	go func() {
		for {
			if s.checkPlayingProcess() {
				slog.Info("Among Us is running, disabling installation and launch")
			}
			// Check every 5 seconds
			<-time.After(5 * time.Second)
		}
	}()

	return &s, nil
}

type State struct {
	Version string
	Window  fyne.Window
	// ModPath          string
	SelectedGamePath binding.String
	DetectedGamePath string
	ModInstalled     binding.Bool
	CanLaunch        binding.Bool
	CanInstall       binding.Bool

	Rest rest.Client

	ModInstalledInfo *widget.Label
	InstallSelect    *widget.Select
	ErrorText        *widget.RichText
}

func (s *State) ModInstallDir() string {
	path, err := s.SelectedGamePath.Get()
	if err != nil || path == "" {
		return ""
	}
	return path
}

type Tab interface {
	Tab() (*container.TabItem, error)
}

func (s *State) SetError(err error) {
	if err == nil {
		s.ErrorText.Hide()
		return
	}
	s.ErrorText.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  lang.LocalizeKey("common.error_occurred", "エラーが発生しました: ") + err.Error(),
			Style: widget.RichTextStyle{ColorName: theme.ColorNameError},
		},
	}
	fyne.Do(func() {
		s.ErrorText.Refresh()
		s.ErrorText.Show()
	})
}

func (s *State) ClearError() {
	s.ErrorText.Hide()
}

func (i *State) RefreshModInstallation() {
	if err := i.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set launchable", "error", err)
	}
	path, err := i.SelectedGamePath.Get()
	if err != nil || path == "" {
		defer i.ModInstalledInfo.Refresh()
		i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.info.select_path", "インストール先を選択してください。"))
		return
	}
	if ok, err := i.ModInstalled.Get(); ok && err == nil {
		defer i.ModInstalledInfo.Refresh()
		detectedLauncher := aumgr.DetectLauncherType(path)
		manifest, err := aumgr.GetManifest(detectedLauncher, path)
		if err != nil {
			slog.Warn("Failed to get game manifest", "error", err)
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_get_version", "Modがインストールされていますが、ゲームのバージョン情報の取得に失敗しました。"))
			return
		}

		modInstallLocation, err := os.OpenRoot(i.ModInstallDir())
		if err != nil {
			slog.Warn("Failed to open game root", "error", err)
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_open_path", "Modがインストールされていますが、インストール先のオープンに失敗しました。"))
			return
		}

		installationInfo, err := modmgr.LoadInstallationInfo(modInstallLocation)
		if err != nil {
			slog.Warn("Failed to load installation info", "error", err)
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_get_installation_info", "Modがインストールされていますが、インストール情報の取得に失敗しました。"))
			return
		}
		if installationInfo.Status == modmgr.InstallStatusBroken {
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.broken_installation", "Modのインストールが壊れています。Modアンインストールしてから再インストールしてください。"))
			return
		}
		canLaunch := false
		info := lang.LocalizeKey("installer.info.mod_installed", "Modがインストールされています。") + "\n"
		if manifest.GetVersion() == installationInfo.InstalledGameVersion {
			info += lang.LocalizeKey("installer.info.game_version", "ゲームバージョン: ") + manifest.GetVersion() + "\n"
			canLaunch = true
			for _, mod := range installationInfo.InstalledMods {
				remoteMod, err := i.Mod(mod.ModID)
				if err != nil {
					slog.Warn("Failed to get mod", "modID", mod.ID, "error", err)
					continue
				}
				if remoteMod.LatestVersion != mod.ModVersion.ID {
					info += lang.LocalizeKey("installer.info.mod_version_outdated", "Modのバージョンが古くなっています: {{.mod}} (インストール済み: {{.version}}, 最新: {{.latest}})",
						map[string]any{
							"mod":     mod.ModID,
							"version": mod.ModVersion.ID,
							"latest":  remoteMod.LatestVersion,
						}) + "\n"
					canLaunch = false
					break
				}
			}
		} else {
			info += lang.LocalizeKey("installer.info.game_version", "ゲームバージョン: ") + manifest.GetVersion() + " (Modインストール時: " + installationInfo.InstalledGameVersion + ")\n"
			info += lang.LocalizeKey("installer.info.mod_incompatible", "Modは現在のゲームバージョンと互換性がありません。") + "\n"
			installationInfo.Status = modmgr.InstallStatusIncompatible
			if err := modmgr.SaveInstallationInfo(modInstallLocation, installationInfo); err != nil {
				slog.Warn("Failed to save installation info", "error", err)
			}
		}
		var modNames string
		idx := 0
		for _, mod := range installationInfo.InstalledMods {
			if idx > 0 {
				modNames += ", "
			}
			modNames += mod.ModID + " (" + mod.ModVersion.ID + ")"
			idx++
		}
		info += lang.LocalizeKey("installer.info.mod_name", "Mod: ") + modNames + "\n"
		i.ModInstalledInfo.SetText(strings.TrimSpace(info))
		if err := i.CanLaunch.Set(canLaunch); err != nil {
			slog.Warn("Failed to set launchable", "error", err)
		}
	} else if err == nil {
		defer i.ModInstalledInfo.Refresh()
		i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.info.mod_not_installed", "Modはインストールされていません。"))
	} else {
		slog.Warn("Failed to get mod installed", "error", err)
	}
}

func (s *State) checkPlayingProcess() bool {
	closed := false
	if ok, err := s.CanInstall.Get(); err == nil && !ok {
		closed = true
	}
	pid, err := aumgr.IsAmongUsRunning()
	if err != nil {
		slog.Error("Failed to check Among Us process", "error", err)
		return false
	}
	if pid != 0 {
		slog.Info("Among Us is currently running", "pid", pid)

		_ = s.CanInstall.Set(false)
		_ = s.CanLaunch.Set(false)

		return true
	} else if closed {
		slog.Info("Among Us is not running, re-enabling installation")
		_ = s.CanInstall.Set(true)
		s.RefreshModInstallation()
	}
	return false
}
