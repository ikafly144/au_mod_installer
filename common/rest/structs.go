package rest

type HealthStatus struct {
	Status           string   `json:"status"`
	WorkingVersion   string   `json:"working_version,omitempty"`
	DisabledVersions []string `json:"disabled_versions,omitempty"`
}
