package discord

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"strings"
	"time"
	"unsafe"

	discord "github.com/ikafly144/discord_social_sdk"
)

var (
	ErrNotConnectedToLobby = errors.New("not connected to a lobby")
	ErrLobbyCreationFailed = errors.New("failed to create or join lobby")
)

type CallStatus = discord.CallStatus

const (
	CallStatusDisconnected = discord.CallStatusDisconnected
	CallStatusConnecting   = discord.CallStatusConnecting
	CallStatusConnected    = discord.CallStatusConnected
	CallStatusReconnecting = discord.CallStatusReconnecting
)

type LinkedChannelInfo struct {
	ID      uint64 `json:"id"`
	GuildID uint64 `json:"guild_id"`
	Name    string `json:"name"`
}

type GuildInfo struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type GuildChannelInfo struct {
	ID         uint64              `json:"id"`
	Name       string              `json:"name"`
	Type       discord.ChannelType `json:"type"`
	Position   int32               `json:"position"`
	IsLinkable bool                `json:"is_linkable"`
}

type LobbyInfo struct {
	ID            uint64             `json:"id"`
	Secret        string             `json:"secret"`
	HostUserID    uint64             `json:"host_user_id"`
	LinkedChannel *LinkedChannelInfo `json:"linked_channel,omitempty"`
	Metadata      map[string]string  `json:"metadata"`
	Members       []LobbyMember      `json:"members"`
	CreatedAt     time.Time          `json:"created_at"`
}

type LobbyMember struct {
	UserID           uint64            `json:"user_id"`
	DisplayName      string            `json:"display_name"`
	Username         string            `json:"username"`
	AvatarURL        string            `json:"avatar_url"`
	Metadata         map[string]string `json:"metadata"`
	IsHost           bool              `json:"is_host"`
	IsSpeaking       bool              `json:"is_speaking"`
	IsVoiceConnected bool              `json:"is_voice_connected"`
	IsMuted          bool              `json:"is_muted"`
	IsDeafened       bool              `json:"is_deafened"`
	CanLinkLobby     bool              `json:"can_link_lobby"`
}

type LobbyMessage struct {
	ID         uint64    `json:"id"`
	LobbyID    uint64    `json:"lobby_id"`
	AuthorID   uint64    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	AvatarURL  string    `json:"avatar_url"`
	Content    string    `json:"content"`
	SentAt     time.Time `json:"sent_at"`
}

// Memory layout mirror for discord.String
type rawDiscordString struct {
	ptr  *uint8
	size uintptr
}

// Memory layout mirror for discord.Properties
type rawProperties struct {
	size   uintptr
	keys   *rawDiscordString
	values *rawDiscordString
}

func propertiesToMap(p discord.Properties) map[string]string {
	raw := (*rawProperties)(unsafe.Pointer(&p))
	if raw.size == 0 || raw.keys == nil || raw.values == nil {
		return make(map[string]string)
	}
	keys := unsafe.Slice(raw.keys, int(raw.size))
	values := unsafe.Slice(raw.values, int(raw.size))
	result := make(map[string]string, int(raw.size))
	for i := 0; i < int(raw.size); i++ {
		var k, v string
		if keys[i].ptr != nil && keys[i].size > 0 {
			k = string(unsafe.Slice(keys[i].ptr, int(keys[i].size)))
		}
		if values[i].ptr != nil && values[i].size > 0 {
			v = string(unsafe.Slice(values[i].ptr, int(values[i].size)))
		}
		if k != "" {
			result[k] = v
		}
	}
	return result
}

func mapToProperties(m map[string]string) (discord.Properties, func()) {
	if len(m) == 0 {
		return discord.Properties{}, func() {}
	}
	keys := make([]rawDiscordString, 0, len(m))
	values := make([]rawDiscordString, 0, len(m))
	pinned := make([][]byte, 0, len(m)*2)

	for k, v := range m {
		kBytes := []byte(k)
		vBytes := []byte(v)
		pinned = append(pinned, kBytes, vBytes)

		var kPtr, vPtr *uint8
		if len(kBytes) > 0 {
			kPtr = &kBytes[0]
		}
		if len(vBytes) > 0 {
			vPtr = &vBytes[0]
		}
		keys = append(keys, rawDiscordString{ptr: kPtr, size: uintptr(len(kBytes))})
		values = append(values, rawDiscordString{ptr: vPtr, size: uintptr(len(vBytes))})
	}

	raw := rawProperties{
		size:   uintptr(len(m)),
		keys:   &keys[0],
		values: &values[0],
	}

	props := *(*discord.Properties)(unsafe.Pointer(&raw))
	cleanup := func() {
		runtime.KeepAlive(keys)
		runtime.KeepAlive(values)
		runtime.KeepAlive(pinned)
	}
	return props, cleanup
}

