package versioning

// repository information
import (
	"context"
	"crypto"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-github/v80/github"
	"github.com/minio/selfupdate"
	"golang.org/x/mod/semver"
)

var (
	repoOwner    = "ikafly144"
	repoName     = "au_mod_installer"
	artifactName = "mod-of-us_${OS}_${ARCH}"
)

func CheckForUpdates(ctx context.Context, branch Branch, currentVersion string) (releaseTag string, err error) {
	client := github.NewClient(http.DefaultClient)
	opt := &github.ListOptions{
		PerPage: 10,
		Page:    1,
	}
	for {
		tags, resp, err := client.Repositories.ListTags(ctx, repoOwner, repoName, opt)
		if err != nil {
			return "", err
		}
		for _, tag := range tags {
			slog.Info("found tag", "tag", tag.GetName())
			if before, _, _ := strings.Cut(strings.TrimPrefix(semver.Prerelease(tag.GetName()), "-"), "."); before != "" && !branch.match(before) {
				slog.Info("skipping tag due to prerelease branch mismatch", "tag", tag.GetName(), "branch", branch)
				continue
			}
			release, _, err := client.Repositories.GetReleaseByTag(ctx, repoOwner, repoName, tag.GetName())
			if err != nil {
				return "", err
			}
			if semver.Compare(release.GetTagName(), currentVersion) <= 0 {
				slog.Info("no newer version found", "current", currentVersion, "found", release.GetTagName())
				return "", nil
			}
			if release.GetTagName() != currentVersion {
				return release.GetTagName(), nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return "", nil
}

func Update(ctx context.Context, tag string) error {
	client := github.NewClient(http.DefaultClient)
	release, _, err := client.Repositories.GetReleaseByTag(ctx, repoOwner, repoName, tag)
	if err != nil {
		return err
	}

	assetName := artifactName
	assetName = replaceOSAndArch(assetName)
	var checkSum []byte
	var binaryAsset *github.ReleaseAsset
	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			binaryAsset = asset
			break
		}
		if asset.GetName() == "checksums.txt" {
			resp, err := http.Get(asset.GetBrowserDownloadURL())
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			buf := new(strings.Builder)
			if _, err = io.Copy(buf, resp.Body); err != nil {
				return err
			}
			lines := strings.Split(buf.String(), "\n")
			for _, line := range lines {
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
	if binaryAsset != nil {
		resp, err := http.Get(binaryAsset.GetBrowserDownloadURL())
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return selfupdate.Apply(resp.Body, selfupdate.Options{
			Checksum: checkSum,
			Hash:     crypto.SHA256,
		})
	}
	return errors.New("no suitable asset found for update")
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
