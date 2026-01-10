package profile

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type Profile struct {
	ID          uuid.UUID                    `json:"id"`
	Name        string                       `json:"name"`
	Author      string                       `json:"author"`
	Description string                       `json:"description,omitempty"`
	ModVersions map[string]modmgr.ModVersion `json:"mod_versions,omitempty"`
	LastUpdated time.Time                    `json:"last_updated"`
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
