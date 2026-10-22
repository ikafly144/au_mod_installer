package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
)

const (
	shareLobbyTTL          = 2 * time.Hour
	shareLobbyRateWindow   = 10 * time.Minute
	shareLobbyMaxPerWindow = 20
)

var (
	ErrShareLobbyRateLimited  = errors.New("share lobby rate limited")
	ErrShareLobbyNotFound     = errors.New("shared lobby not found")
	ErrShareLobbyUnauthorized = errors.New("invalid host key")
	ErrShareLobbyExpired      = errors.New("shared lobby expired")
)

type sharedLobbySession struct {
	SessionID         string
	HostKey           string
	IP                string
	Aupack            []byte
	DiscordLobbyID    uint64
	HostDiscordUserID uint64
	Room              *restcommon.RoomInfo
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

type shareLobbyManager struct {
	mu              sync.Mutex
	discordClient   DiscordLobbyClient
	sessions        map[string]*sharedLobbySession
	sessionByIP     map[string]string
	rateByIP        map[string]*ipRateState
	dedupeByIPLobby map[string]*sharedLobbySession
}

func newShareLobbyManager() *shareLobbyManager {
	return newShareLobbyManagerWithOptions(nil)
}

func newShareLobbyManagerWithOptions(discordClient DiscordLobbyClient) *shareLobbyManager {
	return &shareLobbyManager{
		discordClient:   discordClient,
		sessions:        make(map[string]*sharedLobbySession),
		sessionByIP:     make(map[string]string),
		rateByIP:        make(map[string]*ipRateState),
		dedupeByIPLobby: make(map[string]*sharedLobbySession),
	}
}

func (m *shareLobbyManager) create(ip string, req restcommon.ShareLobbyRequest) (*restcommon.ShareLobbyResponse, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	if err := m.allowRateLocked(ip, now); err != nil {
		return nil, err
	}

	lobbyKey := dedupeLobbyKey(ip, req.Aupack)
	if cached, ok := m.dedupeByIPLobby[lobbyKey]; ok && cached.ExpiresAt.After(now) {
		if req.Room != nil {
			r := *req.Room
			cached.Room = &r
		}
		return &restcommon.ShareLobbyResponse{
			URL:            "/join_lobby?session_id=" + cached.SessionID,
			SessionID:      cached.SessionID,
			HostKey:        cached.HostKey,
			DiscordLobbyID: cached.DiscordLobbyID,
			ExpiresAt:      cached.ExpiresAt,
		}, nil
	}

	if existingID, ok := m.sessionByIP[ip]; ok && existingID != "" {
		m.deleteSessionLocked(existingID)
	}

	sessionID, err := randomURLToken(24)
	if err != nil {
		return nil, err
	}
	hostKey, err := randomURLToken(32)
	if err != nil {
		return nil, err
	}

	var roomCopy *restcommon.RoomInfo
	if req.Room != nil {
		r := *req.Room
		roomCopy = &r
	}

	var discordLobbyID uint64
	if m.discordClient != nil && req.HostDiscordUserID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		dID, err := m.discordClient.CreateLobby(ctx, req.HostDiscordUserID, map[string]string{
			"session_id": sessionID,
		})
		cancel()
		if err != nil {
			slog.Warn("Failed to create server-side Discord lobby; falling back to client secret lobby", "error", err)
		} else {
			discordLobbyID = dID
		}
	}

	s := &sharedLobbySession{
		SessionID:         sessionID,
		HostKey:           hostKey,
		IP:                ip,
		Aupack:            append([]byte(nil), req.Aupack...),
		DiscordLobbyID:    discordLobbyID,
		HostDiscordUserID: req.HostDiscordUserID,
		Room:              roomCopy,
		CreatedAt:         now,
		ExpiresAt:         now.Add(shareLobbyTTL),
	}
	m.sessions[sessionID] = s
	m.sessionByIP[ip] = sessionID
	m.dedupeByIPLobby[lobbyKey] = s

	return &restcommon.ShareLobbyResponse{
		URL:            "/join_lobby?session_id=" + sessionID,
		SessionID:      sessionID,
		HostKey:        hostKey,
		DiscordLobbyID: discordLobbyID,
		ExpiresAt:      s.ExpiresAt,
	}, nil
}

func (m *shareLobbyManager) updateRoom(sessionID, hostKey string, room *restcommon.RoomInfo) (*restcommon.ShareLobbyResponse, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrShareLobbyNotFound
	}
	if s.HostKey != hostKey {
		return nil, ErrShareLobbyUnauthorized
	}
	if now.After(s.ExpiresAt) {
		m.deleteSessionLocked(sessionID)
		return nil, ErrShareLobbyExpired
	}

	if room == nil {
		s.Room = nil
	} else {
		r := *room
		s.Room = &r
	}
	s.ExpiresAt = now.Add(shareLobbyTTL)

	if s.DiscordLobbyID != 0 && m.discordClient != nil {
		discordMeta := map[string]string{
			"session_id": sessionID,
		}
		if s.Room != nil {
			discordMeta["lobby_code"] = s.Room.LobbyCode
			discordMeta["server_ip"] = s.Room.ServerIP
			discordMeta["server_port"] = fmt.Sprint(s.Room.ServerPort)
			discordMeta["game_version"] = s.Room.GameVersion
		}
		dID := s.DiscordLobbyID
		client := m.discordClient
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = client.UpdateLobbyMetadata(ctx, dID, discordMeta)
		}()
	}

	return &restcommon.ShareLobbyResponse{
		URL:            "/join_lobby?session_id=" + sessionID,
		SessionID:      sessionID,
		HostKey:        hostKey,
		DiscordLobbyID: s.DiscordLobbyID,
		ExpiresAt:      s.ExpiresAt,
	}, nil
}

