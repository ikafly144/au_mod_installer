package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/client/core"
	clientrest "github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	commonrest "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
	"github.com/ikafly144/au_mod_installer/pkg/progress"

	_ "image/gif"
	_ "image/jpeg"
)

type Launcher struct {
	state                *uicommon.State
	launchButton         *widget.Button
	shareRoomButton      *widget.Button
	copyRoomLinkButton   *widget.Button
	unpublishRoomButton  *widget.Button
	roomLinkEntry        *widget.Label
	roomLinkLabel        *widget.Label
	roomLinkContainer    *fyne.Container
	roomLinkTray         *fyne.Container
	roomLinkTrayToggle   *widget.Button
	roomLinkTrayExpanded bool
	greetingContent      *widget.Label
	createProfileButton  *widget.Button
	importProfileButton  *widget.Button

	profileList       *widget.List
	profileGrid       *fyne.Container
	profileGridScroll *container.Scroll
	profileViews      *fyne.Container
	toggleViewButton  *widget.Button
	sortOrderButton   *widget.Button
	sortSelect        *widget.Select
	profiles          []profile.Profile
	selectedProfileID uuid.UUID
	isGridView        bool
	sortMode          string
	sortDescending    bool

	modThumbMu             sync.Mutex
	modThumbnailImageCache map[string]image.Image
	modThumbnailFetched    map[string]bool
	modThumbnailLoading    map[string]bool

	canLaunchListener binding.DataListener

	runningProfileMu   sync.Mutex
	runningProfileID   uuid.UUID
	launchingProfileID uuid.UUID
	launchingProfile   bool
	runningDirectJoin  bool
	runningGamePID     int
	runningStartedAt   time.Time
	lobbyPollStop      func()
	lobbyInfo          *core.IPCLobbyInfo

	roomShareMu         sync.Mutex
	roomShareGenerating bool
	roomShareCache      sharedRoomLinkCache

	content *fyne.Container
}

var _ uicommon.Tab = (*Launcher)(nil)

const (
	prefLauncherViewMode       = "launcher.view_mode"
	prefLauncherSortMode       = "launcher.sort_mode"
	prefLauncherSortDescending = "launcher.sort_descending"

	viewModeList = "list"
	viewModeGrid = "grid"

	sortModeName     = "name"
	sortModePlaytime = "playtime"
	sortModeRecent   = "recent"

	launcherListThumbMinSize = float32(88)
	launcherGridCardWidth    = float32(124) // Icon area + inner gaps
	launcherGridCardHeight   = float32(172)
	launcherGridIconAreaSize = float32(116)
	launcherGridIconInset    = float32(2)
	launcherGridMenuSize     = float32(22)
	launcherGridMenuInset    = float32(4)
	launcherRunningBadgeSize = float32(24)
	launcherRunningBadgeGap  = float32(4)

	profileArchiveDownloadTimeout  = 30 * time.Second
	profileArchiveDownloadMaxBytes = int64(64 << 20)
	lobbyPollInterval              = 2 * time.Second
	restoredProcessWatchInterval   = 2 * time.Second
	roomLinkTrayWidth              = float32(320)
)

var roomLinkPlaceholder = lang.LocalizeKey("launcher.join_link.placeholder", "No room shared now")

var launcherRunningProfileStrokeColor = color.NRGBA{R: 56, G: 170, B: 92, A: 255}

type sharedRoomLinkCache struct {
	RoomKey   string
	URL       string
	SessionID string
	HostKey   string
	ExpiresAt time.Time
}

func NewLauncherTab(s *uicommon.State) uicommon.Tab {
	var l Launcher
	revision := fyne.CurrentApp().Metadata().Custom["revision"]
	revision = revision[:min(7, len(revision))]
	viewMode := fyne.CurrentApp().Preferences().StringWithFallback(prefLauncherViewMode, viewModeList)
	sortMode := normalizeSortMode(fyne.CurrentApp().Preferences().StringWithFallback(prefLauncherSortMode, sortModeName))
	sortDescending := fyne.CurrentApp().Preferences().BoolWithFallback(prefLauncherSortDescending, defaultSortDescendingForMode(sortMode))
	l = Launcher{
		state:                  s,
		launchButton:           widget.NewButtonWithIcon(lang.LocalizeKey("launcher.launch", "Launch"), theme.MediaPlayIcon(), l.runLaunch),
		shareRoomButton:        widget.NewButtonWithIcon(lang.LocalizeKey("launcher.join_link.create", "Create Join Link"), theme.MailForwardIcon(), l.shareCurrentRoom),
		copyRoomLinkButton:     widget.NewButtonWithIcon(lang.LocalizeKey("launcher.join_link.copy", "Copy Link"), theme.ContentCopyIcon(), l.copyRoomLinkToClipboard),
		unpublishRoomButton:    widget.NewButtonWithIcon(lang.LocalizeKey("launcher.join_link.unpublish", "Stop Sharing"), theme.MediaStopIcon(), l.unpublishCurrentRoom),
		roomLinkEntry:          widget.NewLabel(""),
		roomLinkLabel:          widget.NewLabel(lang.LocalizeKey("launcher.join_link.title", "Join Link")),
		createProfileButton:    widget.NewButtonWithIcon(lang.LocalizeKey("profile.create", "Create Profile"), theme.ContentAddIcon(), l.createProfile),
		importProfileButton:    widget.NewButtonWithIcon(lang.LocalizeKey("profile.import", "Import"), theme.ContentPasteIcon(), l.showImportDialog),
		greetingContent:        widget.NewLabelWithStyle(fmt.Sprintf("version: %s (%s)", s.Version, revision), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		sortMode:               sortMode,
		sortDescending:         sortDescending,
		isGridView:             viewMode == viewModeGrid,
		modThumbnailImageCache: map[string]image.Image{},
		modThumbnailFetched:    map[string]bool{},
		modThumbnailLoading:    map[string]bool{},
	}
	l.createProfileButton.Importance = widget.HighImportance
	l.shareRoomButton.Importance = widget.MediumImportance
	l.copyRoomLinkButton.Importance = widget.LowImportance
	l.unpublishRoomButton.Importance = widget.LowImportance
	l.roomLinkEntry.Selectable = true
	l.roomLinkEntry.Wrapping = fyne.TextWrapOff

	l.init()

	return &l
}

func (l *Launcher) init() {
	l.state.OnSharedURIReceived = func(uri string) {
		l.state.SharedURI = uri
		fyne.Do(l.checkSharedURI)
	}
	l.state.OnSharedArchiveReceived = func(path string) {
		l.state.SharedArchive = path
		fyne.Do(l.checkSharedArchive)
	}
	l.state.OnDroppedURIs = func(uris []fyne.URI) {
		l.handleDroppedURIs(uris)
	}
	l.state.OnGameStarted = func(profileID uuid.UUID, pid int) {
		l.onGameStarted(profileID, pid)
	}
	l.state.OnGameExited = func(profileID uuid.UUID) {
		l.onGameExited(profileID)
	}
	l.state.OnProfileMetricsUpdated = func(profileID uuid.UUID) {
		fyne.Do(func() {
			l.refreshProfiles()
		})
	}
	l.setupRoomLinkUI()
	bind := binding.NewString()
	bind.AddListener(binding.NewDataListener(func() {
		l.copyRoomLinkButton.SetText(lang.LocalizeKey("launcher.join_link.copy", "Copy Link"))
		if link, err := bind.Get(); err != nil || strings.TrimSpace(link) == "" {
			l.copyRoomLinkButton.Disable()
			return
		}
		l.copyRoomLinkButton.Enable()
	}))
	if l.canLaunchListener == nil {
		l.canLaunchListener = binding.NewDataListener(l.checkLaunchState)
		l.state.CanLaunch.AddListener(l.canLaunchListener)
	}
	l.greetingContent.Wrapping = fyne.TextWrapWord
	l.launchButton.Importance = widget.HighImportance

	l.setupProfileList()
	l.setupProfileGrid()
	l.setupToolbar()
	l.refreshProfiles()
	l.checkLaunchState()
	l.checkSharedURI()
	l.checkSharedArchive()
	l.restoreRunningProfiles()
}

func (l *Launcher) restoreRunningProfiles() {
	runningProfiles, err := l.state.Core.LoadRunningProfilesFromLocks()
	if err != nil {
		slog.Warn("Failed to load running profiles from lock files", "error", err)
		return
	}

	restored := false
	for _, info := range runningProfiles {
		if restored {
			slog.Warn("Skipping extra running profile lock because launcher supports one active profile", "profile_id", info.ProfileID, "game_pid", info.GamePID)
			continue
		}
		prof, ok := l.state.Core.ProfileManager.Get(info.ProfileID)
		if !ok {
			slog.Warn("Skipping running profile lock because profile was not found", "profile_id", info.ProfileID, "game_pid", info.GamePID)
			continue
		}
		directJoinEnabled := info.DirectJoinEnabled
		if !directJoinEnabled && hasDirectJoinFeature(prof.Versions()) {
			directJoinEnabled = true
		}
		slog.Info("Restoring running profile from lock file", "profile_id", info.ProfileID, "game_pid", info.GamePID, "direct_join", directJoinEnabled, "play_started_at", info.PlayStartedAt)
		l.setRunningDirectJoin(directJoinEnabled)
		l.setRunningPlayStartedAt(info.PlayStartedAt)
		l.setRunningProfile(info.ProfileID)
		restored = true
		if l.state.OnGameStarted != nil {
			l.state.OnGameStarted(info.ProfileID, info.GamePID)
		}
		l.watchRestoredRunningProfile(info.ProfileID, info.GamePID, info.PlayStartedAt)
		fyne.Do(l.checkLaunchState)
	}
}

func (l *Launcher) setupRoomLinkUI() {
	l.roomLinkEntry.SetText(roomLinkPlaceholder)
	l.copyRoomLinkButton.Disable()
	l.unpublishRoomButton.Disable()
	panelBackground := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	panelBackground.CornerRadius = theme.InputRadiusSize()
	panelSizer := canvas.NewRectangle(color.Transparent)
	panelSizer.SetMinSize(fyne.NewSize(roomLinkTrayWidth, 0))
	linkBackground := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	linkBackground.CornerRadius = theme.InputRadiusSize()
	l.roomLinkContainer = container.NewStack(
		panelSizer,
		panelBackground,
		container.NewPadded(container.NewVBox(
			l.roomLinkLabel,
			container.NewStack(
				linkBackground,
				container.NewHScroll(l.roomLinkEntry),
			),
			container.NewHBox(l.shareRoomButton, l.copyRoomLinkButton, l.unpublishRoomButton),
		)),
	)
	l.roomLinkTrayToggle = widget.NewButtonWithIcon(lang.LocalizeKey("launcher.join_link.tray_title", "Join Link"), theme.NavigateBackIcon(), func() {
		l.setRoomLinkTrayExpanded(!l.roomLinkTrayExpanded)
	})
	l.roomLinkTrayToggle.Importance = widget.LowImportance
	l.roomLinkTray = container.NewVBox(l.roomLinkContainer, l.roomLinkTrayToggle)
	l.setRoomLinkTrayExpanded(false)
	l.roomLinkTray.Hide()
}

func (l *Launcher) setRoomLinkTrayExpanded(expanded bool) {
	l.roomLinkTrayExpanded = expanded
	if l.roomLinkContainer == nil || l.roomLinkTrayToggle == nil {
		return
	}
	if expanded {
		l.roomLinkContainer.Show()
		l.roomLinkTrayToggle.SetIcon(theme.MoveDownIcon())
		return
	}
	l.roomLinkContainer.Hide()
	l.roomLinkTrayToggle.SetIcon(theme.MoveUpIcon())
}

func (l *Launcher) refreshProfileHighlights() {
	l.profileList.Refresh()
	l.refreshProfileGrid()
}

func (l *Launcher) newRunningProfileBadge() *fyne.Container {
	bg := canvas.NewCircle(launcherRunningProfileStrokeColor)
	bg.StrokeColor = theme.Color(theme.ColorNameBackground)
	bg.StrokeWidth = 1
	bg.Resize(fyne.NewSquareSize(launcherRunningBadgeSize))
	bg.Move(fyne.NewPos(launcherRunningBadgeGap, launcherRunningBadgeGap))

	icon := widget.NewIcon(theme.MediaPlayIcon())
	iconSize := launcherRunningBadgeSize * 0.62
	icon.Resize(fyne.NewSquareSize(iconSize))
	icon.Move(fyne.NewPos(
		launcherRunningBadgeGap+(launcherRunningBadgeSize-iconSize)/2,
		launcherRunningBadgeGap+(launcherRunningBadgeSize-iconSize)/2,
	))

	return container.NewWithoutLayout(bg, icon)
}

func (l *Launcher) applyProfileBorderStyle(bg *canvas.Rectangle, profileID uuid.UUID) {
	bg.StrokeColor = theme.Color(theme.ColorNameButton)
	bg.StrokeWidth = 1
	if l.isProfileRunning(profileID) {
		bg.StrokeColor = launcherRunningProfileStrokeColor
		bg.StrokeWidth = 2
		return
	}
	if profileID == l.selectedProfileID {
		bg.StrokeColor = theme.Color(theme.ColorNamePrimary)
		bg.StrokeWidth = 2
	}
}

func (l *Launcher) isDirectJoinEnabledForRunningProfile() bool {
	l.runningProfileMu.Lock()
	defer l.runningProfileMu.Unlock()
	return l.runningDirectJoin
}

func (l *Launcher) setRunningDirectJoin(enabled bool) {
	l.runningProfileMu.Lock()
	l.runningDirectJoin = enabled
	l.runningProfileMu.Unlock()
}

func (l *Launcher) setRunningPlayStartedAt(startedAt time.Time) {
	l.runningProfileMu.Lock()
	l.runningStartedAt = startedAt
	l.runningProfileMu.Unlock()
}

func hasDirectJoinFeature(versions []modmgr.ModVersion) bool {
	for _, v := range versions {
		if v.HasFeature(modmgr.FeatureDirectJoin) {
			return true
		}
	}
	return false
}

func (l *Launcher) onGameStarted(profileID uuid.UUID, pid int) {
	l.runningProfileMu.Lock()
	wasRunning := l.runningProfileID == profileID && l.runningGamePID > 0
	l.runningGamePID = pid
	directJoin := l.runningDirectJoin
	isRunning := l.runningProfileID == profileID && l.runningGamePID > 0
	l.runningProfileMu.Unlock()
	if wasRunning != isRunning {
		fyne.Do(l.refreshProfileHighlights)
	}
	if !directJoin || pid <= 0 || !l.state.Core.IsLobbyInfoAvailable() {
		fyne.Do(func() {
			l.refreshRoomLinkUI(nil, false)
		})
		return
	}
	l.startLobbyPolling(pid)
}

func (l *Launcher) onGameExited(profileID uuid.UUID) {
	l.stopLobbyPolling()
	l.runningProfileMu.Lock()
	wasRunning := l.runningProfileID == profileID && l.runningGamePID > 0
	l.runningGamePID = 0
	l.runningDirectJoin = false
	l.runningStartedAt = time.Time{}
	isRunning := l.runningProfileID == profileID && l.runningGamePID > 0
	l.runningProfileMu.Unlock()
	l.invalidateCachedRoomShareAsync()
	fyne.Do(func() {
		if wasRunning != isRunning {
			l.refreshProfileHighlights()
		}
		l.refreshRoomLinkUI(nil, false)
	})
}

func (l *Launcher) isCurrentRunningProcess(profileID uuid.UUID, pid int) bool {
	l.runningProfileMu.Lock()
	defer l.runningProfileMu.Unlock()
	return l.runningProfileID == profileID && l.runningGamePID == pid
}

func (l *Launcher) watchRestoredRunningProfile(profileID uuid.UUID, pid int, startedAt time.Time) {
	if profileID == uuid.Nil || pid <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(restoredProcessWatchInterval)
		defer ticker.Stop()
		for range ticker.C {
			if !l.isCurrentRunningProcess(profileID, pid) {
				return
			}
			running, err := aumgr.IsProcessRunning(pid)
			if err != nil {
				slog.Debug("Failed to check restored game process state", "profile_id", profileID, "pid", pid, "error", err)
				continue
			}
			if running {
				continue
			}
			if !l.isCurrentRunningProcess(profileID, pid) {
				return
			}
			l.onGameExited(profileID)
			l.clearRunningProfile(profileID)
			fyne.Do(l.checkLaunchState)
			if err := l.state.UpdateProfileLaunchMetrics(profileID, startedAt, time.Now()); err != nil {
				l.state.SetError(err)
			}
			return
		}
	}()
}

