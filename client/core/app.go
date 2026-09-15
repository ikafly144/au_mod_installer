package core

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"uuid"

	"github.com/ikafly144/modrepo/client/rest"
	"github.com/ikafly144/modrepo/pkg/profile"
	"github.com/ikafly144/modrepo/pkg/progress"
	"github.com/ikafly144/modrepo/pkg/repomgr"
	"github.com/ikafly144/modrepo/pkg/thunderstore"
)

type App struct {
	Version        string
	ConfigDir      string
	Rest           rest.Client
	ProfileManager *profile.Manager

	// Running profile state
	runningProfileMu   sync.Mutex
	runningProfileID   uuid.UUID
	launchingProfileID uuid.UUID
	launchingProfile   bool
	runningGamePID     int
	runningStartedAt   time.Time

	// Callbacks for state changes
	OnGameStarted func(profileID uuid.UUID, pid int)
	OnGameExited  func(profileID uuid.UUID)
}

func New(version string, restClient rest.Client) (*App, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}
	appConfigDir := filepath.Join(configDir, "MODREPO")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	profileManager, err := profile.NewManager(appConfigDir)
	if err != nil {
		slog.Warn("Failed to load profiles, recreating manager", "error", err)
		profileManager, err = profile.NewManager(appConfigDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile manager: %w", err)
		}
	}

	a := &App{
		Version:        version,
		ConfigDir:      appConfigDir,
		Rest:           restClient,
		ProfileManager: profileManager,
	}

	// Ensure at least one profile exists
	if len(profileManager.List()) == 0 {
		_, err := a.CreateProfile("Default")
		if err != nil {
			slog.Warn("Failed to create default profile", "error", err)
		}
	}

	return a, nil
}

// CreateProfile creates a new profile with BepInEx-BepInExPack pre-installed.
func (a *App) CreateProfile(name string) (*profile.Profile, error) {
	prof := profile.Profile{
		ID:        uuid.New(),
		Name:      name,
		UpdatedAt: time.Now(),
	}

	// Automatically add BepInExPack if available
	if a.Rest != nil {
		if bep, err := a.Rest.GetLatestModVersion(thunderstore.BepInExPackModID); err == nil && bep != nil {
			prof.AddModVersion(*bep)
			slog.Info("Automatically added BepInExPack to profile", "profile", name, "version", bep.VersionID)
		}
	}

	if err := a.ProfileManager.Add(prof); err != nil {
		return nil, err
	}
	return &prof, nil
}

func (a *App) DetectGamePath() (string, error) {
	return repomgr.GetRepoDir()
}

func (a *App) DetectLauncherType(path string) repomgr.LauncherType {
	return repomgr.DetectLauncherType(path)
}

func (a *App) GetBinaryType(path string) (repomgr.BinaryType, error) {
	return repomgr.BinaryType64Bit, nil
}

func (a *App) ClearModCache() error {
	modsDir := filepath.Join(a.ConfigDir, "mods")
	cacheDir := filepath.Join(a.ConfigDir, "cache")
	var err1, err2 error
	if _, err := os.Stat(modsDir); err == nil {
		err1 = os.RemoveAll(modsDir)
	}
	if _, err := os.Stat(cacheDir); err == nil {
		err2 = os.RemoveAll(cacheDir)
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (a *App) ExportProfileArchive(prof profile.Profile, iconPNG []byte) ([]byte, error) {
	return profile.EncodeSharedArchive(prof.MakeShared(), iconPNG)
}

func (a *App) HandleSharedProfileArchive(reader io.ReaderAt, size int64) (*profile.SharedProfile, []byte, error) {
	return profile.DecodeSharedArchive(reader, size)
}

func (a *App) HandleSharedProfileArchiveFile(path string) (*profile.SharedProfile, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open archive file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat archive file: %w", err)
	}

	return a.HandleSharedProfileArchive(f, stat.Size())
}

func (a *App) DownloadArchiveURLToTempFile(rawURL string, p progress.Progress) (string, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download archive: HTTP %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "modrepo-archive-*.repopack")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	var writer io.Writer = tempFile
	if p != nil {
		pw := progress.NewProgressWriter(0, 1.0, resp.ContentLength, p, tempFile)
		defer pw.Complete()
		writer = pw
	}

	if _, err := io.Copy(writer, resp.Body); err != nil {
		_ = os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to save archive: %w", err)
	}

	return tempFile.Name(), nil
}

func (a *App) HandleImportReader(reader io.Reader, extension string) (*profile.SharedProfile, []byte, error) {
	ext := strings.ToLower(extension)
	if ext != ".repopack" && ext != ".aupack" && ext != ".zip" {
		return nil, nil, fmt.Errorf("unsupported file extension: %s", extension)
	}

	tempFile, err := os.CreateTemp("", "modrepo-profile-*.repopack")
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
		ID:          uuid.New(), // Give new UUID on import to avoid conflicts
		Name:        shared.Name,
		Author:      shared.Author,
		Description: shared.Description,
		UpdatedAt:   time.Now(),
	}

	// Fetch mod version details
	for modID, versionID := range shared.ModVersions {
		info, err := a.Rest.GetModVersion(modID, versionID)
		if err != nil {
			slog.Warn("Failed to fetch mod version for import, trying latest", "modId", modID, "error", err)
			info, err = a.Rest.GetLatestModVersion(modID)
		}
		if err == nil && info != nil {
			prof.AddModVersion(*info)
		}
	}

	// Ensure BepInEx-BepInExPack is present
	if _, hasBep := prof.ModVersions[thunderstore.BepInExPackModID]; !hasBep {
		if bep, err := a.Rest.GetLatestModVersion(thunderstore.BepInExPackModID); err == nil && bep != nil {
			prof.AddModVersion(*bep)
		}
	}

	if err := a.ProfileManager.Add(prof); err != nil {
		return nil, err
	}
	if len(iconPNG) > 0 {
		_ = a.ProfileManager.SaveIconPNG(prof.ID, iconPNG)
	}
	return &prof, nil
}
