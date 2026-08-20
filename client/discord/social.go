package discord

import (
	"strings"

	discord "github.com/ikafly144/discord_social_sdk"
)

func (s *DiscordService) GetFriends() ([]discord.UserHandle, error) {
	if !s.IsLoggedIn() {
		return nil, ErrNotLoggedIn
	}
	var userList []discord.UserHandle
	seen := make(map[uint64]bool)

	// 1. SearchFriendsByUsername("") returns friends list from Discord Social SDK
	for _, u := range s.client.SearchFriendsByUsername("") {
		id := u.Id()
		if id != 0 && !seen[id] {
			seen[id] = true
			userList = append(userList, u)
		}
	}

	// 2. Also inspect GetRelationships to ensure any friends not returned by search are included
	for _, rel := range s.client.GetRelationships() {
		if u, ok := rel.User(); ok {
			id := u.Id()
			if id != 0 && !seen[id] {
				dType := rel.DiscordRelationshipType()
				gType := rel.GameRelationshipType()
				if dType != discord.RelationshipTypeBlocked && gType != discord.RelationshipTypeBlocked {
					seen[id] = true
					userList = append(userList, u)
				}
			}
		}
	}

	return userList, nil
}

func (s *DiscordService) SearchFriends(query string) ([]discord.UserHandle, error) {
	if !s.IsLoggedIn() {
		return nil, ErrNotLoggedIn
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return s.GetFriends()
	}

	friends := s.client.SearchFriendsByUsername(query)
	if len(friends) > 0 {
		return friends, nil
	}

	all, err := s.GetFriends()
	if err != nil {
		return nil, err
	}
	queryLower := strings.ToLower(query)
	var filtered []discord.UserHandle
	for _, u := range all {
		name := strings.ToLower(u.DisplayName())
		if globalName, ok := u.GlobalName(); ok && strings.Contains(strings.ToLower(globalName), queryLower) {
			filtered = append(filtered, u)
			continue
		}
		if strings.Contains(name, queryLower) || strings.Contains(strings.ToLower(u.Username()), queryLower) {
			filtered = append(filtered, u)
		}
	}
	return filtered, nil
}

func (s *DiscordService) AddRelationshipChangedCallback(callback func([]discord.UserHandle)) int {
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
