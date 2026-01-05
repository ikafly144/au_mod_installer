package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	path     string
	profiles []Profile
	mu       sync.RWMutex
}

func NewManager(storagePath string) (*Manager, error) {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	m := &Manager{
		path: filepath.Join(storagePath, "profiles.json"),
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		m.profiles = []Profile{}
		return nil
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("failed to read profiles: %w", err)
	}

	if err := json.Unmarshal(data, &m.profiles); err != nil {
		return fmt.Errorf("failed to unmarshal profiles: %w", err)
	}

	return nil
}

// save writes the profiles to disk. It assumes the caller holds the lock.
func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.profiles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	if err := os.WriteFile(m.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write profiles: %w", err)
	}

	return nil
}

func (m *Manager) List() []Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make([]Profile, len(m.profiles))
	copy(result, m.profiles)
	return result
}

func (m *Manager) Add(p Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if ID exists, if so replace
	found := false
	for i, existing := range m.profiles {
		if existing.ID == p.ID {
			m.profiles[i] = p
			found = true
			break
		}
	}
	if !found {
		m.profiles = append(m.profiles, p)
	}

	return m.save()
}

func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.profiles {
		if p.ID == id {
			m.profiles = append(m.profiles[:i], m.profiles[i+1:]...)
			return m.save()
		}
	}
	return nil
}

func (m *Manager) Get(id string) (Profile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.profiles {
		if p.ID == id {
			return p, true
		}
	}
	return Profile{}, false
}