func (l *Launcher) startLobbyPolling(pid int) {
	l.stopLobbyPolling()
	stop := l.state.Core.StartLobbyInfoPolling(pid, lobbyPollInterval, func(info *core.IPCLobbyInfo) {
		slog.Info("Received lobby info", "info", fmt.Sprintf("%+v", info))
		fyne.Do(func() {
			l.refreshRoomLinkUI(info, true)
		})
	}, func(err error) {
		slog.Debug("Lobby polling failed", "error", err)
	})
	l.runningProfileMu.Lock()
	l.lobbyPollStop = stop
	l.runningProfileMu.Unlock()
}

func (l *Launcher) stopLobbyPolling() {
	l.runningProfileMu.Lock()
	stop := l.lobbyPollStop
	l.lobbyPollStop = nil
	l.runningProfileMu.Unlock()
	if stop != nil {
		stop()
	}
}

func (l *Launcher) currentRoomInfo(info *core.IPCLobbyInfo) (commonrest.RoomInfo, bool) {
	if info == nil {
		return commonrest.RoomInfo{}, false
	}
	if !info.IsConnected {
		return commonrest.RoomInfo{}, false
	}
	if strings.TrimSpace(info.LobbyCode) == "" {
		return commonrest.RoomInfo{}, false
	}
	room := commonrest.RoomInfo{
		LobbyCode: strings.TrimSpace(info.LobbyCode),
		ServerIP:  strings.TrimSpace(info.ServerIP),
	}
	if info.ServerPort > 0 && info.ServerPort <= 65535 {
		room.ServerPort = uint16(info.ServerPort)
	}
	return room, true
}

func roomKeyForCache(room commonrest.RoomInfo, profileID uuid.UUID) string {
	return strings.ToUpper(strings.TrimSpace(room.LobbyCode)) + "|" + strings.TrimSpace(room.ServerIP) + "|" + fmt.Sprint(room.ServerPort) + "|" + profileID.String()
}

func (l *Launcher) currentRunningProfileAndPID() (uuid.UUID, int) {
	l.runningProfileMu.Lock()
	defer l.runningProfileMu.Unlock()
	return l.runningProfileID, l.runningGamePID
}

func (l *Launcher) profileByID(profileID uuid.UUID) (profile.Profile, bool) {
	for _, prof := range l.profiles {
		if prof.ID == profileID {
			return prof, true
		}
	}
	return profile.Profile{}, false
}

func (l *Launcher) currentRunningProfile() (profile.Profile, int, bool) {
	profileID, runningPID := l.currentRunningProfileAndPID()
	if profileID == uuid.Nil {
		return profile.Profile{}, 0, false
	}
	prof, ok := l.profileByID(profileID)
	if !ok {
		return profile.Profile{}, 0, false
	}
	return prof, runningPID, true
}

func (l *Launcher) refreshRoomLinkUI(info *core.IPCLobbyInfo, running bool) {
	l.lobbyInfo = info
	if !running || !l.isDirectJoinEnabledForRunningProfile() {
		if l.roomLinkTray != nil {
			l.roomLinkTray.Hide()
		}
		l.setRoomLinkTrayExpanded(false)
		l.roomLinkEntry.SetText(roomLinkPlaceholder)
		l.copyRoomLinkButton.Disable()
		l.shareRoomButton.Disable()
		l.unpublishRoomButton.Disable()
		return
	}
	room, ok := l.currentRoomInfo(info)
	if !ok {
		if l.roomLinkTray != nil {
			l.roomLinkTray.Hide()
			l.content.Refresh()
		}
		l.setRoomLinkTrayExpanded(false)
		l.roomLinkEntry.SetText(roomLinkPlaceholder)
		l.copyRoomLinkButton.Disable()
		l.shareRoomButton.Disable()
		l.unpublishRoomButton.Disable()
		l.invalidateCachedRoomShareAsync()
		return
	}

	visible := l.roomLinkTray.Visible()
	if l.roomLinkTray.Show(); !visible {
		l.content.Refresh()
	}
	l.shareRoomButton.Enable()
	runningProfileID, _ := l.currentRunningProfileAndPID()
	key := roomKeyForCache(room, runningProfileID)
	l.roomShareMu.Lock()
	cache := l.roomShareCache
	l.roomShareMu.Unlock()
	if cache.RoomKey != key {
		l.roomLinkEntry.SetText(roomLinkPlaceholder)
		l.copyRoomLinkButton.Disable()
		l.unpublishRoomButton.Disable()
		if cache.SessionID != "" {
			l.invalidateCachedRoomShareAsync()
		}
		return
	}
	if cache.URL != "" && cache.ExpiresAt.After(time.Now()) {
		l.roomLinkEntry.SetText(cache.URL)
		l.copyRoomLinkButton.Enable()
		l.unpublishRoomButton.Enable()
	} else {
		l.roomLinkEntry.SetText(roomLinkPlaceholder)
		l.copyRoomLinkButton.Disable()
		l.unpublishRoomButton.Disable()
	}
}

func (l *Launcher) invalidateCachedRoomShareAsync() {
	l.roomShareMu.Lock()
	cache := l.roomShareCache
	l.roomShareCache = sharedRoomLinkCache{}
	l.roomShareMu.Unlock()
	if cache.SessionID == "" || cache.HostKey == "" {
		return
	}
	go func() {
		if err := l.state.Rest.DeleteSharedGame(cache.SessionID, cache.HostKey); err != nil {
			slog.Debug("Failed to invalidate shared room link", "error", err, "session_id", cache.SessionID)
		}
	}()
}

func (l *Launcher) copyRoomLinkToClipboard() {
	link := strings.TrimSpace(l.roomLinkEntry.Text)
	if link == "" {
		return
	}
	fyne.CurrentApp().Clipboard().SetContent(link)
	l.state.ShowInfoDialog(
		lang.LocalizeKey("common.success", "Success"),
		lang.LocalizeKey("launcher.join_link.copied", "参加リンクをコピーしました。"),
	)
}

func (l *Launcher) unpublishCurrentRoom() {
	l.roomShareMu.Lock()
	cache := l.roomShareCache
	if cache.SessionID == "" || cache.HostKey == "" {
		l.roomShareMu.Unlock()
		return
	}
	l.roomShareCache = sharedRoomLinkCache{}
	l.roomShareMu.Unlock()

	if err := l.state.Rest.DeleteSharedGame(cache.SessionID, cache.HostKey); err != nil {
		slog.Warn("Failed to unpublish room link", "error", err, "session_id", cache.SessionID)
	}

	fyne.Do(func() {
		l.roomLinkEntry.SetText(roomLinkPlaceholder)
		l.copyRoomLinkButton.Disable()
		l.unpublishRoomButton.Disable()
	})
	l.state.ShowInfoDialog(
		lang.LocalizeKey("common.success", "Success"),
		lang.LocalizeKey("launcher.join_link.unpublished", "参加リンクの公開を停止しました。"),
	)
}

func (l *Launcher) shareCurrentRoom() {
	l.roomShareMu.Lock()
	if l.roomShareGenerating {
		l.roomShareMu.Unlock()
		return
	}
	l.roomShareGenerating = true
	l.roomShareMu.Unlock()
	defer func() {
		l.roomShareMu.Lock()
		l.roomShareGenerating = false
		l.roomShareMu.Unlock()
	}()

	prof, _, ok := l.currentRunningProfile()
	if !ok {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.join_link.no_running_profile", "起動中のプロファイルが見つかりません。")))
		return
	}
	room, ok := l.currentRoomInfo(l.lobbyInfo)
	if !ok {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.join_link.no_room", "部屋情報を取得できません。部屋に参加してから再試行してください。")))
		return
	}
	roomKey := roomKeyForCache(room, prof.ID)

	l.roomShareMu.Lock()
	cache := l.roomShareCache
	l.roomShareMu.Unlock()
	if cache.RoomKey == roomKey && cache.URL != "" && cache.ExpiresAt.After(time.Now()) {
		fyne.Do(func() {
			l.roomLinkTray.Show()
			l.setRoomLinkTrayExpanded(true)
			l.roomLinkEntry.SetText(cache.URL)
			l.copyRoomLinkButton.Enable()
			l.unpublishRoomButton.Enable()
		})
		l.copyRoomLinkToClipboard()
		return
	}

	iconPNG, err := l.state.ProfileManager.LoadIconPNG(prof.ID)
	if err != nil {
		l.state.ShowErrorDialog(err)
		return
	}
	base := strings.TrimSpace(l.state.Rest.ServerBaseURL())
	if base == "" {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.join_link.server_unavailable", "このモードでは参加リンクを生成できません。")))
		return
	}
	aupack, err := l.state.Core.ExportProfileArchive(prof, iconPNG)
	if err != nil {
		l.state.ShowErrorDialog(err)
		return
	}
	rs, err := l.state.Rest.ShareGame(aupack, room)
	if err != nil {
		l.state.ShowErrorDialog(err)
		return
	}
	if strings.HasPrefix(rs.URL, "/") {
		rs.URL = strings.TrimRight(base, "/") + rs.URL
	}
	l.roomShareMu.Lock()
	l.roomShareCache = sharedRoomLinkCache{
		RoomKey:   roomKey,
		URL:       rs.URL,
		SessionID: rs.SessionID,
		HostKey:   rs.HostKey,
		ExpiresAt: rs.ExpiresAt,
	}
	l.roomShareMu.Unlock()
	fyne.Do(func() {
		l.roomLinkTray.Show()
		l.setRoomLinkTrayExpanded(true)
		l.roomLinkEntry.SetText(rs.URL)
		l.copyRoomLinkButton.Enable()
		l.unpublishRoomButton.Enable()
	})
	l.copyRoomLinkToClipboard()
}

