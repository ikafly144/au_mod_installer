//go:build windows

package aumgr

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string) error {
	switch launcherType {
	case LauncherSteam:
		return launchSteam(amongUsDir, dllDir)
	case LauncherEpicGames:
		return launchEpicGames(amongUsDir, dllDir)
	default:
		return launchDefault(amongUsDir, dllDir)
	}
}

func launchDefault(amongUsDir string, dllDir string, args ...string) error {
	cmd := exec.Command(filepath.Join(amongUsDir, "Among Us.exe"))
	// if dllDir != "" {
	// 	if err := windows.SetDllDirectory(dllDir); err != nil {
	// 		return fmt.Errorf("SetDllDirectory failed: %v", err)
	// 	}
	// }
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Among Us: %w", err)
	}

	return nil
}

func launchSteam(amongUsDir string, dllDir string) error {
	return launchFromUrl("steam://rungameid/945360")
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

func launchEpicGames(amongUsDir string, dllDir string) error {
	return launchFromUrl("com.epicgames.launcher://apps/" + EpicNamespace + "%3A" + EpicCatalogId + "%3A" + EpicArtifactId + "?action=launch&silent=true")
}

func launchFromUrl(url string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Among Us from url: %s error: %w", url, err)
	}
	for range 100 {
		time.Sleep(500 * time.Millisecond)
		pList, err := getProcesses()
		if err != nil {
			slog.Error("Failed to get process list", "error", err)
			continue
		}
		p := findProcessByName(pList, "Among Us.exe")
		if p == nil {
			continue
		}

		proc, err := os.FindProcess(p.ProcessID)
		if err != nil {
			slog.Error("Failed to find Among Us process", "error", err)
			continue
		}
		slog.Info("Among Us launched successfully", "pid", p.ProcessID)
		if _, err := proc.Wait(); err != nil {
			slog.Error("Failed to wait for Among Us process", "error", err)
		}

		return nil
	}

	return fmt.Errorf("timeout waiting for Among Us to launch")
}
