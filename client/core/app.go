package core

import (
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
)

const (
	ProfileVersion = "v1"
)

type App struct {
	Version            string
	ConfigDir          string
	Rest               rest.Client
	ProfileManager     *profile.Manager
	EpicSessionManager *aumgr.EpicSessionManager
	EpicApi            *aumgr.EpicApi
}

func New(version string, restClient rest.Client) (*App, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}
	appConfigDir := filepath.Join(configDir, "au_mod_installer")
	profileManager, err := profile.NewManager(appConfigDir)
	if err != nil {
		if err := os.RemoveAll(appConfigDir); err != nil {
			return nil, fmt.Errorf("failed to remove profile path: %w", err)
		}
		profileManager, err = profile.NewManager(appConfigDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile manager after removal: %w", err)
		}
	}

	epicSessionManager, err := aumgr.NewEpicSessionManager(appConfigDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create epic session manager: %w", err)
	}

	return &App{
		Version:            version,
		ConfigDir:          appConfigDir,
		Rest:               restClient,
		ProfileManager:     profileManager,
		EpicSessionManager: epicSessionManager,
		EpicApi:            aumgr.NewEpicApi(),
	}, nil
}

func (a *App) DetectGamePath() (string, error) {
	return aumgr.GetAmongUsDir()
}

func (a *App) DetectLauncherType(path string) aumgr.LauncherType {
	return aumgr.DetectLauncherType(path)
}

func (a *App) ClearModCache() error {
	cacheDir := filepath.Join(a.ConfigDir, "mods")
	if _, err := os.Stat(cacheDir); err == nil {
		return os.RemoveAll(cacheDir)
	}
	return nil
}

func (a *App) HandleSharedProfile(uri string) (*profile.SharedProfile, error) {
	var ok bool
	if uri, ok = strings.CutPrefix(uri, "mod-of-us://profile/"); !ok {
		return nil, fmt.Errorf("invalid profile URI")
	}
	if uri, ok = strings.CutPrefix(uri, ProfileVersion+"/"); !ok {
		return nil, fmt.Errorf("invalid profile version")
	}

	reader, err := zlib.NewReader(base64.NewDecoder(base64.RawURLEncoding, strings.NewReader(uri)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode profile data: %w", err)
	}
	defer reader.Close()

	var prof profile.SharedProfile
	if err := json.NewDecoder(reader).Decode(&prof); err != nil {
		return nil, fmt.Errorf("failed to decode profile JSON: %w", err)
	}

	// Reset ID to avoid collision if it's a known one, but maybe better to let user decide?
	// For now, let's keep it but user should confirm import.
	return &prof, nil
}

func (a *App) HandleSharedProfileArchive(reader io.ReaderAt, size int64) (*profile.SharedProfile, []byte, error) {
	prof, iconPNG, err := profile.DecodeSharedArchive(reader, size)
	if err != nil {
		return nil, nil, err
	}
	return prof, iconPNG, nil
}

func (a *App) HandleSharedProfileArchiveFile(path string) (*profile.SharedProfile, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read profile archive: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat profile archive: %w", err)
	}

	return a.HandleSharedProfileArchive(file, stat.Size())
}

func (a *App) ExportProfile(prof profile.Profile) (string, error) {
	builder := &strings.Builder{}
	writer := zlib.NewWriter(base64.NewEncoder(base64.RawURLEncoding, builder))
	defer writer.Close()

	if err := json.NewEncoder(writer).Encode(prof.MakeShared()); err != nil {
		return "", err
	}
	if err := writer.Flush(); err != nil {
		return "", err
	}

	return "mod-of-us://profile/" + ProfileVersion + "/" + builder.String(), nil
}

func (a *App) ExportProfileArchive(prof profile.Profile, iconPNG []byte) ([]byte, error) {
	return profile.EncodeSharedArchive(prof.MakeShared(), iconPNG)
}

type JoinGameLink struct {
	SessionID  string
	ServerBase string
	Error      string
}

func (a *App) ParseJoinGameURI(uri string) (*JoinGameLink, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse join game URI: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "mod-of-us") || !strings.EqualFold(parsed.Host, "join_game") {
		return nil, fmt.Errorf("invalid join game URI")
	}
	path := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasPrefix(path, "v1/") {
		return nil, fmt.Errorf("unsupported join game URI version")
	}
	sessionID := strings.TrimPrefix(path, "v1/")
	values := parsed.Query()
	serverBase := strings.TrimSpace(values.Get("server"))
	if serverBase == "" {
		return nil, fmt.Errorf("join game URI missing server")
	}
	if parsedServer, err := url.Parse(serverBase); err != nil || parsedServer.Scheme == "" || parsedServer.Host == "" {
		return nil, fmt.Errorf("invalid join game URI server")
	}
	return &JoinGameLink{
		SessionID:  sessionID,
		ServerBase: serverBase,
		Error:      strings.TrimSpace(values.Get("error")),
	}, nil
}