func (l *Launcher) shareProfile(prof profile.Profile) {
	var d *dialog.CustomDialog
	shareCodeBtn := widget.NewButtonWithIcon(
		lang.LocalizeKey("profile.share.action.copy_code", "Copy Share Code"),
		theme.ContentCopyIcon(),
		func() {
			if d != nil {
				d.Hide()
			}
			l.shareProfileAsCode(prof, true)
		},
	)
	shareArchiveCopyBtn := widget.NewButtonWithIcon(
		lang.LocalizeKey("profile.share.action.copy_archive", "Copy Archive"),
		theme.ContentCopyIcon(),
		func() {
			if d != nil {
				d.Hide()
			}
			l.shareProfileAsArchive(prof, true)
		},
	)
	shareArchiveSaveBtn := widget.NewButtonWithIcon(
		lang.LocalizeKey("profile.share.action.save_archive", "Save Archive"),
		theme.DocumentSaveIcon(),
		func() {
			if d != nil {
				d.Hide()
			}
			l.shareProfileAsArchive(prof, false)
		},
	)
	content := container.NewVBox(
		widget.NewLabel(lang.LocalizeKey("profile.share.options_hint", "Choose share action.")),
		shareCodeBtn,
		shareArchiveCopyBtn,
		shareArchiveSaveBtn,
	)

	d = dialog.NewCustom(
		lang.LocalizeKey("profile.share.options_title", "Share Profile"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		content,
		l.state.Window,
	)
	d.Resize(fyne.NewSize(420, 240))
	d.Show()
}

func (l *Launcher) shareProfileAsCode(prof profile.Profile, copyToClipboard bool) {
	uri, err := l.state.Core.ExportProfile(prof)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	if copyToClipboard {
		fyne.CurrentApp().Clipboard().SetContent(uri)
		dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("profile.shared_clipboard", "共有コードをコピーしました。"), l.state.Window)
		return
	}
	l.saveProfileShareCodeToFile(prof, uri)
}

func (l *Launcher) saveProfileShareCodeToFile(prof profile.Profile, uri string) {
	path, err := l.state.ExplorerSaveFile(
		lang.LocalizeKey("profile.share.code_file_type", "共有コード"),
		"*.txt",
		profileShareFileBaseName(prof)+".txt",
	)
	if err != nil {
		slog.Info("Save share code cancelled or failed", "error", err)
		return
	}
	if err := os.WriteFile(path, []byte(uri), 0600); err != nil {
		dialog.ShowError(fmt.Errorf("failed to save share code: %w", err), l.state.Window)
		return
	}
	dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("profile.share.saved", "保存しました。"), l.state.Window)
}

func (l *Launcher) shareProfileAsArchive(prof profile.Profile, copyToClipboard bool) {
	iconPNG, err := l.state.ProfileManager.LoadIconPNG(prof.ID)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	archive, err := l.state.Core.ExportProfileArchive(prof, iconPNG)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	if copyToClipboard {
		tempFilePath := filepath.Join(os.TempDir(), profileShareFileBaseName(prof)+".aupack")
		if err := os.WriteFile(tempFilePath, archive, 0600); err != nil {
			dialog.ShowError(fmt.Errorf("failed to create temporary archive file: %w", err), l.state.Window)
			return
		}
		if err := l.state.ClipboardSetFile(tempFilePath); err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("profile.share.archive_clipboard", "アーカイブをコピーしました。"), l.state.Window)
		return
	}

	path, err := l.state.ExplorerSaveFile(
		lang.LocalizeKey("profile.share.archive_file_type", "アーカイブ"),
		"*.aupack",
		profileShareFileBaseName(prof)+".aupack",
	)
	if err != nil {
		slog.Info("Save archive cancelled or failed", "error", err)
		return
	}
	if err := os.WriteFile(path, archive, 0600); err != nil {
		dialog.ShowError(fmt.Errorf("failed to save profile archive: %w", err), l.state.Window)
		return
	}
	dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("profile.share.saved", "保存しました。"), l.state.Window)
}

func profileShareFileBaseName(prof profile.Profile) string {
	name := strings.TrimSpace(prof.Name)
	if name == "" {
		return "mod-of-us-profile"
	}
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	for _, ch := range invalidChars {
		name = strings.ReplaceAll(name, ch, "_")
	}
	return name
}

func (l *Launcher) showImportDialog() {
	var d *dialog.CustomDialog
	importCodeBtn := widget.NewButtonWithIcon(
		lang.LocalizeKey("profile.import_clipboard", "Import from Clipboard"),
		theme.ContentPasteIcon(),
		func() {
			if d != nil {
				d.Hide()
			}
			l.showImportCodeDialog()
		},
	)
	importArchiveBtn := widget.NewButtonWithIcon(
		lang.LocalizeKey("profile.import_file", "アーカイブからインポート"),
		theme.FolderOpenIcon(),
		func() {
			if d != nil {
				d.Hide()
			}
			l.importProfileFromArchiveFileDialog()
		},
	)
	content := container.NewVBox(
		widget.NewLabel(lang.LocalizeKey("profile.import_source_hint", "Choose how to import profile data.")),
		importCodeBtn,
		importArchiveBtn,
		widget.NewSeparator(),
		widget.NewLabel(lang.LocalizeKey("profile.import_drop_hint", "Or drop an archive (.aupack) onto this window to import.")),
	)
	d = dialog.NewCustom(
		lang.LocalizeKey("profile.import_source_title", "Import Profile"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		content,
		l.state.Window,
	)
	d.Resize(fyne.NewSize(500, 220))
	d.Show()
}

func (l *Launcher) showImportCodeDialog() {
	entry := widget.NewMultiLineEntry()
	entry.PlaceHolder = "mod-of-us://profile/..."
	entry.SetMinRowsVisible(3)

	dialog.ShowCustomConfirm(lang.LocalizeKey("profile.import_title", "Import Profile"), lang.LocalizeKey("common.add", "Import"), lang.LocalizeKey("common.cancel", "Cancel"), entry, func(confirm bool) {
		if !confirm {
			return
		}
		l.state.SharedURI = strings.TrimSpace(entry.Text)
		l.checkSharedURI()
	}, l.state.Window)
}

func (l *Launcher) importProfileFromArchiveFileDialog() {
	path, err := l.state.ExplorerOpenFile(
		lang.LocalizeKey("profile.import_file_dialog_type", "アーカイブ"),
		"*.aupack",
	)
	if err != nil {
		slog.Info("Archive selection cancelled or failed", "error", err)
		return
	}
	l.importProfileFromArchiveFile(path)
}

func (l *Launcher) importProfileFromArchiveFile(path string) {
	shared, iconPNG, err := l.state.Core.HandleSharedProfileArchiveFile(path)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	l.confirmAndImportProfile(shared, iconPNG)
}

func (l *Launcher) importProfileFromArchiveURI(uri fyne.URI) {
	if uri == nil {
		dialog.ShowError(errors.New(lang.LocalizeKey("profile.import_drop_unsupported", "Dropped item is not supported.")), l.state.Window)
		return
	}
	if !strings.EqualFold(uri.Extension(), ".aupack") {
		dialog.ShowError(errors.New(lang.LocalizeKey("profile.import_drop_unsupported", "Dropped item is not supported.")), l.state.Window)
		return
	}

	reader, err := storage.Reader(uri)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to read dropped archive: %w", err), l.state.Window)
		return
	}
	defer reader.Close()

	tempFile, err := os.CreateTemp("", "mod-of-us-profile-*.aupack")
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to create temp file for dropped archive: %w", err), l.state.Window)
		return
	}
	defer os.Remove(tempFile.Name())
	if _, err := io.Copy(tempFile, reader); err != nil {
		dialog.ShowError(fmt.Errorf("failed to save dropped archive: %w", err), l.state.Window)
		return
	}

	stat, err := tempFile.Stat()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to stat temp file: %w", err), l.state.Window)
		return
	}

	shared, iconPNG, err := l.state.Core.HandleSharedProfileArchive(tempFile, stat.Size())
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	l.confirmAndImportProfile(shared, iconPNG)
}

func (l *Launcher) handleDroppedURIs(uris []fyne.URI) {
	for _, uri := range uris {
		if uri != nil && strings.EqualFold(uri.Extension(), ".aupack") {
			l.importProfileFromArchiveURI(uri)
			return
		}
	}
	dialog.ShowError(errors.New(lang.LocalizeKey("profile.import_drop_no_zip", "ドロップされた項目にアーカイブ(.aupack)が見つかりませんでした。")), l.state.Window)
}

func (l *Launcher) checkSharedURI() {
	if l.state.SharedURI == "" {
		return
	}
	sharedURI := strings.TrimSpace(l.state.SharedURI)
	l.state.SharedURI = ""
	if sharedURI == "" {
		return
	}

	if parsed, err := neturl.Parse(sharedURI); err == nil {
		switch {
		case strings.EqualFold(parsed.Scheme, "http"), strings.EqualFold(parsed.Scheme, "https"):
			l.importProfileFromArchiveURL(parsed.String())
			return
		case strings.EqualFold(parsed.Scheme, "mod-of-us") && strings.EqualFold(parsed.Host, "join_game"):
			l.handleJoinGameURI(sharedURI)
			return
		}
	}

	prof, err := l.state.Core.HandleSharedProfile(sharedURI)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	l.confirmAndImportProfile(prof, nil)
}

func (l *Launcher) handleJoinGameURI(sharedURI string) {
	joinURI, err := l.state.Core.ParseJoinGameURI(sharedURI)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	if joinURI.Error != "" {
		l.state.ShowErrorDialog(errors.New(joinURI.Error))
		return
	}

	client := clientrest.NewClient(joinURI.ServerBase)
	rs, err := client.GetJoinGameDownload(joinURI.SessionID)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	tmpFile, err := os.CreateTemp("", "mod-of-us-join-*.aupack")
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(rs.Aupack); err != nil {
		_ = tmpFile.Close()
		dialog.ShowError(err, l.state.Window)
		return
	}
	stat, err := tmpFile.Stat()
	if err != nil {
		_ = tmpFile.Close()
		dialog.ShowError(err, l.state.Window)
		return
	}
	shared, iconPNG, err := l.state.Core.HandleSharedProfileArchive(tmpFile, stat.Size())
	_ = tmpFile.Close()
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	joinInfo := &core.LaunchJoinInfo{
		LobbyCode:  rs.Room.LobbyCode,
		ServerIP:   rs.Room.ServerIP,
		ServerPort: rs.Room.ServerPort,
	}
	runningProfileID, runningPID := l.currentRunningProfileAndPID()
	if runningProfile, ok := l.state.Core.ProfileManager.Get(l.runningProfileID); ok && runningPID > 0 && runningProfileID == shared.ID && hasDirectJoinFeature(runningProfile.Versions()) {
		if err := l.state.Core.SendLobbyJoinByPID(runningPID, *joinInfo); err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		l.state.ShowInfoDialog(
			lang.LocalizeKey("common.success", "Success"),
			lang.LocalizeKey("launcher.join_link.join_sent", "起動中のゲームに部屋参加リクエストを送信しました。"),
		)
		return
	}
	if err := l.importProfileWithJoinInfo(shared, iconPNG, joinInfo); err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
}

func (l *Launcher) checkSharedArchive() {
	if l.state.SharedArchive == "" {
		return
	}
	path := l.state.SharedArchive
	l.state.SharedArchive = ""
	l.importProfileFromArchiveFile(path)
}

func (l *Launcher) importProfileFromArchiveURL(archiveURL string) {
	if archiveURL == "" {
		return
	}

	var loadingDialog dialog.Dialog
	var loadingProgress *progress.FyneProgress
	bar := widget.NewProgressBar()
	bar.SetValue(0)
	loadingProgress = progress.NewFyneProgress(bar)
	fyne.DoAndWait(func() {
		content := container.NewVBox(
			widget.NewLabel(lang.LocalizeKey("profile.import_url_loading", "アーカイブをダウンロードしています...")),
			bar,
		)
		loadingDialog = dialog.NewCustomWithoutButtons(
			lang.LocalizeKey("profile.import_title", "Import Profile"),
			content,
			l.state.Window,
		)
		loadingDialog.Resize(fyne.NewSize(420, 130))
		loadingDialog.Show()
	})

	go func() {
		path, err := l.downloadArchiveURLToTempFile(archiveURL, loadingProgress)
		fyne.Do(func() {
			if loadingDialog != nil {
				loadingDialog.Hide()
			}
		})
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, l.state.Window)
			})
			return
		}
		defer os.Remove(path)
		fyne.DoAndWait(func() {
			l.importProfileFromArchiveFile(path)
		})
	}()
}

