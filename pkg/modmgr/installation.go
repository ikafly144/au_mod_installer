package modmgr

import (
	"archive/zip"
	"au_mod_installer/pkg/aumgr"
	"au_mod_installer/pkg/progress"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const currentFileVersion = 2

type ModInstallation struct {
	FileVersion          int           `json:"file_version"`
	InstalledMod         Mod           `json:"installed_mod"`
	InstalledGameVersion string        `json:"installed_game_version"`
	Status               InstallStatus `json:"status"`
	VanillaFiles         []string      `json:"vanilla_files"`
}

type InstallStatus string

const (
	InstallStatusCompatible   InstallStatus = "compatible"
	InstallStatusIncompatible InstallStatus = "incompatible"
	InstallStatusBroken       InstallStatus = "broken"
	InstallStatusUnknown      InstallStatus = "unknown"
)

const installationInfoFileName = ".mod_installation"

func GetInstallationInfoFilePath(gamePath string) string {
	return filepath.Join(gamePath, installationInfoFileName)
}

func LoadInstallationInfo(filePath string) (*ModInstallation, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
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

func SaveInstallationInfo(filePath string, installation *ModInstallation) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(installation); err != nil {
		return err
	}
	return nil
}

func InstallMod(gamePath string, gameManifest aumgr.Manifest, launcherType aumgr.LauncherType, mod Mod, progress progress.Progress) (*ModInstallation, error) {
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
	if !mod.IsCompatible(launcherType, gameManifest.GetVersion()) {
		return nil, fmt.Errorf("mod is not compatible with the current game version: %s", gameManifest.GetVersion())
	}

	installationInfoFilePath := GetInstallationInfoFilePath(gamePath)

	// Remove old installation if exists
	if _, err := os.Stat(installationInfoFilePath); err == nil || !os.IsNotExist(err) {
		if err := UninstallMod(gamePath, installationInfoFilePath, nil); err != nil {
			return nil, fmt.Errorf("failed to remove old mod installation: %w", err)
		}
	}

	vanillaFiles := []string{}
	if err := filepath.Walk(gamePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(gamePath, path)
		if err != nil {
			return err
		}
		vanillaFiles = append(vanillaFiles, relPath)
		return nil
	}); err != nil {
		return nil, err
	}
	installation := &ModInstallation{
		FileVersion:          currentFileVersion,
		InstalledMod:         mod,
		InstalledGameVersion: gameManifest.GetVersion(),
		Status:               InstallStatusBroken,
		VanillaFiles:         vanillaFiles,
	}
	if err := SaveInstallationInfo(installationInfoFilePath, installation); err != nil {
		return nil, fmt.Errorf("failed to save installation info: %w", err)
	}

	gameRoot, err := os.OpenRoot(gamePath)
	if err != nil {
		return nil, err
	}
	hClient := http.DefaultClient
	for file := range mod.Downloads(launcherType) {
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
			if err := extractZip(resp.Body, contentLength, gameRoot, progress, mod.CompatibleFilesCount(launcherType)); err != nil {
				return nil, err
			}
		case FileTypeNormal:
			_ = gameRoot.MkdirAll(filepath.Dir(file.Path), 0755)
			destFile, err := gameRoot.OpenFile(file.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return nil, err
			}
			defer destFile.Close()
			buf := &ProgressWrapper{
				start:    progress.GetValue(),
				goal:     uint64(contentLength),
				scale:    (1.0 / float64(mod.CompatibleFilesCount(launcherType))),
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
	installation.Status = InstallStatusCompatible
	if err := SaveInstallationInfo(installationInfoFilePath, installation); err != nil {
		return nil, err
	}
	return installation, nil
}

func UninstallMod(gamePath string, installationInfoFilePath string, progress progress.Progress) error {
	if err := uninstallMod(gamePath, installationInfoFilePath, progress); err != nil {
		return fmt.Errorf("failed to uninstall mod: %w", err)
	}
	if err := os.Remove(installationInfoFilePath); err != nil {
		return err
	}
	return nil
}

func uninstallMod(gamePath string, installationInfoFilePath string, progress progress.Progress) error {
	if progress != nil {
		progress.SetValue(0.0)
		progress.Start()
		defer progress.Done()
	}
	installation, err := LoadInstallationInfo(installationInfoFilePath)
	if err != nil {
		return err
	}

	if installation.FileVersion == 0 {
		installation.VanillaFiles = append(installation.VanillaFiles, "Among Us_Data")
		installation.VanillaFiles = append(installation.VanillaFiles, ".egstore")
		installation.VanillaFiles = append(installation.VanillaFiles, ".egstore\\4AD6AD0447626FA05A0648B2A5D8C66A.mancpn")
		installation.VanillaFiles = append(installation.VanillaFiles, ".egstore\\4AD6AD0447626FA05A0648B2A5D8C66A.manifest")
		installation.VanillaFiles = append(installation.VanillaFiles, ".egstore\\Pending")
	}

	info, err := os.Open(gamePath)
	if err != nil {
		return err
	}
	defer info.Close()
	fileCount, err := info.Readdirnames(-1)
	if err != nil && err != io.EOF {
		return err
	}
	i := 0

	if err := filepath.Walk(gamePath, func(path string, info os.FileInfo, err error) error {
		i++
		if progress != nil {
			progress.SetValue(float64(i) / float64(len(fileCount)))
		}
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			slog.Warn("Failed to access file during uninstallation", "file", path, "error", err)
		}
		relPath, err := filepath.Rel(gamePath, path)
		if err != nil {
			return err
		}
		if gamePath == path {
			return nil
		}
		if slices.Contains(installation.VanillaFiles, relPath) {
			return nil
		}
		if strings.HasPrefix(relPath, "Among Us_Data") {
			return nil
		}
		if filepath.Ext(path) == ".mod_installation" {
			return nil
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
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

func extractZip(reader io.Reader, contentLength int64, destRoot *os.Root, progress progress.Progress, n int) error {
	buf := &ProgressWriter{
		start:    progress.GetValue(),
		scale:    (1.0 / float64(n)) * 0.9,
		goal:     uint64(contentLength),
		progress: progress,
		buf:      new(bytes.Buffer),
	}
	if _, err := io.CopyN(buf, reader, contentLength); err != nil {
		return err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(buf.buf.Bytes()), contentLength)
	if err != nil {
		return err
	}
	filesCount := len(zipReader.File)
	i := 0
	start := progress.GetValue()
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if err := extractFile(f, destRoot); err != nil {
			return err
		}
		i++
		if progress != nil {
			progress.SetValue(float64(i)/float64(filesCount)*buf.scale*0.2 + start)
		}
	}
	return nil
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
