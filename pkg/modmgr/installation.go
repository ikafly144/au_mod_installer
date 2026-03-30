package modmgr

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

const currentFileVersion = 2

// Deprecated: Should use profile.Profile to track installed mods instead.
type ModInstallation struct {
	FileVersion          int                    `json:"file_version"`
	InstalledMods        []InstalledVersionInfo `json:"installed_mods"`
	InstalledGameVersion string                 `json:"installed_game_version"`
	Status               InstallStatus          `json:"status"`
	raw                  json.RawMessage        `json:"-"`
}

type RestoreInfo struct {
	BackupDir string            `json:"backup_dir"`
	Added     []string          `json:"added"`
	Moved     map[string]string `json:"moved"` // Original Path -> Backup Path
}

func (mi *ModInstallation) UnmarshalJSON(data []byte) error {
	type Alias ModInstallation
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(mi),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	mi.raw = json.RawMessage(data)
	return nil
}

func (mi *ModInstallation) MarshalJSON() ([]byte, error) {
	type Alias ModInstallation
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(mi),
	})
}

// For file versions 0 and 1, return the list of vanilla files stored in the installation data.
func (mi *ModInstallation) OldVanillaFiles() []string {
	if mi.FileVersion == 0 || mi.FileVersion == 1 {
		type OldModInstallationV0 struct {
			VanillaFiles []string `json:"vanilla_files"`
		}
		var oldInst OldModInstallationV0
		if err := json.Unmarshal(mi.raw, &oldInst); err != nil {
			slog.Warn("Failed to unmarshal old installation data", "error", err)
			return nil
		}
		return oldInst.VanillaFiles
	}
	return nil
}

type InstalledVersionInfo struct {
	ModVersion `json:",inline"`
	Paths      []string `json:"paths"`
}

type InstallStatus string

const (
	InstallStatusCompatible   InstallStatus = "compatible"
	InstallStatusIncompatible InstallStatus = "incompatible"
	InstallStatusBroken       InstallStatus = "broken"
	InstallStatusUnknown      InstallStatus = "unknown"
)

const InstallationInfoFileName = ".mod_installation"