func (l *Launcher) downloadArchiveURLToTempFile(archiveURL string, progressListener progress.Progress) (string, error) {
	parsedURL, err := neturl.Parse(archiveURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse archive URL: %w", err)
	}
	if !strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https") {
		return "", fmt.Errorf("unsupported archive URL scheme: %s", parsedURL.Scheme)
	}

	ctx, cancel := context.WithTimeout(context.Background(), profileArchiveDownloadTimeout)
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
	if resp.ContentLength > profileArchiveDownloadMaxBytes {
		return "", fmt.Errorf("archive is too large: %d bytes (max %d)", resp.ContentLength, profileArchiveDownloadMaxBytes)
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
	written, copyErr := io.Copy(buf, io.LimitReader(resp.Body, profileArchiveDownloadMaxBytes+1))
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
	if written > profileArchiveDownloadMaxBytes {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("archive is too large: more than %d bytes", profileArchiveDownloadMaxBytes)
	}
	return tempPath, nil
}

func (l *Launcher) confirmAndImportProfile(prof *profile.SharedProfile, iconPNG []byte) {
	dialog.ShowConfirm(lang.LocalizeKey("profile.import_title", "Import Profile"), lang.LocalizeKey("profile.import_message", "Do you want to import the shared profile '{{.Name}}'?", map[string]any{"Name": prof.Name}), func(confirm bool) {
		if !confirm {
			return
		}

		if existing, found := l.state.ProfileManager.Get(prof.ID); found {
			if existing.UpdatedAt.After(prof.UpdatedAt) {
				dialog.ShowConfirm(lang.LocalizeKey("profile.overwrite_title", "Overwrite Profile"), lang.LocalizeKey("profile.overwrite_message", "The existing profile is newer than the imported one. Do you want to overwrite it?"), func(confirm bool) {
					if !confirm {
						return
					}
					l.importProfile(prof, iconPNG)
				}, l.state.Window)
				return
			}
		}

		l.importProfile(prof, iconPNG)
	}, l.state.Window)
}

func (l *Launcher) importProfile(shared *profile.SharedProfile, iconPNG []byte) {
	prof := profile.Profile{
		ID:          shared.ID,
		Name:        shared.Name,
		Author:      shared.Author,
		Description: shared.Description,
		UpdatedAt:   time.Now(),
	}

	if p, ok := l.state.ProfileManager.Get(shared.ID); ok {
		prof.PlayDurationNS = p.PlayDurationNS
		prof.LastLaunchedAt = p.LastLaunchedAt
	}

	// Fetch mod version infos
	for modID, versionID := range shared.ModVersions {
		info, err := l.state.Rest.GetModVersion(modID, versionID)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to fetch mod version info for %s:%s: %w", modID, versionID, err), l.state.Window)
			return
		}
		prof.AddModVersion(*info)
	}

	if err := l.state.ProfileManager.Add(prof); err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	if len(iconPNG) > 0 {
		if err := l.state.ProfileManager.SaveIconPNG(prof.ID, iconPNG); err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
	}
	l.refreshProfiles()
}

func (l *Launcher) importProfileWithJoinInfo(shared *profile.SharedProfile, iconPNG []byte, joinInfo *core.LaunchJoinInfo) error {
	prof := profile.Profile{
		ID:          shared.ID,
		Name:        shared.Name,
		Author:      shared.Author,
		Description: shared.Description,
		UpdatedAt:   time.Now(),
	}

	if p, ok := l.state.ProfileManager.Get(shared.ID); ok {
		prof.PlayDurationNS = p.PlayDurationNS
		prof.LastLaunchedAt = p.LastLaunchedAt
	}

	for modID, versionID := range shared.ModVersions {
		info, err := l.state.Rest.GetModVersion(modID, versionID)
		if err != nil {
			return fmt.Errorf("failed to fetch mod version info for %s:%s: %w", modID, versionID, err)
		}
		prof.AddModVersion(*info)
	}

	if err := l.state.ProfileManager.Add(prof); err != nil {
		return err
	}
	if len(iconPNG) > 0 {
		if err := l.state.ProfileManager.SaveIconPNG(prof.ID, iconPNG); err != nil {
			return err
		}
	}

	l.refreshProfiles()
	for i, p := range l.profiles {
		if p.ID == prof.ID {
			l.profileList.Select(i)
			break
		}
	}
	if joinInfo != nil {
		l.state.SetPendingJoinInfo(joinInfo)
		l.runLaunch()
	}
	return nil
}

func (l *Launcher) setupProfileList() {
	l.profileList = widget.NewList(
		func() int {
			return len(l.profiles)
		},
		func() fyne.CanvasObject {
			img := l.newProfileIconCanvas(profile.Profile{}, launcherListThumbMinSize, 8)
			imgBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
			imgBg.CornerRadius = 3
			runningBadge := l.newRunningProfileBadge()
			runningBadge.Hide()
			imgArea := container.NewStack(imgBg, img, runningBadge)

			title := widget.NewLabel("Profile Name")
			title.TextStyle = fyne.TextStyle{Bold: true}
			title.SizeName = theme.SizeNameSubHeadingText
			title.Wrapping = fyne.TextWrapOff
			title.Truncation = fyne.TextTruncateEllipsis
			meta := widget.NewLabel("Last launched")
			meta.SizeName = theme.SizeNameCaptionText
			meta.TextStyle = fyne.TextStyle{Monospace: true}
			meta.Wrapping = fyne.TextWrapOff
			stats := widget.NewLabel("Mods and play time")
			stats.SizeName = theme.SizeNameCaptionText
			stats.Wrapping = fyne.TextWrapOff
			textArea := container.NewVBox(title, meta, stats)
			menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
			menuBtn.Importance = widget.LowImportance

			content := container.New(&launcherListItemLayout{
				minThumbSize: launcherListThumbMinSize,
				spacing:      theme.Padding(),
			},
				imgArea,
				menuBtn,
				container.NewPadded(textArea),
			)
			tappable := uicommon.NewTappableContainerWithSecondary(content, nil, nil)

			bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
			bg.StrokeColor = theme.Color(theme.ColorNameButton)
			bg.StrokeWidth = 1
			bg.CornerRadius = theme.InputRadiusSize()

			return container.NewStack(bg, container.NewPadded(tappable))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(l.profiles) {
				return
			}
			prof := l.profiles[id]
			root := item.(*fyne.Container)
			bg := root.Objects[0].(*canvas.Rectangle)
			tappable := root.Objects[1].(*fyne.Container).Objects[0].(*uicommon.TappableContainer)
			content := tappable.Content.(*fyne.Container)

			imgArea := content.Objects[0].(*fyne.Container)
			img := imgArea.Objects[1].(*canvas.Image)
			runningBadge := imgArea.Objects[2]
			l.refreshProfileIconCanvas(img, prof, int(launcherListThumbMinSize))
			if l.isProfileRunning(prof.ID) {
				runningBadge.Show()
			} else {
				runningBadge.Hide()
			}

			textArea := content.Objects[2].(*fyne.Container).Objects[0].(*fyne.Container)
			title := textArea.Objects[0].(*widget.Label)
			meta := textArea.Objects[1].(*widget.Label)
			stats := textArea.Objects[2].(*widget.Label)
			menuBtn := content.Objects[1].(*widget.Button)
			title.SetText(prof.Name)
			meta.SetText(l.profileMetaText(prof))
			stats.SetText(fmt.Sprintf("Mods: %d  Play: %s", len(prof.ModVersions), formatPlayDuration(time.Duration(prof.PlayDurationNS))))

			tappable.OnTapped = func() {
				l.profileList.Select(id)
			}
			tappable.OnSecondaryTapped = func(ev *fyne.PointEvent) {
				l.showProfileMenuAt(ev.AbsolutePosition, prof)
			}

			menuBtn.OnTapped = func() {
				l.showProfileMenuAt(fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn).Add(fyne.NewPos(0, fyne.CanvasObject(menuBtn).Size().Height)), prof)
			}

			l.applyProfileBorderStyle(bg, prof.ID)
			bg.Refresh()
		},
	)

	l.profileList.OnSelected = func(id widget.ListItemID) {
		if id >= len(l.profiles) {
			return
		}
		l.selectedProfileID = l.profiles[id].ID
		_ = l.state.ActiveProfile.Set(l.selectedProfileID.String())
		l.checkLaunchState()
		l.profileList.Refresh()
		l.refreshProfileGrid()
	}
	l.profileList.OnUnselected = func(id widget.ListItemID) {
		l.selectedProfileID = uuid.Nil
		l.checkLaunchState()
		l.profileList.Refresh()
		l.refreshProfileGrid()
	}
}

type launcherListItemLayout struct {
	minThumbSize float32
	spacing      float32
}

func (l *launcherListItemLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	thumb := objects[0]
	menu := objects[1]
	body := objects[2]

	menuSize := menu.MinSize()
	menu.Resize(menuSize)
	menuX := size.Width - menuSize.Width
	if menuX < 0 {
		menuX = 0
	}
	menuY := (size.Height - menuSize.Height) / 2
	if menuY < 0 {
		menuY = 0
	}
	menu.Move(fyne.NewPos(menuX, menuY))

	thumbSide := max(size.Height, l.minThumbSize)
	maxThumbSide := size.Width - menuSize.Width - l.spacing*2
	if maxThumbSide < 0 {
		maxThumbSide = 0
	}
	if thumbSide > maxThumbSide {
		thumbSide = maxThumbSide
	}
	thumb.Resize(fyne.NewSize(thumbSide, thumbSide))
	// thumb.Move(fyne.NewPos(0, (size.Height-thumbSide)/2))

	bodyX := thumbSide + l.spacing
	bodyRight := menuX - l.spacing
	bodyWidth := bodyRight - bodyX
	if bodyWidth < 0 {
		bodyWidth = 0
	}
	body.Resize(fyne.NewSize(bodyWidth, size.Height))
	body.Move(fyne.NewPos(bodyX, 0))
}

func (l *launcherListItemLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 3 {
		return fyne.NewSize(0, 0)
	}
	thumbMinSize := objects[0].MinSize()
	menuMinSize := objects[1].MinSize()
	bodyMinSize := objects[2].MinSize()

	thumbSide := max(l.minThumbSize, max(thumbMinSize.Width, thumbMinSize.Height))
	height := max(thumbSide, max(menuMinSize.Height, bodyMinSize.Height))
	width := thumbSide + l.spacing + l.spacing + menuMinSize.Width
	return fyne.NewSize(width, height)
}

func (l *Launcher) setupProfileGrid() {
	l.profileGrid = container.New(&launcherProfileGridLayout{
		cardSize: fyne.NewSize(launcherGridCardWidth, launcherGridCardHeight),
		spacing:  theme.Padding(),
	})
	l.profileGridScroll = container.NewVScroll(l.profileGrid)
}

type launcherProfileGridLayout struct {
	cardSize fyne.Size
	spacing  float32
}

func (l *launcherProfileGridLayout) columnCount(width float32) int {
	if l.cardSize.Width <= 0 {
		return 1
	}
	cols := int((width + l.spacing) / (l.cardSize.Width + l.spacing))
	if cols < 1 {
		return 1
	}
	return cols
}

func (l *launcherProfileGridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	cols := l.columnCount(size.Width)

	gridWidth := float32(cols)*l.cardSize.Width + float32(cols-1)*l.spacing
	offsetX := (size.Width - gridWidth) / 2
	if offsetX < 0 {
		offsetX = 0
	}

	for i, obj := range objects {
		row := i / cols
		col := i % cols
		x := offsetX + float32(col)*(l.cardSize.Width+l.spacing)
		y := float32(row) * (l.cardSize.Height + l.spacing)
		obj.Move(fyne.NewPos(x, y))
		obj.Resize(l.cardSize)
	}
}

func (l *launcherProfileGridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if l.cardSize.Width <= 0 || l.cardSize.Height <= 0 {
		return fyne.NewSize(0, 0)
	}
	if len(objects) == 0 {
		return fyne.NewSize(l.cardSize.Width, 0)
	}
	rows := len(objects)
	height := float32(rows)*l.cardSize.Height + float32(max(rows-1, 0))*l.spacing
	return fyne.NewSize(l.cardSize.Width, height)
}

