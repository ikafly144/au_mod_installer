package versioning

// repository information
import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-github/v85/github"
	"github.com/minio/selfupdate"
	"golang.org/x/mod/semver"
)

var (
	repoOwner    = "ikafly144"
	repoName     = "au_mod_installer"
	artifactName = "mod-of-us_${OS}_${ARCH}"
)

func CheckForUpdates(ctx context.Context, branch Branch, currentVersion string) (releaseTag string, latestStable string, err error) {
	client := github.NewClient(http.DefaultClient)
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

func Update(ctx context.Context, tag string) error {
	client := github.NewClient(http.DefaultClient)
	release, _, err := client.Repositories.GetReleaseByTag(ctx, repoOwner, repoName, tag)
	if err != nil {
		return err
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
			resp, err := http.Get(asset.GetBrowserDownloadURL())
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			buf := new(strings.Builder)

			var sha256Hash [32]byte
			if hashStr, ok := strings.CutPrefix(asset.GetDigest(), "sha256:"); ok {
				if _, err := hex.Decode(sha256Hash[:], []byte(hashStr)); err != nil {
					return err
				}
			}
			hasher := sha256.New()
			writer := io.MultiWriter(buf, hasher)
			if _, err = io.Copy(writer, resp.Body); err != nil {
				return err
			}
			if !bytes.Equal(sha256Hash[:], hasher.Sum(nil)) {
				return errors.New("checksum verification failed for checksums.txt")
			}
			lines := strings.SplitSeq(buf.String(), "\n")
			for line := range lines {
				parts := strings.Fields(line)
				if len(parts) == 2 && parts[1] == assetName {
					checkSum, err = hex.DecodeString(parts[0])
					if err != nil {
						return err
					}
					break
				}
			}
		}
	}
	if binaryAsset == nil {
		return errors.New("no suitable asset found for update")
	}
	resp, err := http.Get(binaryAsset.GetBrowserDownloadURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	hasher := sha256.New()
	reader := io.TeeReader(resp.Body, hasher)
	if err := selfupdate.Apply(reader, selfupdate.Options{
		Checksum: checkSum,
		Hash:     crypto.SHA256,
	}); err != nil {
		return err
	}
	var sha256Hash [32]byte
	if hashStr, ok := strings.CutPrefix(binaryAsset.GetDigest(), "sha256:"); ok {
		if _, err := hex.Decode(sha256Hash[:], []byte(hashStr)); err != nil {
			slog.Error("failed to decode binary asset digest", "error", err)
			return err
		}
		if !bytes.Equal(sha256Hash[:], hasher.Sum(nil)) {
			return errors.New("checksum verification failed for downloaded binary")
		}
	}
	return nil
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

	if osStr == "windows" {
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
