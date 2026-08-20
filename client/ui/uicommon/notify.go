package uicommon

import (
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"git.sr.ht/~jackmordaunt/go-toast"
)

const AppName = "Mod of Us"

//go:embed icon.png
var appIconBytes []byte

var (
	iconPathOnce   sync.Once
	cachedIconPath string
)

func getAppIconPath() string {
	iconPathOnce.Do(func() {
		if len(appIconBytes) == 0 {
			return
		}
		tmpDir := os.TempDir()
		iconFile := filepath.Join(tmpDir, "mod_of_us_icon.png")
		if err := os.WriteFile(iconFile, appIconBytes, 0644); err != nil {
			slog.Debug("Failed to write temporary icon for notifications", "error", err)
			return
		}
		cachedIconPath = iconFile

		exePath, err := os.Executable()
		if err != nil {
			exePath = ""
		}
		if err := toast.SetAppData(toast.AppData{
			AppID:         AppName,
			IconPath:      cachedIconPath,
			ActivationExe: exePath,
		}); err != nil {
			slog.Debug("Failed to set app data for toast notifications", "error", err)
		}
	})
	return cachedIconPath
}

// Notify sends a standard desktop notification with the application name and icon.
func Notify(title, message string) {
	go func() {
		icon := getAppIconPath()
		exePath, _ := os.Executable()
		n := toast.Notification{
			AppID:         AppName,
			Title:         title,
			Body:          message,
			Icon:          icon,
			Audio:         toast.Default,
			Duration:      toast.Short,
			ActivationExe: exePath,
		}
		if err := n.Push(); err != nil {
			slog.Debug("Failed to push toast notification", "error", err)
		}
	}()
}

// Alert sends an alert desktop notification with the application name and icon.
func Alert(title, message string) {
	go func() {
		icon := getAppIconPath()
		exePath, _ := os.Executable()
		n := toast.Notification{
			AppID:         AppName,
			Title:         title,
			Body:          message,
			Icon:          icon,
			Audio:         toast.Reminder,
			Duration:      toast.Short,
			ActivationExe: exePath,
		}
		if err := n.Push(); err != nil {
			slog.Debug("Failed to push alert toast notification", "error", err)
		}
	}()
}
