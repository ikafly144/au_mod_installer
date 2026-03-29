package modmgr

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/ghactions"
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
		return nil
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
				slog.Info("Mod already cached", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
				if progressListener != nil {
					progressListener.SetValue(progressListener.GetValue() + (float64(modVersions[i].CompatibleFilesCount(binaryType)) / float64(totalDownloadCount)))
				}
				continue
			}
			slog.Info("Force re-downloading mod, clearing cache", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
			if err := os.RemoveAll(modCacheDir); err != nil {
				return fmt.Errorf("failed to clear mod cache: %w", err)
			}
		}

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
			var resolvedBody io.ReadCloser
			var resolvedContentLength int64
			var resolvedURL string
			var resolvedFilename string
			for _, uri := range file.Downloads {
				if ghactions.IsArtifactURL(uri) {
					token := os.Getenv("GITHUB_TOKEN")
					filename, content, err := ghactions.ResolveArtifactURL(context.Background(), uri, token)
					if err != nil {
						slog.Error("Failed to resolve actions artifact url", "url", uri, "error", err)
						continue
					}
					resolvedFilename = filename
					resolvedContentLength = int64(len(content))
					resolvedBody = io.NopCloser(bytes.NewReader(content))
					resolvedURL = uri
					break
				}

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
			if response == nil && resolvedBody == nil {
				return fmt.Errorf("failed to download mod file from all sources: %s@%s (%s)", modVersions[i].ModID, modVersions[i].ID, file.ID)
			}
			var contentLength int64
			var bodyReader io.ReadCloser
			if resolvedBody != nil {
				contentLength = resolvedContentLength
				bodyReader = resolvedBody
				slog.Info("Resolved actions artifact file", "url", resolvedURL, "filename", resolvedFilename, "contentLength", contentLength)
			} else {
				contentLength = response.ContentLength
				bodyReader = response.Body
				slog.Info("Downloading mod file", "url", response.Request.URL, "contentLength", contentLength)
			}

			hashChecker := checkDownloadedFileHash(&file)

			body := io.TeeReader(bodyReader, hashChecker)

			switch file.ContentType {
			case model.ContentTypeArchive:
				_, err := extractZip(body, contentLength, modCacheRoot, progressListener, totalDownloadCount)
				if err != nil {
					return err
				}
			case model.ContentTypeBinary, model.ContentTypePluginDll:
				path := file.ExtractPath
				var filename string
				// RFC-6266 parsing for filename from Content-Disposition header
				if v := response.Header.Get("Content-Disposition"); v != "" {
					_, params, err := mime.ParseMediaType(v)
					if err == nil {
						if fn, ok := params["filename*"]; ok {
							if strings.HasPrefix(fn, "UTF-8''") {
								filename, err = url.QueryUnescape(fn[7:])
								if err != nil {
									filename = fn[7:]
								}
							} else {
								filename = fn
							}
						} else if fn, ok := params["filename"]; ok {
							filename = fn
						}
					}
				}
				if filepath.Dir(path) == path || path == "" {
					// Invalid or missing path, use filename from header if available
					if filename != "" {
						path = filepath.Join(filepath.Base(path), filename)
					} else if resolvedFilename != "" {
						path = filepath.Join(filepath.Base(path), resolvedFilename)
					} else {
						return fmt.Errorf("file path is empty for normal file type")
					}
				}
				if file.ContentType == model.ContentTypePluginDll {
					if filename == "" {
						if resolvedFilename != "" {
							filename = resolvedFilename
						} else {
							return fmt.Errorf("failed to determine plugin filename for URL: %v", file.Downloads)
						}
					}
					// For plugin files, use a fixed naming scheme
					path = filepath.Join("BepInEx", "plugins", filename)
				}
				if path == "" {
					return fmt.Errorf("file path is empty for normal file type")
				}
				_ = modCacheRoot.MkdirAll(filepath.Dir(path), 0755)
				slog.Info("Saving mod file to cache", "path", path)
				destFile, err := modCacheRoot.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
				return fmt.Errorf("downloaded file hash mismatch: %w", err)
			} else {
				slog.Info("File hash verified", "hash", computedHash)
			}

			if resolvedBody != nil {
				_ = resolvedBody.Close()
			}
		}
	}
	return nil
}

