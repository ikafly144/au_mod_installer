//go:build windows

package aumgr

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string, args ...string) error {
	switch launcherType {
	case LauncherEpicGames:
		return launchEpicGames(amongUsDir, dllDir, args...)
	default:
		return launchDefault(amongUsDir, dllDir, args...)
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

func launchEpicGames(amongUsDir string, dllDir string, args ...string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "com.epicgames.launcher://apps/"+epicNamespace+"%3A"+epicCatalogId+"%3A"+epicArtifactId+"?action=launch&silent=true")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Epic Games Launcher: %w", err)
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

const TH32CS_SNAPPROCESS = 0x00000002

type WindowsProcess struct {
	ProcessID       int
	ParentProcessID int
	Exe             string
}

func getProcesses() ([]WindowsProcess, error) {
	handle, err := windows.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := windows.CloseHandle(handle); err != nil {
			slog.Error("Failed to close process snapshot handle", "error", err)
		}
	}()

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = windows.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]WindowsProcess, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		err = windows.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

func findProcessByName(processes []WindowsProcess, name string) *WindowsProcess {
	for _, p := range processes {
		if strings.EqualFold(p.Exe, name) {
			return &p
		}
	}
	return nil
}

func newWindowsProcess(e *windows.ProcessEntry32) WindowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return WindowsProcess{
		ProcessID:       int(e.ProcessID),
		ParentProcessID: int(e.ParentProcessID),
		Exe:             syscall.UTF16ToString(e.ExeFile[:end]),
	}
}
