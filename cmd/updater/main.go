//go:build windows

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"golang.org/x/mod/semver"

	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/common/versioning"
)

var defaultServer = "https://modofus.sabafly.net/api/v1"

func main() {
	var (
		serverFlag  string
		localMode   string
		offlineFlag bool
		silentFlag  bool
		targetFlag  string
	)
	flag.StringVar(&serverFlag, "server", "", "URL of the mod server")
	flag.StringVar(&localMode, "local", "", "Path to local mods.json file for local mode")
	flag.BoolVar(&offlineFlag, "offline", false, "Run in offline mode")
	flag.BoolVar(&silentFlag, "silent", false, "Start minimized in system tray")
	flag.StringVar(&targetFlag, "target", "", "Path to target executable to launch after update")
	flag.Parse()

	branchName := readUpdateBranchPreference()
	branch := versioning.BranchFromString(branchName)

	serverURL := serverFlag
	if serverURL == "" {
		serverURL = defaultServer
	}

	currentVersion := readCurrentVersion()

	// If not in offline or local mode, perform update check without confirmation
	if !offlineFlag && localMode == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		client := rest.NewClient(serverURL)
		info, err := client.GetVersionInfo()
		if err != nil {
			slog.Warn("Failed to check for updates on startup", "error", err)
		} else if info != nil {
			tag := versioning.FindBranchVersion(info, branch.String())
			if tag != "" && shouldPerformUpdate(currentVersion, tag) {
				slog.Info("Update available on startup, downloading and installing with /passive without confirmation", "target", tag, "current", currentVersion)
				msiPath, err := versioning.DownloadUpdate(ctx, tag)
				if err != nil {
					slog.Error("Failed to download update", "error", err)
				} else {
					defer os.Remove(msiPath)
					if err := versioning.RunMsiPassive(ctx, msiPath); err != nil {
						slog.Error("Passive MSI installation failed", "error", err)
					} else {
						slog.Info("Passive MSI installation completed successfully")
					}
				}
			}
		}
	}

	launchMainApp(targetFlag)
}

func readPreferences() map[string]any {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	prefPath := filepath.Join(configDir, "fyne", "com.github.ikafly.au_mod_installer", "preferences.json")
	data, err := os.ReadFile(prefPath)
	if err != nil {
		return nil
	}
	var prefs map[string]any
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil
	}
	return prefs
}

func readUpdateBranchPreference() string {
	prefs := readPreferences()
	if prefs != nil {
		if val, ok := prefs["core.update_branch"].(string); ok && val != "" {
			return val
		}
	}
	return "stable"
}

func shouldPerformUpdate(currentVersion, targetVersion string) bool {
	if targetVersion == "" {
		return false
	}
	if currentVersion == "" || currentVersion == "unknown" || semver.Build(currentVersion) != "" || semver.Prerelease(currentVersion) != "" {
		return true
	}
	return semver.Compare(targetVersion, currentVersion) > 0
}

func readCurrentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return ""
}

func resolveTargetPath(targetFlag string) string {
	execPath, err := os.Executable()
	if err != nil {
		execPath = os.Args[0]
	}
	dir := filepath.Dir(execPath)

	if targetFlag != "" {
		if filepath.IsAbs(targetFlag) {
			return targetFlag
		}
		relPath := filepath.Join(dir, targetFlag)
		if _, err := os.Stat(relPath); err == nil {
			return relPath
		}
		return targetFlag
	}

	mainExe := filepath.Join(dir, "Mod of Us.exe")
	if _, err := os.Stat(mainExe); os.IsNotExist(err) {
		mainExe = filepath.Join(dir, "client.exe")
	}
	return mainExe
}

func buildLaunchArgs(args []string) []string {
	var forwarded []string
	hasInitial := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-target" || arg == "--target" {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-target=") || strings.HasPrefix(arg, "--target=") {
			continue
		}
		if arg == "-initial" || arg == "--initial" {
			hasInitial = true
		}
		forwarded = append(forwarded, arg)
	}
	if !hasInitial {
		forwarded = append(forwarded, "-initial")
	}
	return forwarded
}

func launchMainApp(targetFlag string) {
	targetExe := resolveTargetPath(targetFlag)
	args := buildLaunchArgs(os.Args[1:])

	cmd := exec.Command(targetExe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start target application", "path", targetExe, "error", err)
		os.Exit(1)
	}
	os.Exit(0)
}
