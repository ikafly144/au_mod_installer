package thunderstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ikafly144/modrepo/pkg/progress"
)

const (
	DefaultCommunityURL = "https://thunderstore.io/c/repo"
	PackageAPIEndpoint  = "https://thunderstore.io/c/repo/api/v1/package/"
	BepInExPackModID    = "BepInEx-BepInExPack"
)

type Client struct {
	cacheDir   string
	httpClient *http.Client

	mu       sync.RWMutex
	packages map[string]*Package // keyed by FullName (e.g. "BepInEx-BepInExPack")
	pkgList  []*Package
}

func NewClient(cacheDir string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		cacheDir:   cacheDir,
		httpClient: httpClient,
		packages:   make(map[string]*Package),
	}
}

func (c *Client) CacheDir() string {
	return c.cacheDir
}

// FetchPackages fetches all packages from Thunderstore API, with HTTP ETag caching.
func (c *Client) FetchPackages(ctx context.Context) error {
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheFile := filepath.Join(c.cacheDir, "packages.json")
	etagFile := filepath.Join(c.cacheDir, "packages.etag")

	var etag string
	if data, err := os.ReadFile(etagFile); err == nil {
		etag = strings.TrimSpace(string(data))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, PackageAPIEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "MODREPO/1.0 (R.E.P.O. Mod Launcher)")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("Failed to fetch Thunderstore packages, trying local cache", "error", err)
		return c.loadFromDisk(cacheFile)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		slog.Info("Thunderstore packages not modified, loading from cache")
		return c.loadFromDisk(cacheFile)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Thunderstore API returned unexpected status", "status", resp.StatusCode)
		return c.loadFromDisk(cacheFile)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var pkgs []Package
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return fmt.Errorf("failed to decode Thunderstore package JSON: %w", err)
	}

	// Save to cache
	_ = os.WriteFile(cacheFile, data, 0644)
	newETag := resp.Header.Get("ETag")
	if newETag != "" {
		_ = os.WriteFile(etagFile, []byte(newETag), 0644)
	}

	c.loadPackages(pkgs)
	slog.Info("Thunderstore packages loaded", "count", len(pkgs))
	return nil
}

func (c *Client) loadFromDisk(cacheFile string) error {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return fmt.Errorf("no cache file available: %w", err)
	}
	var pkgs []Package
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return fmt.Errorf("failed to decode cached package JSON: %w", err)
	}
	c.loadPackages(pkgs)
	slog.Info("Thunderstore packages loaded from disk cache", "count", len(pkgs))
	return nil
}

func (c *Client) loadPackages(pkgs []Package) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.packages = make(map[string]*Package, len(pkgs))
	c.pkgList = make([]*Package, 0, len(pkgs))

	for i := range pkgs {
		p := &pkgs[i]
		total := 0
		for _, v := range p.Versions {
			total += v.Downloads
		}
		p.TotalDownloads = total
		c.packages[p.FullName] = p
		c.pkgList = append(c.pkgList, p)
	}
}

func (c *Client) GetPackage(fullName string) (*Package, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.packages[fullName]
	return p, ok
}

func (c *Client) GetPackageVersion(fullName string, versionNumber string) (*PackageVersion, bool) {
	p, ok := c.GetPackage(fullName)
	if !ok {
		return nil, false
	}
	for i := range p.Versions {
		if p.Versions[i].VersionNumber == versionNumber {
			return &p.Versions[i], true
		}
	}
	return nil, false
}

func (c *Client) GetLatestVersion(fullName string) (*PackageVersion, bool) {
	p, ok := c.GetPackage(fullName)
	if !ok || len(p.Versions) == 0 {
		return nil, false
	}
	return &p.Versions[0], true
}

func (c *Client) GetAllPackages() []*Package {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*Package, len(c.pkgList))
	copy(result, c.pkgList)
	return result
}