func (l *Launcher) setupToolbar() {
	l.toggleViewButton = widget.NewButtonWithIcon("", theme.GridIcon(), func() {
		l.isGridView = !l.isGridView
		if l.isGridView {
			fyne.CurrentApp().Preferences().SetString(prefLauncherViewMode, viewModeGrid)
		} else {
			fyne.CurrentApp().Preferences().SetString(prefLauncherViewMode, viewModeList)
		}
		l.updateViewToggleButton()
		l.switchProfileView()
	})
	l.toggleViewButton.Importance = widget.LowImportance
	l.sortSelect = widget.NewSelect([]string{
		lang.LocalizeKey("launcher.sort.name", "Name"),
		lang.LocalizeKey("launcher.sort.playtime", "Play Time"),
		lang.LocalizeKey("launcher.sort.recent", "Recently Launched"),
	}, func(selected string) {
		switch selected {
		case lang.LocalizeKey("launcher.sort.playtime", "Play Time"):
			l.sortMode = sortModePlaytime
		case lang.LocalizeKey("launcher.sort.recent", "Recently Launched"):
			l.sortMode = sortModeRecent
		default:
			l.sortMode = sortModeName
		}
		fyne.CurrentApp().Preferences().SetString(prefLauncherSortMode, l.sortMode)
		l.refreshProfiles()
	})
	l.sortOrderButton = widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		l.sortDescending = !l.sortDescending
		fyne.CurrentApp().Preferences().SetBool(prefLauncherSortDescending, l.sortDescending)
		l.updateSortOrderButton()
		l.refreshProfiles()
	})
	l.sortOrderButton.Importance = widget.LowImportance
	l.sortSelect.SetSelected(l.sortModeLabel(l.sortMode))
	l.updateSortOrderButton()
	l.updateViewToggleButton()
}

func (l *Launcher) updateViewToggleButton() {
	if l.isGridView {
		l.toggleViewButton.SetIcon(theme.ListIcon())
		// l.toggleViewButton.SetText(lang.LocalizeKey("launcher.view.list", "List"))
		return
	}
	l.toggleViewButton.SetIcon(theme.GridIcon())
	// l.toggleViewButton.SetText(lang.LocalizeKey("launcher.view.grid", "Grid"))
}

func (l *Launcher) switchProfileView() {
	if l.profileViews == nil {
		return
	}
	if l.isGridView {
		l.profileGridScroll.Show()
		l.profileList.Hide()
		return
	}
	l.profileList.Show()
	l.profileGridScroll.Hide()
}

func (l *Launcher) refreshProfileGrid() {
	if l.profileGrid == nil {
		return
	}
	var items []fyne.CanvasObject
	for i, prof := range l.profiles {
		index := i
		p := prof

		iconSize := launcherGridIconAreaSize - launcherGridIconInset*2
		img := l.newProfileIconCanvas(p, iconSize, 3)

		text := widget.NewLabelWithStyle(prof.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		text.Wrapping = fyne.TextWrapOff
		text.Truncation = fyne.TextTruncateEllipsis
		desc := canvas.NewText(l.profileMetaText(p), theme.Color(theme.ColorNameDisabled))
		desc.TextSize = theme.TextSize() * 0.76

		menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
		menuBtn.Importance = widget.LowImportance
		menuBtn.Resize(fyne.NewSquareSize(launcherGridMenuSize))
		menuBtn.Move(fyne.NewPos(
			launcherGridIconAreaSize-launcherGridMenuSize-launcherGridMenuInset,
			launcherGridMenuInset,
		))
		menuBtn.OnTapped = func() {
			l.showProfileMenuAt(fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn).Add(fyne.NewPos(0, fyne.CanvasObject(menuBtn).Size().Height)), p)
		}

		runningBadge := l.newRunningProfileBadge()
		if !l.isProfileRunning(p.ID) {
			runningBadge.Hide()
		}

		iconAreaBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		iconAreaBg.CornerRadius = 6
		iconAreaBg.Resize(fyne.NewSquareSize(launcherGridIconAreaSize))
		iconAreaBg.SetMinSize(fyne.NewSquareSize(launcherGridIconAreaSize))
		img.Move(fyne.NewPos(launcherGridIconInset, launcherGridIconInset))
		img.Resize(fyne.NewSquareSize(iconSize))
		iconAreaSizer := canvas.NewRectangle(color.Transparent)
		iconAreaSizer.SetMinSize(fyne.NewSquareSize(launcherGridIconAreaSize))
		iconArea := container.NewStack(
			iconAreaSizer,
			container.NewWithoutLayout(iconAreaBg, img, menuBtn, runningBadge),
		)

		cardContent := container.NewBorder(
			nil,
			desc,
			nil,
			nil,
			container.NewVBox(
				container.New(
					layout.NewGridWrapLayout(
						fyne.NewSquareSize(launcherGridIconAreaSize),
					),
					iconArea,
				),
				text,
			),
		)
		tappable := uicommon.NewTappableContainerWithSecondary(cardContent, func() {
			l.profileList.Select(index)
		}, func(ev *fyne.PointEvent) {
			l.showProfileMenuAt(ev.AbsolutePosition, p)
		})

		bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
		bg.CornerRadius = theme.InputRadiusSize()
		l.applyProfileBorderStyle(bg, p.ID)

		items = append(items, container.NewStack(bg, container.NewPadded(tappable)))
	}
	if len(items) == 0 {
		items = append(items, container.NewCenter(widget.NewLabel(lang.LocalizeKey("launcher.no_profiles", "No profiles found."))))
	}
	fyne.Do(func() {
		l.profileGrid.Objects = items
		l.profileGrid.Refresh()
	})
}

func (l *Launcher) Tab() (*container.TabItem, error) {
	header := container.NewVBox(
		widget.NewCard(lang.LocalizeKey("launcher.card_title", "Mod of Us"), lang.LocalizeKey("launcher.card_subtitle", "Among Us Mod Manager"), l.greetingContent),
		container.NewPadded(container.NewBorder(
			nil,
			nil,
			container.NewHBox(l.toggleViewButton, l.sortOrderButton, l.sortSelect),
			container.NewHBox(l.createProfileButton, l.importProfileButton),
		)),
	)

	footer := container.NewVBox(
		l.roomLinkTray,
		l.launchButton,
		l.state.ErrorText,
	)
	l.profileViews = container.NewStack(l.profileList, l.profileGridScroll)
	l.switchProfileView()

	l.content = container.NewBorder(
		header,
		footer,
		nil,
		nil,
		l.profileViews,
	)
	return container.NewTabItem(lang.LocalizeKey("launcher.tab_name", "Launcher"), l.content), nil
}

func (l *Launcher) runLaunch() {
	l.state.ClearError()
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.error.no_path", "Game path is not specified.")))
		return
	}

	if l.selectedProfileID == uuid.Nil {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.error.no_profile", "Please select a profile to launch.")))
		return
	}

	binaryType, err := aumgr.GetBinaryType(path)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	// Find selected profile
	var targetProfile profile.Profile
	for _, prof := range l.profiles {
		if prof.ID == l.selectedProfileID {
			targetProfile = prof
			break
		}
	}
	if targetProfile.ID == uuid.Nil {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.error.no_profile", "Please select a profile to launch.")))
		return
	}
	if l.isAnyProfileBusy() {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("error.game_already_running", "Already running.")))
		return
	}
	l.setRunningDirectJoin(false)

	launchDialog, launchProgress := l.newLaunchProgressDialog()
	l.setLaunchingProfile(targetProfile.ID, true)
	l.checkLaunchState()
	l.launchButton.Disable()
	if err := l.state.CanInstall.Set(false); err != nil {
		slog.Warn("Failed to set canInstall", "error", err)
	}
	if err := l.state.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set canLaunch", "error", err)
	} // Disable launch while downloading
	fyne.Do(launchDialog.Show)

	go func() {
		var launchErr error
		progressShown := true
		defer func() {
			l.setLaunchingProfile(targetProfile.ID, false)
			fyne.DoAndWait(func() {
				if progressShown {
					launchDialog.Hide()
				}
				l.checkLaunchState()
			})
			if launchErr != nil {
				l.state.SetError(launchErr)
			}
		}()

		// Resolve dependencies
		resolvedVersions, err := l.state.Core.ResolveDependencies(targetProfile.Versions())
		if err != nil {
			launchErr = fmt.Errorf("failed to resolve dependencies: %w", err)
			return
		}

		// Download mods to cache
		configDir, err := os.UserConfigDir()
		if err != nil {
			launchErr = err
			return
		}
		cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")

		if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, launchProgress, false); err != nil {
			launchErr = err
			return
		}

		// Set active profile
		if err := l.state.ActiveProfile.Set(targetProfile.ID.String()); err != nil {
			launchErr = err
			return
		}

		l.state.ClearError()
		fyne.DoAndWait(func() {
			launchDialog.Hide()
			progressShown = false
		})
		l.setRunningDirectJoin(hasDirectJoinFeature(resolvedVersions))
		l.setLaunchingProfile(targetProfile.ID, false)
		l.setRunningProfile(targetProfile.ID)
		fyne.DoAndWait(l.checkLaunchState)

		// Proceed to Launch
		l.state.Launch(path, hasDirectJoinFeature(resolvedVersions))
		l.stopLobbyPolling()
		l.setRunningDirectJoin(false)
		l.setRunningPlayStartedAt(time.Time{})
		l.clearRunningProfile(targetProfile.ID)
		fyne.DoAndWait(l.checkLaunchState)
	}()
}

func (l *Launcher) newLaunchProgressDialog() (*dialog.CustomDialog, *progress.FyneProgress) {
	return l.newProgressDialog(
		"launcher.launch.title",
		"Launching",
		"launcher.launch.in_progress",
		"Preparing launch. Please wait...",
	)
}

func (l *Launcher) checkLaunchState() {
	runningProfileID, launching := l.currentBusyProfile()
	if runningProfileID != uuid.Nil && launching {
		l.launchButton.SetText(lang.LocalizeKey("launcher.launch.preparing", "Preparing launch..."))
		l.launchButton.SetIcon(theme.MediaStopIcon())
		l.launchButton.Disable()
		return
	}
	if runningProfileID != uuid.Nil {
		l.launchButton.SetText(lang.LocalizeKey("launcher.launch.running", "Running..."))
		l.launchButton.SetIcon(theme.MediaStopIcon())
		l.launchButton.Disable()
		return
	}
	l.launchButton.SetText(lang.LocalizeKey("launcher.launch", "Launch"))
	l.launchButton.SetIcon(theme.MediaPlayIcon())

	// Enable launch if profile selected and game path exists
	// We might also check if game is running (handled in state.Launch but button state is good to have)

	// Check Game Path
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		l.launchButton.Disable()
		return
	}
	if _, err := os.Stat(filepath.Join(path, "Among Us.exe")); os.IsNotExist(err) {
		l.launchButton.Disable()
		return
	}

	// Check Profile Selected
	if l.selectedProfileID == uuid.Nil {
		l.launchButton.Disable()
		return
	}

	l.launchButton.Enable()
}

func (l *Launcher) setRunningProfile(profileID uuid.UUID) {
	if profileID == uuid.Nil {
		return
	}
	l.runningProfileMu.Lock()
	changed := l.runningProfileID != profileID
	l.runningProfileID = profileID
	l.runningProfileMu.Unlock()
	if changed {
		fyne.Do(l.refreshProfileHighlights)
	}
}

func (l *Launcher) clearRunningProfile(profileID uuid.UUID) {
	l.runningProfileMu.Lock()
	changed := false
	if l.runningProfileID == profileID {
		l.runningProfileID = uuid.Nil
		changed = true
	}
	l.runningProfileMu.Unlock()
	if changed {
		fyne.Do(l.refreshProfileHighlights)
	}
}

func (l *Launcher) setLaunchingProfile(profileID uuid.UUID, launching bool) {
	l.runningProfileMu.Lock()
	l.launchingProfile = launching
	if launching {
		l.launchingProfileID = profileID
	} else if l.launchingProfileID == profileID {
		l.launchingProfileID = uuid.Nil
	}
	l.runningProfileMu.Unlock()
}

func (l *Launcher) currentBusyProfile() (uuid.UUID, bool) {
	l.runningProfileMu.Lock()
	defer l.runningProfileMu.Unlock()
	if l.launchingProfile {
		return l.launchingProfileID, true
	}
	return l.runningProfileID, false
}

func (l *Launcher) isAnyProfileBusy() bool {
	runningProfileID, launching := l.currentBusyProfile()
	return launching || runningProfileID != uuid.Nil
}

func (l *Launcher) isProfileBusy(profileID uuid.UUID) bool {
	if profileID == uuid.Nil {
		return false
	}
	l.runningProfileMu.Lock()
	defer l.runningProfileMu.Unlock()
	if l.launchingProfile && l.launchingProfileID == profileID {
		return true
	}
	return l.runningProfileID == profileID
}

func (l *Launcher) isProfileRunning(profileID uuid.UUID) bool {
	if profileID == uuid.Nil {
		return false
	}
	l.runningProfileMu.Lock()
	defer l.runningProfileMu.Unlock()
	return l.runningProfileID == profileID && l.runningGamePID > 0
}

