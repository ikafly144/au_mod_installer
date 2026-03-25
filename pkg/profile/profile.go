package profile

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type Profile struct {
	ID             uuid.UUID                    `json:"id"`
	Name           string                       `json:"name"`
	Author         string                       `json:"author"`
	Description    string                       `json:"description,omitempty"`
	ModVersions    map[string]modmgr.ModVersion `json:"mod_versions,omitempty"`
	UpdatedAt      time.Time                    `json:"updated_at"`
	PlayDurationNS int64                        `json:"play_duration_ns,omitempty"`
	LastLaunchedAt time.Time                    `json:"last_launched_at,omitempty"`
}

const SharedProfileVersion = "1"

type SharedProfile struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Author      string            `json:"author"`
	Description string            `json:"description,omitempty"`
	ModVersions map[string]string `json:"mod_versions,omitempty"` // map of mod ID to version ID
	UpdatedAt   time.Time         `json:"updated_at"`
}

func (p *Profile) Versions() []modmgr.ModVersion {
	versions := make([]modmgr.ModVersion, 0, len(p.ModVersions))
	for _, v := range p.ModVersions {
		versions = append(versions, v)
	}
	slices.SortFunc(versions, func(a, b modmgr.ModVersion) int {
		return strings.Compare(a.ModID, b.ModID)
	})
	return versions
}

func (p *Profile) AddModVersion(version modmgr.ModVersion) {
	if p.ModVersions == nil {
		p.ModVersions = make(map[string]modmgr.ModVersion)
	}
	p.ModVersions[version.ModID] = version
}

func (p *Profile) RemoveModVersion(modID string) {
	delete(p.ModVersions, modID)
}

func (p *Profile) MakeShared() SharedProfile {
	shared := SharedProfile{
		ID:          p.ID,
		Name:        p.Name,
		Author:      p.Author,
		Description: p.Description,
		UpdatedAt:   p.UpdatedAt,
		ModVersions: make(map[string]string, len(p.ModVersions)),
	}

	for modID, version := range p.ModVersions {
		shared.ModVersions[modID] = version.ID
	}

	return shared
}

func (p *Profile) PlayDuration() time.Duration {
	return time.Duration(p.PlayDurationNS)
}

func (p *Profile) AddPlayDuration(d time.Duration) {
	if d <= 0 {
		return
	}
	p.PlayDurationNS += int64(d)
}
