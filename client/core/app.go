package core

import (
	"compress/zlib"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2/lang"
	"github.com/google/uuid"

	sdk "github.com/ikafly144/discord_social_sdk"

	"github.com/ikafly144/au_mod_installer/client/discord"
	"github.com/ikafly144/au_mod_installer/client/rest"
	commonrest "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
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

	DiscordService *discord.DiscordService

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

	// Shared lobby state
	lobbyShareMu         sync.Mutex
	lobbyShareGenerating bool
	lobbyShareCache      SharedLobbyLink
}

type SharedRoomLink struct {
	RoomKey   string
	URL       string
	SessionID string
	HostKey   string
	ExpiresAt time.Time
	InFlight  bool
}

type SharedLobbyLink struct {
	URL       string
	SessionID string
	HostKey   string
	ExpiresAt time.Time
	InFlight  bool
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

func (a *App) GetSharedLobby() SharedLobbyLink {
	a.lobbyShareMu.Lock()
	defer a.lobbyShareMu.Unlock()
	return a.lobbyShareCache
}

func (a *App) SetSharedLobby(link SharedLobbyLink) {
	a.lobbyShareMu.Lock()
	a.lobbyShareCache = link
	a.lobbyShareMu.Unlock()
}

func (a *App) SetLobbyShareGenerating(generating bool) {
	a.lobbyShareMu.Lock()
	a.lobbyShareGenerating = generating
	a.lobbyShareMu.Unlock()
}

func (a *App) IsLobbyShareGenerating() bool {
	a.lobbyShareMu.Lock()
	defer a.lobbyShareMu.Unlock()
	return a.lobbyShareGenerating
}

func (a *App) InvalidateCachedLobbyShareAsync() {
	a.lobbyShareMu.Lock()
	cache := a.lobbyShareCache
	a.lobbyShareCache = SharedLobbyLink{}
	a.lobbyShareMu.Unlock()
	if cache.SessionID == "" || cache.HostKey == "" {
		return
	}
	go func() {
		if err := a.Rest.DeleteSharedLobby(cache.SessionID, cache.HostKey); err != nil {
			slog.Warn("Failed to invalidate shared lobby link", "error", err)
		}
	}()
}

func (a *App) HeartbeatLobbyShareAsync() {
	a.lobbyShareMu.Lock()
	if a.lobbyShareCache.InFlight || a.lobbyShareCache.SessionID == "" || a.lobbyShareCache.HostKey == "" {
		a.lobbyShareMu.Unlock()
		return
	}
	cache := a.lobbyShareCache
	a.lobbyShareMu.Unlock()

	// Only heartbeat if it expires within 30 minutes
	if cache.ExpiresAt.After(time.Now().Add(30 * time.Minute)) {
		return
	}

	a.lobbyShareMu.Lock()
	if a.lobbyShareCache.SessionID != cache.SessionID || a.lobbyShareCache.InFlight {
		a.lobbyShareMu.Unlock()
		return
	}
	a.lobbyShareCache.InFlight = true
	a.lobbyShareMu.Unlock()

	go func() {
		defer func() {
			a.lobbyShareMu.Lock()
			if a.lobbyShareCache.SessionID == cache.SessionID {
				a.lobbyShareCache.InFlight = false
			}
			a.lobbyShareMu.Unlock()
		}()

		rs, err := a.Rest.UpdateSharedLobbyExpiration(cache.SessionID, cache.HostKey)
		if err != nil {
			slog.Warn("Failed to heartbeat shared lobby link", "error", err)
			return
		}
		a.lobbyShareMu.Lock()
		if a.lobbyShareCache.SessionID == cache.SessionID {
			a.lobbyShareCache.ExpiresAt = rs.ExpiresAt
		}
		a.lobbyShareMu.Unlock()
	}()
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

func (a *App) HeartbeatRoomShareAsync() {
	a.roomShareMu.Lock()
	if a.roomShareCache.InFlight || a.roomShareCache.SessionID == "" || a.roomShareCache.HostKey == "" {
		a.roomShareMu.Unlock()
		return
	}
	cache := a.roomShareCache
	a.roomShareMu.Unlock()

	// Check if current room matches
	a.runningProfileMu.Lock()
	profileID := a.runningProfileID
	lobby := a.lobbyInfo
	a.runningProfileMu.Unlock()

	room, ok := a.CurrentRoomInfo(lobby)
	if !ok {
		return
	}
	roomKey := RoomKeyForCache(room, profileID)
	if cache.RoomKey != roomKey {
		return
	}

	// Only heartbeat if it expires within 30 minutes
	if cache.ExpiresAt.After(time.Now().Add(30 * time.Minute)) {
		return
	}

	a.roomShareMu.Lock()
	if a.roomShareCache.SessionID != cache.SessionID || a.roomShareCache.InFlight {
		a.roomShareMu.Unlock()
		return
	}
	a.roomShareCache.InFlight = true
	a.roomShareMu.Unlock()

	go func() {
		defer func() {
			a.roomShareMu.Lock()
			if a.roomShareCache.SessionID == cache.SessionID {
				a.roomShareCache.InFlight = false
			}
			a.roomShareMu.Unlock()
		}()

		rs, err := a.Rest.UpdateSharedGameExpiration(cache.SessionID, cache.HostKey)
		if err != nil {
			slog.Warn("Failed to heartbeat shared room link", "error", err)
			return
		}
		a.roomShareMu.Lock()
		if a.roomShareCache.SessionID == cache.SessionID {
			a.roomShareCache.ExpiresAt = rs.ExpiresAt
		}
		a.roomShareMu.Unlock()
	}()
}

func RoomKeyForCache(room commonrest.RoomInfo, profileID uuid.UUID) string {
	return strings.ToUpper(strings.TrimSpace(room.LobbyCode)) + "|" + strings.TrimSpace(room.ServerIP) + "|" + fmt.Sprint(room.ServerPort) + "|" + strings.TrimSpace(room.GameVersion) + "|" + profileID.String()
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
				a.updateRichPresence()
			}
		}
	}()
}

