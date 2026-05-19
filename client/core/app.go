package core

import (
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/client/activity"
	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
	sdk "github.com/ikafly144/discord_social_sdk"
)

const (
	ProfileVersion                 = "v1"
	ProfileArchiveDownloadTimeout  = 30 * time.Second
	ProfileArchiveDownloadMaxBytes = int64(64 << 20)
)

type App struct {
	Version            string
	ConfigDir          string
	Rest               rest.Client
	ProfileManager     *profile.Manager
	EpicSessionManager *aumgr.EpicSessionManager
	EpicApi            *aumgr.EpicApi

	ActivityService *activity.ActivityService

	// Running profile state
	runningProfileMu   sync.Mutex
	runningProfileID   uuid.UUID
	launchingProfileID uuid.UUID
	launchingProfile   bool
	runningDirectJoin  bool
	runningGamePID     int
	runningStartedAt   time.Time
	lobbyPollStop      func()
	lobbyInfo          *IPCLobbyInfo

	// Callbacks for state changes
	OnGameStarted      func(profileID uuid.UUID, pid int)
	OnGameExited       func(profileID uuid.UUID)
	OnLobbyInfoUpdated func(info *IPCLobbyInfo)

	// Shared room state
	roomShareMu         sync.Mutex
	roomShareGenerating bool
	roomShareCache      SharedRoomLink
}

type SharedRoomLink struct {
	RoomKey   string
	URL       string
	SessionID string
	HostKey   string
	ExpiresAt time.Time
}

func (a *App) GetSharedRoom() SharedRoomLink {
	a.roomShareMu.Lock()
	defer a.roomShareMu.Unlock()
	return a.roomShareCache
}

func (a *App) SetSharedRoom(link SharedRoomLink) {
	a.roomShareMu.Lock()
	a.roomShareCache = link
	a.roomShareMu.Unlock()
}

func (a *App) SetRoomShareGenerating(generating bool) {
	a.roomShareMu.Lock()
	a.roomShareGenerating = generating
	a.roomShareMu.Unlock()
}

func (a *App) IsRoomShareGenerating() bool {
	a.roomShareMu.Lock()
	defer a.roomShareMu.Unlock()
	return a.roomShareGenerating
}

func (a *App) InvalidateCachedRoomShareAsync() {
	a.roomShareMu.Lock()
	cache := a.roomShareCache
	a.roomShareCache = SharedRoomLink{}
	a.roomShareMu.Unlock()
	if cache.SessionID == "" || cache.HostKey == "" {
		return
	}
	go func() {
		if err := a.Rest.DeleteSharedGame(cache.SessionID, cache.HostKey); err != nil {
			slog.Warn("Failed to invalidate shared room link", "error", err)
		}
	}()
}

func (a *App) GetLobbyInfo() *IPCLobbyInfo {
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	return a.lobbyInfo
}

func (a *App) StartActivityPolling(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				slog.Info("updating presence")
				a.updateRichPresence()
			}
		}
	}()
}

func (a *App) updateRichPresence() {
	if a.ActivityService == nil {
		return
	}

	a.runningProfileMu.Lock()
	profileID := a.runningProfileID
	lobby := a.lobbyInfo
	a.runningProfileMu.Unlock()

	if profileID == uuid.Nil {
		a.ActivityService.ClearActivity()
		return
	}

	prof, ok := a.ProfileManager.Get(profileID)
	if !ok {
		a.ActivityService.ClearActivity()
		return
	}

	act := sdk.NewActivity()
	act.SetType(sdk.ActivityTypePlaying)
	act.SetName("Mod of Us")
	act.SetDetails(fmt.Sprintf("Playing %s", prof.Name))

	if lobby != nil && lobby.IsConnected {
		if lobby.LobbyCode != "" {
			act.SetState(fmt.Sprintf("In Lobby (%s)", lobby.LobbyCode))
		} else {
			act.SetState("In Lobby")
		}
		p := sdk.NewActivityParty()
		p.SetID(lobby.MatchMakerIp + ":" + strconv.Itoa(lobby.MatchMakerPort) + "@" + lobby.LobbyCode)
		if lobby.MaxPlayers > 0 {
			p.SetMaxSize(lobby.MaxPlayers)
			p.SetCurrentSize(lobby.JoinedPlayers)
		}
		act.SetParty(p)
	} else {
		act.SetState("In Main Menu")
	}

	share := a.GetSharedRoom()
	if share.URL != "" && share.ExpiresAt.After(time.Now()) {
		secrets := sdk.NewActivitySecrets()
		secrets.SetJoin(share.URL)
		act.SetSecrets(secrets)
	}

	a.ActivityService.SetActivity(act, func(et sdk.ErrorType) {
		if et != sdk.ErrorTypeNone {
			slog.Warn("Failed to set activity", "error", et)
		}
	})
}

