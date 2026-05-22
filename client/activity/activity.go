package activity

import (
	"log/slog"
	"sync"

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

	idleActivity    *sdk.Activity
	currentActivity *sdk.Activity

	queueMu sync.Mutex
	queue   []string
}

func (s *ActivityService) Client() *sdk.Client {
	return s.client
}

func (s *ActivityService) PushQueue(uri string) {
	s.queueMu.Lock()
	s.queue = append(s.queue, uri)
	s.queueMu.Unlock()
}

func (s *ActivityService) PopQueue() (string, bool) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	if len(s.queue) == 0 {
		return "", false
	}
	uri := s.queue[0]
	s.queue = s.queue[1:]
	return uri, true
}

func (s *ActivityService) SetIdleActivity(provider func() *sdk.Activity, callback func(sdk.ErrorType)) {
	s.idleActivityProvider = provider
	s.idleActivityCallback = callback
	s.updateIdleActivity()
}

func (s *ActivityService) updateIdleActivity() {
	if s.idleActivityProvider != nil && s.currentActivity == nil {
		activity := s.idleActivityProvider()
		s.idleActivity = activity
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
		slog.Warn("SetActivity called with nil activity")
		return
	}
	s.currentActivity = activity
	if s.idleActivity != nil && activity != s.idleActivity {
		s.idleActivity = nil
	}
	s.client.UpdateRichPresence(activity, func(err sdk.ErrorType) {
		if callback != nil {
			callback(err)
		}
	})
}

func (s *ActivityService) ClearActivity() {
	s.currentActivity = nil
	s.updateIdleActivity()
}

func (s *ActivityService) CurrentActivity() (*sdk.Activity, bool) {
	return s.currentActivity, s.currentActivity != nil
}
