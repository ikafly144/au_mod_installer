package service

import (
	"crypto/sha256"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
)

const (
	shareGameTTL          = 10 * time.Minute
	shareGameRateWindow   = time.Minute
	shareGameMaxPerWindow = 10
)

var (
	ErrShareGameRateLimited  = errors.New("share game rate limited")
	ErrShareGameNotFound     = errors.New("shared game not found")
	ErrShareGameUnauthorized = errors.New("invalid host key")
	ErrShareGameExpired      = errors.New("shared game expired")
)

type sharedGameSession struct {
	SessionID string
	HostKey   string
	IP        string
	Aupack    []byte
	Room      restcommon.RoomInfo
	CreatedAt time.Time
	ExpiresAt time.Time
}

type ipRateState struct {
	WindowStart time.Time
	Count       int
}

type shareGameManager struct {
	mu             sync.Mutex
	sessions       map[string]*sharedGameSession
	sessionByIP    map[string]string
	rateByIP       map[string]*ipRateState
	dedupeByIPRoom map[string]*sharedGameSession
}

func newShareGameManager() *shareGameManager {
	return &shareGameManager{
		sessions:       make(map[string]*sharedGameSession),
		sessionByIP:    make(map[string]string),
		rateByIP:       make(map[string]*ipRateState),
		dedupeByIPRoom: make(map[string]*sharedGameSession),
	}
}

func (m *shareGameManager) create(ip string, req restcommon.ShareGameRequest) (*restcommon.ShareGameResponse, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	if err := m.allowRateLocked(ip, now); err != nil {
		return nil, err
	}

	roomKey := dedupeRoomKey(ip, req.Aupack, req.Room)
	if cached, ok := m.dedupeByIPRoom[roomKey]; ok && cached.ExpiresAt.After(now) {
		return &restcommon.ShareGameResponse{
			URL:       "/join_game?session_id=" + cached.SessionID,
			SessionID: cached.SessionID,
			HostKey:   cached.HostKey,
			ExpiresAt: cached.ExpiresAt,
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

	s := &sharedGameSession{
		SessionID: sessionID,
		HostKey:   hostKey,
		IP:        ip,
		Aupack:    append([]byte(nil), req.Aupack...),
		Room:      req.Room,
		CreatedAt: now,
		ExpiresAt: now.Add(shareGameTTL),
	}
	m.sessions[sessionID] = s
	m.sessionByIP[ip] = sessionID
	m.dedupeByIPRoom[roomKey] = s

	return &restcommon.ShareGameResponse{
		URL:       "/join_game?session_id=" + sessionID,
		SessionID: sessionID,
		HostKey:   hostKey,
		ExpiresAt: s.ExpiresAt,
	}, nil
}

func (m *shareGameManager) getDownload(sessionID string) (*restcommon.JoinGameDownloadResponse, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrShareGameNotFound
	}
	if now.After(s.ExpiresAt) {
		m.deleteSessionLocked(sessionID)
		return nil, ErrShareGameExpired
	}
	return &restcommon.JoinGameDownloadResponse{
		SessionID: sessionID,
		Aupack:    append([]byte(nil), s.Aupack...),
		Room:      s.Room,
		ExpiresAt: s.ExpiresAt,
	}, nil
}

func (m *shareGameManager) getSessionMeta(sessionID string) (*sharedGameSession, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrShareGameNotFound
	}
	if now.After(s.ExpiresAt) {
		m.deleteSessionLocked(sessionID)
		return nil, ErrShareGameExpired
	}
	cp := *s
	cp.Aupack = nil
	return &cp, nil
}

func (m *shareGameManager) delete(sessionID, hostKey string) error {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)

	s, ok := m.sessions[sessionID]
	if !ok {
		return ErrShareGameNotFound
	}
	if s.HostKey != hostKey {
		return ErrShareGameUnauthorized
	}
	m.deleteSessionLocked(sessionID)
	return nil
}

func (m *shareGameManager) allowRateLocked(ip string, now time.Time) error {
	state, ok := m.rateByIP[ip]
	if !ok || now.Sub(state.WindowStart) >= shareGameRateWindow {
		m.rateByIP[ip] = &ipRateState{
			WindowStart: now,
			Count:       1,
		}
		return nil
	}
	if state.Count >= shareGameMaxPerWindow {
		return ErrShareGameRateLimited
	}
	state.Count++
	return nil
}

func (m *shareGameManager) cleanupLocked(now time.Time) {
	for id, s := range m.sessions {
		if now.After(s.ExpiresAt) {
			m.deleteSessionLocked(id)
		}
	}
}

func (m *shareGameManager) deleteSessionLocked(sessionID string) {
	s, ok := m.sessions[sessionID]
	if !ok {
		return
	}
	delete(m.sessions, sessionID)
	if current, ok := m.sessionByIP[s.IP]; ok && current == sessionID {
		delete(m.sessionByIP, s.IP)
	}
	roomKey := dedupeRoomKey(s.IP, s.Aupack, s.Room)
	if current, ok := m.dedupeByIPRoom[roomKey]; ok && current.SessionID == sessionID {
		delete(m.dedupeByIPRoom, roomKey)
	}
}

func dedupeRoomKey(ip string, aupack []byte, room restcommon.RoomInfo) string {
	sum := sha256.Sum256(aupack)
	return fmt.Sprintf("%s|%s|%s|%s|%d", ip, hex.EncodeToString(sum[:]), room.LobbyCode, room.ServerIP, room.ServerPort)
}

func randomURLToken(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("invalid token size")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
