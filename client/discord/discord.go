package discord

import (
	"log/slog"
	"sync"

	discord "github.com/ikafly144/discord_social_sdk"
)

func NewDiscordService(client *discord.Client) *DiscordService {
	ds := &DiscordService{
		client:                       client,
		ready:                        make(chan struct{}),
		relationShipChangedCallbacks: make(map[int]func([]discord.RelationshipHandle)),
	}
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
	return ds
}

type DiscordService struct {
	client    *discord.Client
	ready     chan struct{}
	readyOnce sync.Once

	idleActivityProvider func() *discord.Activity
	idleActivityCallback func(*discord.ClientResult)

	idleActivity    *discord.Activity
	currentActivity *discord.Activity
	activityMu      sync.Mutex

	queueMu sync.Mutex
	queue   []string

	signInMu  sync.Mutex
	signingIn bool
	loggedIn  bool

	relationShipChangedCallbacks map[int]func([]discord.RelationshipHandle)
	nextRelationshipCallbackID   int
	relationshipsMu              sync.Mutex
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
