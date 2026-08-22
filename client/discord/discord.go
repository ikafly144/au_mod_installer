package discord

import (
	"log/slog"
	"sync"
	"time"

	discord "github.com/ikafly144/discord_social_sdk"
)

const ApplicationID uint64 = 1472154358291501171

func NewDiscordService(client *discord.Client) *DiscordService {
	ds := &DiscordService{
		client:                       client,
		relationShipChangedCallbacks: make(map[int]func([]discord.UserHandle)),
		activityInviteCallbacks:      make(map[int]func(*discord.ActivityInvite)),
		sentJoinRequests:             make(map[uint64]time.Time),
		lobbyUpdatedCallbacks:        make(map[int]func(*LobbyInfo)),
		lobbyMessageCallbacks:        make(map[int]func(LobbyMessage)),
		speakingUsers:                make(map[uint64]bool),
		voiceStatusCallbacks:         make(map[int]func(discord.CallStatus)),
		speakingCallbacks:            make(map[int]func(map[uint64]bool)),
	}
	ds.resetReady()
	client.SetRelationshipGroupsUpdatedCallback(func(userId uint64) {
		ds.relationshipsMu.Lock()
		friends, err := ds.GetFriends()
		if err != nil {
			slog.Warn("Failed to get friends during relationship update", "error", err)
			ds.relationshipsMu.Unlock()
			return
		}
		for _, callback := range ds.relationShipChangedCallbacks {
			callback(friends)
		}
		ds.relationshipsMu.Unlock()
	})
	client.SetActivityInviteCreatedCallback(func(invite *discord.ActivityInvite) {
		ds.activityInviteMu.Lock()
		callbacks := make([]func(*discord.ActivityInvite), 0, len(ds.activityInviteCallbacks))
		for _, cb := range ds.activityInviteCallbacks {
			callbacks = append(callbacks, cb)
		}
		ds.activityInviteMu.Unlock()
		for _, cb := range callbacks {
			cb(invite)
		}
	})

	// Lobby event callbacks
	client.SetLobbyUpdatedCallback(func(lobbyID uint64) {
		ds.lobbyMu.RLock()
		activeID := ds.activeLobbyID
		ds.lobbyMu.RUnlock()
		if activeID != 0 && activeID == lobbyID {
			info := ds.refreshActiveLobbyInfo(lobbyID)
			ds.notifyLobbyUpdated(info)
		}
	})
	client.SetLobbyMemberAddedCallback(func(lobbyID uint64, memberID uint64) {
		ds.lobbyMu.RLock()
		activeID := ds.activeLobbyID
		ds.lobbyMu.RUnlock()
		if activeID != 0 && activeID == lobbyID {
			info := ds.refreshActiveLobbyInfo(lobbyID)
			ds.notifyLobbyUpdated(info)
		}
	})
	client.SetLobbyMemberRemovedCallback(func(lobbyID uint64, memberID uint64) {
		ds.lobbyMu.RLock()
		activeID := ds.activeLobbyID
		ds.lobbyMu.RUnlock()
		if activeID != 0 && activeID == lobbyID {
			info := ds.refreshActiveLobbyInfo(lobbyID)
			ds.notifyLobbyUpdated(info)
		}
	})
	client.SetLobbyMemberUpdatedCallback(func(lobbyID uint64, memberID uint64) {
		ds.lobbyMu.RLock()
		activeID := ds.activeLobbyID
		ds.lobbyMu.RUnlock()
		if activeID != 0 && activeID == lobbyID {
			info := ds.refreshActiveLobbyInfo(lobbyID)
			ds.notifyLobbyUpdated(info)
		}
	})
	client.SetLobbyDeletedCallback(func(lobbyID uint64) {
		ds.lobbyMu.Lock()
		if ds.activeLobbyID == lobbyID {
			ds.activeLobbyID = 0
			ds.activeLobbySecret = ""
			ds.activeLobbyInfo = nil
		}
		ds.lobbyMu.Unlock()
		ds.notifyLobbyUpdated(nil)
	})

	// Message callback
	client.SetMessageCreatedCallback(func(messageID uint64) {
		handle, ok := client.GetMessageHandle(messageID)
		if !ok {
			return
		}
		ds.lobbyMu.RLock()
		activeID := ds.activeLobbyID
		ds.lobbyMu.RUnlock()

		if activeID == 0 {
			return
		}

		if lobby, ok := handle.Lobby(); ok && lobby.Id() == activeID {
			msg := ds.convertMessageHandle(&handle, activeID)
			ds.notifyLobbyMessage(msg)
		}
	})

	// Discord Activity Join callback (when friend clicks Join in Discord client)
	client.SetActivityJoinCallback(func(joinSecret string) {
		slog.Info("Received Discord Activity Join callback", "joinSecret", joinSecret)
		if joinSecret != "" {
			ds.CreateOrJoinLobby(joinSecret, nil, nil, func(err error, lobbyID uint64) {
				if err != nil {
					slog.Warn("Failed to auto-join lobby from Activity Join callback", "error", err)
				}
			})
		}
	})

	return ds
}

