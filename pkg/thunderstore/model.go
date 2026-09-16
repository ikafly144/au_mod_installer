package thunderstore

import (
	"strings"
	"time"
)

type Package struct {
	Name           string           `json:"name"`
	FullName       string           `json:"full_name"`
	Owner          string           `json:"owner"`
	PackageURL     string           `json:"package_url"`
	DateCreated    time.Time        `json:"date_created"`
	DateUpdated    time.Time        `json:"date_updated"`
	UUID4          string           `json:"uuid4"`
	RatingScore    int              `json:"rating_score"`
	IsPinned       bool             `json:"is_pinned"`
	IsDeprecated   bool             `json:"is_deprecated"`
	HasNSFW        bool             `json:"has_nsfw_content"`
	Categories     []string         `json:"categories"`
	Versions       []PackageVersion `json:"versions"`
	TotalDownloads int              `json:"-"`
}

type PackageVersion struct {
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description"`
	Icon          string    `json:"icon"`
	VersionNumber string    `json:"version_number"`
	Dependencies  []string  `json:"dependencies"`
	DownloadURL   string    `json:"download_url"`
	Downloads     int       `json:"downloads"`
	DateCreated   time.Time `json:"date_created"`
	WebsiteURL    string    `json:"website_url"`
	IsActive      bool      `json:"is_active"`
	UUID4         string    `json:"uuid4"`
	FileSize      int64     `json:"file_size"`
}

// ParseDependency parses a Thunderstore dependency string (e.g. "BepInEx-BepInExPack-5.4.2100")
// into modID ("BepInEx-BepInExPack") and version ("5.4.2100").
func ParseDependency(dep string) (modID string, version string) {
	dep = strings.TrimSpace(dep)
	idx := strings.LastIndex(dep, "-")
	if idx <= 0 {
		return dep, ""
	}
	return dep[:idx], dep[idx+1:]
}
