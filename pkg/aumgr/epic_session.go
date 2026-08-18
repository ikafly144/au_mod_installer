package aumgr

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/danieljoos/wincred"
)

const epicCredentialsKey = "ModOfUs_EpicCredentials"

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

	// Try loading from Windows Credential Manager
	creds, err := wincred.GetGenericCredential(epicCredentialsKey)
	if err == nil && creds != nil {
		var session EpicSession
		if err := json.Unmarshal(creds.CredentialBlob, &session); err == nil {
			m.session = &session
			// Clean up legacy session file if present
			if _, err := os.Stat(m.path); err == nil {
				_ = os.Remove(m.path)
			}
			return nil
		}
	}

	// Legacy file fallback migration if not found in Windows Credential Manager
	if _, err := os.Stat(m.path); err == nil {
		data, err := os.ReadFile(m.path)
		if err == nil {
			var session EpicSession
			if err := json.Unmarshal(data, &session); err == nil {
				m.session = &session
				// Migrate session to Windows Credential Manager
				blob, err := json.Marshal(&session)
				if err == nil {
					cred := wincred.NewGenericCredential(epicCredentialsKey)
					cred.CredentialBlob = blob
					cred.Persist = wincred.PersistLocalMachine
					if err := cred.Write(); err == nil {
						_ = os.Remove(m.path)
					}
				}
				return nil
			}
		}
	}

	m.session = nil
	return nil
}

func (m *EpicSessionManager) Save(session *EpicSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	blob, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal epic session: %w", err)
	}

	cred, err := wincred.GetGenericCredential(epicCredentialsKey)
	if err != nil {
		cred = wincred.NewGenericCredential(epicCredentialsKey)
	}
	cred.CredentialBlob = blob
	cred.Persist = wincred.PersistLocalMachine
	if err := cred.Write(); err != nil {
		return fmt.Errorf("failed to save epic session to credential manager: %w", err)
	}

	// Remove legacy file if it exists
	if _, err := os.Stat(m.path); err == nil {
		_ = os.Remove(m.path)
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
	if cred, err := wincred.GetGenericCredential(epicCredentialsKey); err == nil && cred != nil {
		if err := cred.Delete(); err != nil {
			slog.Warn("Failed to delete epic credentials from Windows Credential Manager", "error", err)
		}
	}
	if err := os.Remove(m.path); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to remove legacy epic session file", "error", err)
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
