package versioning

import (
	"strings"

	"golang.org/x/mod/semver"
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
