//go:build windows

package repomgr

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// GetRepoDir attempts to automatically locate the R.E.P.O. installation directory.
func GetRepoDir() (string, error) {
	// 1. Try Windows Uninstall registry key for Steam App 3241660
	if p, err := getFromUninstallRegistry(); err == nil && isRepoDir(p) {
		return p, nil
	}

	// 2. Try Steam library folders
	if p, err := getFromSteamLibraries(); err == nil && isRepoDir(p) {
		return p, nil
	}

	// 3. Try common drive locations
	commonDrives := []string{"C:", "D:", "E:", "F:", "G:"}
	for _, drive := range commonDrives {
		candidates := []string{
			filepath.Join(drive, "Program Files (x86)", "Steam", "steamapps", "common", "REPO"),
			filepath.Join(drive, "Program Files", "Steam", "steamapps", "common", "REPO"),
			filepath.Join(drive, "SteamLibrary", "steamapps", "common", "REPO"),
			filepath.Join(drive, "Steam", "steamapps", "common", "REPO"),
		}
		for _, c := range candidates {
			if isRepoDir(c) {
				return c, nil
			}
		}
	}

	return "", fmt.Errorf("R.E.P.O. installation directory not found")
}

func isRepoDir(path string) bool {
	if path == "" {
		return false
	}
	exe := filepath.Join(path, ExecutableName)
	info, err := os.Stat(exe)
	return err == nil && !info.IsDir()
}

func getFromUninstallRegistry() (string, error) {
	keys := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Steam App ` + SteamAppID,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\Steam App ` + SteamAppID,
	}

	for _, keyPath := range keys {
		for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
			k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			val, _, err := k.GetStringValue("InstallLocation")
			k.Close()
			if err == nil && val != "" {
				return val, nil
			}
		}
	}
	return "", fmt.Errorf("registry key not found")
}

func getFromSteamLibraries() (string, error) {
	steamPath := ""
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, sub := range []string{`Software\Valve\Steam`, `SOFTWARE\WOW6432Node\Valve\Steam`} {
			k, err := registry.OpenKey(root, sub, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			val, _, err := k.GetStringValue("SteamPath")
			k.Close()
			if err == nil && val != "" {
				steamPath = val
				break
			}
		}
		if steamPath != "" {
			break
		}
	}

	if steamPath == "" {
		return "", fmt.Errorf("steam path not found in registry")
	}

	vdfPath := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")
	libraryPaths := parseLibraryFolders(vdfPath)
	libraryPaths = append([]string{steamPath}, libraryPaths...)

	for _, lib := range libraryPaths {
		candidate := filepath.Join(lib, "steamapps", "common", "REPO")
		if isRepoDir(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("R.E.P.O. not found in steam libraries")
}

func parseLibraryFolders(vdfPath string) []string {
	f, err := os.Open(vdfPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var paths []string
	scanner := bufio.NewScanner(f)
	re := regexp.MustCompile(`"path"\s+"([^"]+)"`)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			p := strings.ReplaceAll(matches[1], `\\`, `\`)
			paths = append(paths, p)
		}
	}
	return paths
}