func (a *App) updateRichPresence() {
	if a.DiscordService == nil || !a.DiscordService.IsLoggedIn() {
		return
	}

	a.runningProfileMu.Lock()
	profileID := a.runningProfileID
	lobby := a.lobbyInfo
	runningStartedAt := a.runningStartedAt
	a.runningProfileMu.Unlock()

	lobbyShare := a.GetSharedLobby()
	activeLobby, hasActiveLobby := a.DiscordService.GetActiveLobby()

	// 1. If game is running
	if profileID != uuid.Nil {
		prof, ok := a.ProfileManager.Get(profileID)
		if !ok {
			a.DiscordService.ClearActivity()
			return
		}

		act := sdk.NewActivity()
		act.SetType(sdk.ActivityTypesPlaying)
		act.SetName("Mod of Us")
		act.SetDetails(fmt.Sprintf("Playing %s", prof.Name))
		act.SetSupportedPlatforms(sdk.ActivityGamePlatformsDesktop)

		assets := sdk.NewActivityAssets()
		assets.SetLargeImage("icon")
		if a.Version != "" {
			assets.SetLargeText(fmt.Sprintf("Mod of Us %s", a.Version))
		} else {
			assets.SetLargeText("Mod of Us")
		}
		act.SetAssets(assets)

		if !runningStartedAt.IsZero() {
			timestamp := sdk.NewActivityTimestamps()
			timestamp.SetStart(uint64(runningStartedAt.UnixMilli()))
			act.SetTimestamps(timestamp)
		}

		partyID := ""
		currentSize := int32(1)
		maxSize := int32(10)

		if lobby != nil && lobby.IsConnected {
			if lobby.GameState == "Started" {
				act.SetState(lang.LocalizeKey("discord.status.in_game", "In Game"))
			} else {
				act.SetState(lang.LocalizeKey("discord.status.in_lobby", "In Lobby"))
			}
			if lobby.MaxPlayers > 0 && lobby.JoinedPlayers > 0 {
				currentSize = int32(lobby.JoinedPlayers)
				maxSize = int32(lobby.MaxPlayers)
				partyID = strings.ToLower(lobby.GameState) + "/" + hex.EncodeToString(new(sha256.Sum256([]byte(lobby.MatchMakerIp + ":" + strconv.Itoa(lobby.MatchMakerPort) + "@" + lobby.LobbyCode)))[:])
			}
		} else {
			act.SetState(lang.LocalizeKey("discord.status.in_main_menu", "In Main Menu"))
		}

		joinSecret := ""
		if lobbyShare.URL != "" && lobbyShare.ExpiresAt.After(time.Now()) {
			joinSecret = lobbyShare.URL
			if partyID == "" {
				partyID = "lobby-" + lobbyShare.SessionID
			}
			a.HeartbeatLobbyShareAsync()
		} else if hasActiveLobby && activeLobby.Secret != "" {
			joinSecret = activeLobby.Secret
			if partyID == "" {
				partyID = fmt.Sprintf("discord-lobby-%d", activeLobby.ID)
			}
			if len(activeLobby.Members) > 0 {
				currentSize = int32(len(activeLobby.Members))
			}
		} else {
			share := a.GetSharedRoom()
			if lobby != nil && lobby.GameState == "Joined" && share.URL != "" && share.ExpiresAt.After(time.Now()) {
				joinSecret = share.URL
				a.HeartbeatRoomShareAsync()
			}
		}

		if partyID != "" || joinSecret != "" {
			if partyID == "" {
				partyID = "mod-of-us-party"
			}
			p := sdk.NewActivityParty()
			p.SetId(partyID)
			p.SetCurrentSize(currentSize)
			p.SetMaxSize(maxSize)
			p.SetPrivacy(sdk.ActivityPartyPrivacyPublic)
			act.SetParty(p)
		}

		if joinSecret != "" {
			secrets := sdk.NewActivitySecrets()
			secrets.SetJoin(joinSecret)
			act.SetSecrets(secrets)
		}

		a.DiscordService.SetActivity(act, func(d *sdk.ClientResult) {
			if !d.Successful() {
				slog.Warn("Failed to update Discord activity", "error", d.ErrorCode())
			}
		})
		return
	}

	// 2. If game is NOT running, but user has an active or shared lobby in launcher
	if (lobbyShare.URL != "" && lobbyShare.ExpiresAt.After(time.Now())) || hasActiveLobby {
		act := sdk.NewActivity()
		act.SetType(sdk.ActivityTypesPlaying)
		act.SetName("Mod of Us")
		act.SetState(lang.LocalizeKey("discord.status.in_lobby", "In Lobby"))
		act.SetDetails(lang.LocalizeKey("launcher.lobby.title", "Lobby"))
		act.SetSupportedPlatforms(sdk.ActivityGamePlatformsDesktop)

		assets := sdk.NewActivityAssets()
		assets.SetLargeImage("icon")
		if a.Version != "" {
			assets.SetLargeText(fmt.Sprintf("Mod of Us %s", a.Version))
		} else {
			assets.SetLargeText("Mod of Us")
		}
		act.SetAssets(assets)

		partyID := "lobby"
		currentSize := int32(1)
		maxSize := int32(10)
		joinSecret := ""

		if lobbyShare.URL != "" && lobbyShare.ExpiresAt.After(time.Now()) {
			joinSecret = lobbyShare.URL
			partyID = "lobby-" + lobbyShare.SessionID
			a.HeartbeatLobbyShareAsync()
		}
		if hasActiveLobby {
			if joinSecret == "" && activeLobby.Secret != "" {
				joinSecret = activeLobby.Secret
			}
			if partyID == "lobby" && activeLobby.ID != 0 {
				partyID = fmt.Sprintf("discord-lobby-%d", activeLobby.ID)
			}
			if len(activeLobby.Members) > 0 {
				currentSize = int32(len(activeLobby.Members))
			}
		}

		p := sdk.NewActivityParty()
		p.SetId(partyID)
		p.SetCurrentSize(currentSize)
		p.SetMaxSize(maxSize)
		p.SetPrivacy(sdk.ActivityPartyPrivacyPublic)
		act.SetParty(p)

		if joinSecret != "" {
			secrets := sdk.NewActivitySecrets()
			secrets.SetJoin(joinSecret)
			act.SetSecrets(secrets)
		}

		a.DiscordService.SetActivity(act, func(d *sdk.ClientResult) {
			if !d.Successful() {
				slog.Warn("Failed to update Discord activity for launcher lobby", "error", d.ErrorCode())
			}
		})
		return
	}

	a.DiscordService.ClearActivity()
	a.InvalidateCachedRoomShareAsync()
}

