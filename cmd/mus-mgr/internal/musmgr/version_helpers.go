package musmgr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func parseFileFlag(val string) *parsedFile {
	pf := &parsedFile{
		Type:           string(model.FileTypeArchive),
		TargetPlatform: string(model.TargetPlatformAny),
	}

	if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
		if !strings.Contains(val, "type=") && !strings.Contains(val, "path=") && !strings.Contains(val, "url=") {
			pf.URLs = append(pf.URLs, val)
			return pf
		}
	}

	if !strings.Contains(val, "=") {
		pf.Path = val
		return pf
	}

	for part := range strings.SplitSeq(val, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			kStr := strings.TrimSpace(kv[0])
			vStr := strings.TrimSpace(kv[1])
			switch kStr {
			case "path":
				pf.Path = vStr
			case "type":
				pf.Type = vStr
			case "url":
				pf.URLs = append(pf.URLs, strings.Split(vStr, "|")...)
			case "extract_path":
				pf.ExtractPath = &vStr
			case "target_platform":
				pf.TargetPlatform = vStr
			}
		}
	}

	if pf.Path == "" && len(pf.URLs) == 0 {
		pf.Path = val
	}

	return pf
}

func parseFeatures(raw []string) map[string]any {
	features := make(map[string]any)
	for _, item := range raw {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		if key == "" {
			continue
		}
		value := strings.ToLower(strings.TrimSpace(kv[1]))
		switch value {
		case "1", "true", "yes", "on":
			features[key] = true
		case "0", "false", "no", "off":
			features[key] = false
		default:
			features[key] = strings.TrimSpace(kv[1])
		}
	}
	return features
}

func nextVersionID(existingIDs []string) string {
	highest := ""
	for _, id := range existingIDs {
		semID := id
		if !strings.HasPrefix(semID, "v") {
			semID = "v" + semID
		}
		if semver.IsValid(semID) {
			if highest == "" || semver.Compare(semID, highest) > 0 {
				highest = semID
			}
		}
	}

	if highest == "" {
		return "1.0.0"
	}

	base := strings.SplitN(strings.TrimPrefix(highest, "v"), "-", 2)[0]
	base = strings.SplitN(base, "+", 2)[0]
	parts := strings.Split(base, ".")
	var major, minor, patch int
	if len(parts) > 0 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
}

func parseDependencies(rawDeps []string) []model.ModVersionDependency {
	var deps []model.ModVersionDependency
	for _, d := range rawDeps {
		parts := strings.Split(d, ":")
		if len(parts) < 2 {
			continue
		}
		depType := model.DependencyTypeRequired
		if len(parts) > 2 {
			depType = model.DependencyType(parts[2])
		}
		deps = append(deps, model.ModVersionDependency{
			ModID:          parts[0],
			VersionID:      parts[1],
			DependencyType: depType,
		})
	}
	return deps
}

func fileMetadataFromParsedFile(pf *parsedFile) (filename string, size int64, hash string, err error) {
	hasher := sha256.New()

	if pf.Path != "" {
		filename = filepath.Base(pf.Path)
		file, openErr := os.Open(pf.Path)
		if openErr != nil {
			return "", 0, "", fmt.Errorf("failed to open file %s: %w", pf.Path, openErr)
		}
		defer file.Close()

		stat, statErr := file.Stat()
		if statErr != nil {
			return "", 0, "", fmt.Errorf("failed to stat file %s: %w", pf.Path, statErr)
		}
		size = stat.Size()

		if _, copyErr := io.Copy(hasher, file); copyErr != nil {
			return "", 0, "", fmt.Errorf("failed to hash file: %w", copyErr)
		}
	} else if len(pf.URLs) > 0 {
		dlURL := pf.URLs[0]
		parsedPath := dlURL
		if strings.Contains(dlURL, "?") {
			parsedPath = strings.SplitN(dlURL, "?", 2)[0]
		}
		filename = filepath.Base(parsedPath)
		if filename == "" || filename == "/" || filename == "." {
			filename = "downloaded_file"
		}

		fmt.Printf("Downloading %s to compute size and hash...\n", dlURL)
		resp, getErr := http.Get(dlURL)
		if getErr != nil {
			return "", 0, "", fmt.Errorf("failed to download url %s: %w", dlURL, getErr)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return "", 0, "", fmt.Errorf("failed to download url %s: status %d", dlURL, resp.StatusCode)
		}

		var copyErr error
		size, copyErr = io.Copy(hasher, resp.Body)
		if copyErr != nil {
			return "", 0, "", fmt.Errorf("failed to read from url %s: %w", dlURL, copyErr)
		}
	} else {
		return "", 0, "", fmt.Errorf("invalid file specifier: neither path nor url provided")
	}

	return filename, size, hex.EncodeToString(hasher.Sum(nil)), nil
}