func (l *Launcher) syncProfile(prof profile.Profile) {
	if l.isProfileBusy(prof.ID) {
		dialog.ShowError(errors.New(lang.LocalizeKey("error.game_already_running", "Already running.")), l.state.Window)
		return
	}
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		dialog.ShowError(errors.New(lang.LocalizeKey("launcher.error.no_path", "Game path is not specified.")), l.state.Window)
		return
	}

	binaryType, err := aumgr.GetBinaryType(path)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	gameVersion, err := aumgr.GetVersion(path)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}

	syncDialog, syncProgress := l.newSyncProgressDialog()
	l.launchButton.Disable()
	if err := l.state.CanInstall.Set(false); err != nil {
		slog.Warn("Failed to set canInstall", "error", err)
	}
	if err := l.state.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set canLaunch", "error", err)
	}
	fyne.Do(syncDialog.Show)

	go func() {
		var syncErr error
		defer func() {
			fyne.DoAndWait(func() {
				syncDialog.Hide()
				l.checkLaunchState()
			})
			if syncErr != nil {
				l.state.SetError(syncErr)
				return
			}
			l.state.ClearError()
			l.state.ShowInfoDialog(
				lang.LocalizeKey("common.success", "Success"),
				lang.LocalizeKey("launcher.sync.success", "Profile has been re-synced and mods re-downloaded."),
			)
		}()

		// Resolve dependencies
		resolvedVersions, err := l.state.Core.ResolveDependencies(prof.Versions())
		if err != nil {
			syncErr = fmt.Errorf("failed to resolve dependencies: %w", err)
			return
		}

		// Download mods to cache with force=false (don't clear global cache)
		configDir, err := os.UserConfigDir()
		if err != nil {
			syncErr = err
			return
		}
		cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")

		downloadProgress := progress.NewPhaseProgress(syncProgress, 0.0, 0.5)
		if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, downloadProgress, true); err != nil {
			syncErr = err
			return
		}

		// Sync profile directory
		copyProgress := progress.NewPhaseProgress(syncProgress, 0.5, 0.5)
		if err := l.state.Core.SyncProfile(prof.ID, binaryType, gameVersion, copyProgress); err != nil {
			syncErr = err
			return
		}
	}()
}

func (l *Launcher) newSyncProgressDialog() (*dialog.CustomDialog, *progress.FyneProgress) {
	return l.newProgressDialog(
		"launcher.sync.title",
		"Syncing Profile",
		"launcher.sync.in_progress",
		"Syncing profile. Please wait...",
	)
}

func (l *Launcher) newProgressDialog(titleKey, titleDefault, messageKey, messageDefault string) (*dialog.CustomDialog, *progress.FyneProgress) {
	bar := widget.NewProgressBar()
	bar.SetValue(0)
	progressBar := progress.NewFyneProgress(bar)
	content := container.NewVBox(
		widget.NewLabel(lang.LocalizeKey(messageKey, messageDefault)),
		bar,
	)
	d := dialog.NewCustomWithoutButtons(
		lang.LocalizeKey(titleKey, titleDefault),
		content,
		l.state.Window,
	)
	d.Resize(fyne.NewSize(420, 140))
	return d, progressBar
}

func (l *Launcher) refreshProfiles() {
	l.profiles = l.state.ProfileManager.List()
	l.sortProfiles()
	l.profileList.Refresh()
	l.refreshProfileGrid()

	// Select active profile
	activeIDStr, _ := l.state.ActiveProfile.Get()
	if activeIDStr != "" {
		activeID, _ := uuid.Parse(activeIDStr)
		for i, p := range l.profiles {
			if p.ID == activeID {
				l.profileList.Select(i)
				break
			}
		}
	} else {
		l.profileList.UnselectAll()
		l.selectedProfileID = uuid.Nil
		l.checkLaunchState()
		l.refreshProfileGrid()
	}
}

func (l *Launcher) sortProfiles() {
	sort.SliceStable(l.profiles, func(i, j int) bool {
		cmp := l.compareProfiles(l.profiles[i], l.profiles[j])
		if cmp == 0 {
			return false
		}
		if l.sortDescending {
			return cmp > 0
		}
		return cmp < 0
	})
}

func (l *Launcher) profileMetaText(p profile.Profile) string {
	if p.LastLaunchedAt.IsZero() {
		return lang.LocalizeKey("launcher.meta.never_launched", "Never launched")
	}
	return lang.LocalizeKey("launcher.meta.last_launched", "Last: {{.Date}}", map[string]any{
		"Date": p.LastLaunchedAt.Format("2006-01-02"),
	})
}

func normalizeSortMode(mode string) string {
	switch mode {
	case sortModePlaytime, sortModeRecent, sortModeName:
		return mode
	default:
		return sortModeName
	}
}

func defaultSortDescendingForMode(mode string) bool {
	switch mode {
	case sortModePlaytime, sortModeRecent:
		return true
	default:
		return false
	}
}

func (l *Launcher) compareProfiles(a, b profile.Profile) int {
	nameCmp := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	switch l.sortMode {
	case sortModePlaytime:
		if a.PlayDurationNS < b.PlayDurationNS {
			return -1
		}
		if a.PlayDurationNS > b.PlayDurationNS {
			return 1
		}
		return nameCmp
	case sortModeRecent:
		if a.LastLaunchedAt.Before(b.LastLaunchedAt) {
			return -1
		}
		if a.LastLaunchedAt.After(b.LastLaunchedAt) {
			return 1
		}
		return nameCmp
	default:
		return nameCmp
	}
}

func (l *Launcher) updateSortOrderButton() {
	if l.sortDescending {
		l.sortOrderButton.SetIcon(theme.MoveDownIcon())
		return
	}
	l.sortOrderButton.SetIcon(theme.MoveUpIcon())
}

func (l *Launcher) sortModeLabel(mode string) string {
	switch mode {
	case sortModePlaytime:
		return lang.LocalizeKey("launcher.sort.playtime", "Play Time")
	case sortModeRecent:
		return lang.LocalizeKey("launcher.sort.recent", "Recently Launched")
	default:
		return lang.LocalizeKey("launcher.sort.name", "Name")
	}
}

func formatPlayDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int64(d.Round(time.Second).Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func (l *Launcher) showProfileMenuAt(pos fyne.Position, prof profile.Profile) {
	isBusy := l.isProfileBusy(prof.ID)
	editItem := fyne.NewMenuItem(lang.LocalizeKey("profile.edit", "Edit"), func() {
		l.openProfileEditor(prof)
	})
	syncItem := fyne.NewMenuItem(lang.LocalizeKey("profile.sync", "Sync (Clear & Re-download)"), func() {
		l.syncProfile(prof)
	})
	shareItem := fyne.NewMenuItem(lang.LocalizeKey("profile.share", "Share"), func() {
		l.shareProfile(prof)
	})
	openFolderItem := fyne.NewMenuItem(lang.LocalizeKey("profile.open_folder", "Open Folder"), func() {
		l.openProfileFolder(prof)
	})
	duplicateItem := fyne.NewMenuItem(lang.LocalizeKey("profile.duplicate", "Duplicate"), func() {
		l.showDuplicateDialog(prof)
	})
	deleteItem := fyne.NewMenuItem(lang.LocalizeKey("profile.delete", "Delete"), func() {
		l.deleteProfile(prof.ID)
	})
	if isBusy {
		editItem.Disabled = true
		syncItem.Disabled = true
		shareItem.Disabled = true
		openFolderItem.Disabled = true
		duplicateItem.Disabled = true
		deleteItem.Disabled = true
	}
	menu := fyne.NewMenu("",
		editItem,
		syncItem,
		shareItem,
		openFolderItem,
		duplicateItem,
		deleteItem,
	)
	widget.ShowPopUpMenuAtPosition(menu, l.state.Window.Canvas(), pos)
}

func (l *Launcher) openProfileFolder(prof profile.Profile) {
	dir, err := l.state.ProfileManager.ProfileDir(prof.ID)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		dialog.ShowError(fmt.Errorf("failed to create profile directory: %w", err), l.state.Window)
		return
	}
	if err := l.state.ExplorerOpenFolder(dir); err != nil {
		dialog.ShowError(err, l.state.Window)
	}
}

// -- Profile Management Methods --

func (l *Launcher) createProfile() {
	baseName := "New Profile"
	name := baseName
	counter := 1
	existing := l.state.ProfileManager.List()
	for {
		found := false
		for _, prof := range existing {
			if prof.Name == name {
				found = true
				break
			}
		}
		if !found {
			break
		}
		counter++
		name = fmt.Sprintf("%s (%d)", baseName, counter)
	}

	prof := profile.Profile{
		ID:          uuid.New(),
		Name:        name,
		ModVersions: map[string]modmgr.ModVersion{},
		UpdatedAt:   time.Now(),
	}

	if err := l.state.ProfileManager.Add(prof); err != nil {
		dialog.ShowError(err, l.state.Window)
		return
	}
	l.refreshProfiles()

	// Select the new profile
	for i, pr := range l.profiles {
		if pr.ID == prof.ID {
			l.profileList.Select(i)
			break
		}
	}

	l.openProfileEditor(prof)
}

func (l *Launcher) deleteProfile(id uuid.UUID) {
	if id == uuid.Nil {
		return
	}

	dialog.ShowConfirm(lang.LocalizeKey("profile.delete_confirm_title", "Delete Profile"), lang.LocalizeKey("profile.delete_confirm_message", "Are you sure you want to delete this profile?"), func(confirm bool) {
		if !confirm {
			return
		}

		if err := l.state.ProfileManager.Remove(id); err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		l.refreshProfiles()
		l.profileList.UnselectAll()
	}, l.state.Window)
}

func (l *Launcher) showDuplicateDialog(prof profile.Profile) {
	entry := widget.NewEntry()
	entry.SetText(prof.Name + " - Copy")
	entry.Validator = func(s string) error {
		if s == "" {
			return os.ErrInvalid
		}
		return nil
	}

	d := dialog.NewForm(lang.LocalizeKey("profile.duplicate_title", "Duplicate Profile"), lang.LocalizeKey("common.save", "Save"), lang.LocalizeKey("common.cancel", "Cancel"), []*widget.FormItem{
		widget.NewFormItem(lang.LocalizeKey("profile.name", "Profile Name"), entry),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		newName := entry.Text

		newProf := prof
		newProf.ID = uuid.New()
		newProf.Name = newName
		newProf.UpdatedAt = time.Now()

		if err := l.state.ProfileManager.Add(newProf); err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		iconPNG, err := l.state.ProfileManager.LoadIconPNG(prof.ID)
		if err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		if len(iconPNG) > 0 {
			if err := l.state.ProfileManager.SaveIconPNG(newProf.ID, iconPNG); err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}
		}
		l.refreshProfiles()
	}, l.state.Window)
	d.Resize(fyne.NewSize(400, 200))
	d.Show()
}

