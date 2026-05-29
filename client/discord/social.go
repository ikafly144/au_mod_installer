package discord

import discord "github.com/ikafly144/discord_social_sdk"

func (s *DiscordService) GetFriends() ([]discord.Discord_RelationshipHandle, error) {
	if !s.IsLoggedIn() {
		return nil, ErrNotLoggedIn
	}
	friends := s.client.GetRelationships()
	var friendList []discord.Discord_RelationshipHandle
	for _, friend := range friends {
		if friend.DiscordRelationshipType() != discord.Discord_RelationshipType_Friend {
			continue
		}
		friendList = append(friendList, friend)
	}
	return friendList, nil
}

func (s *DiscordService) SearchFriends(query string) ([]discord.Discord_UserHandle, error) {
	if !s.IsLoggedIn() {
		return nil, ErrNotLoggedIn
	}
	friends := s.client.SearchFriendsByUsername(query)
	return friends, nil
}

func (s *DiscordService) AddRelationshipChangedCallback(callback func([]discord.Discord_RelationshipHandle)) int {
	s.relationshipsMu.Lock()
	defer s.relationshipsMu.Unlock()
	id := s.nextRelationshipCallbackID
	s.relationShipChangedCallbacks[id] = callback
	s.nextRelationshipCallbackID++
	return id
}

func (s *DiscordService) RemoveRelationshipChangedCallback(id int) {
	s.relationshipsMu.Lock()
	defer s.relationshipsMu.Unlock()
	delete(s.relationShipChangedCallbacks, id)
}
