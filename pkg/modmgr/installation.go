package modmgr

import (
	"archive/zip"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ikafly144/modrepo/common/rest/model"
	"github.com/ikafly144/modrepo/pkg/progress"
	"github.com/ikafly144/modrepo/pkg/repomgr"
)

type CacheMetadata struct {
	ModVersion ModVersion `json:"mod_version"`
}

func fileDestinationPath(file model.ModVersionFile) string {
	path := file.ExtractPath
	filename := filepath.Base(file.Filename)
	if path == "" && file.ContentType == model.ContentTypePluginDll {
		path = filepath.Join("BepInEx", "plugins", filename)
	}
	if path == "" {
		path = filename
	}
	if filepath.Base(path) != filename {
		path = filepath.Join(filepath.Dir(path), filename)
	}
	return path
}

func DownloadMods(cacheDir string, modVersions []ModVersion, binaryType repomgr.BinaryType, progressListener progress.Progress, force bool) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	totalDownloadCount := len(modVersions)
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
				metaFile, err := os.Open(filepath.Join(modCacheDir, "metadata.json"))
				if err == nil {
					var metadata CacheMetadata
					if err := json.UnmarshalRead(metaFile, &metadata); err == nil && metadata.ModVersion.VersionID == modVersions[i].VersionID {
						metaFile.Close()
						// Check if package zip exists
						zipName := modVersions[i].ModID + ".zip"
						if len(modVersions[i].Files) > 0 && modVersions[i].Files[0].Filename != "" {
							zipName = modVersions[i].Files[0].Filename
						}
						if _, err := os.Stat(filepath.Join(modCacheDir, zipName)); err == nil {
							slog.Info("Mod already cached", "modId", modVersions[i].ModID, "versionId", modVersions[i].VersionID)
							if progressListener != nil {
								progressListener.SetValue(progressListener.GetValue() + (1.0 / float64(totalDownloadCount)))
							}
							continue
						}
					} else {
						metaFile.Close()
					}
				}
			}
			_ = os.RemoveAll(modCacheDir)
		}

		if err := os.MkdirAll(modCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create mod cache directory: %w", err)
		}

		downloadURL := modVersions[i].DownloadURL
		zipName := modVersions[i].ModID + ".zip"
		if len(modVersions[i].Files) > 0 {
			if len(modVersions[i].Files[0].Downloads) > 0 {
				downloadURL = modVersions[i].Files[0].Downloads[0]
			}
			if modVersions[i].Files[0].Filename != "" {
				zipName = modVersions[i].Files[0].Filename
			}
		}

		if downloadURL == "" {
			return fmt.Errorf("mod version has no download URL: %s@%s", modVersions[i].ModID, modVersions[i].VersionID)
		}

		slog.Info("Downloading mod", "modId", modVersions[i].ModID, "versionId", modVersions[i].VersionID, "url", downloadURL)
		req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for %s: %w", downloadURL, err)
		}
		req.Header.Set("User-Agent", "MODREPO/1.0 (R.E.P.O. Mod Launcher)")

		resp, err := hClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download mod %s: %w", modVersions[i].ModID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download mod %s, HTTP %d", modVersions[i].ModID, resp.StatusCode)
		}

		cachedZipPath := filepath.Join(modCacheDir, zipName)
		outFile, err := os.Create(cachedZipPath)
		if err != nil {
			return err
		}

		startVal := 0.0
		if progressListener != nil {
			startVal = progressListener.GetValue()
		}
		pw := progress.NewProgressWriter(startVal, (1.0 / float64(totalDownloadCount)), resp.ContentLength, progressListener, outFile)
		if _, err := io.Copy(pw, resp.Body); err != nil {
			outFile.Close()
			_ = os.Remove(cachedZipPath)
			return fmt.Errorf("failed to save mod zip %s: %w", modVersions[i].ModID, err)
		}
		pw.Complete()
		outFile.Close()

		// Write metadata.json
		metadata := CacheMetadata{ModVersion: modVersions[i]}
		metaFile, err := os.Create(filepath.Join(modCacheDir, "metadata.json"))
		if err != nil {
			return err
		}
		if err := json.MarshalWrite(metaFile, metadata); err != nil {
			metaFile.Close()
			return err
		}
		metaFile.Close()
	}

	return nil
}

func removeEmptyDirs(root *os.Root, dir string) error {
	dirInfo, err := root.Stat(dir)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return nil
	}
	d, err := root.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	entries, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		if err := root.Remove(dir); err != nil {
			return err
		}
		parent := filepath.Dir(dir)
		if parent != "." && parent != "/" && parent != dir {
			return removeEmptyDirs(root, parent)
		}
	}
	return nil
}

// extractThunderstoreZip extracts a Thunderstore mod ZIP according to r2modman rules.
func extractThunderstoreZip(reader io.ReaderAt, contentLength int64, modID string, profileRoot *os.Root) ([]string, error) {
	zipReader, err := zip.NewReader(reader, contentLength)
	if err != nil {
		return nil, err
	}

	var extractedFiles []string
	isBepInExPack := strings.EqualFold(modID, "BepInEx-BepInExPack")

	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}

		cleanName := filepath.Clean(f.Name)
		normalized := strings.ReplaceAll(cleanName, "\\", "/")

		var destPath string
		if isBepInExPack {
			// BepInExPack usually has BepInExPack/BepInEx/... or BepInExPack/winhttp.dll
			trimmed := normalized
			if strings.HasPrefix(strings.ToLower(trimmed), "bepinexpack/") {
				trimmed = trimmed[len("bepinexpack/"):]
			}
			destPath = filepath.FromSlash(trimmed)
		} else {
			lower := strings.ToLower(normalized)
			if strings.HasPrefix(lower, "bepinex/") {
				destPath = filepath.FromSlash(normalized)
			} else if strings.HasPrefix(lower, "plugins/") {
				sub := normalized[len("plugins/"):]
				destPath = filepath.Join("BepInEx", "plugins", modID, filepath.FromSlash(sub))
			} else if strings.HasPrefix(lower, "patchers/") {
				sub := normalized[len("patchers/"):]
				destPath = filepath.Join("BepInEx", "patchers", modID, filepath.FromSlash(sub))
			} else if strings.HasPrefix(lower, "config/") {
				destPath = filepath.Join("BepInEx", filepath.FromSlash(normalized))
			} else {
				// Root level files: dll, bundle, etc.
				destPath = filepath.Join("BepInEx", "plugins", modID, filepath.FromSlash(normalized))
			}
		}

		if destPath == "" {
			continue
		}

		if err := extractZipFile(f, destPath, profileRoot); err != nil {
			return nil, fmt.Errorf("failed to extract %s to %s: %w", f.Name, destPath, err)
		}
		extractedFiles = append(extractedFiles, filepath.Clean(destPath))
	}

	return extractedFiles, nil
}

func extractZipFile(f *zip.File, destPath string, profileRoot *os.Root) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dir := filepath.Dir(destPath)
	if dir != "." && dir != "/" {
		_ = profileRoot.MkdirAll(dir, 0755)
	}

	destFile, err := profileRoot.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, rc)
	return err
}