func (l *Launcher) openProfileEditor(prof profile.Profile) {
	currentProfile := prof

	var saveBtn *widget.Button
	var removeIconBtn *widget.Button
	var d *dialog.CustomDialog
	nameEntry := widget.NewEntry()
	nameEntry.SetText(currentProfile.Name)
	nameEntry.OnChanged = func(s string) {
		if nameEntry.Validate() != nil {
			saveBtn.Disable()
		} else {
			saveBtn.Enable()
		}
	}
	nameEntry.Validator = func(s string) (err error) {
		if s == "" {
			return errors.New(lang.LocalizeKey("profile.error_name_empty", "Profile name cannot be empty"))
		}
		return nil
	}
	nameForm := widget.NewForm(widget.NewFormItem(lang.LocalizeKey("profile.name", "Profile Name"), nameEntry))

	lastLaunchedText := lang.LocalizeKey("profile.stats.never_launched", "Last Launch: Never")
	if !currentProfile.LastLaunchedAt.IsZero() {
		lastLaunchedText = lang.LocalizeKey("profile.stats.last_launched", "Last Launch: {{.Date}}", map[string]any{
			"Date": currentProfile.LastLaunchedAt.Format("2006-01-02 15:04:05"),
		})
	}
	playDurationText := lang.LocalizeKey("profile.stats.play_time", "Play Time: {{.Duration}}", map[string]any{
		"Duration": formatPlayDuration(currentProfile.PlayDuration()),
	})
	statsContent := container.NewVBox(
		widget.NewLabel(lastLaunchedText),
		widget.NewLabel(playDurationText),
	)
	statsCard := widget.NewCard(lang.LocalizeKey("profile.stats.title", "Play Stats"), "", statsContent)

	currentIconPNG, err := l.state.ProfileManager.LoadIconPNG(currentProfile.ID)
	if err != nil {
		dialog.ShowError(err, l.state.Window)
		currentIconPNG = nil
	}
	selectedIconPNG := []byte(nil)
	removeIcon := false

	iconPlaceholder := l.newProfileIconCanvasFromPNG(currentIconPNG, 128, 8)
	selectIconBtn := widget.NewButtonWithIcon(lang.LocalizeKey("profile.icon.select", "Select Icon"), theme.FolderOpenIcon(), func() {
		l.showProfileIconSelectionDialog(currentProfile, func(iconPNG []byte) {
			selectedIconPNG = iconPNG
			currentIconPNG = iconPNG
			removeIcon = false
			l.refreshProfileIconCanvasFromPNG(iconPlaceholder, currentIconPNG, 128)
			removeIconBtn.Enable()
		})
	})
	removeIconBtn = widget.NewButtonWithIcon(lang.LocalizeKey("profile.icon.remove", "Remove Icon"), theme.DeleteIcon(), func() {
		selectedIconPNG = nil
		currentIconPNG = nil
		removeIcon = true
		l.refreshProfileIconCanvasFromPNG(iconPlaceholder, currentIconPNG, 128)
		removeIconBtn.Disable()
	})
	if len(currentIconPNG) == 0 {
		removeIconBtn.Disable()
	}
	iconArea := container.NewVBox(
		container.NewCenter(iconPlaceholder),
		container.NewGridWithRows(2, selectIconBtn, removeIconBtn),
	)

	modList := widget.NewList(
		func() int { return len(currentProfile.Versions()) },
		func() fyne.CanvasObject {
			thumb := l.newModThumbnailCanvas("", 64, 6)
			thumbBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
			thumbBg.CornerRadius = 6
			thumbArea := container.NewStack(thumbBg, container.NewCenter(thumb))

			label := widget.NewLabel("Mod Name")
			label.Wrapping = fyne.TextWrapWord
			badge := widget.NewLabel("")
			badge.Hide()
			textArea := container.NewVBox(label, badge)
			updateBtn := widget.NewButtonWithIcon("", theme.DownloadIcon(), nil)
			updateBtn.Hide()
			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			buttonsArea := container.NewHBox(updateBtn, deleteBtn)
			content := container.New(layout.NewBorderLayout(nil, nil, thumbArea, buttonsArea),
				thumbArea,
				buttonsArea,
				container.NewPadded(textArea),
			)
			return container.NewPadded(content)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// This will be overridden by UpdateItem below
		},
	)

	updatesAvailable := make(map[string]string) // ModID -> LatestVersionID
	var applyLatestBtn *widget.Button
	applyLatestBtn = widget.NewButtonWithIcon(
		lang.LocalizeKey("profile.apply_latest", "最新バージョンを適用"),
		theme.DownloadIcon(),
		func() {
			if len(updatesAvailable) == 0 {
				return
			}
			for modID, latestID := range updatesAvailable {
				version, fetchErr := l.state.Rest.GetModVersion(modID, latestID)
				if fetchErr != nil {
					dialog.ShowError(fmt.Errorf("failed to fetch latest version for %s:%s: %w", modID, latestID, fetchErr), l.state.Window)
					return
				}
				currentProfile.AddModVersion(*version)
				delete(updatesAvailable, modID)
			}
			applyLatestBtn.Hide()
			modList.Refresh()
		},
	)
	applyLatestBtn.Hide()
	go func() {
		installed := make(map[string]string)
		for _, v := range currentProfile.Versions() {
			installed[v.ModID] = v.VersionID
		}
		updates, err := l.state.Rest.CheckForUpdates(installed)
		if err == nil {
			for modID, latest := range updates {
				updatesAvailable[modID] = latest.VersionID
			}
			fyne.Do(func() {
				if len(updatesAvailable) > 0 {
					applyLatestBtn.Show()
				} else {
					applyLatestBtn.Hide()
				}
				modList.Refresh()
			})
		}
	}()

	// Hook up update item to ensure closure correctness
	modList.UpdateItem = func(id widget.ListItemID, item fyne.CanvasObject) {
		if id >= len(currentProfile.Versions()) {
			return
		}
		v := currentProfile.Versions()[id]
		c := item.(*fyne.Container).Objects[0].(*fyne.Container)
		thumbArea := c.Objects[0].(*fyne.Container)
		thumb := thumbArea.Objects[1].(*fyne.Container).Objects[0].(*canvas.Image)
		textArea := c.Objects[2].(*fyne.Container).Objects[0].(*fyne.Container)
		label := textArea.Objects[0].(*widget.Label)
		badge := textArea.Objects[1].(*widget.Label)
		buttonsArea := c.Objects[1].(*fyne.Container)
		updateBtn := buttonsArea.Objects[0].(*widget.Button)
		delBtn := buttonsArea.Objects[1].(*widget.Button)

		l.refreshModThumbnailCanvas(thumb, v.ModID, 64)
		l.ensureModThumbnailLoaded(v.ModID, modList.Refresh)
		label.SetText(v.ModID + " (" + v.VersionID + ")")
		label.Wrapping = fyne.TextWrapOff
		label.Truncation = fyne.TextTruncateEllipsis

		if latestID, ok := updatesAvailable[v.ModID]; ok {
			badge.SetText(lang.LocalizeKey("repository.update_available", "Update Available") + " (" + latestID + ")")
			badge.Importance = widget.WarningImportance
			badge.Show()
			updateBtn.Show()
			updateBtn.OnTapped = func() {
				modID := v.ModID
				latestVersionID := latestID
				version, fetchErr := l.state.Rest.GetModVersion(modID, latestVersionID)
				if fetchErr != nil {
					dialog.ShowError(fmt.Errorf("failed to fetch latest version for %s:%s: %w", modID, latestVersionID, fetchErr), l.state.Window)
					return
				}
				currentProfile.AddModVersion(*version)
				delete(updatesAvailable, modID)
				if len(updatesAvailable) == 0 {
					applyLatestBtn.Hide()
				}
				modList.Refresh()
			}
		} else {
			badge.Hide()
			updateBtn.Hide()
		}

		delBtn.OnTapped = func() {
			currentProfile.RemoveModVersion(v.ModID)
			delete(updatesAvailable, v.ModID)
			if len(updatesAvailable) == 0 {
				applyLatestBtn.Hide()
			}
			modList.Refresh()
		}
	}

	addModBtn := widget.NewButtonWithIcon(lang.LocalizeKey("profile.add_mod", "Add Mod"), theme.ContentAddIcon(), func() {
		l.showAddModDialog(func(addedMods []modmgr.ModVersion) {
			for _, m := range addedMods {
				currentProfile.AddModVersion(m)
			}
			modList.Refresh()
		})
	})

	saveBtn = widget.NewButtonWithIcon(lang.LocalizeKey("common.save", "Save"), theme.DocumentSaveIcon(),
		func() {
			if err := nameForm.Validate(); err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}
			newName := nameEntry.Text
			if err := nameEntry.Validate(); err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}

			oldID := prof.ID
			currentProfile.Name = newName
			currentProfile.UpdatedAt = time.Now()

			if err := l.state.ProfileManager.Add(currentProfile); err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}
			if removeIcon && selectedIconPNG == nil {
				if err := l.state.ProfileManager.RemoveIcon(currentProfile.ID); err != nil {
					dialog.ShowError(err, l.state.Window)
					return
				}
			}
			if selectedIconPNG != nil {
				if err := l.state.ProfileManager.SaveIconPNG(currentProfile.ID, selectedIconPNG); err != nil {
					dialog.ShowError(err, l.state.Window)
					return
				}
			}

			if oldID != currentProfile.ID {
				if err := l.state.ProfileManager.Remove(oldID); err != nil {
					slog.Warn("Failed to remove old profile", "error", err)
				}
			}

			l.refreshProfiles()
			for i, pr := range l.profiles {
				if pr.ID == currentProfile.ID {
					l.profileList.Select(i)
					break
				}
			}
			d.Dismiss()
		})

	content := container.NewBorder(
		container.NewVBox(
			container.NewBorder(
				nil,
				nil,
				iconArea,
				nil,
				container.NewBorder(
					nil,
					statsCard,
					nil,
					nil,
					nameForm,
				),
			),
			widget.NewSeparator(),
			container.NewBorder(
				nil,
				nil,
				widget.NewLabel(lang.LocalizeKey("profile.mods", "Mods")),
				applyLatestBtn,
			),
		),
		addModBtn, nil, nil,
		modList,
	)

	d = dialog.NewCustomWithoutButtons(
		lang.LocalizeKey("profile.edit_title", "Edit Profile"),
		container.NewVScroll(content),
		l.state.Window,
	)
	d.SetButtons([]fyne.CanvasObject{
		widget.NewButtonWithIcon(lang.LocalizeKey("common.cancel", "Cancel"), theme.CancelIcon(), func() {
			d.Dismiss()
		}),
		saveBtn,
	})
	d.Resize(fyne.NewSize(500, 600))
	d.Show()
}

