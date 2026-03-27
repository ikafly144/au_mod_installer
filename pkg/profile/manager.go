package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type Manager struct {
	path       string
	storageDir string
	profiles   []Profile
	mu         sync.RWMutex
}

func NewManager(storagePath string) (*Manager, error) {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	m := &Manager{
		path:       filepath.Join(storagePath, "profiles.json"),
		storageDir: storagePath,
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

	if p.ID == uuid.Nil {
		return fmt.Errorf("profile ID cannot be nil")
	}

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

func (m *Manager) Remove(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.profiles {
		if p.ID == id {
			m.profiles = append(m.profiles[:i], m.profiles[i+1:]...)
			if err := m.save(); err != nil {
				return err
			}
			if err := os.RemoveAll(m.profileDir(id)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove profile directory: %w", err)
			}
			return nil
		}
	}
	return nil
}

func (m *Manager) Get(id uuid.UUID) (Profile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.profiles {
		if p.ID == id {
			return p, true
		}
	}
	return Profile{}, false
}

func (m *Manager) ProfileDir(id uuid.UUID) (string, error) {
	if id == uuid.Nil {
		return "", fmt.Errorf("profile ID cannot be nil")
	}
	return m.profileDir(id), nil
}

func (m *Manager) profileDir(id uuid.UUID) string {
	return filepath.Join(m.storageDir, "profiles", id.String())
}

func (m *Manager) profileIconPath(id uuid.UUID) string {
	return filepath.Join(m.profileDir(id), "icon.png")
}

func (m *Manager) SaveIconPNG(id uuid.UUID, png []byte) error {
	if id == uuid.Nil {
		return fmt.Errorf("profile ID cannot be nil")
	}
	if len(png) == 0 {
		return fmt.Errorf("icon data is empty")
	}

	iconPath := m.profileIconPath(id)
	if err := os.MkdirAll(filepath.Dir(iconPath), 0755); err != nil {
		return fmt.Errorf("failed to create profile icon directory: %w", err)
	}
	if err := os.WriteFile(iconPath, png, 0644); err != nil {
		return fmt.Errorf("failed to write profile icon: %w", err)
	}
	return nil
}

func (m *Manager) LoadIconPNG(id uuid.UUID) ([]byte, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("profile ID cannot be nil")
	}
	iconPath := m.profileIconPath(id)
	data, err := os.ReadFile(iconPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read profile icon: %w", err)
	}
	return data, nil
}

func (m *Manager) RemoveIcon(id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("profile ID cannot be nil")
	}
	iconPath := m.profileIconPath(id)
	if err := os.Remove(iconPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove profile icon: %w", err)
	}
	return nil
}