func (s *DiscordService) CreateOrJoinLobby(sessionID string, lobbyMeta, memberMeta map[string]string, callback func(err error, lobbyID uint64)) {
	if !s.IsLoggedIn() {
		if callback != nil {
			callback(ErrNotLoggedIn, 0)
		}
		return
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		if callback != nil {
			callback(errors.New("lobby session_id cannot be empty"), 0)
		}
		return
	}

	lobbyProps, cleanupLobby := mapToProperties(lobbyMeta)
	defer cleanupLobby()
	memberProps, cleanupMember := mapToProperties(memberMeta)
	defer cleanupMember()

	s.lobbyMu.Lock()
	s.activeLobbySessionID = sessionID
	s.lobbyMu.Unlock()

	s.client.CreateOrJoinLobbyWithMetadata(sessionID, lobbyProps, memberProps, func(result *discord.ClientResult, lobbyID uint64) {
		if !result.Successful() {
			slog.Warn("Failed to create or join Discord lobby", "error", result.ErrorCode(), "sessionID", sessionID)
			s.lobbyMu.Lock()
			if s.activeLobbySessionID == sessionID {
				s.activeLobbySessionID = ""
				s.activeLobbyID = 0
				s.activeLobbyInfo = nil
			}
			s.lobbyMu.Unlock()
			if callback != nil {
				callback(fmt.Errorf("%w: code %d", ErrLobbyCreationFailed, result.ErrorCode()), 0)
			}
			return
		}

		slog.Info("Successfully created/joined Discord lobby", "lobbyID", lobbyID, "sessionID", sessionID)
		s.lobbyMu.Lock()
		s.activeLobbyID = lobbyID
		s.lobbyMu.Unlock()

		info := s.refreshActiveLobbyInfo(lobbyID)
		if callback != nil {
			callback(nil, lobbyID)
		}
		s.notifyLobbyUpdated(info)
	})
}

func (s *DiscordService) LeaveLobby(callback func(error)) {
	s.lobbyMu.Lock()
	lobbyID := s.activeLobbyID
	s.activeLobbyID = 0
	s.activeLobbySessionID = ""
	s.activeLobbyInfo = nil
	s.lobbyMu.Unlock()

	// Disconnect voice if active
	s.DisconnectVoice(nil)

	if lobbyID == 0 {
		if callback != nil {
			callback(nil)
		}
		s.notifyLobbyUpdated(nil)
		return
	}

	s.client.LeaveLobby(lobbyID, func(result *discord.ClientResult) {
		if !result.Successful() {
			slog.Warn("Failed to leave Discord lobby cleanly", "lobbyID", lobbyID, "error", result.ErrorCode())
			if callback != nil {
				callback(fmt.Errorf("failed to leave lobby: code %d", result.ErrorCode()))
			}
		} else {
			slog.Info("Successfully left Discord lobby", "lobbyID", lobbyID)
			if callback != nil {
				callback(nil)
			}
		}
		s.notifyLobbyUpdated(nil)
	})
}

func (s *DiscordService) SetActiveLobbyID(lobbyID uint64) *LobbyInfo {
	s.lobbyMu.Lock()
	s.activeLobbyID = lobbyID
	s.lobbyMu.Unlock()

	info := s.refreshActiveLobbyInfo(lobbyID)
	s.notifyLobbyUpdated(info)
	return info
}

func (s *DiscordService) GetActiveLobby() (*LobbyInfo, bool) {
	s.lobbyMu.RLock()
	defer s.lobbyMu.RUnlock()
	if s.activeLobbyID == 0 || s.activeLobbyInfo == nil {
		return nil, false
	}
	return s.activeLobbyInfo, true
}

