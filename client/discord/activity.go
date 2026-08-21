package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2/lang"
	discord "github.com/ikafly144/discord_social_sdk"
)

func (s *DiscordService) SetIdleActivityEnabled(enabled bool) {
	s.activityMu.Lock()
	if s.idleActivityEnabled == enabled {
		s.activityMu.Unlock()
		return
	}
	s.idleActivityEnabled = enabled
	if !enabled {
		if s.currentActivity != nil && s.currentActivity == s.idleActivity {
			s.currentActivity = nil
			s.idleActivity = nil
			s.activityMu.Unlock()
			s.client.ClearRichPresence()
			return
		}
		s.activityMu.Unlock()
		return
	}
	s.activityMu.Unlock()
	s.updateIdleActivity()
}

func (s *DiscordService) IsIdleActivityEnabled() bool {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	return s.idleActivityEnabled
}

func (s *DiscordService) SetIdleActivity(provider func() *discord.Activity, callback func(*discord.ClientResult)) {
	s.activityMu.Lock()
	s.idleActivityProvider = provider
	s.idleActivityCallback = callback
	s.activityMu.Unlock()
	s.updateIdleActivity()
}

func (s *DiscordService) updateIdleActivity() {
	s.activityMu.Lock()
	if s.idleActivityProvider != nil && s.currentActivity == nil && s.idleActivityEnabled {
		activity := s.idleActivityProvider()
		s.idleActivity = activity
		callback := s.idleActivityCallback
		s.activityMu.Unlock()

		if activity != nil {
			if callback == nil {
				callback = func(result *discord.ClientResult) {
					if !result.Successful() {
						slog.Warn("No callback set for idle activity update error", "error", result.ErrorCode())
					}
				}
			}
			s.SetActivity(activity, callback)
		}
	} else if s.currentActivity == nil {
		s.idleActivity = nil
		s.activityMu.Unlock()
		s.client.ClearRichPresence()
	} else {
		s.activityMu.Unlock()
	}
}

func (s *DiscordService) SetActivity(activity *discord.Activity, callback func(*discord.ClientResult)) {
	if activity == nil {
		slog.Warn("SetActivity called with nil activity")
		return
	}
	s.activityMu.Lock()
	s.currentActivity = activity
	if s.idleActivity != nil && activity != s.idleActivity {
		s.idleActivity = nil
	}
	s.activityMu.Unlock()

	s.client.UpdateRichPresence(activity, func(arg0 *discord.ClientResult) {
		if callback != nil {
			callback(arg0)
		}
	})
}

func (s *DiscordService) ClearActivity() {
	s.activityMu.Lock()
	s.currentActivity = nil
	s.activityMu.Unlock()
	s.updateIdleActivity()
}

func (s *DiscordService) CurrentActivity() (*discord.Activity, bool) {
	s.activityMu.Lock()
	defer s.activityMu.Unlock()
	return s.currentActivity, s.currentActivity != nil
}

func (s *DiscordService) SendInvite(userId uint64, inviteUrl string) {
	s.activityMu.Lock()
	activity := s.currentActivity
	s.activityMu.Unlock()

	if activity == nil {
		slog.Warn("Cannot send invite, no current activity")
		return
	}

	name := ""
	if user, ok := s.UserInfo(); ok {
		if dName := strings.TrimSpace(user.DisplayName()); dName != "" {
			name = dName
		} else if uName := strings.TrimSpace(user.Username()); uName != "" {
			name = uName
		}
	}
	if name == "" {
		name = "Unknown"
	}

	s.client.SendActivityInvite(userId, lang.LocalizeKey("discord.invite_message", "Join {{.Name}} in Mod of Us!\n\n{{.Link}}",
		map[string]any{
			"Name": name,
			"Link": inviteUrl,
		}), func(result *discord.ClientResult) {
		if !result.Successful() {
			slog.Warn("Failed to send Discord invite", "error", result.ErrorCode())
		} else {
			slog.Info("Successfully sent Discord invite")
		}
	})
}

const joinRequestTTL = 10 * time.Minute

func (s *DiscordService) RecordSentJoinRequest(userId uint64) {
	s.sentJoinRequestsMu.Lock()
	defer s.sentJoinRequestsMu.Unlock()
	if s.sentJoinRequests == nil {
		s.sentJoinRequests = make(map[uint64]time.Time)
	}
	s.sentJoinRequests[userId] = time.Now()
}

func (s *DiscordService) RemoveSentJoinRequest(userId uint64) {
	s.sentJoinRequestsMu.Lock()
	defer s.sentJoinRequestsMu.Unlock()
	delete(s.sentJoinRequests, userId)
}

func (s *DiscordService) ConsumeSentJoinRequest(userId uint64) bool {
	s.sentJoinRequestsMu.Lock()
	defer s.sentJoinRequestsMu.Unlock()
	if s.sentJoinRequests == nil {
		return false
	}
	t, ok := s.sentJoinRequests[userId]
	if !ok {
		return false
	}
	delete(s.sentJoinRequests, userId)
	return time.Since(t) <= joinRequestTTL
}

func (s *DiscordService) SendActivityJoinRequest(userId uint64, callback func(error)) {
	s.RecordSentJoinRequest(userId)
	s.client.SendActivityJoinRequest(userId, func(result *discord.ClientResult) {
		if !result.Successful() {
			s.RemoveSentJoinRequest(userId)
			slog.Warn("Failed to send Discord activity join request", "error", result.ErrorCode(), "userId", userId)
			if callback != nil {
				callback(fmt.Errorf("failed with error code: %d", result.ErrorCode()))
			}
		} else {
			slog.Info("Successfully sent Discord activity join request", "userId", userId)
			if callback != nil {
				callback(nil)
			}
		}
	})
}

func (s *DiscordService) SendActivityJoinRequestReply(invite *discord.ActivityInvite, callback func(error)) {
	s.client.SendActivityJoinRequestReply(invite, func(result *discord.ClientResult) {
		if !result.Successful() {
			slog.Warn("Failed to send Discord activity join request reply", "error", result.ErrorCode())
			if callback != nil {
				callback(fmt.Errorf("failed with error code: %d", result.ErrorCode()))
			}
		} else {
			slog.Info("Successfully replied to Discord activity join request")
			if callback != nil {
				callback(nil)
			}
		}
	})
}
