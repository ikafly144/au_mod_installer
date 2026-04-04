package githubrelease

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v84/github"
)

// GetLatestReleaseIncludingPrereleases returns the latest non-draft release
// based on published time, including pre-releases.
func GetLatestReleaseIncludingPrereleases(
	ctx context.Context,
	client *github.Client,
	owner, repo string,
) (*github.RepositoryRelease, error) {
	opt := &github.ListOptions{
		PerPage: 100,
		Page:    1,
	}

	var latest *github.RepositoryRelease
	var latestPublishedAt time.Time

	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}

		for _, release := range releases {
			if release.GetDraft() {
				continue
			}

			publishedAt := releasePublishedAt(release)
			if latest == nil || publishedAt.After(latestPublishedAt) {
				latest = release
				latestPublishedAt = publishedAt
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if latest == nil {
		return nil, fmt.Errorf("no published releases found for %s/%s", owner, repo)
	}

	return latest, nil
}

func releasePublishedAt(release *github.RepositoryRelease) time.Time {
	if publishedAt := release.GetPublishedAt().Time; !publishedAt.IsZero() {
		return publishedAt
	}
	return release.GetCreatedAt().Time
}
