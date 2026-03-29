package ghactions

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v80/github"
)

func IsArtifactURL(raw string) bool {
	return strings.HasPrefix(raw, "gha://")
}

func ResolveArtifactURL(ctx context.Context, rawURL, token string) (filename string, data []byte, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid actions artifact url: %w", err)
	}
	if u.Scheme != "gha" {
		return "", nil, fmt.Errorf("unsupported actions artifact url scheme: %s", u.Scheme)
	}

	owner := strings.TrimSpace(u.Host)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if owner == "" || len(parts) != 3 || parts[1] != "artifact" {
		return "", nil, fmt.Errorf("invalid actions artifact url path: %s", rawURL)
	}

	repo := strings.TrimSpace(parts[0])
	artifactID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", nil, fmt.Errorf("invalid artifact id in url: %w", err)
	}

	filePattern := strings.TrimSpace(u.Query().Get("file_pattern"))

	client := github.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}

	downloadURL, _, err := client.Actions.DownloadArtifact(ctx, owner, repo, artifactID, 5)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get artifact download URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL.String(), nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create artifact request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to download artifact: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("artifact download failed: status %s", resp.Status)
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read artifact body: %w", err)
	}

	// If no inner file pattern is specified, return the artifact zip itself.
	if filePattern == "" {
		return fmt.Sprintf("artifact-%d.zip", artifactID), zipData, nil
	}

	return extractFromArtifactZip(zipData, filePattern)
}

func extractFromArtifactZip(zipData []byte, filePattern string) (filename string, data []byte, err error) {
	re, err := regexp.Compile(filePattern)
	if err != nil {
		return "", nil, fmt.Errorf("invalid file_pattern regex: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", nil, fmt.Errorf("failed to read artifact zip: %w", err)
	}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !re.MatchString(f.Name) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", nil, fmt.Errorf("failed to open file in artifact: %w", err)
		}
		defer rc.Close()

		content, err := io.ReadAll(rc)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read file in artifact: %w", err)
		}

		name := filepath.Base(f.Name)
		if name == "" || name == "." || name == "/" {
			name = f.Name
		}
		return name, content, nil
	}

	return "", nil, fmt.Errorf("no file matching pattern '%s' found in artifact", filePattern)
}