// GetCategories returns all unique category names from loaded packages, sorted alphabetically.
func (c *Client) GetCategories() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[string]struct{})
	for _, p := range c.pkgList {
		for _, cat := range p.Categories {
			seen[cat] = struct{}{}
		}
	}

	cats := make([]string, 0, len(seen))
	for cat := range seen {
		cats = append(cats, cat)
	}
	slices.Sort(cats)
	return cats
}

// SearchPackages searches packages by query, category and sorting criteria.
func (c *Client) SearchPackages(query string, category string, sortBy string) []*Package {
	c.mu.RLock()
	defer c.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	cat := strings.ToLower(strings.TrimSpace(category))

	var matches []*Package
	for _, p := range c.pkgList {
		if cat != "" {
			hasCat := false
			for _, cName := range p.Categories {
				if strings.EqualFold(cName, cat) {
					hasCat = true
					break
				}
			}
			if !hasCat {
				continue
			}
		}

		if q != "" {
			nameMatch := strings.Contains(strings.ToLower(p.Name), q) ||
				strings.Contains(strings.ToLower(p.FullName), q) ||
				strings.Contains(strings.ToLower(p.Owner), q)
			descMatch := false
			if len(p.Versions) > 0 && strings.Contains(strings.ToLower(p.Versions[0].Description), q) {
				descMatch = true
			}
			if !nameMatch && !descMatch {
				continue
			}
		}

		matches = append(matches, p)
	}

	// Sort results
	switch sortBy {
	case "rating":
		slices.SortFunc(matches, func(a, b *Package) int {
			return b.RatingScore - a.RatingScore
		})
	case "updated":
		slices.SortFunc(matches, func(a, b *Package) int {
			if b.DateUpdated.After(a.DateUpdated) {
				return 1
			} else if b.DateUpdated.Before(a.DateUpdated) {
				return -1
			}
			return 0
		})
	case "name":
		slices.SortFunc(matches, func(a, b *Package) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
	default: // "downloads"
		slices.SortFunc(matches, func(a, b *Package) int {
			return b.TotalDownloads - a.TotalDownloads
		})
	}

	return matches
}

// DownloadPackage downloads a mod zip file from Thunderstore to destFile.
func (c *Client) DownloadPackage(ctx context.Context, version *PackageVersion, destFile string, progressListener progress.Progress) error {
	if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, version.DownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "MODREPO/1.0 (R.E.P.O. Mod Launcher)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", version.FullName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with HTTP %d for %s", resp.StatusCode, version.DownloadURL)
	}

	tmpFile := destFile + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	totalBytes := version.FileSize
	if totalBytes <= 0 {
		totalBytes = resp.ContentLength
	}

	var currentBytes int64
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			out.Close()
			_ = os.Remove(tmpFile)
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				out.Close()
				_ = os.Remove(tmpFile)
				return writeErr
			}
			currentBytes += int64(n)
			if progressListener != nil && totalBytes > 0 {
				progressListener.SetValue(float64(currentBytes) / float64(totalBytes))
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			out.Close()
			_ = os.Remove(tmpFile)
			return readErr
		}
	}
	out.Close()

	return os.Rename(tmpFile, destFile)
}

// GetPackageIcon fetches and caches the icon for a package.
func (c *Client) GetPackageIcon(ctx context.Context, iconURL string) ([]byte, error) {
	if iconURL == "" {
		return nil, fmt.Errorf("empty icon URL")
	}

	h := sha256.Sum256([]byte(iconURL))
	hashStr := hex.EncodeToString(h[:16])
	iconDir := filepath.Join(c.cacheDir, "icons")
	_ = os.MkdirAll(iconDir, 0755)
	iconPath := filepath.Join(iconDir, hashStr+".png")

	if data, err := os.ReadFile(iconPath); err == nil && len(data) > 0 {
		return data, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MODREPO/1.0 (R.E.P.O. Mod Launcher)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("icon fetch failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	_ = os.WriteFile(iconPath, data, 0644)
	return data, nil
}
