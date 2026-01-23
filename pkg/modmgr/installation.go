package modmgr

import (
	"archive/zip"
	"bytes"
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

func DownloadMods(cacheDir string, modVersions []ModVersion, binaryType aumgr.BinaryType, progress progress.Progress, force bool) error {
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

	if progress != nil {
		progress.SetValue(0.0)
		progress.Start()
		defer progress.Done()
	}

	hClient := http.DefaultClient
	for i := range modVersions {
		modCacheDir := filepath.Join(cacheDir, modVersions[i].ModID, hashId(modVersions[i].ID))
		if _, err := os.Stat(modCacheDir); err == nil {
			if !force {
				slog.Info("Mod already cached", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
				progress.SetValue(progress.GetValue() + (float64(modVersions[i].CompatibleFilesCount(binaryType)) / float64(totalDownloadCount)))
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
			req, err := http.NewRequest(http.MethodGet, file.URL, nil)
			if err != nil {
				return err
			}
			resp, err := hClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			contentLength := resp.ContentLength
			slog.Info("Downloading mod file", "url", file.URL, "contentLength", contentLength)

			switch file.FileType {
			case FileTypeZip:
				_, err := extractZip(resp.Body, contentLength, modCacheRoot, progress, totalDownloadCount)
				if err != nil {
					return err
				}
			case FileTypeNormal, FileTypePlugin:
				path := file.Path
				var filename string
				// RFC-6266 parsing for filename from Content-Disposition header
				if v := resp.Header.Get("Content-Disposition"); v != "" {
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
					} else {
						return fmt.Errorf("file path is empty for normal file type")
					}
				}
				if file.FileType == FileTypePlugin {
					if filename == "" {
						return fmt.Errorf("failed to determine plugin filename for URL: %s", file.URL)
					}
					// For plugin files, use a fixed naming scheme
					path = filepath.Join("BepInEx", "plugins", filename)
				}
				if path == "" {
					return fmt.Errorf("file path is empty for normal file type")
				}
				_ = modCacheRoot.MkdirAll(filepath.Dir(path), 0755)
				destFile, err := modCacheRoot.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return err
				}
				defer destFile.Close()
				buf := &ProgressWrapper{
					start:    progress.GetValue(),
					goal:     uint64(contentLength),
					scale:    (1.0 / float64(totalDownloadCount)),
					progress: progress,
					buf:      destFile,
				}
				if _, err := io.Copy(buf, resp.Body); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown file type: %s", file.FileType)
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
				isAdded := false
				for _, added := range restoreInfo.Added {
					if added == relPath {
						isAdded = true
						break
					}
				}

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
				isAdded := false
				for _, added := range restoreInfo.Added {
					if added == relPath {
						isAdded = true
						break
					}
				}
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

func RestoreGame(gameDir string, restoreInfo *RestoreInfo) error {
	if restoreInfo == nil {
		return nil
	}

	// Delete added files
	// Sort by length desc to delete files before directories?
	// Actually we only recorded files.
	for _, path := range restoreInfo.Added {
		fullPath := filepath.Join(gameDir, path)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to remove added file", "path", path, "error", err)
		}
	}

	// Restore moved files
	for origPath, backupPath := range restoreInfo.Moved {
		destPath := filepath.Join(gameDir, origPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			slog.Warn("Failed to create directory for restore", "path", destPath, "error", err)
			continue
		}
		// Remove the modded file if it exists (it should, unless we deleted it above)
		_ = os.Remove(destPath)

		if err := os.Rename(backupPath, destPath); err != nil {
			slog.Warn("Failed to restore file", "path", origPath, "error", err)
			// Try copy
			data, err := os.ReadFile(backupPath)
			if err == nil {
				_ = os.WriteFile(destPath, data, 0644)
			}
		}
	}

	// Cleanup backup dir
	_ = os.RemoveAll(restoreInfo.BackupDir)

	// Cleanup empty directories in game dir?
	// Maybe too risky.

	return nil
}

func InstallMod(modInstallLocation *os.Root, gameVersion string, launcherType aumgr.LauncherType, binaryType aumgr.BinaryType, modVersions []ModVersion, progress progress.Progress) (*ModInstallation, error) {
	slog.Info("Starting mod installation", "mods", modVersions)
	if progress != nil {
		progress.SetValue(0.0)
		progress.Start()
		defer progress.Done()
	}

	if binaryType == aumgr.BinaryTypeUnknown {
		return nil, fmt.Errorf("unknown binary type")
	}
	for _, mod := range modVersions {
		if !mod.IsCompatible(launcherType, binaryType, gameVersion) {
			return nil, fmt.Errorf("mod is not compatible with the current game version: %s", gameVersion)
		}
	}

	var remainMods []InstalledVersionInfo
	// Remove old installation if exists
	if _, err := modInstallLocation.Stat(InstallationInfoFileName); err == nil || !os.IsNotExist(err) {
		remainModInfos, err := UninstallRemainingMods(modInstallLocation, progress, modVersions)
		if err != nil {
			return nil, fmt.Errorf("failed to remove old mod installation: %w", err)
		}
		progress.SetValue(0.0)
		slog.Info("Filtered remaining mods after uninstallation", "remainMods", remainModInfos)
		for _, remainModInfo := range remainModInfos {
			for _, modVersion := range modVersions {
				if remainModInfo.ModID == modVersion.ModID && remainModInfo.ID == modVersion.ID {
					remainMods = append(remainMods, remainModInfo)
					break
				}
			}
		}
	}

	var installedMods []InstalledVersionInfo
	for _, modVersion := range modVersions {
		installedMods = append(installedMods, InstalledVersionInfo{
			ModVersion: modVersion,
			Paths:      nil,
		})
	}

	installation := &ModInstallation{
		FileVersion:          currentFileVersion,
		InstalledMods:        installedMods,
		InstalledGameVersion: gameVersion,
		Status:               InstallStatusBroken,
	}
	if err := SaveInstallationInfo(modInstallLocation, installation); err != nil {
		return nil, fmt.Errorf("failed to save installation info: %w", err)
	}
	defer func(installation *ModInstallation) {
		if err := SaveInstallationInfo(modInstallLocation, installation); err != nil {
			slog.Error("Failed to finalize installation info", "error", err)
		}
	}(installation)

	totalDownloadCount := func() int {
		count := 0
		for i := range modVersions {
			count += modVersions[i].CompatibleFilesCount(binaryType)
		}
		return count
	}()
	hClient := http.DefaultClient
	for i := range modVersions {
		if remainMods != nil {
			shouldSkip := false
			var remainModInfo InstalledVersionInfo
			for _, remainMod := range remainMods {
				if modVersions[i].ModID == remainMod.ModID && modVersions[i].ID == remainMod.ID {
					shouldSkip = true
					remainModInfo = remainMod
					break
				}
			}
			if shouldSkip {
				slog.Info("Skipping already installed mod", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
				installation.InstalledMods[i] = remainModInfo
				progress.SetValue(progress.GetValue() + (1.0 / float64(totalDownloadCount)))
				continue
			}
		}
		slog.Info("Installing mod", "modId", modVersions[i].ModID, "versionId", modVersions[i].ID)
		for file := range modVersions[i].Downloads(binaryType) {
			req, err := http.NewRequest(http.MethodGet, file.URL, nil)
			if err != nil {
				return nil, err
			}
			resp, err := hClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			contentLength := resp.ContentLength
			slog.Info("Downloading mod", "url", file.URL, "contentLength", contentLength)
			switch file.FileType {
			case FileTypeZip:
				extractFiles, err := extractZip(resp.Body, contentLength, modInstallLocation, progress, totalDownloadCount)
				installation.InstalledMods[i].Paths = append(installation.InstalledMods[i].Paths, extractFiles...)
				if err != nil {
					if e := SaveInstallationInfo(modInstallLocation, installation); e != nil {
						return nil, fmt.Errorf("failed to install mod: %w (%v)", err, e)
					}
					return nil, err
				}
			case FileTypeNormal:
				if file.Path == "" {
					return nil, fmt.Errorf("file path is empty for normal file type")
				}
				_ = modInstallLocation.MkdirAll(filepath.Dir(file.Path), 0755)
				destFile, err := modInstallLocation.OpenFile(file.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return nil, err
				}
				defer destFile.Close()
				buf := &ProgressWrapper{
					start:    progress.GetValue(),
					goal:     uint64(contentLength),
					scale:    (1.0 / float64(totalDownloadCount)),
					progress: progress,
					buf:      destFile,
				}
				installation.InstalledMods[i].Paths = append(installation.InstalledMods[i].Paths, file.Path)
				if _, err := io.Copy(buf, resp.Body); err != nil {
					if e := SaveInstallationInfo(modInstallLocation, installation); e != nil {
						return nil, fmt.Errorf("failed to install mod: %w (%v)", err, e)
					}
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown file type: %s", file.FileType)
			}
		}
	}
	installation.Status = InstallStatusCompatible
	return installation, nil
}

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

type ProgressWrapper struct {
	start    float64
	goal     uint64
	scale    float64
	progress progress.Progress
	bytes    int
	buf      io.Writer
}

func (pw *ProgressWrapper) Write(data []byte) (n int, err error) {
	if pw.buf != nil {
		n, err = pw.buf.Write(data)
	}
	pw.bytes += n
	if pw.scale > 0 && pw.progress != nil {
		pw.progress.SetValue(float64(pw.bytes)/float64(pw.goal)*pw.scale + pw.start)
	}
	return
}

type ProgressWriter struct {
	start    float64
	scale    float64
	goal     uint64
	progress progress.Progress
	buf      *bytes.Buffer
}

func (pw *ProgressWriter) Write(data []byte) (n int, err error) {
	if pw.buf != nil {
		n, err = pw.buf.Write(data)
	}
	if pw.goal > 0 && pw.progress != nil {
		pw.progress.SetValue(float64(pw.buf.Len())/float64(pw.goal)*pw.scale + pw.start)
	}
	return
}

func extractZip(reader io.Reader, contentLength int64, destRoot *os.Root, progress progress.Progress, n int) ([]string, error) {
	buf := &ProgressWriter{
		start:    progress.GetValue(),
		scale:    (1.0 / float64(n)) * 0.9,
		goal:     uint64(contentLength),
		progress: progress,
		buf:      new(bytes.Buffer),
	}
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
	zipReader, err := zip.NewReader(bytes.NewReader(buf.buf.Bytes()), written)
	if err != nil {
		return nil, err
	}
	filesCount := len(zipReader.File)
	i := 0
	start := progress.GetValue()
	var extractErr error
	var extractFiles []string
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		extractFiles = append(extractFiles, f.Name)
		if err := extractFile(f, destRoot); err != nil {
			slog.Warn("Failed to extract file from zip", "file", f.Name, "error", err)
			extractErr = err
			break
		}
		i++
		if progress != nil {
			progress.SetValue(float64(i)/float64(filesCount)*buf.scale*0.2 + start)
		}
	}
	return extractFiles, extractErr
}

func extractFile(f *zip.File, destRoot *os.Root) error {
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

	if _, err := io.Copy(destFile, rc); err != nil {
		return err
	}
	return nil
}