func ApplyMods(gameDir string, cacheDir string, modVersions []ModVersion, binaryType aumgr.BinaryType) (*RestoreInfo, error) {
	backupDir, err := os.MkdirTemp("", "au_mod_backup_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	restoreInfo := &RestoreInfo{
		BackupDir: backupDir,
		Added:     []string{},
		Moved:     make(map[string]string),
	}

	gameRoot, err := os.OpenRoot(gameDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open game directory: %w", err)
	}
	defer gameRoot.Close()

	for _, mod := range modVersions {
		modCacheDir := filepath.Join(cacheDir, mod.ModID, mod.ID)
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

			// Check if file exists in game dir
			if _, err := gameRoot.Stat(relPath); err == nil {
				// Check if it was added by us in this session
				isAdded := slices.Contains(restoreInfo.Added, relPath)

				if !isAdded {
					// It exists and was NOT added by us.
					// Check if we already backed it up.
					if _, ok := restoreInfo.Moved[relPath]; !ok {
						// Not backed up yet, so this is the original file
						// Backup existing file
						backupPath := filepath.Join(backupDir, relPath)
						if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
							return err
						}

						// Move it to backup
						if err := os.Rename(filepath.Join(gameDir, relPath), backupPath); err != nil {
							return fmt.Errorf("failed to backup file %s: %w", relPath, err)
						}
						restoreInfo.Moved[relPath] = backupPath
					}
				}
			} else {
				// File doesn't exist
				// Check if we already marked it as added?
				isAdded := slices.Contains(restoreInfo.Added, relPath)
				if !isAdded {
					restoreInfo.Added = append(restoreInfo.Added, relPath)
				}
			}

			// Copy file from cache to game dir
			destPath := filepath.Join(gameDir, relPath)
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
			return nil, err
		}
	}

	return restoreInfo, nil
}

// func RestoreGame(gameDir string, restoreInfo *RestoreInfo) error {
// 	if restoreInfo == nil {
// 		return nil
// 	}

// 	// Delete added files
// 	// Sort by length desc to delete files before directories?
// 	// Actually we only recorded files.
// 	for _, path := range restoreInfo.Added {
// 		fullPath := filepath.Join(gameDir, path)
// 		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
// 			slog.Warn("Failed to remove added file", "path", path, "error", err)
// 		}
// 	}

// 	// Restore moved files
// 	for origPath, backupPath := range restoreInfo.Moved {
// 		destPath := filepath.Join(gameDir, origPath)
// 		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
// 			slog.Warn("Failed to create directory for restore", "path", destPath, "error", err)
// 			continue
// 		}
// 		// Remove the modded file if it exists (it should, unless we deleted it above)
// 		_ = os.Remove(destPath)

// 		if err := os.Rename(backupPath, destPath); err != nil {
// 			slog.Warn("Failed to restore file", "path", origPath, "error", err)
// 			// Try copy
// 			data, err := os.ReadFile(backupPath)
// 			if err == nil {
// 				_ = os.WriteFile(destPath, data, 0644)
// 			}
// 		}
// 	}

// 	// Cleanup backup dir
// 	_ = os.RemoveAll(restoreInfo.BackupDir)

// 	// Cleanup empty directories in game dir?
// 	// Maybe too risky.

// 	return nil
// }

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

func extractZip(reader io.Reader, contentLength int64, destRoot *os.Root, progressListener progress.Progress, n int) ([]string, error) {
	startVal := 0.0
	if progressListener != nil {
		startVal = progressListener.GetValue()
	}
	perFileScale := 1.0 / float64(n)
	downloadScale := perFileScale * 0.9
	extractScale := perFileScale - downloadScale
	zipBuffer := new(bytes.Buffer)
	buf := progress.NewProgressWriter(startVal, downloadScale, contentLength, progressListener, zipBuffer)
	var written int64
	if contentLength <= 0 {
		w, err := io.Copy(buf, reader)
		if err != nil {
			return nil, err
		}
		written = w
	} else {
		w, err := io.CopyN(buf, reader, contentLength)
		if err != nil {
			return nil, err
		}
		written = w
	}
	buf.Complete()
	zipReader, err := zip.NewReader(bytes.NewReader(zipBuffer.Bytes()), written)
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
		extractFiles = append(extractFiles, f.Name)
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
