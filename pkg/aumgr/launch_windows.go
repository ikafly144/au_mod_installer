//go:build windows

package aumgr

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"golang.org/x/sys/windows"
)

var gameLock = sync.Mutex{}

type LaunchContext struct {
	GameExe     string
	ProfilePath string
	DllDir      string
	BepInExDll  string
	DotNetDir   string
	CoreClrPath string
	Platform    LauncherType
}

func LaunchAmongUs(ctx LaunchContext) error {
	slog.Info("Launching Among Us", "launcher", ctx.Platform)
	if i, err := os.Stat(ctx.GameExe); err != nil {
		return fmt.Errorf("game executable not found: %w", err)
	} else if i.IsDir() {
		return fmt.Errorf("game executable path is a directory")
	}

	if err := windows.SetDllDirectory(ctx.DllDir); err != nil {
		return fmt.Errorf("SetDllDirectory failed: %v", err)
	}

	cmd := exec.Command(ctx.GameExe)
	cmd.Args = append(cmd.Args, "--doorstop-enabled", "true")
	cmd.Args = append(cmd.Args, "--doorstop-target-assembly", ctx.BepInExDll)
	cmd.Args = append(cmd.Args, "--doorstop-clr-corlib-dir", ctx.DotNetDir)
	cmd.Args = append(cmd.Args, "--doorstop-clr-runtime-coreclr-path", ctx.CoreClrPath)

	if ctx.Platform == LauncherEpicGames {
		// cmd.Args = append(cmd.Args, fmt.Sprintf("-AUTH_PASSWORD=%s", "TODO:get from EGL"))
	}

	return launch(cmd)
}

func launch(cmd *exec.Cmd) error {
	if !gameLock.TryLock() {
		slog.Warn("Another launch is already in progress")
		return fmt.Errorf("another launch is already in progress")
	}
	defer gameLock.Unlock()

	slog.Info("Starting Among Us")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Among Us: %w", err)
	}

	slog.Info("Among Us started successfully", "pid", cmd.Process.Pid)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Among Us exited with error: %w", err)
	}

	slog.Info("Among Us exited successfully")

	return nil
}

func launchVanilla(gameExe string, platform LauncherType) error {
	cmd := exec.Command(gameExe)
	if platform == LauncherEpicGames {
		// cmd.Args = append(cmd.Args, fmt.Sprintf("-AUTH_PASSWORD=%s", "TODO:get from EGL"))
	}
	return launch(cmd)
}

// const (
// 	EglVersion = "11.0.1-14907503+++Portal+Release-Live"

// 	OAuth2Agent = "EpicGamesLauncher/" + EglVersion

// 	UserAgent = "UELauncher/" + EglVersion + " Windows/10.0.19041.1.256.64bit"

// 	OAuthHost = "account-public-service-prod03.ol.epicgames.com"
// 	UserBasic = "34a02cf8f4414e29b15921876da36f9a"
// 	PwBasic   = "daafbccc737745039dffe53d94fc76cf"
// 	Label     = "Live-EternalKnight"

// 	ExchangeEndpoint = "https://" + OAuthHost + "/account/api/oauth/exchange"
// 	TokenEndpoint    = "https://" + OAuthHost + "/account/api/oauth/token"
// )
