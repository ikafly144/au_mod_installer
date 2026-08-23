package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type DiscordLobbyClient interface {
	CreateLobby(ctx context.Context, hostUserID uint64, metadata map[string]string) (uint64, error)
	AddMember(ctx context.Context, lobbyID uint64, userID uint64, metadata map[string]string) error
	RemoveMember(ctx context.Context, lobbyID uint64, userID uint64) error
	DeleteLobby(ctx context.Context, lobbyID uint64) error
}

type discordLobbyHTTPClient struct {
	botToken   string
	apiBase    string
	httpClient *http.Client
}

func NewDiscordLobbyClient(botToken string, httpClient *http.Client) DiscordLobbyClient {
	if botToken == "" {
		return nil
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &discordLobbyHTTPClient{
		botToken:   botToken,
		apiBase:    "https://discord.com/api/v10",
		httpClient: httpClient,
	}
}

type discordLobbyMemberPayload struct {
	ID       string            `json:"id"`
	Flags    int               `json:"flags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type discordCreateLobbyPayload struct {
	IdleTimeoutSeconds int                         `json:"idle_timeout_seconds,omitempty"`
	Metadata           map[string]string           `json:"metadata,omitempty"`
	Members            []discordLobbyMemberPayload `json:"members,omitempty"`
}

type discordLobbyResponse struct {
	ID            string                      `json:"id"`
	ApplicationID string                      `json:"application_id"`
	Metadata      map[string]string           `json:"metadata"`
	Members       []discordLobbyMemberPayload `json:"members"`
}

func (c *discordLobbyHTTPClient) CreateLobby(ctx context.Context, hostUserID uint64, metadata map[string]string) (uint64, error) {
	members := []discordLobbyMemberPayload{}
	if hostUserID != 0 {
		members = append(members, discordLobbyMemberPayload{
			ID:    fmt.Sprintf("%d", hostUserID),
			Flags: 1, // CanLinkLobby = 1 << 0
			Metadata: map[string]string{
				"is_host": "true",
			},
		})
	}
	payload := discordCreateLobbyPayload{
		IdleTimeoutSeconds: 7200, // 2 hours
		Metadata:           metadata,
		Members:            members,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/lobbies", bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bot "+c.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("discord create lobby returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var lobbyResp discordLobbyResponse
	if err := json.NewDecoder(resp.Body).Decode(&lobbyResp); err != nil {
		return 0, err
	}

	var lobbyID uint64
	_, _ = fmt.Sscanf(lobbyResp.ID, "%d", &lobbyID)
	return lobbyID, nil
}

func (c *discordLobbyHTTPClient) AddMember(ctx context.Context, lobbyID uint64, userID uint64, metadata map[string]string) error {
	if lobbyID == 0 || userID == 0 {
		return nil
	}
	payload := map[string]any{
		"metadata": metadata,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/lobbies/%d/members/%d", c.apiBase, lobbyID, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+c.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord add lobby member returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *discordLobbyHTTPClient) RemoveMember(ctx context.Context, lobbyID uint64, userID uint64) error {
	if lobbyID == 0 || userID == 0 {
		return nil
	}
	url := fmt.Sprintf("%s/lobbies/%d/members/%d", c.apiBase, lobbyID, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *discordLobbyHTTPClient) DeleteLobby(ctx context.Context, lobbyID uint64) error {
	if lobbyID == 0 {
		return nil
	}
	url := fmt.Sprintf("%s/lobbies/%d", c.apiBase, lobbyID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
