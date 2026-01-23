package modmgr

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

// PrepareProfileDirectory installs mods from cache to the profile directory and generates doorstop_config.ini.
func PrepareProfileDirectory(profileDir string, cacheDir string, modVersions []ModVersion, binaryType aumgr.BinaryType) error {
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Install mods
	for _, mod := range modVersions {
		modCacheDir := filepath.Join(cacheDir, mod.ModID, hashId(mod.ID))
		if err := filepath.WalkDir(modCacheDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(modCacheDir, path)
			if err != nil {
				return err
			}

			destPath := filepath.Join(profileDir, relPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			srcData, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(destPath, srcData, 0644); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return fmt.Errorf("failed to install mod %s: %w", mod.ModID, err)
		}
	}

	// Generate doorstop_config.ini
	doorstopConfig := generateDoorstopConfig(profileDir)
	if err := os.WriteFile(filepath.Join(profileDir, "doorstop_config.ini"), []byte(doorstopConfig), 0644); err != nil {
		return fmt.Errorf("failed to write doorstop_config.ini: %w", err)
	}

	return nil
}

func generateDoorstopConfig(basePath string) string {
	// Paths must be absolute or relative to the executable?
	// With SetDllDirectory, winhttp.dll is loaded from basePath.
	// Doorstop usually resolves relative paths against the game executable.
	// So we should use absolute paths here to be safe, pointing to files inside basePath.
	
	targetAssembly := filepath.Join(basePath, "BepInEx", "core", "BepInEx.Unity.IL2CPP.dll")
	coreClrPath := filepath.Join(basePath, "dotnet", "coreclr.dll")
	corlibDir := filepath.Join(basePath, "dotnet")

	return fmt.Sprintf(`# General options for Unity Doorstop
[General]
enabled = true
target_assembly = %s
redirect_output_log = false
boot_config_override =
ignore_disable_switch = false

[UnityMono]
dll_search_path_override =
debug_enabled = false
debug_start_server = true
debug_address = 127.0.0.1:10000
debug_suspend = false

[Il2Cpp]
coreclr_path = %s
corlib_dir = %s
`, targetAssembly, coreClrPath, corlibDir)
}
