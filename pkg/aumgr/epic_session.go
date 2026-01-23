package aumgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type EpicSessionManager struct {
	path    string
	session *EpicSession
	mu      sync.RWMutex
}

func NewEpicSessionManager(storagePath string) (*EpicSessionManager, error) {
	if err := os.MkdirAll(storagePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	m := &EpicSessionManager{
		path: filepath.Join(storagePath, "epic_session.json"),
	}
	if err := m.Load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *EpicSessionManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		m.session = nil
		return nil
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("failed to read epic session: %w", err)
	}

	var session EpicSession
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("failed to unmarshal epic session: %w", err)
	}

	m.session = &session
	return nil
}

func (m *EpicSessionManager) Save(session *EpicSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal epic session: %w", err)
	}

	if err := os.WriteFile(m.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write epic session: %w", err)
	}

	m.session = session
	return nil
}

func (m *EpicSessionManager) GetSession() *EpicSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.session
}

func (m *EpicSessionManager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.session = nil
	if err := os.Remove(m.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove epic session file: %w", err)
	}
	return nil
}

func (m *EpicSessionManager) GetValidSession(api *EpicApi) (*EpicSession, error) {
	session := m.GetSession()
	if session == nil {
		return nil, fmt.Errorf("no epic session found")
	}

	// If access token is still valid (with 1 minute buffer)
	if time.Now().Add(1 * time.Minute).Before(session.ExpiresAt) {
		return session, nil
	}

	// Try to refresh
	newSession, err := api.RefreshSession(session.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh epic session: %w", err)
	}

	if err := m.Save(newSession); err != nil {
		return nil, err
	}

	return newSession, nil
}
