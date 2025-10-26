package modmgr

import (
	"archive/zip"
	"au_mod_installer/pkg/aumgr"
	"au_mod_installer/pkg/progress"
	"bytes"
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
)

const currentFileVersion = 2

type ModInstallation struct {
	FileVersion          int                `json:"file_version"`
	InstalledMods        []InstalledModInfo `json:"installed_mods"`
	InstalledGameVersion string             `json:"installed_game_version"`
	Status               InstallStatus      `json:"status"`
	raw                  json.RawMessage    `json:"-"`
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

type InstalledModInfo struct {
	Mod   `json:",inline"`
	Paths []string `json:"paths"`
}

type InstallStatus string

const (
	InstallStatusCompatible   InstallStatus = "compatible"
	InstallStatusIncompatible InstallStatus = "incompatible"
	InstallStatusBroken       InstallStatus = "broken"
	InstallStatusUnknown      InstallStatus = "unknown"
)

const InstallationInfoFileName = ".mod_installation"

func LoadInstallationInfo(root *os.Root) (*ModInstallation, error) {
	file, err := root.OpenFile(InstallationInfoFileName, os.O_RDONLY, 0644)
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

func InstallMod(gameRoot *os.Root, gameManifest aumgr.Manifest, launcherType aumgr.LauncherType, mods []Mod, progress progress.Progress) (*ModInstallation, error) {
	slog.Info("Starting mod installation", "mods", mods)
	if progress != nil {
		progress.SetValue(0.0)
		progress.Start()
		defer progress.Done()
	}

	if gameManifest == nil {
		return nil, fmt.Errorf("game manifest is nil")
	}
	if launcherType == aumgr.LauncherUnknown {
		return nil, fmt.Errorf("unknown launcher type")
	}
	for _, mod := range mods {
		if !mod.IsCompatible(launcherType, gameManifest.GetVersion()) {
			return nil, fmt.Errorf("mod is not compatible with the current game version: %s", gameManifest.GetVersion())
		}
	}

	// Remove old installation if exists
	if _, err := gameRoot.Stat(InstallationInfoFileName); err == nil || !os.IsNotExist(err) {
		if err := UninstallMod(gameRoot, progress); err != nil {
			return nil, fmt.Errorf("failed to remove old mod installation: %w", err)
		}
		progress.SetValue(0.0)
	}

	var installedMods []InstalledModInfo
	for _, mod := range mods {
		installedMods = append(installedMods, InstalledModInfo{
			Mod:   mod,
			Paths: nil,
		})
	}

	installation := &ModInstallation{
		FileVersion:          currentFileVersion,
		InstalledMods:        installedMods,
		InstalledGameVersion: gameManifest.GetVersion(),
		Status:               InstallStatusBroken,
	}
	if err := SaveInstallationInfo(gameRoot, installation); err != nil {
		return nil, fmt.Errorf("failed to save installation info: %w", err)
	}
	defer func(installation *ModInstallation) {
		if err := SaveInstallationInfo(gameRoot, installation); err != nil {
			slog.Error("Failed to finalize installation info", "error", err)
		}
	}(installation)

	hClient := http.DefaultClient
	for i := range mods {
		for file := range mods[i].Downloads(launcherType) {
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
			if contentLength <= 0 {
				return nil, fmt.Errorf("invalid content length: %d", contentLength)
			}
			slog.Info("Downloading mod", "url", file.URL, "contentLength", contentLength)
			switch file.FileType {
			case FileTypeZip:
				extractFiles, err := extractZip(resp.Body, contentLength, gameRoot, progress, mods[i].CompatibleFilesCount(launcherType)*len(mods))
				installation.InstalledMods[i].Paths = append(installation.InstalledMods[i].Paths, extractFiles...)
				if err != nil {
					if e := SaveInstallationInfo(gameRoot, installation); e != nil {
						return nil, fmt.Errorf("failed to install mod: %w (%v)", err, e)
					}
					return nil, err
				}
			case FileTypeNormal:
				installation.InstalledMods[i].Paths = append(installation.InstalledMods[i].Paths, file.Path)
				_ = gameRoot.MkdirAll(filepath.Dir(file.Path), 0755)
				destFile, err := gameRoot.OpenFile(file.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return nil, err
				}
				defer destFile.Close()
				buf := &ProgressWrapper{
					start:    progress.GetValue(),
					goal:     uint64(contentLength),
					scale:    (1.0 / float64(mods[i].CompatibleFilesCount(launcherType)*len(mods))),
					progress: progress,
					buf:      destFile,
				}
				if _, err := io.Copy(buf, resp.Body); err != nil {
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

func UninstallMod(gameRoot *os.Root, progress progress.Progress) error {
	if err := uninstallMod(gameRoot, progress); err != nil {
		return fmt.Errorf("failed to uninstall mod: %w", err)
	}
	if err := gameRoot.Remove(InstallationInfoFileName); err != nil {
		return err
	}
	return nil
}

func uninstallMod(gameRoot *os.Root, progress progress.Progress) error {
	if progress != nil {
		progress.SetValue(0.0)
		progress.Start()
		defer progress.Done()
	}
	installation, err := LoadInstallationInfo(gameRoot)
	if err != nil {
		return err
	}

	dirInfo, err := gameRoot.Open(".")
	if err != nil {
		return err
	}
	defer dirInfo.Close()
	fileCount, err := dirInfo.Readdirnames(-1)
	if err != nil && err != io.EOF {
		return err
	}

	switch installation.FileVersion {
	case 0, 1:
		i := 0
		if err := fs.WalkDir(gameRoot.FS(), ".", func(path string, info fs.DirEntry, err error) error {
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
			if err := gameRoot.RemoveAll(path); err != nil {
				slog.Warn("Failed to delete file during uninstallation", "file", path, "error", err)
				return nil
			}
			return nil
		}); err != nil {
			return err
		}
	case 2:
		var paths []string
		for _, mod := range installation.InstalledMods {
			paths = append(paths, mod.Paths...)
		}
		sort.SliceStable(paths, func(i, j int) bool {
			return len(paths[i]) > len(paths[j])
		})
		for _, path := range paths {
			if err := gameRoot.RemoveAll(path); err != nil {
				slog.Warn("Failed to remove mod file during uninstallation", "file", path, "error", err)
			}
			if err := removeEmptyDirs(gameRoot, filepath.Dir(path)); err != nil {
				slog.Warn("Failed to remove empty directory during uninstallation", "dir", filepath.Dir(path), "error", err)
			}
		}
	}
	return nil
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
	if _, err := io.CopyN(buf, reader, contentLength); err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(buf.buf.Bytes()), contentLength)
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