type DiscordService struct {
	client    *discord.Client
	ready     chan struct{}
	readyOnce sync.Once

	idleActivityProvider func() *discord.Activity
	idleActivityCallback func(*discord.ClientResult)
	idleActivityEnabled  bool

	idleActivity    *discord.Activity
	currentActivity *discord.Activity
	activityMu      sync.Mutex

	queueMu sync.Mutex
	queue   []string

	signInMu  sync.Mutex
	signingIn bool
	loggedIn  bool

	relationShipChangedCallbacks map[int]func([]discord.UserHandle)
	nextRelationshipCallbackID   int
	relationshipsMu              sync.Mutex

	activityInviteCallbacks      map[int]func(*discord.ActivityInvite)
	nextActivityInviteCallbackID int
	activityInviteMu             sync.Mutex

	sentJoinRequestsMu sync.Mutex
	sentJoinRequests   map[uint64]time.Time

	lobbyMu                    sync.RWMutex
	activeLobbyID              uint64
	activeLobbySecret          string
	activeLobbyInfo            *LobbyInfo
	lobbyUpdatedCallbacks      map[int]func(*LobbyInfo)
	nextLobbyCallbackID        int
	lobbyMessageCallbacks      map[int]func(LobbyMessage)
	nextLobbyMessageCallbackID int

	voiceMu                   sync.RWMutex
	activeCall                *discord.Call
	voiceStatus               discord.CallStatus
	speakingUsers             map[uint64]bool
	voiceStatusCallbacks      map[int]func(discord.CallStatus)
	nextVoiceStatusCallbackID int
	speakingCallbacks         map[int]func(map[uint64]bool)
	nextSpeakingCallbackID    int
}

func (s *DiscordService) AddActivityInviteCallback(callback func(*discord.ActivityInvite)) int {
	s.activityInviteMu.Lock()
	defer s.activityInviteMu.Unlock()
	id := s.nextActivityInviteCallbackID
	s.activityInviteCallbacks[id] = callback
	s.nextActivityInviteCallbackID++
	return id
}

func (s *DiscordService) RemoveActivityInviteCallback(id int) {
	s.activityInviteMu.Lock()
	defer s.activityInviteMu.Unlock()
	delete(s.activityInviteCallbacks, id)
}

func (s *DiscordService) resetReady() {
	if s.ready != nil {
		select {
		case <-s.ready:
		default:
			s.readyOnce.Do(func() {
				close(s.ready)
			})
		}
	}
	s.readyOnce = sync.Once{}
	s.ready = make(chan struct{})
}

func (s *DiscordService) Client() *discord.Client {
	return s.client
}

func (s *DiscordService) PushQueue(uri string) {
	s.queueMu.Lock()
	s.queue = append(s.queue, uri)
	s.queueMu.Unlock()
}

func (s *DiscordService) PopQueue() (string, bool) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	if len(s.queue) == 0 {
		return "", false
	}
	uri := s.queue[0]
	s.queue = s.queue[1:]
	return uri, true
}
