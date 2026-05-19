package activity

import (
	"log/slog"

	sdk "github.com/ikafly144/discord_social_sdk"
)

func NewActivityService(client *sdk.Client) *ActivityService {
	return &ActivityService{
		client: client,
	}
}

type ActivityService struct {
	client *sdk.Client

	idleActivityProvider func() *sdk.Activity
	idleActivityCallback func(sdk.ErrorType)

	currentActivity *sdk.Activity
}

func (s *ActivityService) SetIdleActivity(provider func() *sdk.Activity, callback func(sdk.ErrorType)) {
	s.idleActivityProvider = provider
	s.idleActivityCallback = callback
	s.updateIdleActivity()
}

func (s *ActivityService) updateIdleActivity() {
	if s.idleActivityProvider != nil && s.currentActivity == nil {
		activity := s.idleActivityProvider()
		if activity != nil {
			callback := s.idleActivityCallback
			if callback == nil {
				callback = func(et sdk.ErrorType) {
					slog.Warn("No callback set for idle activity update error", "et", et)
				}
			}
			s.SetActivity(activity, callback)
		}
	}
}

func (s *ActivityService) SetActivity(activity *sdk.Activity, callback func(sdk.ErrorType)) {
	if activity == nil {
		panic("activity cannot be nil")
	}
	s.currentActivity = activity
	s.client.UpdateRichPresence(activity, func(err sdk.ErrorType) {
		if callback != nil {
			callback(err)
		}
	})
	s.updateIdleActivity()
}

func (s *ActivityService) ClearActivity() {
	s.currentActivity = nil
	s.client.ClearRichPresence()
	s.updateIdleActivity()
}

func (s *ActivityService) CurrentActivity() (*sdk.Activity, bool) {
	return s.currentActivity, s.currentActivity != nil
}
