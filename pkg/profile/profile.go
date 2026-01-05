package profile

import (
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type Profile struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Versions    []modmgr.ModVersion `json:"versions"`
	LastUpdated time.Time           `json:"last_updated"`
}
