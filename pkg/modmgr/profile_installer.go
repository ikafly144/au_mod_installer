package modmgr

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

type ProfileMetadata struct {
	ModVersions []ModVersion     `json:"mod_versions"`
	GameVersion string           `json:"game_version"`
	BinaryType  aumgr.BinaryType `json:"binary_type"`
}

func getProfileMetadataPath(profileDir string) string {
	return filepath.Join(profileDir, "profile_meta.json")
}

func GetProfileMetadata(profileDir string) (*ProfileMetadata, error) {
	path := getProfileMetadataPath(profileDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var meta ProfileMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func saveProfileMetadata(profileDir string, meta *ProfileMetadata) error {
	path := getProfileMetadataPath(profileDir)
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func modVersionsEqual(a, b []ModVersion) bool {
	if len(a) != len(b) {
		return false
	}
	// Sort or map? They might be in different order.
	ma := make(map[string]string)
	for _, v := range a {
		ma[v.ModID] = v.ID
	}
	for _, v := range b {
		if id, ok := ma[v.ModID]; !ok || id != v.ID {
			return false
		}
	}
	return true
}

// PrepareProfileDirectory installs mods from cache to the profile directory and generates doorstop_config.ini.
func PrepareProfileDirectory(profileDir string, cacheDir string, modVersions []ModVersion, binaryType aumgr.BinaryType, gameVersion string, force bool, progressListener progress.Progress) error {
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	meta, err := GetProfileMetadata(profileDir)
	if err != nil {
		return fmt.Errorf("failed to load profile metadata: %w", err)
	}

	shouldInstall := force || meta == nil || !modVersionsEqual(meta.ModVersions, modVersions) || meta.GameVersion != gameVersion || meta.BinaryType != binaryType

	if shouldInstall {
		if progressListener != nil {
			progressListener.SetValue(0)
			progressListener.Start()
			defer progressListener.Done()
		}
		// Clear BepInEx folder
		bepInExDir := filepath.Join(profileDir, "BepInEx")
		if _, err := os.Stat(bepInExDir); err == nil {
			if err := os.RemoveAll(bepInExDir); err != nil {
				return fmt.Errorf("failed to clear BepInEx directory: %w", err)
			}
		}
		// Also clear dotnet folder if it exists (for IL2CPP)
		dotnetDir := filepath.Join(profileDir, "dotnet")
		if _, err := os.Stat(dotnetDir); err == nil {
			if err := os.RemoveAll(dotnetDir); err != nil {
				return fmt.Errorf("failed to clear dotnet directory: %w", err)
			}
		}

		// Install mods
		copyFilesCount := 0
		for _, mod := range modVersions {
			hashStr, err := hashModVersion(mod)
			if err != nil {
				return fmt.Errorf("failed to hash mod version: %w", err)
			}
			modCacheDir := filepath.Join(cacheDir, string(binaryType), mod.ModID, hashStr)
			if err := filepath.WalkDir(modCacheDir, func(_ string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				copyFilesCount++
				return nil
			}); err != nil {
				return fmt.Errorf("failed to enumerate mod files for %s: %w", mod.ModID, err)
			}
		}
		completedCopies := 0
		for _, mod := range modVersions {
			hashStr, err := hashModVersion(mod)
			if err != nil {
				return fmt.Errorf("failed to hash mod version: %w", err)
			}
			modCacheDir := filepath.Join(cacheDir, string(binaryType), mod.ModID, hashStr)
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
				srcFile, err := os.Open(path)
				if err != nil {
					return err
				}
				srcInfo, err := srcFile.Stat()
				if err != nil {
					_ = srcFile.Close()
					return err
				}
				destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					_ = srcFile.Close()
					return err
				}
				writer := io.Writer(destFile)
				if progressListener != nil && copyFilesCount > 0 {
					scale := 1.0 / float64(copyFilesCount)
					start := float64(completedCopies) * scale
					pw := progress.NewProgressWriter(start, scale, srcInfo.Size(), progressListener, destFile)
					writer = pw
					if _, err := io.Copy(writer, srcFile); err != nil {
						_ = destFile.Close()
						_ = srcFile.Close()
						return err
					}
					pw.Complete()
				} else {
					if _, err := io.Copy(writer, srcFile); err != nil {
						_ = destFile.Close()
						_ = srcFile.Close()
						return err
					}
				}
				if err := destFile.Close(); err != nil {
					_ = srcFile.Close()
					return err
				}
				if err := srcFile.Close(); err != nil {
					return err
				}
				completedCopies++

				return nil
			}); err != nil {
				return fmt.Errorf("failed to install mod %s: %w", mod.ModID, err)
			}
		}

		// Save metadata
		newMeta := &ProfileMetadata{
			ModVersions: modVersions,
			GameVersion: gameVersion,
			BinaryType:  binaryType,
		}
		if err := saveProfileMetadata(profileDir, newMeta); err != nil {
			return fmt.Errorf("failed to save profile metadata: %w", err)
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
