package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ikafly144/modrepo/common/rest"
	"github.com/ikafly144/modrepo/common/rest/model"
	"github.com/ikafly144/modrepo/pkg/modmgr"
	"github.com/ikafly144/modrepo/pkg/thunderstore"
)

type ThunderstoreClient struct {
	tsClient *thunderstore.Client
}

func NewThunderstoreClient(cacheDir string, httpClient *http.Client) *ThunderstoreClient {
	return &ThunderstoreClient{
		tsClient: thunderstore.NewClient(cacheDir, httpClient),
	}
}

func (c *ThunderstoreClient) ThunderstoreClient() *thunderstore.Client {
	return c.tsClient
}

func (c *ThunderstoreClient) ServerBaseURL() string {
	return thunderstore.DefaultCommunityURL
}

func (c *ThunderstoreClient) GetHealthStatus() (*rest.HealthStatus, error) {
	return &rest.HealthStatus{Status: "OK"}, nil
}

func (c *ThunderstoreClient) GetVersionInfo() (*rest.VersionInfo, error) {
	return &rest.VersionInfo{
		Branches: []rest.BranchInfo{{Name: "main", Version: "1.0.0"}},
	}, nil
}

func (c *ThunderstoreClient) RefreshPackages(ctx context.Context) error {
	return c.tsClient.FetchPackages(ctx)
}

func (c *ThunderstoreClient) GetModIDs(limit int, after string, before string) ([]string, error) {
	pkgs := c.tsClient.GetAllPackages()
	ids := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		ids = append(ids, p.FullName)
	}
	return ids, nil
}

func (c *ThunderstoreClient) GetMod(modID string) (*modmgr.Mod, error) {
	pkg, ok := c.tsClient.GetPackage(modID)
	if !ok {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}
	return packageToMod(pkg), nil
}

func (c *ThunderstoreClient) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	pkg, ok := c.tsClient.GetPackage(modID)
	if !ok {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}
	ids := make([]string, 0, len(pkg.Versions))
	for _, v := range pkg.Versions {
		ids = append(ids, v.VersionNumber)
	}
	return ids, nil
}

func (c *ThunderstoreClient) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	v, ok := c.tsClient.GetPackageVersion(modID, versionID)
	if !ok {
		return nil, fmt.Errorf("mod version not found: %s@%s", modID, versionID)
	}
	return packageVersionToModVersion(v, modID), nil
}

func (c *ThunderstoreClient) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	v, ok := c.tsClient.GetLatestVersion(modID)
	if !ok {
		return nil, fmt.Errorf("latest version not found: %s", modID)
	}
	return packageVersionToModVersion(v, modID), nil
}

func (c *ThunderstoreClient) GetModThumbnail(modID string) ([]byte, error) {
	pkg, ok := c.tsClient.GetPackage(modID)
	if !ok || len(pkg.Versions) == 0 || pkg.Versions[0].Icon == "" {
		return nil, fmt.Errorf("no icon for %s", modID)
	}
	return c.tsClient.GetPackageIcon(context.Background(), pkg.Versions[0].Icon)
}

func (c *ThunderstoreClient) CheckForUpdates(installed map[string]string) (map[string]*modmgr.ModVersion, error) {
	updates := make(map[string]*modmgr.ModVersion)
	for modID, currentVer := range installed {
		latest, err := c.GetLatestModVersion(modID)
		if err != nil {
			continue
		}
		if latest != nil && latest.VersionID != currentVer {
			updates[modID] = latest
		}
	}
	return updates, nil
}

func (c *ThunderstoreClient) SearchMods(query string, category string, sortBy string) ([]*modmgr.Mod, error) {
	pkgs := c.tsClient.SearchPackages(query, category, sortBy)
	mods := make([]*modmgr.Mod, 0, len(pkgs))
	for _, p := range pkgs {
		mods = append(mods, packageToMod(p))
	}
	return mods, nil
}

func (c *ThunderstoreClient) GetCategories() []string {
	return c.tsClient.GetCategories()
}

func packageToMod(p *thunderstore.Package) *modmgr.Mod {
	m := &modmgr.Mod{
		ModDetails: model.ModDetails{
			ID:             p.FullName,
			Name:           p.Name,
			Author:         p.Owner,
			PackageURL:     p.PackageURL,
			RatingScore:    p.RatingScore,
			IsPinned:       p.IsPinned,
			IsDeprecated:   p.IsDeprecated,
			Categories:     p.Categories,
			TotalDownloads: p.TotalDownloads,
			CreatedAt:      p.DateCreated,
			UpdatedAt:      p.DateUpdated,
		},
	}
	if len(p.Versions) > 0 {
		m.LatestVersionID = p.Versions[0].VersionNumber
		m.Description = p.Versions[0].Description
		m.IconURL = p.Versions[0].Icon
	}
	return m
}

func packageVersionToModVersion(v *thunderstore.PackageVersion, modID string) *modmgr.ModVersion {
	deps := make([]model.ModVersionDependency, 0, len(v.Dependencies))
	for _, depStr := range v.Dependencies {
		dModID, dVer := thunderstore.ParseDependency(depStr)
		deps = append(deps, model.ModVersionDependency{
			ModID:          dModID,
			VersionID:      dVer,
			DependencyType: model.DependencyTypeRequired,
		})
	}

	return &modmgr.ModVersion{
		ModVersionDetails: model.ModVersionDetails{
			VersionID:    v.VersionNumber,
			ModID:        modID,
			Dependencies: deps,
			DownloadURL:  v.DownloadURL,
			FileSize:     v.FileSize,
			IconURL:      v.Icon,
			Description:  v.Description,
			CreatedAt:    v.DateCreated,
			UpdatedAt:    v.DateCreated,
			Files: []model.ModVersionFile{
				{
					ID:             v.UUID4,
					Filename:       v.FullName + ".zip",
					ContentType:    model.ContentTypeArchive,
					Size:           v.FileSize,
					Downloads:      []string{v.DownloadURL},
					TargetPlatform: model.TargetPlatformAny,
					CreatedAt:      v.DateCreated,
				},
			},
		},
	}
}
