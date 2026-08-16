package versioning

import (
	"strings"

	"golang.org/x/mod/semver"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
)

// LatestVersionsFromTags returns the latest tag per branch based on semver ordering.
// Tags that are not valid semver are ignored.
func LatestVersionsFromTags(tags []string) map[Branch]string {
	latest := make(map[Branch]string)
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || !semver.IsValid(tag) {
			continue
		}
		prerelease := strings.TrimPrefix(semver.Prerelease(tag), "-")
		before, _, _ := strings.Cut(prerelease, ".")
		for b := BranchStable; b <= BranchDev; b++ {
			if !b.match(before) {
				continue
			}
			if current, ok := latest[b]; !ok || semver.Compare(tag, current) > 0 {
				latest[b] = tag
			}
		}
	}
	return latest
}

// FindBranchVersion searches VersionInfo for the specified branch name.
func FindBranchVersion(info *restcommon.VersionInfo, branch string) string {
	if info == nil {
		return ""
	}
	for _, b := range info.Branches {
		if strings.EqualFold(b.Name, branch) {
			return b.Version
		}
	}
	return ""
}