func (s *DiscordService) UpdateLobbyMemberMetadata(meta map[string]string, callback func(error)) {
	s.lobbyMu.RLock()
	sessionID := s.activeLobbySessionID
	s.lobbyMu.RUnlock()

	if sessionID == "" {
		if callback != nil {
			callback(ErrNotConnectedToLobby)
		}
		return
	}

	memberProps, cleanupMember := mapToProperties(meta)
	defer cleanupMember()

	s.client.CreateOrJoinLobbyWithMetadata(sessionID, discord.Properties{}, memberProps, func(result *discord.ClientResult, lobbyID uint64) {
		if !result.Successful() {
			slog.Warn("Failed to update lobby member metadata", "error", result.ErrorCode())
			if callback != nil {
				callback(fmt.Errorf("failed to update member metadata: code %d", result.ErrorCode()))
			}
			return
		}
		info := s.refreshActiveLobbyInfo(lobbyID)
		if callback != nil {
			callback(nil)
		}
		s.notifyLobbyUpdated(info)
	})
}

func (s *DiscordService) SendLobbyMessage(content string, callback func(err error, messageID uint64)) {
	s.lobbyMu.RLock()
	lobbyID := s.activeLobbyID
	s.lobbyMu.RUnlock()

	if lobbyID == 0 {
		if callback != nil {
			callback(ErrNotConnectedToLobby, 0)
		}
		return
	}

	s.client.SendLobbyMessage(lobbyID, content, func(result *discord.ClientResult, messageID uint64) {
		if !result.Successful() {
			slog.Warn("Failed to send lobby message", "error", result.ErrorCode())
			if callback != nil {
				callback(fmt.Errorf("failed to send message: code %d", result.ErrorCode()), 0)
			}
			return
		}
		if callback != nil {
			callback(nil, messageID)
		}
	})
}

