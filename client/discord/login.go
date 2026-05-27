package discord

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/danieljoos/wincred"
	discord "github.com/ikafly144/discord_social_sdk"
)

const discordCredentialsKey = "au_mod_installer_discord"

func (s *DiscordService) Connect() {
	s.client.SetStatusChangedCallback(func(arg0 discord.Discord_Client_Status, arg1 discord.Discord_Client_Error, arg2 int32) {
		if arg0 == discord.Discord_Client_Status_Ready {
			s.login()
		}
		slog.Info("Discord client status changed", "status", arg0, "error", arg1, "code", arg2)
	})
	s.client.Connect()
}

func (s *DiscordService) Disconnect() {
	s.client.Disconnect()
}

func (s *DiscordService) IsLoggedIn() bool {
	s.signInMu.Lock()
	defer s.signInMu.Unlock()
	return s.loggedIn
}

func (s *DiscordService) UserInfo() (*discord.Discord_UserHandle, bool) {
	if !s.IsLoggedIn() {
		return nil, false
	}
	user, ok := s.client.GetCurrentUserV2()
	if !ok {
		return nil, false
	}
	return &user, true
}

func (s *DiscordService) StartSignIn() (started bool) {
	if !s.signInMu.TryLock() {
		return false
	}
	codeVerifier := s.client.CreateAuthorizationCodeVerifier()
	authArgs := discord.NewAuthorizationArgs()
	authArgs.SetClientId(s.client.GetApplicationId())
	authArgs.SetScopes(discord.Client_GetDefaultCommunicationScopes())
	authArgs.SetCodeChallenge(new(codeVerifier.Challenge()))

	s.client.Authorize(authArgs, func(arg0 *discord.Discord_ClientResult, arg1, arg2 string) {
		if !arg0.Successful() {
			slog.Warn("Failed to authorize Discord client", "error", arg0.ErrorCode())
			s.signInMu.Unlock()
			return
		}
		s.client.GetToken(s.client.GetApplicationId(), arg1, codeVerifier.Verifier(), arg2,
			func(arg0 *discord.Discord_ClientResult, arg1, arg2 string, arg3 discord.Discord_AuthorizationTokenType, arg4 int32, arg5 string) {
				if !arg0.Successful() {
					slog.Warn("Failed to get Discord token", "error", arg0.ErrorCode())
					s.signInMu.Unlock()
					return
				}
				creds := &discordCredentials{
					ClientID:     s.client.GetApplicationId(),
					AccessToken:  arg1,
					RefreshToken: arg2,
					TokenType:    arg3,
					ExpiresAt:    time.Now().Add(time.Duration(arg4) * time.Second).Unix(),
				}
				if err := s.saveCredentials(creds); err != nil {
					slog.Error("Failed to save Discord credentials", "error", err)
				}
				s.loggedIn = true
				s.signInMu.Unlock()
				s.login()
			})
	})
	return true
}

func (s *DiscordService) login() {
	creds, ok := s.loadCredentials()
	if !ok {
		slog.Info("No existing Discord credentials found, skipping login")
		return
	}

	s.client.SetTokenExpirationCallback(func() {
		s.client.RefreshToken(creds.ClientID, creds.RefreshToken, func(arg0 *discord.Discord_ClientResult, arg1, arg2 string, arg3 discord.Discord_AuthorizationTokenType, arg4 int32, arg5 string) {
			if !arg0.Successful() {
				slog.Warn("Failed to refresh Discord token", "error", arg0.ErrorCode())
				s.loggedIn = false
				return
			}
			creds.AccessToken = arg1
			creds.RefreshToken = arg2
			creds.TokenType = arg3
			creds.ExpiresAt = time.Now().Add(time.Duration(arg4) * time.Second).Unix()
			if err := s.saveCredentials(creds); err != nil {
				slog.Error("Failed to save updated Discord credentials", "error", err)
			}
			s.loggedIn = true
		})
	})

	s.client.UpdateToken(creds.TokenType, creds.AccessToken, func(arg0 *discord.Discord_ClientResult) {
		if !arg0.Successful() {
			slog.Warn("Failed to update Discord token", "error", arg0.ErrorCode())
			s.loggedIn = false
			return
		}
		s.loggedIn = true
		slog.Info("Successfully logged in to Discord")
	})
}

func (s *DiscordService) Logout() {
	creds, ok := s.loadCredentials()
	if !ok {
		slog.Info("No existing Discord credentials found, skipping logout")
		return
	}

	s.client.RevokeToken(creds.ClientID, creds.AccessToken, func(arg0 *discord.Discord_ClientResult) {
		if !arg0.Successful() {
			slog.Warn("Failed to revoke Discord token", "error", arg0.ErrorCode())
		} else {
			slog.Info("Successfully revoked Discord token")
			s.clearCredentials()
		}
	})
	s.loggedIn = false
}

type discordCredentials struct {
	ClientID     uint64                                 `json:"client_id"`
	AccessToken  string                                 `json:"access_token"`
	RefreshToken string                                 `json:"refresh_token"`
	TokenType    discord.Discord_AuthorizationTokenType `json:"token_type"`
	ExpiresAt    int64                                  `json:"expires_at"`
}

func (s *DiscordService) loadCredentials() (*discordCredentials, bool) {
	creds, err := wincred.GetGenericCredential(discordCredentialsKey)
	if err != nil {
		return nil, false
	}
	var discordCreds discordCredentials
	if err := json.Unmarshal(creds.CredentialBlob, &discordCreds); err != nil {
		return nil, false
	}
	return &discordCreds, true
}

func (s *DiscordService) saveCredentials(creds *discordCredentials) error {
	blob, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	cred, err := wincred.GetGenericCredential(discordCredentialsKey)
	if err != nil {
		cred = wincred.NewGenericCredential(discordCredentialsKey)
	}
	cred.CredentialBlob = blob
	cred.Persist = wincred.PersistLocalMachine
	return cred.Write()
}

func (s *DiscordService) clearCredentials() error {
	cred, err := wincred.GetGenericCredential(discordCredentialsKey)
	if err != nil {
		slog.Warn("No existing Discord credentials to clear", "error", err)
		return nil // Credential doesn't exist, consider it cleared
	}
	return cred.Delete()
}
