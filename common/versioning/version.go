package versioning

// repository information
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/v87/github"
	"golang.org/x/mod/semver"
)

var (
	repoOwner    = "ikafly144"
	repoName     = "au_mod_installer"
	artifactName = "mod-of-us_${OS}_${ARCH}.msi"
)

func CheckForUpdates(ctx context.Context, branch Branch, currentVersion string) (releaseTag string, latestStable string, err error) {
	var opts []github.ClientOptionsFunc
	client, err := github.NewClient(opts...)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GitHub client: %w", err)
	}
	opt := &github.ListOptions{
		PerPage: 10,
		Page:    1,
	}
outer:
	for {
		tags, resp, err := client.Repositories.ListTags(ctx, repoOwner, repoName, opt)
		if err != nil {
			return "", "", err
		}
		for _, tag := range tags {
			slog.Info("found tag", "tag", tag.GetName())
			if before, _, _ := strings.Cut(strings.TrimPrefix(semver.Prerelease(tag.GetName()), "-"), "."); before != "" && !branch.match(before) {
				slog.Info("skipping tag due to prerelease branch mismatch", "tag", tag.GetName(), "branch", branch)
			}
			if semver.Compare(tag.GetName(), currentVersion) <= 0 {
				slog.Info("no newer version found", "current", currentVersion, "found", tag.GetName())
				return "", "", nil
			}
			if semver.Prerelease(tag.GetName()) != "" && semver.Compare(tag.GetName(), releaseTag) <= 0 {
				slog.Info("already found a newer version, skipping", "current", currentVersion, "found", tag.GetName(), "existing", releaseTag)
				continue
			}
			release, _, err := client.Repositories.GetReleaseByTag(ctx, repoOwner, repoName, tag.GetName())
			if err != nil {
				slog.Error("failed to get release by tag", "tag", tag.GetName(), "error", err)
				continue
			}
			if release.GetTagName() != currentVersion && releaseTag == "" {
				releaseTag = release.GetTagName()
			}
			if semver.Prerelease(release.GetTagName()) == "" && latestStable == "" {
				latestStable = release.GetTagName()
			}
			if releaseTag != "" && latestStable != "" {
				break outer
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return releaseTag, latestStable, nil
}

func Update(ctx context.Context, tag string) (bool, error) {
	var opts []github.ClientOptionsFunc
	client, err := github.NewClient(opts...)
	if err != nil {
		return false, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	release, _, err := client.Repositories.GetReleaseByTag(ctx, repoOwner, repoName, tag)
	if err != nil {
		return false, err
	}

	assetName := replaceOSAndArch(artifactName)
	var checkSum []byte
	var binaryAsset *github.ReleaseAsset

	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			binaryAsset = asset
			continue
		}
		if asset.GetName() == "checksums.txt" {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.GetBrowserDownloadURL(), nil)
			if err != nil {
				return false, fmt.Errorf("failed to create request for checksums.txt: %w", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return false, fmt.Errorf("failed to download checksums.txt: %w", err)
			}
			defer resp.Body.Close()
			buf := new(strings.Builder)

			var sha256Hash [32]byte
			if hashStr, ok := strings.CutPrefix(asset.GetDigest(), "sha256:"); ok {
				if _, err := hex.Decode(sha256Hash[:], []byte(hashStr)); err != nil {
					return false, fmt.Errorf("failed to decode checksum: %w", err)
				}
			}
			hasher := sha256.New()
			writer := io.MultiWriter(buf, hasher)
			if _, err = io.Copy(writer, resp.Body); err != nil {
				return false, err
			}
			if !bytes.Equal(sha256Hash[:], hasher.Sum(nil)) {
				return false, errors.New("checksum verification failed for checksums.txt")
			}
			lines := strings.SplitSeq(buf.String(), "\n")
			for line := range lines {
				parts := strings.Fields(line)
				if len(parts) == 2 && parts[1] == assetName {
					checkSum, err = hex.DecodeString(parts[0])
					if err != nil {
						return false, err
					}
					break
				}
			}
		}
	}
	if binaryAsset == nil {
		return false, errors.New("no suitable asset found for update")
	}
	if len(checkSum) == 0 {
		return false, errors.New("checksum for MSI not found in checksums.txt")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, binaryAsset.GetBrowserDownloadURL(), nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	hasher := sha256.New()
	tempFile, err := os.CreateTemp("", "mod-of-us-*.msi")
	if err != nil {
		return false, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()
	reader := io.TeeReader(resp.Body, hasher)
	if _, err := io.Copy(tempFile, reader); err != nil {
		return false, err
	}
	if !bytes.Equal(checkSum, hasher.Sum(nil)) {
		return false, errors.New("checksum verification failed for downloaded MSI")
	}
	if err := tempFile.Close(); err != nil {
		return false, err
	}

	cmd := exec.Command("msiexec", "/i", tempFile.Name(), "/norestart")
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("failed to start installer: %w", err)
	}
	slog.Info("Started MSI installer", "path", tempFile.Name())
	return true, nil
}

func replaceOSAndArch(name string) string {
	// Replace OS
	var osStr string
	switch runtime.GOOS {
	case "windows":
		osStr = "windows"
	case "linux":
		osStr = "linux"
	case "darwin":
		osStr = "darwin"
	default:
		osStr = runtime.GOOS
	}
	name = strings.ReplaceAll(name, "${OS}", osStr)

	// Replace Arch
	var archStr string
	switch runtime.GOARCH {
	case "amd64":
		archStr = "x86_64"
	case "386":
		archStr = "i386"
	default:
		archStr = runtime.GOARCH
	}
	name = strings.ReplaceAll(name, "${ARCH}", archStr)

	if osStr == "windows" && filepath.Ext(name) == "" {
		name += ".exe"
	}

	return name
}

func BranchFromString(s string) Branch {
	for b, str := range branchString {
		if str == s {
			return b
		}
	}
	return BranchStable
}

type Branch int

const (
	BranchStable Branch = iota
	BranchPreview
	BranchBeta
	BranchCanary
	BranchDev
)

var branchString = map[Branch]string{
	BranchStable:  "stable",
	BranchPreview: "preview",
	BranchBeta:    "beta",
	BranchCanary:  "canary",
	BranchDev:     "dev",
}

var prereleaseToBranch = map[string]Branch{
	"rc":    BranchPreview,
	"pre":   BranchBeta,
	"beta":  BranchCanary,
	"alpha": BranchDev,
}

func (b Branch) match(prerelease string) bool {
	if p, ok := prereleaseToBranch[prerelease]; (ok && p <= b) || prerelease == "" {
		return true
	}
	return false
}

func (b Branch) String() string {
	return branchString[b]
}
