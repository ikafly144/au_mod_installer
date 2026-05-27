package discord

import (
	"sync"

	discord "github.com/ikafly144/discord_social_sdk"
)

func NewDiscordService(client *discord.Discord_Client) *DiscordService {
	return &DiscordService{
		client: client,
		ready:  make(chan struct{}),
	}
}

type DiscordService struct {
	client    *discord.Discord_Client
	ready     chan struct{}
	readyOnce sync.Once

	idleActivityProvider func() *discord.Discord_Activity
	idleActivityCallback func(*discord.Discord_ClientResult)

	idleActivity    *discord.Discord_Activity
	currentActivity *discord.Discord_Activity

	queueMu sync.Mutex
	queue   []string

	signInMu sync.Mutex
	loggedIn bool
}

func (s *DiscordService) Client() *discord.Discord_Client {
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
