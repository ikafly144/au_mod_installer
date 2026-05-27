package discord

import (
	"log/slog"
	"sync"

	discord "github.com/ikafly144/discord_social_sdk"
)

func NewDiscordService(client *discord.Discord_Client) *DiscordService {
	return &DiscordService{
		client: client,
	}
}

type DiscordService struct {
	client *discord.Discord_Client

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

func (s *DiscordService) SetIdleActivity(provider func() *discord.Discord_Activity, callback func(*discord.Discord_ClientResult)) {
	s.idleActivityProvider = provider
	s.idleActivityCallback = callback
	s.updateIdleActivity()
}

func (s *DiscordService) updateIdleActivity() {
	if s.idleActivityProvider != nil && s.currentActivity == nil {
		activity := s.idleActivityProvider()
		s.idleActivity = activity
		if activity != nil {
			callback := s.idleActivityCallback
			if callback == nil {
				callback = func(result *discord.Discord_ClientResult) {
					if !result.Successful() {
						slog.Warn("No callback set for idle activity update error", "error", result.ErrorCode())
					}
				}
			}
			s.SetActivity(activity, callback)
		}
	}
}

func (s *DiscordService) SetActivity(activity *discord.Discord_Activity, callback func(*discord.Discord_ClientResult)) {
	if activity == nil {
		slog.Warn("SetActivity called with nil activity")
		return
	}
	s.currentActivity = activity
	if s.idleActivity != nil && activity != s.idleActivity {
		s.idleActivity = nil
	}
	s.client.UpdateRichPresence(activity, func(arg0 *discord.Discord_ClientResult) {
		if callback != nil {
			callback(arg0)
		}
	})
}

func (s *DiscordService) ClearActivity() {
	s.currentActivity = nil
	s.updateIdleActivity()
}

func (s *DiscordService) CurrentActivity() (*discord.Discord_Activity, bool) {
	return s.currentActivity, s.currentActivity != nil
}