func ComputeProfileHash(prof profile.Profile) string {
	keys := make([]string, 0, len(prof.ModVersions))
	for modID := range prof.ModVersions {
		keys = append(keys, modID)
	}
	slices.Sort(keys)
	builder := strings.Builder{}
	for _, modID := range keys {
		builder.WriteString(modID)
		builder.WriteString(":")
		builder.WriteString(prof.ModVersions[modID].VersionID)
		builder.WriteString(";")
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:8])
}

func (a *App) CheckProfileCompatibility(profileID uuid.UUID, lobbyMeta map[string]string) bool {
	if lobbyMeta == nil {
		return true
	}
	expectedHash := lobbyMeta["profile_hash"]
	if expectedHash == "" {
		return true
	}
	prof, ok := a.ProfileManager.Get(profileID)
	if !ok {
		return false
	}
	return ComputeProfileHash(prof) == expectedHash
}

func (a *App) SyncGameDiscordLobby(info *IPCLobbyInfo) {
	if a.DiscordService == nil || !a.DiscordService.IsLoggedIn() {
		return
	}

	a.runningProfileMu.Lock()
	profileID := a.runningProfileID
	a.runningProfileMu.Unlock()

	if profileID == uuid.Nil || info == nil || !info.IsConnected || strings.TrimSpace(info.LobbyCode) == "" {
		return
	}

	// If already in an active Discord lobby, just update room metadata on server
	if _, ok := a.DiscordService.GetActiveLobby(); ok {
		_ = a.UpdateCurrentLobbyRoom(info)
		return
	}

	_, err := a.ShareCurrentLobby(profileID)
	if err != nil {
		slog.Warn("Failed to create server-managed lobby for in-game session", "error", err)
	}
}