func (l *Launcher) showProfileIconSelectionDialog(prof profile.Profile, onSelect func([]byte)) {
	var d *dialog.CustomDialog
	selectFromExplorerBtn := widget.NewButtonWithIcon(
		lang.LocalizeKey("profile.icon.select_from_explorer", "Choose from Explorer"),
		theme.FolderOpenIcon(),
		func() {
			iconPNG, err := l.pickProfileIconFromExplorer()
			if err != nil {
				dialog.ShowError(err, l.state.Window)
				return
			}
			if len(iconPNG) == 0 {
				return
			}
			if onSelect != nil {
				onSelect(iconPNG)
			}
			d.Dismiss()
		},
	)

	modIDSet := map[string]struct{}{}
	modIDs := make([]string, 0, len(prof.Versions()))
	for _, version := range prof.Versions() {
		if version.ModID == "" {
			continue
		}
		if _, exists := modIDSet[version.ModID]; exists {
			continue
		}
		modIDSet[version.ModID] = struct{}{}
		modIDs = append(modIDs, version.ModID)
	}
	sort.Strings(modIDs)

	modsLabel := widget.NewLabelWithStyle(
		lang.LocalizeKey("profile.icon.select_from_mod_thumbnails", "Choose from MOD thumbnails"),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	var modArea fyne.CanvasObject
	if len(modIDs) == 0 {
		modArea = widget.NewLabel(lang.LocalizeKey("profile.icon.no_mods_in_profile", "No MODs in this profile."))
	} else {
		modGrid := container.NewGridWrap(fyne.NewSize(130, 148))
		for _, modID := range modIDs {
			thumb := l.newModThumbnailCanvas(modID, 88, 8)
			l.ensureModThumbnailLoaded(modID, modGrid.Refresh)
			thumbBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
			thumbBg.CornerRadius = 8

			name := widget.NewLabel(modID)
			name.Alignment = fyne.TextAlignCenter
			name.Wrapping = fyne.TextWrapOff
			name.Truncation = fyne.TextTruncateEllipsis

			itemBody := container.NewVBox(
				container.NewCenter(container.NewStack(thumbBg, container.NewCenter(thumb))),
				name,
			)
			item := uicommon.NewTappableContainer(itemBody, func() {
				iconPNG, err := l.profileIconPNGFromModThumbnail(modID)
				if err != nil {
					dialog.ShowError(err, l.state.Window)
					return
				}
				if onSelect != nil {
					onSelect(iconPNG)
				}
				d.Dismiss()
			})

			itemBg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
			itemBg.StrokeColor = theme.Color(theme.ColorNameButton)
			itemBg.StrokeWidth = 1
			itemBg.CornerRadius = theme.InputRadiusSize()
			modGrid.Add(container.NewStack(itemBg, container.NewPadded(item)))
		}
		scroll := container.NewVScroll(modGrid)
		scroll.SetMinSize(fyne.NewSize(0, 300))
		modArea = scroll
	}

	content := container.NewVBox(
		widget.NewLabel(lang.LocalizeKey("profile.icon.select_source_hint", "Select how to choose the profile icon.")),
		selectFromExplorerBtn,
		widget.NewSeparator(),
		modsLabel,
		modArea,
	)

	d = dialog.NewCustom(
		lang.LocalizeKey("profile.icon.select_source_title", "Select Profile Icon"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		content,
		l.state.Window,
	)
	d.Resize(fyne.NewSize(560, 500))
	d.Show()
}

func (l *Launcher) pickProfileIconFromExplorer() ([]byte, error) {
	path, err := l.state.ExplorerOpenFile("Profile Icon", "*.png;*.jpg;*.jpeg;*.gif")
	if err != nil {
		slog.Info("File selection cancelled or failed", "error", err)
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoded, _, err := image.Decode(f)
	if err != nil {
		return nil, errors.New(lang.LocalizeKey("profile.icon.invalid", "Selected file is not a valid image."))
	}

	iconPNG, err := encodeSquarePNG(decoded)
	if err != nil {
		return nil, err
	}
	return iconPNG, nil
}

func (l *Launcher) profileIconPNGFromModThumbnail(modID string) ([]byte, error) {
	if modID == "" {
		return nil, errors.New(lang.LocalizeKey("profile.icon.mod_thumbnail_unavailable", "MOD thumbnail is unavailable."))
	}

	l.modThumbMu.Lock()
	cached := l.modThumbnailImageCache[modID]
	l.modThumbMu.Unlock()
	if cached != nil {
		return encodeSquarePNG(cached)
	}
	if l.state.Rest == nil {
		return nil, errors.New(lang.LocalizeKey("profile.icon.mod_thumbnail_unavailable", "MOD thumbnail is unavailable."))
	}

	thumbBytes, err := l.state.Rest.GetModThumbnail(modID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", lang.LocalizeKey("profile.icon.mod_thumbnail_load_failed", "Failed to load MOD thumbnail."), err)
	}
	if len(thumbBytes) == 0 {
		return nil, errors.New(lang.LocalizeKey("profile.icon.mod_thumbnail_unavailable", "MOD thumbnail is unavailable."))
	}

	decoded, _, err := image.Decode(bytes.NewReader(thumbBytes))
	if err != nil {
		return nil, errors.New(lang.LocalizeKey("profile.icon.mod_thumbnail_invalid", "MOD thumbnail is not a valid image."))
	}
	decoded = centerCropSquare(decoded)

	l.modThumbMu.Lock()
	l.modThumbnailImageCache[modID] = decoded
	l.modThumbnailFetched[modID] = true
	l.modThumbMu.Unlock()

	return encodeSquarePNG(decoded)
}

func (l *Launcher) showAddModDialog(onAdd func([]modmgr.ModVersion)) {
	contentBox := container.NewVBox()
	scroll := container.NewVScroll(contentBox)

	// Create dialog first
	var d *dialog.CustomDialog

	buildItem := func(modID, title, subtitle string, onTap func()) fyne.CanvasObject {
		thumb := l.newModThumbnailCanvas(modID, 80, 6)
		l.ensureModThumbnailLoaded(modID, func() {
			l.refreshModThumbnailCanvas(thumb, modID, 80)
		})
		thumbBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		thumbBg.CornerRadius = 6
		thumbArea := container.NewStack(thumbBg, container.NewCenter(thumb))

		titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		titleLabel.Wrapping = fyne.TextWrapOff
		titleLabel.Truncation = fyne.TextTruncateEllipsis
		subtitleLabel := widget.NewLabel(subtitle)
		subtitleLabel.Wrapping = fyne.TextWrapOff
		subtitleLabel.Truncation = fyne.TextTruncateEllipsis
		textContainer := container.NewVBox(titleLabel, subtitleLabel)

		itemContent := container.New(layout.NewBorderLayout(nil, nil, thumbArea, nil),
			thumbArea,
			container.NewPadded(textContainer),
		)

		card := uicommon.NewTappableContainer(itemContent, onTap)
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
		bg.StrokeColor = theme.Color(theme.ColorNameButton)
		bg.StrokeWidth = 1
		bg.CornerRadius = theme.InputRadiusSize()
		return container.NewStack(bg, container.NewPadded(card))
	}

	go func() {
		modIDs, err := l.state.Rest.GetModIDs(100, "", "")
		if err != nil {
			dialog.ShowError(err, l.state.Window)
			return
		}
		fyne.DoAndWait(func() {
			contentBox.Objects = nil
			for range modIDs {
				contentBox.Add(buildItem("", lang.LocalizeKey("profile.loading_mod", "Loading mod details..."), "", nil))
			}
			endLabel := widget.NewLabel(lang.LocalizeKey("common.scroll_end_reached", "Reached the bottom."))
			endLabel.Alignment = fyne.TextAlignCenter
			endLabel.Importance = widget.LowImportance
			contentBox.Add(container.NewCenter(endLabel))
			contentBox.Refresh()
		})

		for i, modID := range modIDs {
			go func(index int, id string) {
				mod, fetchErr := l.state.Rest.GetMod(id)
				fyne.Do(func() {
					if index >= len(contentBox.Objects) {
						return
					}
					if fetchErr != nil || mod == nil {
						if fetchErr != nil {
							slog.Warn("Failed to fetch mod details", "modID", id, "error", fetchErr)
						}
						title := lang.LocalizeKey("profile.failed_mod", "Failed to load mod '{{.ID}}'", map[string]any{"ID": id})
						subtitle := lang.LocalizeKey("profile.failed_mod_description", "Reopen this dialog to retry")
						contentBox.Objects[index] = buildItem(id, title, subtitle, nil)
						contentBox.Refresh()
						return
					}

					contentBox.Objects[index] = buildItem(mod.ID, mod.Name, mod.Author, func() {
						detailsDialog := l.newModDetailsDialog(mod, func(v modmgr.ModVersion) {
							onAdd([]modmgr.ModVersion{v})
							d.Dismiss()
						})
						detailsDialog.Show()
					})
					contentBox.Refresh()
				})
			}(i, modID)
		}
	}()

	d = dialog.NewCustom(
		lang.LocalizeKey("profile.add_mod_title", "Add Mods"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		scroll,
		l.state.Window,
	)
	d.Resize(fyne.NewSize(600, 600))
	d.Show()
}

func placeholderProfileIcon(size int) image.Image {
	return image.NewPaletted(image.Rect(0, 0, size, size), color.Palette{theme.Color(theme.ColorNameDisabled)})
}

func centerCropSquare(src image.Image) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return placeholderProfileIcon(1)
	}

	side := min(width, height)
	startX := bounds.Min.X + (width-side)/2
	startY := bounds.Min.Y + (height-side)/2
	dstRect := image.Rect(0, 0, side, side)
	dst := image.NewRGBA(dstRect)
	imagedraw.Draw(dst, dstRect, src, image.Point{X: startX, Y: startY}, imagedraw.Src)
	return dst
}

func encodeSquarePNG(src image.Image) ([]byte, error) {
	cropped := centerCropSquare(src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, cropped); err != nil {
		return nil, fmt.Errorf("failed to encode profile icon: %w", err)
	}
	return buf.Bytes(), nil
}

func (l *Launcher) profileSquareIconImage(prof profile.Profile, fallbackSize int) image.Image {
	if prof.ID == uuid.Nil {
		return placeholderProfileIcon(fallbackSize)
	}

	iconPNG, err := l.state.ProfileManager.LoadIconPNG(prof.ID)
	if err != nil {
		slog.Warn("Failed to load profile icon image", "profileID", prof.ID.String(), "error", err)
		return placeholderProfileIcon(fallbackSize)
	}
	return l.squareIconImageFromPNG(iconPNG, fallbackSize, prof.ID)
}

func (l *Launcher) squareIconImageFromPNG(iconPNG []byte, fallbackSize int, profileID uuid.UUID) image.Image {
	if len(iconPNG) == 0 {
		return placeholderProfileIcon(fallbackSize)
	}
	decoded, _, err := image.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		slog.Warn("Failed to decode profile icon image", "profileID", profileID.String(), "error", err)
		return placeholderProfileIcon(fallbackSize)
	}
	return centerCropSquare(decoded)
}

func (l *Launcher) newProfileIconCanvas(prof profile.Profile, size float32, cornerRadius float32) *canvas.Image {
	img := canvas.NewImageFromImage(l.profileSquareIconImage(prof, int(size)))
	img.CornerRadius = cornerRadius
	img.SetMinSize(fyne.NewSquareSize(size))
	img.FillMode = canvas.ImageFillContain
	return img
}

func (l *Launcher) newProfileIconCanvasFromPNG(iconPNG []byte, size float32, cornerRadius float32) *canvas.Image {
	img := canvas.NewImageFromImage(l.squareIconImageFromPNG(iconPNG, int(size), uuid.Nil))
	img.CornerRadius = cornerRadius
	img.SetMinSize(fyne.NewSquareSize(size))
	img.FillMode = canvas.ImageFillContain
	return img
}

func (l *Launcher) refreshProfileIconCanvas(target *canvas.Image, prof profile.Profile, fallbackSize int) {
	target.Image = l.profileSquareIconImage(prof, fallbackSize)
	target.SetMinSize(fyne.NewSquareSize(float32(fallbackSize)))
	target.Refresh()
}

func (l *Launcher) refreshProfileIconCanvasFromPNG(target *canvas.Image, iconPNG []byte, fallbackSize int) {
	target.Image = l.squareIconImageFromPNG(iconPNG, fallbackSize, uuid.Nil)
	target.SetMinSize(fyne.NewSquareSize(float32(fallbackSize)))
	target.Refresh()
}

func placeholderModThumbnail(size int) image.Image {
	return image.NewPaletted(image.Rect(0, 0, max(size, 1), max(size, 1)), color.Palette{theme.Color(theme.ColorNameDisabled)})
}

func (l *Launcher) modThumbnailImage(modID string, fallbackSize int) image.Image {
	l.modThumbMu.Lock()
	img := l.modThumbnailImageCache[modID]
	l.modThumbMu.Unlock()
	if img == nil {
		return placeholderModThumbnail(fallbackSize)
	}
	return img
}

func (l *Launcher) newModThumbnailCanvas(modID string, size float32, cornerRadius float32) *canvas.Image {
	img := canvas.NewImageFromImage(l.modThumbnailImage(modID, int(size)))
	img.CornerRadius = cornerRadius
	img.SetMinSize(fyne.NewSquareSize(size))
	img.FillMode = canvas.ImageFillContain
	return img
}

func (l *Launcher) refreshModThumbnailCanvas(target *canvas.Image, modID string, fallbackSize int) {
	target.Image = l.modThumbnailImage(modID, fallbackSize)
	target.SetMinSize(fyne.NewSquareSize(float32(fallbackSize)))
	target.Refresh()
}

func (l *Launcher) ensureModThumbnailLoaded(modID string, onLoaded func()) {
	if modID == "" || l.state.Rest == nil {
		return
	}
	l.modThumbMu.Lock()
	if l.modThumbnailFetched[modID] || l.modThumbnailLoading[modID] {
		l.modThumbMu.Unlock()
		return
	}
	l.modThumbnailLoading[modID] = true
	l.modThumbMu.Unlock()

	go func(targetModID string) {
		thumbBytes, err := l.state.Rest.GetModThumbnail(targetModID)
		var decoded image.Image
		if err == nil && len(thumbBytes) > 0 {
			decoded, _, err = image.Decode(bytes.NewReader(thumbBytes))
			if err == nil {
				decoded = centerCropSquare(decoded)
			}
		}
		if err != nil {
			slog.Debug("Failed to load mod thumbnail", "modID", targetModID, "error", err)
		}

		l.modThumbMu.Lock()
		delete(l.modThumbnailLoading, targetModID)
		l.modThumbnailFetched[targetModID] = true
		if decoded != nil {
			l.modThumbnailImageCache[targetModID] = decoded
		}
		l.modThumbMu.Unlock()

		if onLoaded != nil {
			fyne.Do(onLoaded)
		}
	}(modID)
}

func (l *Launcher) newModDetailsDialog(mod *modmgr.Mod, onSelect func(modmgr.ModVersion)) *dialog.CustomDialog {
	loading := widget.NewProgressBarInfinite()
	loading.Start()

	type versionRow struct {
		versionID string
		version   *modmgr.ModVersion
		err       error
		loading   bool
	}

	var rows []versionRow
	var d *dialog.CustomDialog

	versionList := widget.NewList(
		func() int { return len(rows) },
		func() fyne.CanvasObject {
			return widget.NewButton("ver", nil)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(rows) {
				return
			}
			row := rows[id]
			btn := item.(*widget.Button)

			btn.OnTapped = nil
			btn.Disable()

			switch {
			case row.loading:
				btn.SetText(lang.LocalizeKey("profile.loading_version", "Loading version '{{.ID}}'...", map[string]any{"ID": row.versionID}))
			case row.err != nil:
				btn.SetText(lang.LocalizeKey("profile.failed_version", "Failed to load version '{{.ID}}'", map[string]any{"ID": row.versionID}))
			case row.version != nil:
				btn.SetText(row.version.VersionID)
				btn.Enable()
				version := *row.version
				btn.OnTapped = func() {
					d.Dismiss()
					onSelect(version)
				}
			default:
				btn.SetText(lang.LocalizeKey("profile.unavailable_version", "Version '{{.ID}}' unavailable", map[string]any{"ID": row.versionID}))
			}
		},
	)

	description := widget.NewRichTextFromMarkdown(mod.Description)
	content := container.NewBorder(description,
		loading, nil, nil,
		description,
		loading,
		versionList,
	)

	d = dialog.NewCustom(mod.Name, lang.LocalizeKey("common.cancel", "Cancel"), content, l.state.Window)
	d.Resize(fyne.NewSize(400, 300))

	go func() {
		v, err := l.state.Rest.GetModVersionIDs(mod.ID, 100, "")
		if err != nil {
			d.Hide()
			dialog.ShowError(err, l.state.Window)
			return
		}
		fyne.Do(func() {
			rows = make([]versionRow, len(v))
			for i, versionID := range v {
				rows[i] = versionRow{
					versionID: versionID,
					loading:   true,
				}
			}
			versionList.Refresh()
		})

		var wg sync.WaitGroup
		for i, id := range v {
			wg.Add(1)
			go func(index int, versionID string) {
				defer wg.Done()
				version, fetchErr := l.state.Rest.GetModVersion(mod.ID, versionID)
				fyne.Do(func() {
					if index >= len(rows) {
						return
					}
					rows[index].loading = false
					rows[index].err = fetchErr
					if fetchErr == nil && version != nil {
						rows[index].version = version
					}
					versionList.RefreshItem(index)
				})
			}(i, id)
		}
		wg.Wait()
		fyne.Do(loading.Hide)
	}()
	return d
}
