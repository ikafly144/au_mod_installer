package discord

import discord "github.com/ikafly144/discord_social_sdk"

func (s *DiscordService) GetFriends() ([]discord.Discord_RelationshipHandle, error) {
	if !s.IsLoggedIn() {
		return nil, ErrNotLoggedIn
	}
	friends := s.client.GetRelationshipsByGroup(discord.Discord_RelationshipGroupType(discord.Discord_RelationshipType_Friend))
	return friends, nil
}

func (s *DiscordService) SearchFriends(query string) ([]discord.Discord_UserHandle, error) {
	if !s.IsLoggedIn() {
		return nil, ErrNotLoggedIn
	}
	friends := s.client.SearchFriendsByUsername(query)
	return friends, nil
}