func (s *DiscordService) GetLobbyMessages(limit int32, callback func(err error, messages []LobbyMessage)) {
	s.lobbyMu.RLock()
	lobbyID := s.activeLobbyID
	s.lobbyMu.RUnlock()

	if lobbyID == 0 {
		if callback != nil {
			callback(ErrNotConnectedToLobby, nil)
		}
		return
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	s.client.GetLobbyMessagesWithLimit(lobbyID, limit, func(result *discord.ClientResult, rawMessages []discord.MessageHandle) {
		if !result.Successful() {
			slog.Warn("Failed to fetch lobby messages", "error", result.ErrorCode())
			if callback != nil {
				callback(fmt.Errorf("failed to fetch messages: code %d", result.ErrorCode()), nil)
			}
			return
		}

		messages := make([]LobbyMessage, 0, len(rawMessages))
		for _, raw := range rawMessages {
			msg := s.convertMessageHandle(&raw, lobbyID)
			messages = append(messages, msg)
		}
		if callback != nil {
			callback(nil, messages)
		}
	})
}

func (s *DiscordService) convertMessageHandle(raw *discord.MessageHandle, lobbyID uint64) LobbyMessage {
	msg := LobbyMessage{
		ID:       raw.Id(),
		LobbyID:  lobbyID,
		AuthorID: raw.AuthorId(),
		Content:  raw.Content(),
		SentAt:   time.UnixMilli(int64(raw.SentTimestamp())),
	}
	if author, ok := raw.Author(); ok {
		if dName := strings.TrimSpace(author.DisplayName()); dName != "" {
			msg.AuthorName = dName
		} else {
			msg.AuthorName = author.Username()
		}
		msg.AvatarURL = author.AvatarUrl(discord.UserHandleAvatarTypeGif, discord.UserHandleAvatarTypePng)
	}
	if msg.AuthorName == "" {
		msg.AuthorName = fmt.Sprintf("User %d", raw.AuthorId())
	}
	return msg
}

func (s *DiscordService) refreshActiveLobbyInfo(lobbyID uint64) *LobbyInfo {
	handle, ok := s.client.GetLobbyHandle(lobbyID)
	if !ok {
		return nil
	}

	s.voiceMu.RLock()
	speakingMap := make(map[uint64]bool, len(s.speakingUsers))
	maps.Copy(speakingMap, s.speakingUsers)
	s.voiceMu.RUnlock()

	var voiceParticipants map[uint64]bool
	if callInfo, ok := handle.GetCallInfoHandle(); ok {
		voiceParticipants = make(map[uint64]bool)
		for _, pID := range callInfo.GetParticipants() {
			voiceParticipants[pID] = true
		}
	}

	var linkedChan *LinkedChannelInfo
	if lc, ok := handle.LinkedChannel(); ok {
		linkedChan = &LinkedChannelInfo{
			ID:      lc.Id(),
			GuildID: lc.GuildId(),
			Name:    lc.Name(),
		}
	}

	meta := propertiesToMap(handle.Metadata())
	membersRaw := handle.LobbyMembers()
	members := make([]LobbyMember, 0, len(membersRaw))

	for _, m := range membersRaw {
		mID := m.Id()
		mMeta := propertiesToMap(m.Metadata())
		member := LobbyMember{
			UserID:           mID,
			Metadata:         mMeta,
			IsHost:           strings.EqualFold(mMeta["is_host"], "true"),
			IsSpeaking:       speakingMap[mID],
			IsVoiceConnected: voiceParticipants[mID],
			CanLinkLobby:     m.CanLinkLobby(),
		}
		if user, ok := m.User(); ok {
			if dName := strings.TrimSpace(user.DisplayName()); dName != "" {
				member.DisplayName = dName
			} else {
				member.DisplayName = user.Username()
			}
			member.Username = user.Username()
			member.AvatarURL = user.AvatarUrl(discord.UserHandleAvatarTypeGif, discord.UserHandleAvatarTypePng)
		}
		if member.DisplayName == "" {
			member.DisplayName = fmt.Sprintf("Player %d", mID)
		}
		members = append(members, member)
	}

	s.lobbyMu.Lock()
	var hostID uint64
	for _, m := range members {
		if m.IsHost {
			hostID = m.UserID
			break
		}
	}
	if hostID == 0 && len(members) > 0 {
		hostID = members[0].UserID
	}

	info := &LobbyInfo{
		ID:            lobbyID,
		Secret:        s.activeLobbySessionID,
		HostUserID:    hostID,
		LinkedChannel: linkedChan,
		Metadata:      meta,
		Members:       members,
		CreatedAt:     time.Now(),
	}
	s.activeLobbyInfo = info
	s.lobbyMu.Unlock()

	return info
}

func (s *DiscordService) notifyLobbyUpdated(info *LobbyInfo) {
	s.lobbyMu.RLock()
	callbacks := make([]func(*LobbyInfo), 0, len(s.lobbyUpdatedCallbacks))
	for _, cb := range s.lobbyUpdatedCallbacks {
		callbacks = append(callbacks, cb)
	}
	s.lobbyMu.RUnlock()

	for _, cb := range callbacks {
		cb(info)
	}
}

func (s *DiscordService) notifyLobbyMessage(msg LobbyMessage) {
	s.lobbyMu.RLock()
	callbacks := make([]func(LobbyMessage), 0, len(s.lobbyMessageCallbacks))
	for _, cb := range s.lobbyMessageCallbacks {
		callbacks = append(callbacks, cb)
	}
	s.lobbyMu.RUnlock()

	for _, cb := range callbacks {
		cb(msg)
	}
}

func (s *DiscordService) AddLobbyUpdatedCallback(cb func(*LobbyInfo)) int {
	s.lobbyMu.Lock()
	defer s.lobbyMu.Unlock()
	id := s.nextLobbyCallbackID
	s.nextLobbyCallbackID++
	s.lobbyUpdatedCallbacks[id] = cb
	return id
}

func (s *DiscordService) RemoveLobbyUpdatedCallback(id int) {
	s.lobbyMu.Lock()
	defer s.lobbyMu.Unlock()
	delete(s.lobbyUpdatedCallbacks, id)
}

func (s *DiscordService) AddLobbyMessageCallback(cb func(LobbyMessage)) int {
	s.lobbyMu.Lock()
	defer s.lobbyMu.Unlock()
	id := s.nextLobbyMessageCallbackID
	s.nextLobbyMessageCallbackID++
	s.lobbyMessageCallbacks[id] = cb
	return id
}

func (s *DiscordService) RemoveLobbyMessageCallback(id int) {
	s.lobbyMu.Lock()
	defer s.lobbyMu.Unlock()
	delete(s.lobbyMessageCallbacks, id)
}

// ---------------- Voice Chat Operations ----------------

func (s *DiscordService) ConnectVoice(callback func(error)) {
	s.lobbyMu.RLock()
	lobbyID := s.activeLobbyID
	s.lobbyMu.RUnlock()

	if lobbyID == 0 {
		if callback != nil {
			callback(ErrNotConnectedToLobby)
		}
		return
	}

	s.voiceMu.RLock()
	alreadyActive := s.activeCall != nil
	s.voiceMu.RUnlock()

	if alreadyActive {
		if callback != nil {
			callback(nil)
		}
		return
	}

	call, ok := s.client.StartCall(lobbyID)
	if !ok {
		if callback != nil {
			callback(errors.New("failed to start voice call"))
		}
		return
	}

	call.SetStatusChangedCallback(func(status discord.CallStatus, callErr discord.CallError, detail int32) {
		slog.Info("Voice call status changed", "status", status, "error", callErr, "detail", detail)
		s.voiceMu.Lock()
		s.voiceStatus = status
		s.voiceMu.Unlock()
		s.notifyVoiceStatus(status)
		s.lobbyMu.RLock()
		activeID := s.activeLobbyID
		s.lobbyMu.RUnlock()
		if activeID != 0 {
			info := s.refreshActiveLobbyInfo(activeID)
			s.notifyLobbyUpdated(info)
		}
	})

	call.SetSpeakingStatusChangedCallback(func(userID uint64, speaking bool) {
		s.voiceMu.Lock()
		if s.speakingUsers == nil {
			s.speakingUsers = make(map[uint64]bool)
		}
		if speaking {
			s.speakingUsers[userID] = true
		} else {
			delete(s.speakingUsers, userID)
		}
		speakingCopy := make(map[uint64]bool, len(s.speakingUsers))
		maps.Copy(speakingCopy, s.speakingUsers)
		s.voiceMu.Unlock()
		s.notifySpeaking(speakingCopy)
	})

	call.SetParticipantChangedCallback(func(userID uint64, joined bool) {
		slog.Info("Voice call participant changed", "userID", userID, "joined", joined)
		s.lobbyMu.RLock()
		activeID := s.activeLobbyID
		s.lobbyMu.RUnlock()
		if activeID != 0 {
			info := s.refreshActiveLobbyInfo(activeID)
			s.notifyLobbyUpdated(info)
		}
	})

	status := call.GetStatus()
	s.voiceMu.Lock()
	s.activeCall = &call
	s.voiceStatus = status
	s.voiceMu.Unlock()

	if callback != nil {
		callback(nil)
	}
	s.notifyVoiceStatus(status)
}

func (s *DiscordService) DisconnectVoice(callback func(error)) {
	s.voiceMu.Lock()
	call := s.activeCall
	s.activeCall = nil
	s.voiceStatus = discord.CallStatusDisconnected
	s.speakingUsers = make(map[uint64]bool)
	s.voiceMu.Unlock()

	s.lobbyMu.RLock()
	lobbyID := s.activeLobbyID
	s.lobbyMu.RUnlock()

	if call == nil || lobbyID == 0 {
		if callback != nil {
			callback(nil)
		}
		s.notifyVoiceStatus(discord.CallStatusDisconnected)
		return
	}

	s.client.EndCall(lobbyID, func() {
		slog.Info("Voice call ended", "lobbyID", lobbyID)
		if callback != nil {
			callback(nil)
		}
		s.notifyVoiceStatus(discord.CallStatusDisconnected)
		s.lobbyMu.RLock()
		activeID := s.activeLobbyID
		s.lobbyMu.RUnlock()
		if activeID != 0 {
			info := s.refreshActiveLobbyInfo(activeID)
			s.notifyLobbyUpdated(info)
		}
	})
}

func (s *DiscordService) IsVoiceConnected() bool {
	s.voiceMu.RLock()
	defer s.voiceMu.RUnlock()
	return s.activeCall != nil && s.voiceStatus == discord.CallStatusConnected
}

func (s *DiscordService) GetVoiceStatus() discord.CallStatus {
	s.voiceMu.RLock()
	defer s.voiceMu.RUnlock()
	return s.voiceStatus
}

func (s *DiscordService) IsSelfMuted() bool {
	s.voiceMu.RLock()
	defer s.voiceMu.RUnlock()
	if s.activeCall != nil {
		return s.activeCall.GetSelfMute()
	}
	return s.client.GetSelfMuteAll()
}

func (s *DiscordService) SetSelfMuted(muted bool) {
	s.voiceMu.RLock()
	if s.activeCall != nil {
		s.activeCall.SetSelfMute(muted)
	}
	s.voiceMu.RUnlock()
	s.client.SetSelfMuteAll(muted)
}

func (s *DiscordService) IsSelfDeafened() bool {
	s.voiceMu.RLock()
	defer s.voiceMu.RUnlock()
	if s.activeCall != nil {
		return s.activeCall.GetSelfDeaf()
	}
	return s.client.GetSelfDeafAll()
}

func (s *DiscordService) SetSelfDeafened(deafened bool) {
	s.voiceMu.RLock()
	if s.activeCall != nil {
		s.activeCall.SetSelfDeaf(deafened)
	}
	s.voiceMu.RUnlock()
	s.client.SetSelfDeafAll(deafened)
}

func (s *DiscordService) notifyVoiceStatus(status discord.CallStatus) {
	s.voiceMu.RLock()
	callbacks := make([]func(discord.CallStatus), 0, len(s.voiceStatusCallbacks))
	for _, cb := range s.voiceStatusCallbacks {
		callbacks = append(callbacks, cb)
	}
	s.voiceMu.RUnlock()

	for _, cb := range callbacks {
		cb(status)
	}
}

func (s *DiscordService) notifySpeaking(speaking map[uint64]bool) {
	s.voiceMu.RLock()
	callbacks := make([]func(map[uint64]bool), 0, len(s.speakingCallbacks))
	for _, cb := range s.speakingCallbacks {
		callbacks = append(callbacks, cb)
	}
	s.voiceMu.RUnlock()

	for _, cb := range callbacks {
		cb(speaking)
	}
}

func (s *DiscordService) AddVoiceStatusCallback(cb func(discord.CallStatus)) int {
	s.voiceMu.Lock()
	defer s.voiceMu.Unlock()
	id := s.nextVoiceStatusCallbackID
	s.nextVoiceStatusCallbackID++
	s.voiceStatusCallbacks[id] = cb
	return id
}

func (s *DiscordService) RemoveVoiceStatusCallback(id int) {
	s.voiceMu.Lock()
	defer s.voiceMu.Unlock()
	delete(s.voiceStatusCallbacks, id)
}

func (s *DiscordService) AddSpeakingCallback(cb func(map[uint64]bool)) int {
	s.voiceMu.Lock()
	defer s.voiceMu.Unlock()
	id := s.nextSpeakingCallbackID
	s.nextSpeakingCallbackID++
	s.speakingCallbacks[id] = cb
	return id
}

func (s *DiscordService) RemoveSpeakingCallback(id int) {
	s.voiceMu.Lock()
	defer s.voiceMu.Unlock()
	delete(s.speakingCallbacks, id)
}

// ---------------- Connected Channel (Linked Channel) Operations ----------------

func (s *DiscordService) GetUserGuilds(callback func(err error, guilds []GuildInfo)) {
	if s.client == nil {
		if callback != nil {
			callback(errors.New("discord client not initialized"), nil)
		}
		return
	}

	s.client.GetUserGuilds(func(result *discord.ClientResult, rawGuilds []discord.GuildMinimal) {
		if !result.Successful() {
			err := fmt.Errorf("failed to get guilds: code %d", result.ErrorCode())
			slog.Warn("Failed to get user guilds", "error", err)
			if callback != nil {
				callback(err, nil)
			}
			return
		}

		guilds := make([]GuildInfo, 0, len(rawGuilds))
		for _, g := range rawGuilds {
			guilds = append(guilds, GuildInfo{
				ID:   g.Id(),
				Name: g.Name(),
			})
		}
		if callback != nil {
			callback(nil, guilds)
		}
	})
}

func (s *DiscordService) GetGuildChannels(guildID uint64, callback func(err error, channels []GuildChannelInfo)) {
	if s.client == nil {
		if callback != nil {
			callback(errors.New("discord client not initialized"), nil)
		}
		return
	}

	s.client.GetGuildChannels(guildID, func(result *discord.ClientResult, rawChannels []discord.GuildChannel) {
		if !result.Successful() {
			err := fmt.Errorf("failed to get guild channels: code %d", result.ErrorCode())
			slog.Warn("Failed to get guild channels", "guildID", guildID, "error", err)
			if callback != nil {
				callback(err, nil)
			}
			return
		}

		channels := make([]GuildChannelInfo, 0, len(rawChannels))
		for _, ch := range rawChannels {
			channels = append(channels, GuildChannelInfo{
				ID:         ch.Id(),
				Name:       ch.Name(),
				Type:       ch.Type(),
				Position:   ch.Position(),
				IsLinkable: ch.IsLinkable(),
			})
		}
		if callback != nil {
			callback(nil, channels)
		}
	})
}

func (s *DiscordService) LinkChannelToLobby(channelID uint64, callback func(error)) {
	s.lobbyMu.RLock()
	lobbyID := s.activeLobbyID
	s.lobbyMu.RUnlock()

	if lobbyID == 0 {
		if callback != nil {
			callback(ErrNotConnectedToLobby)
		}
		return
	}

	s.client.LinkChannelToLobby(lobbyID, channelID, func(result *discord.ClientResult) {
		if !result.Successful() {
			err := fmt.Errorf("failed to link channel: code %d", result.ErrorCode())
			slog.Warn("Failed to link channel to lobby", "lobbyID", lobbyID, "channelID", channelID, "error", err)
			if callback != nil {
				callback(err)
			}
			return
		}
		slog.Info("Successfully linked channel to lobby", "lobbyID", lobbyID, "channelID", channelID)
		info := s.refreshActiveLobbyInfo(lobbyID)
		if callback != nil {
			callback(nil)
		}
		s.notifyLobbyUpdated(info)
	})
}

func (s *DiscordService) UnlinkChannelFromLobby(callback func(error)) {
	s.lobbyMu.RLock()
	lobbyID := s.activeLobbyID
	s.lobbyMu.RUnlock()

	if lobbyID == 0 {
		if callback != nil {
			callback(ErrNotConnectedToLobby)
		}
		return
	}

	s.client.UnlinkChannelFromLobby(lobbyID, func(result *discord.ClientResult) {
		if !result.Successful() {
			err := fmt.Errorf("failed to unlink channel: code %d", result.ErrorCode())
			slog.Warn("Failed to unlink channel from lobby", "lobbyID", lobbyID, "error", err)
			if callback != nil {
				callback(err)
			}
			return
		}
		slog.Info("Successfully unlinked channel from lobby", "lobbyID", lobbyID)
		info := s.refreshActiveLobbyInfo(lobbyID)
		if callback != nil {
			callback(nil)
		}
		s.notifyLobbyUpdated(info)
	})
}

func (s *DiscordService) GetLinkedChannel() (*LinkedChannelInfo, bool) {
	s.lobbyMu.RLock()
	defer s.lobbyMu.RUnlock()
	if s.activeLobbyInfo == nil || s.activeLobbyInfo.LinkedChannel == nil {
		return nil, false
	}
	return s.activeLobbyInfo.LinkedChannel, true
}

func (s *DiscordService) CanCurrentUserLinkLobby() bool {
	s.lobbyMu.RLock()
	defer s.lobbyMu.RUnlock()
	if s.activeLobbyInfo == nil {
		return false
	}
	user, ok := s.UserInfo()
	if !ok || user == nil {
		return false
	}
	myID := user.Id()
	for _, m := range s.activeLobbyInfo.Members {
		if m.UserID == myID {
			return m.CanLinkLobby
		}
	}
	return false
}