// Deprecated: Should use profile.Profile to track installed mods instead.
func LoadInstallationInfo(modInstallLocation *os.Root) (*ModInstallation, error) {
	file, err := modInstallLocation.OpenFile(InstallationInfoFileName, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var installation ModInstallation
	if err := decoder.Decode(&installation); err != nil {
		return nil, err
	}
	if installation.FileVersion > currentFileVersion {
		return nil, fmt.Errorf("unsupported installation file version: %d", installation.FileVersion)
	}
	return &installation, nil
}

// Deprecated: Should use profile.Profile to track installed mods instead.
func SaveInstallationInfo(gameRoot *os.Root, installation *ModInstallation) error {
	slog.Info("Saving installation info", "installation", installation)
	if installation == nil {
		return fmt.Errorf("installation is nil")
	}
	file, err := gameRoot.OpenFile(InstallationInfoFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(installation); err != nil {
		return err
	}
	return nil
}

type CacheMetadata struct {
	ModVersion ModVersion `json:"mod_version"`
}

func DownloadMods(cacheDir string, modVersions []ModVersion, binaryType aumgr.BinaryType, progressListener progress.Progress, force bool) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	totalDownloadCount := func() int {
		count := 0
		for i := range modVersions {
			count += modVersions[i].CompatibleFilesCount(binaryType)
		}
		return count
	}()
	if totalDownloadCount == 0 {
		return fmt.Errorf("no compatible files to download for the selected mods and binary type")
	}

	if progressListener != nil {
		progressListener.SetValue(0.0)
		progressListener.Start()
		defer progressListener.Done()
	}

	hClient := http.DefaultClient
	for i := range modVersions {
		hashStr, err := hashModVersion(modVersions[i])
		if err != nil {
			return fmt.Errorf("failed to hash mod version: %w", err)
		}
		modCacheDir := filepath.Join(cacheDir, string(binaryType), modVersions[i].ModID, hashStr)
		if _, err := os.Stat(modCacheDir); err == nil {
			if !force {
				// Load metadata and check if it matches the mod version
				metaFile, err := os.Open(filepath.Join(modCacheDir, "metadata.json"))
				if err != nil {
					slog.Warn("Failed to open mod cache metadata, will re-download", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID, "error", err)
					goto download
				}
				var metadata CacheMetadata
				if err := json.NewDecoder(metaFile).Decode(&metadata); err != nil {
					slog.Warn("Failed to decode mod cache metadata, will re-download", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID, "error", err)
					goto download
				} else if metadata.ModVersion.ID != modVersions[i].ID {
					slog.Warn("Mod cache metadata version mismatch, will re-download", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID, "cachedVersionId", metadata.ModVersion.ID)
					goto download
				}

				// Check if all files exist in cache
				allFilesExist := true
				for file := range modVersions[i].Downloads(binaryType) {
					cachedFilePath := filepath.Join(modCacheDir, file.ExtractPath)
					if _, err := os.Stat(cachedFilePath); os.IsNotExist(err) {
						allFilesExist = false
						slog.Info("Cached mod file not found, need to re-download", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID, "file", file.ExtractPath)
						goto download
					}
				}
				// If all files exist, Check the hash of one file
				if allFilesExist {
					for file := range modVersions[i].Downloads(binaryType) {
						cachedFilePath := filepath.Join(modCacheDir, file.ExtractPath)
						hashChecker := newHashWriters(file.Hashes)
						hashFile, err := os.Open(cachedFilePath)
						if err != nil {
							slog.Error("Failed to open cached mod file for hashing", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID, "file", file.ExtractPath, "error", err)
							goto download
						}
						if _, err := io.Copy(io.Discard, io.TeeReader(hashFile, hashChecker)); err != nil {
							slog.Error("Failed to hash cached mod file", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID, "file", file.ExtractPath, "error", err)
							hashFile.Close()
							goto download
						}
						hashFile.Close()
					}

					slog.Info("Mod already cached", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
					if progressListener != nil {
						progressListener.SetValue(progressListener.GetValue() + (float64(modVersions[i].CompatibleFilesCount(binaryType)) / float64(totalDownloadCount)))
					}
					continue
				}
			} else {
				slog.Info("Force re-downloading mod, clearing cache", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
				if err := os.RemoveAll(modCacheDir); err != nil {
					return fmt.Errorf("failed to clear mod cache: %w", err)
				}
			}
		}
	download:

		if err := os.MkdirAll(modCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create mod cache directory: %w", err)
		}

		modCacheRoot, err := os.OpenRoot(modCacheDir)
		if err != nil {
			return fmt.Errorf("failed to open mod cache root: %w", err)
		}
		defer modCacheRoot.Close()

		slog.Info("Downloading mod", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
		for file := range modVersions[i].Downloads(binaryType) {
			var response *http.Response
			for _, uri := range file.Downloads {
				req, err := http.NewRequest(http.MethodGet, uri, nil)
				if err != nil {
					slog.Error("Failed to create HTTP request for mod file", "url", uri, "error", err)
					continue
				}
				resp, err := hClient.Do(req)
				if err != nil {
					slog.Error("Failed to download mod file", "url", uri, "error", err)
					continue
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					slog.Error("Failed to download mod file, non-OK status", "url", uri, "status", resp.Status)
					continue
				}
				response = resp
				break
			}
			if response == nil {
				return fmt.Errorf("failed to download mod file from all sources: %s@%s (%s)", modVersions[i].ModID, modVersions[i].ID, file.ID)
			}
			contentLength := response.ContentLength
			slog.Info("Downloading mod file", "url", response.Request.URL, "contentLength", contentLength)

			hashChecker := newHashWriters(file.Hashes)

			body := io.TeeReader(response.Body, hashChecker)
			var extractPath string

			switch file.ContentType {
			case model.ContentTypeArchive:
				fallthrough
			case model.ContentTypeBinary, model.ContentTypePluginDll:
				extractPath = file.ExtractPath
				filename := file.Filename
				if filepath.Dir(extractPath) == extractPath || extractPath == "" {
					// Invalid or missing path, use filename from header if available
					if filename != "" {
						extractPath = filepath.Join(filepath.Base(extractPath), filename)
					} else {
						return fmt.Errorf("file path is empty")
					}
				}
				if file.ContentType == model.ContentTypePluginDll {
					if filename == "" {
						return fmt.Errorf("failed to determine plugin filename for URL: %v", file.Downloads)
					}
					// For plugin files, use a fixed naming scheme
					extractPath = filepath.Join("BepInEx", "plugins", filename)
				}
				if extractPath == "" {
					return fmt.Errorf("file path is empty")
				}
				_ = modCacheRoot.MkdirAll(filepath.Dir(extractPath), 0755)
				slog.Info("Saving mod file to cache", "path", extractPath)
				destFile, err := modCacheRoot.OpenFile(extractPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return err
				}
				defer destFile.Close()
				startVal := 0.0
				if progressListener != nil {
					startVal = progressListener.GetValue()
				}
				buf := progress.NewProgressWriter(startVal, (1.0 / float64(totalDownloadCount)), contentLength, progressListener, destFile)
				if _, err := io.Copy(buf, body); err != nil {
					return err
				}
				buf.Complete()
			default:
				return fmt.Errorf("unknown file type: %s", file.ContentType)
			}

			if computedHash, err := hashChecker.Sum(); err != nil {
				if extractPath != "" {
					slog.Warn("File hash mismatch for extracted file, deleting cached file", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID, "file", extractPath, "error", err)
					if err := modCacheRoot.RemoveAll(extractPath); err != nil {
						slog.Warn("Failed to remove cached file after hash mismatch", "file", extractPath, "error", err)
					}
				}
				return fmt.Errorf("downloaded file hash mismatch: %w", err)
			} else {
				slog.Info("File hash verified", "hash", computedHash)
			}
		}

		// Write metadata.json to cache directory
		metadata := CacheMetadata{
			ModVersion: modVersions[i],
		}
		metaFile, err := modCacheRoot.OpenFile("metadata.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer metaFile.Close()
		if err := json.NewEncoder(metaFile).Encode(metadata); err != nil {
			return err
		}
	}
	return nil
}

// For legacy support
func UninstallMod(modInstallLocation *os.Root, progress progress.Progress, remainMods []ModVersion) error {
	if _, err := uninstallMod(modInstallLocation, progress, remainMods); err != nil {
		return fmt.Errorf("failed to uninstall mod: %w", err)
	}
	if err := modInstallLocation.Remove(InstallationInfoFileName); err != nil {
		return err
	}
	return nil
}

func UninstallRemainingMods(modInstallLocation *os.Root, progress progress.Progress, remainMods []ModVersion) ([]InstalledVersionInfo, error) {
	remainModInfos, err := uninstallMod(modInstallLocation, progress, remainMods)
	if err != nil {
		return nil, fmt.Errorf("failed to uninstall remaining mods: %w", err)
	}
	return remainModInfos, nil
}

func uninstallMod(modInstallLocation *os.Root, progress progress.Progress, remainMods []ModVersion) ([]InstalledVersionInfo, error) {
	if progress != nil {
		progress.SetValue(0.0)
		progress.Start()
		defer progress.Done()
	}
	installation, err := LoadInstallationInfo(modInstallLocation)
	if err != nil {
		return nil, err
	}

	dirInfo, err := modInstallLocation.Open(".")
	if err != nil {
		return nil, err
	}
	defer dirInfo.Close()
	fileCount, err := dirInfo.Readdirnames(-1)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var remainModInfos []InstalledVersionInfo
	switch installation.FileVersion {
	case 0, 1:
		i := 0
		if err := fs.WalkDir(modInstallLocation.FS(), ".", func(path string, info fs.DirEntry, err error) error {
			i++
			if progress != nil {
				progress.SetValue(float64(i) / float64(len(fileCount)))
			}
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				slog.Warn("Failed to access file during uninstallation", "file", path, "error", err)
				return err
			}
			if slices.Contains(installation.OldVanillaFiles(), path) {
				return nil
			}
			if strings.HasPrefix(path, "Among Us_Data") {
				return nil
			}
			if filepath.Ext(path) == InstallationInfoFileName {
				return nil
			}
			if err := modInstallLocation.RemoveAll(path); err != nil {
				slog.Warn("Failed to delete file during uninstallation", "file", path, "error", err)
				return nil
			}
			return nil
		}); err != nil {
			return nil, err
		}
	case 2:
		var paths []string
		if installation.Status == InstallStatusCompatible {
			for _, version := range installation.InstalledMods {
				if remainMods != nil {
					shouldRemain := false
					for _, remainVersion := range remainMods {
						if version.ModID == remainVersion.ModID && version.ID == remainVersion.ID {
							shouldRemain = true
							break
						}
					}
					if shouldRemain {
						slog.Info("Keeping mod during uninstallation", "modId", version.ModID, "versionId", version.ID)
						remainModInfos = append(remainModInfos, version)
						continue
					}
				}
				paths = append(paths, version.Paths...)
			}
		} else {
			for _, mod := range installation.InstalledMods {
				paths = append(paths, mod.Paths...)
			}
		}
		sort.SliceStable(paths, func(i, j int) bool {
			return len(paths[i]) > len(paths[j])
		})
		for _, path := range paths {
			slog.Info("Removing mod file during uninstallation", "file", path)
			if err := modInstallLocation.RemoveAll(path); err != nil {
				slog.Warn("Failed to remove mod file during uninstallation", "file", path, "error", err)
			}
			if err := removeEmptyDirs(modInstallLocation, filepath.Dir(path)); err != nil {
				slog.Warn("Failed to remove empty directory during uninstallation", "dir", filepath.Dir(path), "error", err)
			}
		}
	}
	return remainModInfos, nil
}

func removeEmptyDirs(root *os.Root, dir string) error {
	dirInfo, err := root.Stat(dir)
	if err != nil {
		slog.Warn("Failed to stat directory during cleanup", "dir", dir, "error", err)
		return err
	}
	if !dirInfo.IsDir() {
		return nil
	}
	d, err := root.Open(dir)
	if err != nil {
		slog.Warn("Failed to open directory during cleanup", "dir", dir, "error", err)
		return err
	}
	defer d.Close()
	entries, err := d.Readdirnames(-1)
	if err != nil {
		slog.Warn("Failed to read directory entries during cleanup", "dir", dir, "error", err)
		return err
	}
	if len(entries) == 0 {
		if err := root.Remove(dir); err != nil {
			slog.Warn("Failed to remove empty directory during cleanup", "dir", dir, "error", err)
		}
		return removeEmptyDirs(root, filepath.Dir(dir))
	}
	return nil
}

func extractZip(reader io.ReaderAt, contentLength int64, destRoot *os.Root, progressListener progress.Progress, n int) ([]string, error) {
	startVal := 0.0
	if progressListener != nil {
		startVal = progressListener.GetValue()
	}
	perFileScale := 1.0 / float64(n)
	downloadScale := perFileScale * 0.75
	extractScale := perFileScale - downloadScale
	zipReader, err := zip.NewReader(reader, contentLength)
	if err != nil {
		return nil, err
	}
	var files []*zip.File
	var totalExtractBytes uint64
	var extractFiles []string
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		files = append(files, f)
		extractFiles = append(extractFiles, filepath.Clean(f.Name))
		totalExtractBytes += f.UncompressedSize64
	}
	if len(files) == 0 {
		if progressListener != nil {
			progressListener.SetValue(startVal + perFileScale)
		}
		return extractFiles, nil
	}
	extractStart := startVal + downloadScale
	writtenExtractBytes := uint64(0)
	var extractErr error
	for _, f := range files {
		fileScale := 0.0
		if totalExtractBytes > 0 {
			fileScale = (float64(f.UncompressedSize64) / float64(totalExtractBytes)) * extractScale
		}
		fileStart := extractStart
		if totalExtractBytes > 0 {
			fileStart += (float64(writtenExtractBytes) / float64(totalExtractBytes)) * extractScale
		}
		pw := progress.NewProgressWriter(fileStart, fileScale, int64(f.UncompressedSize64), progressListener, nil)
		if err := extractFile(f, destRoot, pw); err != nil {
			slog.Warn("Failed to extract file from zip", "file", f.Name, "error", err)
			extractErr = err
			break
		}
		pw.Complete()
		writtenExtractBytes += f.UncompressedSize64
	}
	if extractErr == nil && progressListener != nil {
		progressListener.SetValue(startVal + perFileScale)
	}
	return extractFiles, extractErr
}

func extractFile(f *zip.File, destRoot *os.Root, progressWriter *progress.ProgressWriter) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if filepath.Dir(f.Name) == f.Name {
		slog.Warn("Skipping file with invalid path", "file", f.Name)
		return nil
	}

	_ = destRoot.MkdirAll(filepath.Dir(f.Name), 0755)
	destFile, err := destRoot.OpenFile(f.Name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	writer := io.Writer(destFile)
	if progressWriter != nil {
		progressWriter.SetWriter(destFile)
		writer = progressWriter
	}
	if _, err := io.Copy(writer, rc); err != nil {
		return err
	}
	return nil
}
