package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v88/github"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/common/versioning"
)

const (
	defaultVersionInfoTTL = 10 * time.Minute
	repoOwner             = "ikafly144"
	repoName              = "au_mod_installer"
)

var errVersionInfoNotModified = errors.New("version info not modified")

type VersionInfoProvider interface {
	GetVersionInfo(ctx context.Context) (*restcommon.VersionInfo, error)
}

type VersionInfoOptions struct {
	HTTPClient *http.Client
	Token      string
	TTL        time.Duration
}

type VersionInfoService struct {
	githubClient *github.Client
	ttl          time.Duration

	mu       sync.Mutex
	cached   *restcommon.VersionInfo
	cachedAt time.Time
	etag     string
}

func NewVersionInfoService(opts VersionInfoOptions) *VersionInfoService {
	var githubOpts []github.ClientOptionsFunc
	if opts.HTTPClient != nil {
		githubOpts = append(githubOpts, github.WithHTTPClient(opts.HTTPClient))
	}
	if opts.Token != "" {
		githubOpts = append(githubOpts, github.WithAuthToken(opts.Token))
	}

	client, err := github.NewClient(githubOpts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create GitHub client: %v", err))
	}

	ttl := opts.TTL
	if ttl <= 0 {
		ttl = defaultVersionInfoTTL
	}
	return &VersionInfoService{
		githubClient: client,
		ttl:          ttl,
	}
}

func (s *VersionInfoService) GetVersionInfo(ctx context.Context) (*restcommon.VersionInfo, error) {
	now := time.Now()
	s.mu.Lock()
	if s.cached != nil && now.Sub(s.cachedAt) < s.ttl {
		info := s.cached
		s.mu.Unlock()
		return info, nil
	}
	etag := s.etag
	s.mu.Unlock()

	info, newETag, err := s.fetchVersionInfo(ctx, etag)
	if err != nil {
		if errors.Is(err, errVersionInfoNotModified) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.cached == nil {
				return nil, fmt.Errorf("version info not modified but cache is empty")
			}
			s.cachedAt = now
			return s.cached, nil
		}
		return nil, err
	}

	s.mu.Lock()
	s.cached = info
	s.cachedAt = now
	s.etag = newETag
	s.mu.Unlock()
	return info, nil
}

func (s *VersionInfoService) fetchVersionInfo(ctx context.Context, etag string) (*restcommon.VersionInfo, string, error) {
	u := fmt.Sprintf("repos/%s/%s/releases?per_page=100", repoOwner, repoName)
	req, err := s.githubClient.NewRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	var releases []*github.RepositoryRelease
	resp, err := s.githubClient.Do(req, &releases)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotModified {
			return nil, etag, errVersionInfoNotModified
		}
		return nil, "", err
	}

	tags := make([]string, 0, len(releases))
	for _, release := range releases {
		if release.GetTagName() == "" {
			continue
		}
		tags = append(tags, release.GetTagName())
	}

	latest := versioning.LatestVersionsFromTags(tags)
	branches := make([]restcommon.BranchInfo, 0, int(versioning.BranchDev)+1)
	for b := versioning.BranchStable; b <= versioning.BranchDev; b++ {
		branches = append(branches, restcommon.BranchInfo{
			Name:    b.String(),
			Version: latest[b],
		})
	}

	info := &restcommon.VersionInfo{
		Branches: branches,
	}
	return info, resp.Header.Get("ETag"), nil
}
