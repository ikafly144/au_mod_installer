package discord

import (
	"log/slog"

	"fyne.io/fyne/v2/lang"
	discord "github.com/ikafly144/discord_social_sdk"
)

func (s *DiscordService) SetIdleActivity(provider func() *discord.Activity, callback func(*discord.ClientResult)) {
	s.activityMu.Lock()
	s.idleActivityProvider = provider
	s.idleActivityCallback = callback
	s.activityMu.Unlock()
	s.updateIdleActivity()
}

func (s *DiscordService) updateIdleActivity() {
	s.activityMu.Lock()
	if s.idleActivityProvider != nil && s.currentActivity == nil {
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

func (s *DiscordService) SendInvite(userId uint64) {
	s.activityMu.Lock()
	activity := s.currentActivity
	s.activityMu.Unlock()

	if activity == nil {
		slog.Warn("Cannot send invite, no current activity")
		return
	}
	s.client.SendActivityInvite(userId, lang.LocalizeKey("discord.invite_message", "Join me in {{.Name}}!", map[string]any{"Name": activity.Name()}), func(result *discord.ClientResult) {
		if !result.Successful() {
			slog.Warn("Failed to send Discord invite", "error", result.ErrorCode())
		} else {
			slog.Info("Successfully sent Discord invite")
		}
	})
}