func New(version string, restClient rest.Client, activityService *activity.ActivityService) (*App, error) {
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

	a := &App{
		Version:            version,
		ConfigDir:          appConfigDir,
		Rest:               restClient,
		ProfileManager:     profileManager,
		EpicSessionManager: epicSessionManager,
		EpicApi:            aumgr.NewEpicApi(),
		ActivityService:    activityService,
	}

	return a, nil
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

func (a *App) DownloadArchiveURLToTempFile(archiveURL string, progressListener progress.Progress) (string, error) {
	parsedURL, err := url.Parse(archiveURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse archive URL: %w", err)
	}
	if !strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https") {
		return "", fmt.Errorf("unsupported archive URL scheme: %s", parsedURL.Scheme)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ProfileArchiveDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create archive request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download archive: unexpected status %s", resp.Status)
	}
	if resp.ContentLength > ProfileArchiveDownloadMaxBytes {
		return "", fmt.Errorf("archive is too large: %d bytes (max %d)", resp.ContentLength, ProfileArchiveDownloadMaxBytes)
	}
	if progressListener != nil {
		progressListener.SetValue(0)
		progressListener.Start()
		defer progressListener.Done()
	}

	tempFile, err := os.CreateTemp("", "mod-of-us-profile-url-*.aupack")
	if err != nil {
		return "", fmt.Errorf("failed to create temp archive file: %w", err)
	}
	tempPath := tempFile.Name()
	buf := progress.NewProgressWriter(0, 1, resp.ContentLength, progressListener, tempFile)
	written, copyErr := io.Copy(buf, io.LimitReader(resp.Body, ProfileArchiveDownloadMaxBytes+1))
	buf.Complete()
	closeErr := tempFile.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("failed to save downloaded archive: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("failed to finalize downloaded archive: %w", closeErr)
	}
	if written > ProfileArchiveDownloadMaxBytes {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("archive is too large: more than %d bytes", ProfileArchiveDownloadMaxBytes)
	}
	return tempPath, nil
}

func (a *App) HandleJoinGameDownload(sessionID string, serverBase string) (*profile.SharedProfile, []byte, *LaunchJoinInfo, error) {
	client := rest.NewClient(serverBase)
	rs, err := client.GetJoinGameDownload(sessionID)
	if err != nil {
		return nil, nil, nil, err
	}

	tmpFile, err := os.CreateTemp("", "mod-of-us-join-*.aupack")
	if err != nil {
		return nil, nil, nil, err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(rs.Aupack); err != nil {
		_ = tmpFile.Close()
		return nil, nil, nil, err
	}
	stat, err := tmpFile.Stat()
	if err != nil {
		_ = tmpFile.Close()
		return nil, nil, nil, err
	}
	shared, iconPNG, err := a.HandleSharedProfileArchive(tmpFile, stat.Size())
	_ = tmpFile.Close()
	if err != nil {
		return nil, nil, nil, err
	}
	joinInfo := &LaunchJoinInfo{
		LobbyCode:      rs.Room.LobbyCode,
		ServerIP:       rs.Room.ServerIP,
		ServerPort:     rs.Room.ServerPort,
		MatchMakerIp:   rs.Room.MatchMakerIp,
		MatchMakerPort: rs.Room.MatchMakerPort,
	}
	return shared, iconPNG, joinInfo, nil
}

func (a *App) HandleImportReader(reader io.Reader, extension string) (*profile.SharedProfile, []byte, error) {
	if !strings.EqualFold(extension, ".aupack") {
		return nil, nil, fmt.Errorf("unsupported file extension: %s", extension)
	}

	tempFile, err := os.CreateTemp("", "mod-of-us-profile-*.aupack")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if _, err := io.Copy(tempFile, reader); err != nil {
		_ = tempFile.Close()
		return nil, nil, fmt.Errorf("failed to save archive: %w", err)
	}

	stat, err := tempFile.Stat()
	if err != nil {
		_ = tempFile.Close()
		return nil, nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	shared, iconPNG, err := a.HandleSharedProfileArchive(tempFile, stat.Size())
	_ = tempFile.Close()
	return shared, iconPNG, err
}

func (a *App) ImportSharedProfile(shared *profile.SharedProfile, iconPNG []byte) (*profile.Profile, error) {
	prof := profile.Profile{
		ID:          shared.ID,
		Name:        shared.Name,
		Author:      shared.Author,
		Description: shared.Description,
		UpdatedAt:   time.Now(),
	}

	if p, ok := a.ProfileManager.Get(shared.ID); ok {
		prof.PlayDurationNS = p.PlayDurationNS
		prof.LastLaunchedAt = p.LastLaunchedAt
	}

	// Fetch mod version infos
	for modID, versionID := range shared.ModVersions {
		info, err := a.Rest.GetModVersion(modID, versionID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch mod version info for %s:%s: %w", modID, versionID, err)
		}
		prof.AddModVersion(*info)
	}

	if err := a.ProfileManager.Add(prof); err != nil {
		return nil, err
	}
	if len(iconPNG) > 0 {
		if err := a.ProfileManager.SaveIconPNG(prof.ID, iconPNG); err != nil {
			return nil, err
		}
	}
	return &prof, nil
}

type JoinGameLink struct {
	SessionID  string
	ServerBase string
	Error      string
}

func (a *App) ParseJoinGameURI(uri string) (*JoinGameLink, error) {
	slog.Info("parsing join game URI", "uri", uri)
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
