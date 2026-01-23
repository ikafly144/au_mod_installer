package rest

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func NewFileClient(file string) (*FileClient, error) {
	return &FileClient{
		file: file,
	}, nil
}

type FileClient struct {
	file         string
	modStore     map[string]modmgr.Mod
	versionStore map[string]map[string][]modmgr.ModVersion
}

func (f *FileClient) LoadData() error {
	source, err := os.Open(f.file)
	if err != nil {
		return err
	}
	defer source.Close()

	type fileMod struct {
		modmgr.Mod
		Versions []modmgr.ModVersion `json:"versions"`
	}
	var fileMods []fileMod
	if err = json.NewDecoder(source).Decode(&fileMods); err != nil {
		return err
	}
	f.modStore = make(map[string]modmgr.Mod)
	f.versionStore = make(map[string]map[string][]modmgr.ModVersion)
	for _, m := range fileMods {
		f.modStore[m.ID] = m.Mod
		if _, ok := f.versionStore[m.ID]; !ok {
			f.versionStore[m.ID] = make(map[string][]modmgr.ModVersion)
		}
		f.versionStore[m.ID]["all"] = m.Versions
		for _, v := range m.Versions {
			f.versionStore[m.ID][v.ID] = []modmgr.ModVersion{v}
		}
	}

	slog.Info("mods loaded from file", "file", f.file, "count", len(f.modStore))

	return nil
}

var _ Client = (*FileClient)(nil)

func (f *FileClient) GetHealthStatus() (*rest.HealthStatus, error) {
	return &rest.HealthStatus{
		Status: "OK",
	}, nil
}

func (f *FileClient) GetModList(limit int, after string, before string) ([]modmgr.Mod, error) {
	var mods []modmgr.Mod
	for _, m := range f.modStore {
		mods = append(mods, m)
	}
	return mods, nil
}

func (f *FileClient) GetMod(modID string) (*modmgr.Mod, error) {
	m, ok := f.modStore[modID]
	if !ok {
		return nil, nil
	}
	return &m, nil
}

func (f *FileClient) GetModVersions(modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	versionsMap, ok := f.versionStore[modID]
	if !ok {
		return nil, nil
	}
	return versionsMap["all"], nil
}

func (f *FileClient) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	versionsMap, ok := f.versionStore[modID]
	if !ok {
		return nil, nil
	}
	versions, ok := versionsMap[versionID]
	if !ok || len(versions) == 0 {
		return nil, nil
	}
	return &versions[0], nil
}

func (f *FileClient) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	mod, ok := f.modStore[modID]
	if !ok || mod.LatestVersion == "" {
		return nil, nil
	}
	return f.GetModVersion(modID, mod.LatestVersion)
}