func (m *shareLobbyManager) getDownload(sessionID string) (*restcommon.JoinLobbyDownloadResponse, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrShareLobbyNotFound
	}
	if now.After(s.ExpiresAt) {
		m.deleteSessionLocked(sessionID)
		return nil, ErrShareLobbyExpired
	}

	var roomCopy *restcommon.RoomInfo
	if s.Room != nil {
		r := *s.Room
		roomCopy = &r
	}

	return &restcommon.JoinLobbyDownloadResponse{
		SessionID:      sessionID,
		Aupack:         append([]byte(nil), s.Aupack...),
		DiscordLobbyID: s.DiscordLobbyID,
		Room:           roomCopy,
		ExpiresAt:      s.ExpiresAt,
	}, nil
}

func (m *shareLobbyManager) addMember(sessionID string, userID uint64) error {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return ErrShareLobbyNotFound
	}
	if now.After(s.ExpiresAt) {
		m.deleteSessionLocked(sessionID)
		return ErrShareLobbyExpired
	}

	if s.DiscordLobbyID != 0 && m.discordClient != nil && userID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return m.discordClient.AddMember(ctx, s.DiscordLobbyID, userID, map[string]string{
			"is_host": "false",
		})
	}
	return nil
}

func (m *shareLobbyManager) removeMember(sessionID string, userID uint64) error {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return ErrShareLobbyNotFound
	}
	if now.After(s.ExpiresAt) {
		m.deleteSessionLocked(sessionID)
		return ErrShareLobbyExpired
	}

	if s.DiscordLobbyID != 0 && m.discordClient != nil && userID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return m.discordClient.RemoveMember(ctx, s.DiscordLobbyID, userID)
	}
	return nil
}

func (m *shareLobbyManager) getSessionMeta(sessionID string) (*sharedLobbySession, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrShareLobbyNotFound
	}
	if now.After(s.ExpiresAt) {
		m.deleteSessionLocked(sessionID)
		return nil, ErrShareLobbyExpired
	}
	cp := *s
	cp.Aupack = nil
	return &cp, nil
}

func (m *shareLobbyManager) delete(sessionID, hostKey string) error {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return ErrShareLobbyNotFound
	}
	if s.HostKey != hostKey {
		return ErrShareLobbyUnauthorized
	}
	m.deleteSessionLocked(sessionID)
	return nil
}

func (m *shareLobbyManager) updateExpiration(sessionID, hostKey string) (*restcommon.ShareLobbyResponse, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrShareLobbyNotFound
	}
	if s.HostKey != hostKey {
		return nil, ErrShareLobbyUnauthorized
	}

	s.ExpiresAt = now.Add(shareLobbyTTL)

	return &restcommon.ShareLobbyResponse{
		URL:            "/join_lobby?session_id=" + sessionID,
		SessionID:      sessionID,
		HostKey:        hostKey,
		DiscordLobbyID: s.DiscordLobbyID,
		ExpiresAt:      s.ExpiresAt,
	}, nil
}

func (m *shareLobbyManager) allowRateLocked(ip string, now time.Time) error {
	state, ok := m.rateByIP[ip]
	if !ok || now.Sub(state.WindowStart) >= shareLobbyRateWindow {
		m.rateByIP[ip] = &ipRateState{
			WindowStart: now,
			Count:       1,
		}
		return nil
	}
	if state.Count >= shareLobbyMaxPerWindow {
		return ErrShareLobbyRateLimited
	}
	state.Count++
	return nil
}

func (m *shareLobbyManager) cleanupLocked(now time.Time) {
	for id, s := range m.sessions {
		if now.After(s.ExpiresAt) {
			m.deleteSessionLocked(id)
		}
	}
}

func (m *shareLobbyManager) deleteSessionLocked(sessionID string) {
	s, ok := m.sessions[sessionID]
	if !ok {
		return
	}
	if s.DiscordLobbyID != 0 && m.discordClient != nil {
		dID := s.DiscordLobbyID
		client := m.discordClient
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = client.DeleteLobby(ctx, dID)
		}()
	}
	delete(m.sessions, sessionID)
	if current, ok := m.sessionByIP[s.IP]; ok && current == sessionID {
		delete(m.sessionByIP, s.IP)
	}
	lobbyKey := dedupeLobbyKey(s.IP, s.Aupack)
	if current, ok := m.dedupeByIPLobby[lobbyKey]; ok && current.SessionID == sessionID {
		delete(m.dedupeByIPLobby, lobbyKey)
	}
}

func dedupeLobbyKey(ip string, aupack []byte) string {
	sum := sha256.Sum256(aupack)
	return fmt.Sprintf("%s|%s", ip, hex.EncodeToString(sum[:]))
}