func (a *App) CreateManualLobby(profileID uuid.UUID, callback func(err error, lobbyID uint64)) {
	if a.DiscordService == nil || !a.DiscordService.IsLoggedIn() {
		if callback != nil {
			callback(discord.ErrNotLoggedIn, 0)
		}
		return
	}
	_, err := a.ShareCurrentLobby(profileID)
	if err != nil {
		if callback != nil {
			callback(err, 0)
		}
		return
	}
	if callback != nil {
		var dID uint64
		if a.DiscordService != nil {
			if active, ok := a.DiscordService.GetActiveLobby(); ok {
				dID = active.ID
			}
		}
		callback(nil, dID)
	}
}

func New(version string, restClient rest.Client, activityService *discord.DiscordService) (*App, error) {
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
		DiscordService:     activityService,
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
	if err := json.UnmarshalRead(reader, &prof); err != nil {
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

	if err := json.MarshalWrite(writer, prof.MakeShared()); err != nil {
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
	if strings.TrimSpace(serverBase) == "" && a.Rest != nil {
		serverBase = a.Rest.ServerBaseURL()
	}
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
		GameVersion:    rs.Room.GameVersion,
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
	ErrorType  string
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
		ErrorType:  strings.TrimSpace(values.Get("error_type")),
	}, nil
}

func (a *App) ShareCurrentLobby(profileID uuid.UUID) (*SharedLobbyLink, error) {
	a.lobbyShareMu.Lock()
	if a.lobbyShareCache.SessionID != "" && a.lobbyShareCache.ExpiresAt.After(time.Now()) {
		cached := a.lobbyShareCache
		a.lobbyShareMu.Unlock()
		return &cached, nil
	}
	a.lobbyShareMu.Unlock()

	prof, ok := a.ProfileManager.Get(profileID)
	if !ok {
		return nil, fmt.Errorf("profile not found: %s", profileID)
	}

	var iconPNG []byte
	if icon, err := a.ProfileManager.LoadIconPNG(profileID); err == nil {
		iconPNG = icon
	}

	aupack, err := a.ExportProfileArchive(prof, iconPNG)
	if err != nil {
		return nil, fmt.Errorf("failed to export profile archive: %w", err)
	}

	// Current room if in-game
	var roomPtr *commonrest.RoomInfo
	a.runningProfileMu.Lock()
	lobby := a.lobbyInfo
	a.runningProfileMu.Unlock()
	if room, ok := a.CurrentRoomInfo(lobby); ok {
		roomPtr = &room
	}

	// Determine host discord user ID
	var hostDiscordUserID uint64
	if a.DiscordService != nil {
		if user, ok := a.DiscordService.UserInfo(); ok && user != nil {
			hostDiscordUserID = user.Id()
		}
	}

	rs, err := a.Rest.ShareLobby(aupack, hostDiscordUserID, roomPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to share lobby: %w", err)
	}

	if rs.DiscordLobbyID != 0 && a.DiscordService != nil {
		a.DiscordService.SetActiveLobbyID(rs.DiscordLobbyID)
	}

	link := SharedLobbyLink{
		URL:       rs.URL,
		SessionID: rs.SessionID,
		HostKey:   rs.HostKey,
		ExpiresAt: rs.ExpiresAt,
	}
	a.SetSharedLobby(link)
	return &link, nil
}

func (a *App) UpdateCurrentLobbyRoom(info *IPCLobbyInfo) error {
	a.runningProfileMu.Lock()
	profileID := a.runningProfileID
	a.runningProfileMu.Unlock()

	var roomPtr *commonrest.RoomInfo
	if info != nil && info.IsConnected && strings.TrimSpace(info.LobbyCode) != "" {
		gameVersion := ""
		if profileID != uuid.Nil {
			profileDir := filepath.Join(a.ConfigDir, "profiles", profileID.String())
			if meta, err := modmgr.GetProfileMetadata(profileDir); err == nil && meta != nil {
				gameVersion = meta.GameVersion
			}
		}
		if gameVersion == "" {
			if gamePath, err := a.DetectGamePath(); err == nil && gamePath != "" {
				if v, err := aumgr.GetVersion(gamePath); err == nil {
					gameVersion = v
				}
			}
		}
		roomPtr = &commonrest.RoomInfo{
			LobbyCode:      info.LobbyCode,
			ServerIP:       info.ServerIP,
			ServerPort:     uint16(info.ServerPort),
			MatchMakerIp:   info.MatchMakerIp,
			MatchMakerPort: uint16(info.MatchMakerPort),
			GameVersion:    gameVersion,
		}
	}

	// 1. Update server shared lobby session if active
	a.lobbyShareMu.Lock()
	cache := a.lobbyShareCache
	a.lobbyShareMu.Unlock()
	if cache.SessionID != "" && cache.HostKey != "" {
		go func() {
			if rs, err := a.Rest.UpdateSharedLobbyRoom(cache.SessionID, cache.HostKey, roomPtr); err != nil {
				slog.Warn("Failed to update shared lobby room on server", "error", err)
			} else {
				a.lobbyShareMu.Lock()
				if a.lobbyShareCache.SessionID == cache.SessionID {
					a.lobbyShareCache.ExpiresAt = rs.ExpiresAt
				}
				a.lobbyShareMu.Unlock()
			}
		}()
	}

	// 2. Update Discord Social SDK Lobby metadata if active
	if a.DiscordService != nil && a.DiscordService.IsLoggedIn() {
		if active, ok := a.DiscordService.GetActiveLobby(); ok {
			meta := make(map[string]string)
			maps.Copy(meta, active.Metadata)
			if roomPtr != nil {
				meta["room_code"] = roomPtr.LobbyCode
				meta["server_ip"] = roomPtr.ServerIP
				meta["server_port"] = strconv.Itoa(int(roomPtr.ServerPort))
				meta["match_maker_ip"] = roomPtr.MatchMakerIp
				meta["match_maker_port"] = strconv.Itoa(int(roomPtr.MatchMakerPort))
				meta["game_version"] = roomPtr.GameVersion
			} else {
				delete(meta, "room_code")
				delete(meta, "server_ip")
				delete(meta, "server_port")
				delete(meta, "match_maker_ip")
				delete(meta, "match_maker_port")
			}
			a.DiscordService.UpdateLobbyMemberMetadata(meta, nil)
		}
	}

	return nil
}

type JoinLobbyLink struct {
	SessionID  string
	ServerBase string
	ErrorType  string
}

func (a *App) ParseJoinLobbyURI(uri string) (*JoinLobbyLink, error) {
	slog.Info("parsing join lobby URI", "uri", uri)
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse join lobby URI: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "mod-of-us") || !strings.EqualFold(parsed.Host, "join_lobby") {
		return nil, fmt.Errorf("invalid join lobby URI")
	}
	path := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasPrefix(path, "v1/") {
		return nil, fmt.Errorf("unsupported join lobby URI version")
	}
	sessionID := strings.TrimPrefix(path, "v1/")
	values := parsed.Query()
	serverBase := strings.TrimSpace(values.Get("server"))
	if serverBase == "" {
		return nil, fmt.Errorf("join lobby URI missing server")
	}
	if parsedServer, err := url.Parse(serverBase); err != nil || parsedServer.Scheme == "" || parsedServer.Host == "" {
		return nil, fmt.Errorf("invalid join lobby URI server")
	}
	return &JoinLobbyLink{
		SessionID:  sessionID,
		ServerBase: serverBase,
		ErrorType:  strings.TrimSpace(values.Get("error_type")),
	}, nil
}

