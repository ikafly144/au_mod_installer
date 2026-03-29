package rest

import (
	"encoding/json"
	"fmt"
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

func (f *FileClient) ServerBaseURL() string {
	return ""
}

func (f *FileClient) GetHealthStatus() (*rest.HealthStatus, error) {
	return &rest.HealthStatus{
		Status: "OK",
	}, nil
}

func (f *FileClient) GetModIDs(limit int, after string, before string) ([]string, error) {
	var modIDs []string
	for _, m := range f.modStore {
		modIDs = append(modIDs, m.ID)
	}
	return modIDs, nil
}

func (f *FileClient) GetMod(modID string) (*modmgr.Mod, error) {
	m, ok := f.modStore[modID]
	if !ok {
		return nil, nil
	}
	return &m, nil
}

func (f *FileClient) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	versionsMap, ok := f.versionStore[modID]
	if !ok {
		return nil, nil
	}
	var versionIDs []string
	for _, v := range versionsMap["all"] {
		versionIDs = append(versionIDs, v.ID)
	}
	return versionIDs, nil
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
	if !ok || mod.LatestVersionID == "" {
		return nil, nil
	}
	return f.GetModVersion(modID, mod.LatestVersionID)
}

func (f *FileClient) CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error) {
	updates := make(map[string]*modmgr.ModVersion)
	for modID, currentVersion := range installedVersions {
		latest, err := f.GetLatestModVersion(modID)
		if err != nil {
			continue
		}
		if latest != nil && latest.ID != currentVersion {
			updates[modID] = latest
		}
	}
	return updates, nil
}

func (f *FileClient) GetModThumbnail(modID string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FileClient) ShareGame(aupack []byte, room rest.RoomInfo) (*rest.ShareGameResponse, error) {
	return nil, fmt.Errorf("local mode: share game not available")
}

func (f *FileClient) DeleteSharedGame(sessionID, hostKey string) error {
	return fmt.Errorf("local mode: delete shared game not available")
}

func (f *FileClient) GetJoinGameDownload(sessionID string) (*rest.JoinGameDownloadResponse, error) {
	return nil, fmt.Errorf("local mode: join game download not available")
}
