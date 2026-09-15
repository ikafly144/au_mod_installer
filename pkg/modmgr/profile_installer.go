package modmgr

import (
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/ikafly144/modrepo/pkg/progress"
	"github.com/ikafly144/modrepo/pkg/repomgr"
)

type ProfileMetadata struct {
	GameVersion string             `json:"game_version"`
	BinaryType  repomgr.BinaryType `json:"binary_type"`
	ModVersions []ModVersion       `json:"mod_versions"`
	ModFiles    []string           `json:"mod_files,omitempty"`
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
	ma := make(map[string]string)
	for _, v := range a {
		ma[v.ModID] = v.VersionID
	}
	for _, v := range b {
		if id, ok := ma[v.ModID]; !ok || id != v.VersionID {
			return false
		}
	}
	return true
}

// PrepareProfileDirectory installs mods from cache to the profile directory and generates doorstop_config.ini.
// The game directory is NEVER modified.
func PrepareProfileDirectory(profileDir string, gamePath string, cacheDir string, modVersions []ModVersion, binaryType repomgr.BinaryType, gameVersion string, force bool, progressListener progress.Progress) error {
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	meta, err := GetProfileMetadata(profileDir)
	if err != nil {
		return fmt.Errorf("failed to load profile metadata: %w", err)
	}
	profileRoot, err := os.OpenRoot(profileDir)
	if err != nil {
		return fmt.Errorf("failed to open profile directory: %w", err)
	}
	defer profileRoot.Close()

	shouldInstall := force || meta == nil || !modVersionsEqual(meta.ModVersions, modVersions) || meta.GameVersion != gameVersion

	if shouldInstall {
		if meta != nil {
			// Delete existing mod files in profile directory
			files := make([]string, len(meta.ModFiles))
			copy(files, meta.ModFiles)
			slices.SortFunc(files, func(a, b string) int {
				return len(filepath.Dir(b)) - len(filepath.Dir(a))
			})

			for _, path := range files {
				if err := profileRoot.Remove(path); err != nil && !os.IsNotExist(err) {
					slog.Warn("Failed to remove existing mod file", "file", path, "error", err)
				}
				_ = removeEmptyDirs(profileRoot, filepath.Dir(path))
			}
		}

		if progressListener != nil {
			progressListener.SetValue(0)
			progressListener.Start()
			defer progressListener.Done()
		}

		// Clear BepInEx folder if resetting
		bepInExDir := filepath.Join(profileDir, "BepInEx")
		if _, err := os.Stat(bepInExDir); err == nil && force {
			_ = os.RemoveAll(bepInExDir)
		}

		var modPaths []string
		for idx, mod := range modVersions {
			hashStr, err := hashModVersion(mod)
			if err != nil {
				return fmt.Errorf("failed to hash mod version: %w", err)
			}
			modCacheDir := filepath.Join(cacheDir, string(binaryType), mod.ModID, hashStr)

			zipName := mod.ModID + ".zip"
			if len(mod.Files) > 0 && mod.Files[0].Filename != "" {
				zipName = mod.Files[0].Filename
			}
			cachedZipPath := filepath.Join(modCacheDir, zipName)

			zipFile, err := os.Open(cachedZipPath)
			if err != nil {
				return fmt.Errorf("failed to open cached mod zip %s: %w", cachedZipPath, err)
			}
			zipInfo, err := zipFile.Stat()
			if err != nil {
				zipFile.Close()
				return fmt.Errorf("failed to stat cached mod zip %s: %w", cachedZipPath, err)
			}

			extracted, err := extractThunderstoreZip(zipFile, zipInfo.Size(), mod.ModID, profileRoot)
			zipFile.Close()
			if err != nil {
				return fmt.Errorf("failed to extract mod %s: %w", mod.ModID, err)
			}
			modPaths = append(modPaths, extracted...)

			if progressListener != nil {
				progressListener.SetValue(float64(idx+1) / float64(len(modVersions)))
			}
		}

		slices.SortStableFunc(modPaths, func(a, b string) int {
			return len(filepath.Dir(b)) - len(filepath.Dir(a))
		})

		newMeta := &ProfileMetadata{
			ModVersions: modVersions,
			GameVersion: gameVersion,
			BinaryType:  binaryType,
			ModFiles:    modPaths,
		}
		if err := saveProfileMetadata(profileDir, newMeta); err != nil {
			return fmt.Errorf("failed to save profile metadata: %w", err)
		}
	}

	// Always ensure doorstop_config.ini exists in profileDir
	doorstopConfig := GenerateDoorstopConfig(profileDir)
	writePath := filepath.Join(profileDir, "doorstop_config.ini")
	if err := os.WriteFile(writePath, []byte(doorstopConfig), 0644); err != nil {
		return fmt.Errorf("failed to write doorstop_config.ini: %w", err)
	}

	return nil
}

func GenerateDoorstopConfig(basePath string) string {
	targetAssembly := filepath.Join(basePath, "BepInEx", "core", "BepInEx.Preloader.dll")

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
`, targetAssembly)
}
