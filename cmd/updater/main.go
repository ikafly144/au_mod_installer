//go:build windows

package main

import (
	"context"
	"encoding/json/v2"
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
	"golang.org/x/sys/windows"

	"github.com/nightlyone/lockfile"

	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/common/versioning"
)

var defaultServer = "https://modofus.sabafly.net/api/v1"

func isMainAppRunning() bool {
	pd, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return false
	}
	lockPath := filepath.Join(pd, "au_mod_installer.lock")
	lock, err := lockfile.New(lockPath)
	if err != nil {
		return false
	}
	proc, err := lock.GetOwner()
	if err == nil && proc != nil {
		return true
	}
	return false
}

func main() {
	var (
		serverFlag  string
		localMode   string
		offlineFlag bool
		silentFlag  bool
		targetFlag  string
		fromTemp    bool
	)
	flag.StringVar(&serverFlag, "server", "", "URL of the mod server")
	flag.StringVar(&localMode, "local", "", "Path to local mods.json file for local mode")
	flag.BoolVar(&offlineFlag, "offline", false, "Run in offline mode")
	flag.BoolVar(&silentFlag, "silent", false, "Start minimized in system tray")
	flag.StringVar(&targetFlag, "target", "", "Path to target executable to launch after update")
	flag.BoolVar(&fromTemp, "from-temp", false, "Internal flag indicating updater is running from temp directory")
	flag.Parse()

	// If the main application is already running, skip update check and directly forward to main app
	if isMainAppRunning() {
		slog.Info("Main application is already running, skipping update check and activating instance")
		launchMainApp(targetFlag)
		return
	}

	if !fromTemp {
		maybeRelaunchFromTemp(targetFlag)
	}

	branchName := readUpdateBranchPreference()
	branch := versioning.BranchFromString(branchName)

	serverURL := serverFlag
	if serverURL == "" {
		serverURL = defaultServer
	}

	currentVersion := readCurrentVersion()

	// If not in offline or local mode, and main app is not running, perform update check without confirmation
	if !offlineFlag && localMode == "" && !isMainAppRunning() {
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
		return currentVersion != targetVersion
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

func maybeRelaunchFromTemp(targetFlag string) {
	execPath, err := os.Executable()
	if err != nil {
		execPath = os.Args[0]
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return
	}

	tempDir := os.TempDir()
	rel, err := filepath.Rel(tempDir, execPath)
	if err == nil && !strings.HasPrefix(rel, "..") {
		// Already executing inside Temp directory
		return
	}

	dir, err := os.MkdirTemp("", "modofus-updater-*")
	if err != nil {
		slog.Warn("Failed to create temp directory for updater, running in place", "error", err)
		return
	}

	tempExe := filepath.Join(dir, "updater.exe")
	input, err := os.ReadFile(execPath)
	if err != nil {
		slog.Warn("Failed to read updater binary, running in place", "error", err)
		return
	}
	if err := os.WriteFile(tempExe, input, 0755); err != nil {
		slog.Warn("Failed to write updater binary to temp, running in place", "error", err)
		return
	}

	absTarget := resolveTargetPath(targetFlag)
	var args []string
	args = append(args, "-from-temp", "-target", absTarget)
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-target" || arg == "--target" {
			if i+1 < len(os.Args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-target=") || strings.HasPrefix(arg, "--target=") {
			continue
		}
		if arg == "-from-temp" || arg == "--from-temp" {
			continue
		}
		args = append(args, arg)
	}

	cmd := exec.Command(tempExe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		slog.Warn("Failed to start temp updater, running in place", "error", err)
		return
	}

	// Terminate original updater so the file lock on INSTALLFOLDER\updater.exe is freed!
	os.Exit(0)
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
		if arg == "-from-temp" || arg == "--from-temp" {
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