func (a *App) HandleJoinLobbyDownload(sessionID string, serverBase string) (*profile.SharedProfile, []byte, *LaunchJoinInfo, error) {
	if strings.TrimSpace(serverBase) == "" && a.Rest != nil {
		serverBase = a.Rest.ServerBaseURL()
	}
	client := rest.NewClient(serverBase)
	rs, err := client.GetJoinLobbyDownload(sessionID)
	if err != nil {
		return nil, nil, nil, err
	}

	if rs.DiscordLobbyID != 0 && a.DiscordService != nil {
		if user, ok := a.DiscordService.UserInfo(); ok && user != nil {
			_ = client.AddLobbyMember(sessionID, user.Id())
		}
	}

	tmpFile, err := os.CreateTemp("", "mod-of-us-lobby-*.aupack")
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

	var joinInfo *LaunchJoinInfo
	if rs.Room != nil && rs.Room.LobbyCode != "" {
		joinInfo = &LaunchJoinInfo{
			LobbyCode:      rs.Room.LobbyCode,
			ServerIP:       rs.Room.ServerIP,
			ServerPort:     rs.Room.ServerPort,
			MatchMakerIp:   rs.Room.MatchMakerIp,
			MatchMakerPort: rs.Room.MatchMakerPort,
			GameVersion:    rs.Room.GameVersion,
		}
	}
	return shared, iconPNG, joinInfo, nil
}
