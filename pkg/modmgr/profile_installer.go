package modmgr

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

type ProfileMetadata struct {
	GameVersion string           `json:"game_version"`
	BinaryType  aumgr.BinaryType `json:"binary_type"`
	ModVersions []ModVersion     `json:"mod_versions"`
	ModFiles    []string         `json:"mod_files,omitempty"`
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
	profileRoot, err := os.OpenRoot(profileDir)
	if err != nil {
		return fmt.Errorf("failed to open profile directory: %w", err)
	}
	defer profileRoot.Close()
	cacheRoot, err := os.OpenRoot(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to open cache directory: %w", err)
	}
	defer cacheRoot.Close()

	shouldInstall := force || meta == nil || !modVersionsEqual(meta.ModVersions, modVersions) || meta.GameVersion != gameVersion || meta.BinaryType != binaryType

	if shouldInstall {
		if meta != nil {
			// Delete existing mod files in profile directory
			var files []string
			copy(files, meta.ModFiles)
			// sort files by length desc to delete nested files before their parents
			slices.SortFunc(files, func(a, b string) int {
				return len(filepath.Dir(b)) - len(filepath.Dir(a))
			})

			for _, path := range meta.ModFiles {
				if err := profileRoot.Remove(path); err != nil && !os.IsNotExist(err) {
					slog.Warn("Failed to remove existing mod file, will attempt to overwrite", "file", path, "error", err)
				}
				if err := removeEmptyDirs(profileRoot, filepath.Dir(path)); err != nil {
					slog.Warn("Failed to remove empty directories after deleting mod file", "file", path, "error", err)
				}
			}
		}

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

		var totalFiles int
		for _, mod := range modVersions {
			totalFiles += mod.CompatibleFilesCount(binaryType)
		}

		completedCopies := 0
		var modPaths []string
		for _, mod := range modVersions {
			hashStr, err := hashModVersion(mod)
			if err != nil {
				return fmt.Errorf("failed to hash mod version: %w", err)
			}
			modCacheDir := filepath.Join(string(binaryType), mod.ModID, hashStr)
			cacheRoot, err := cacheRoot.OpenRoot(modCacheDir)
			if err != nil {
				return fmt.Errorf("failed to open mod cache directory for %s: %w", mod.ModID, err)
			}

			var metadata CacheMetadata
			if metaFile, err := cacheRoot.Open("metadata.json"); err != nil {
				slog.Warn("Failed to open mod cache metadata, will re-download", "modId", mod.ModID, "versionId", mod.ID, "error", err)
				return fmt.Errorf("mod cache metadata not found for %s: %w", mod.ModID, err)
			} else if err := json.NewDecoder(metaFile).Decode(&metadata); err != nil {
				_ = metaFile.Close()
				slog.Warn("Failed to decode mod cache metadata, will re-download", "modId", mod.ModID, "versionId", mod.ID, "error", err)
				return fmt.Errorf("failed to decode mod cache metadata for %s: %w", mod.ModID, err)
			} else if metadata.ModVersion.ID != mod.ID {
				_ = metaFile.Close()
				slog.Warn("Mod cache metadata version mismatch, will re-download", "modId", mod.ModID, "versionId", mod.ID, "cachedVersionId", metadata.ModVersion.ID)
				return fmt.Errorf("mod cache metadata version mismatch for %s: cached %s but expected %s", mod.ModID, metadata.ModVersion.ID, mod.ID)
			} else {
				_ = metaFile.Close()
			}

			for _, file := range mod.Files {
				if !binaryType.IsCompatibleWith(file.TargetPlatform) {
					slog.Info("Skipping incompatible file in cache", "modId", mod.ModID, "versionId", mod.ID, "file", file, "binaryType", binaryType)
					continue
				}

				path := file.ExtractPath
				filename := filepath.Base(file.Filename)
				if path == "" && file.ContentType == model.ContentTypePluginDll {
					path = filepath.Join("BepInEx", "plugins", filepath.Base(file.Filename))
				}
				if path == "" {
					path = filename
				}
				if filepath.Base(path) != filename {
					path = filepath.Join(filepath.Dir(path), filename)
				}
				if path == "" {
					slog.Warn("File has no valid path, skipping", "modId", mod.ModID, "versionId", mod.ID, "file", file)
					return fmt.Errorf("file has no valid path for mod %s version %s: %s", mod.ModID, mod.ID, file.Filename)
				}
				srcFile, err := cacheRoot.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open cached file for %s: %w", path, err)
				}
				srcInfo, err := srcFile.Stat()
				if err != nil {
					_ = srcFile.Close()
					return fmt.Errorf("failed to stat cached file for %s: %w", path, err)
				}
				if err := profileRoot.MkdirAll(filepath.Dir(path), 0755); err != nil {
					_ = srcFile.Close()
					return fmt.Errorf("failed to create directories for %s: %w", path, err)
				}

				if file.ContentType == model.ContentTypeArchive {
					// Check zip hash
					newHashChecker := newHashWriters(file.Hashes)
					if _, err := io.Copy(io.Discard, io.TeeReader(srcFile, newHashChecker)); err != nil {
						_ = srcFile.Close()
						return fmt.Errorf("failed to read zip file for hashing: %w", err)
					}
					computedHash, err := newHashChecker.Sum()
					if err != nil {
						_ = srcFile.Close()
						return fmt.Errorf("failed to compute hash for zip file: %w", err)
					}
					for hashType, hashStr := range file.Hashes {
						if computedHash[hashType] != hashStr {
							slog.Warn("Zip file hash mismatch for cached file", "modId", mod.ModID, "versionId", mod.ID, "file", path, "hashType", hashType, "expectedHash", hashStr, "computedHash", computedHash[hashType])
							return fmt.Errorf("zip file hash mismatch for %s: expected %s but got %s", path, hashStr, computedHash[hashType])
						}
						slog.Info("Zip file hash verified for cached file", "modId", mod.ModID, "versionId", mod.ID, "file", path, "hashType", hashType, "hash", hashStr)
					}

					_, _ = srcFile.Seek(0, io.SeekStart)

					destRoot, err := profileRoot.OpenRoot(filepath.Dir(path))
					if err != nil {
						_ = srcFile.Close()
						return fmt.Errorf("failed to open destination directory for %s: %w", path, err)
					}

					zipPaths, err := extractZip(srcFile, srcInfo.Size(), destRoot, progressListener, totalFiles)
					if err != nil {
						_ = srcFile.Close()
						return fmt.Errorf("failed to extract zip file: %w", err)
					}
					for i, zipPath := range zipPaths {
						zipPaths[i] = filepath.Clean(filepath.Join(filepath.Dir(path), zipPath))
					}
					modPaths = append(modPaths, zipPaths...)
					completedCopies++
					continue
				}

				destFile, err := profileRoot.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					_ = srcFile.Close()
					return fmt.Errorf("failed to create destination file for %s: %w", path, err)
				}

				hashChecker := newHashWriters(file.Hashes)

				writer := io.MultiWriter(destFile, hashChecker)
				if progressListener != nil && totalFiles > 0 {
					scale := 1.0 / float64(totalFiles)
					start := float64(completedCopies) * scale
					pw := progress.NewProgressWriter(start, scale, srcInfo.Size(), progressListener, writer)
					writer = pw
					if _, err := io.Copy(writer, srcFile); err != nil {
						_ = destFile.Close()
						_ = srcFile.Close()
						return fmt.Errorf("failed to copy file: %w", err)
					}
					pw.Complete()
				} else {
					if _, err := io.Copy(writer, srcFile); err != nil {
						_ = destFile.Close()
						_ = srcFile.Close()
						return fmt.Errorf("failed to copy file: %w", err)
					}
				}
				if err := destFile.Close(); err != nil {
					_ = srcFile.Close()
					return fmt.Errorf("failed to close destination file: %w", err)
				}
				if err := srcFile.Close(); err != nil {
					return fmt.Errorf("failed to close source file: %w", err)
				}
				computedHash, err := hashChecker.Sum()
				if err != nil {
					return fmt.Errorf("failed to compute hash for %s: %w", path, err)
				}
				for hashType, hashStr := range file.Hashes {
					if computedHash[hashType] != hashStr {
						slog.Warn("File hash mismatch for copied file, deleting profile file", "modId", mod.ModID, "versionId", mod.ID, "file", path, "hashType", hashType, "expectedHash", hashStr, "computedHash", computedHash[hashType])
						return fmt.Errorf("file hash mismatch for %s: expected %s but got %s", path, hashStr, computedHash[hashType])
					}
					slog.Info("File hash verified for copied file", "modId", mod.ModID, "versionId", mod.ID, "file", path, "hashType", hashType, "hash", hashStr)
				}
				modPaths = append(modPaths, filepath.Clean(path))
				completedCopies++
			}
		}

		// sort modPaths for consistent metadata (not strictly necessary but cleaner)
		slices.SortStableFunc(modPaths, func(a, b string) int {
			return len(filepath.Dir(b)) - len(filepath.Dir(a))
		})

		// Save metadata
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
