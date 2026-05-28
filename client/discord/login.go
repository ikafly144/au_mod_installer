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
			slog.Info("Discord client is ready")
			s.readyOnce.Do(func() {
				close(s.ready)
			})
		}
		if arg0 == discord.Discord_Client_Status_Disconnected {
			slog.Info("Discord client disconnected")
			s.signInMu.Lock()
			s.loggedIn = false
			s.signInMu.Unlock()
			s.readyOnce.Do(func() {
				close(s.ready)
			})
		}
		slog.Info("Discord client status changed", "status", arg0, "error", arg1, "code", arg2)
	})
	s.login(true, func(success bool) {
		if !success {
			slog.Warn("Discord login failed during Connect")
			s.readyOnce.Do(func() {
				close(s.ready)
			})
		}
	})
}

func (s *DiscordService) Disconnect() {
	s.client.Disconnect()
}

func (s *DiscordService) WaitReady() {
	<-s.ready
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

func (s *DiscordService) StartSignIn(callback func(bool)) (started bool) {
	s.signInMu.Lock()
	if s.signingIn {
		s.signInMu.Unlock()
		return false
	}
	s.signingIn = true
	s.signInMu.Unlock()

	codeVerifier := s.client.CreateAuthorizationCodeVerifier()
	authArgs := discord.NewAuthorizationArgs()
	authArgs.SetClientId(s.client.GetApplicationId())
	authArgs.SetScopes(discord.Client_GetDefaultCommunicationScopes())
	authArgs.SetCodeChallenge(new(codeVerifier.Challenge()))

	s.client.Authorize(authArgs, func(arg0 *discord.Discord_ClientResult, arg1, arg2 string) {
		if !arg0.Successful() {
			slog.Warn("Failed to authorize Discord client", "error", arg0.ErrorCode())
			s.signInMu.Lock()
			s.signingIn = false
			s.signInMu.Unlock()
			if callback != nil {
				callback(false)
			}
			return
		}
		s.client.GetToken(s.client.GetApplicationId(), arg1, codeVerifier.Verifier(), arg2,
			func(result *discord.Discord_ClientResult, accessToken, refreshToken string, tokenType discord.Discord_AuthorizationTokenType, expiresIn int32, scopes string) {
				if !result.Successful() {
					slog.Warn("Failed to get Discord token", "error", result.ErrorCode())
					s.signInMu.Lock()
					s.signingIn = false
					s.signInMu.Unlock()
					if callback != nil {
						callback(false)
					}
					return
				}
				creds := &discordCredentials{
					ClientID:     s.client.GetApplicationId(),
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
					TokenType:    tokenType,
					ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second).Unix(),
				}
				if err := s.saveCredentials(creds); err != nil {
					slog.Error("Failed to save Discord credentials", "error", err)
				}
				s.login(true, func(success bool) {
					s.signInMu.Lock()
					s.signingIn = false
					// loggedIn state is updated within s.login callbacks
					s.signInMu.Unlock()
					if callback != nil {
						callback(success)
					}
				})
			})
	})
	return true
}

func (s *DiscordService) login(connect bool, callbacks ...func(bool)) {
	creds, ok := s.loadCredentials()
	if !ok {
		s.StartSignIn(func(b bool) {
			if !b {
				slog.Warn("Discord sign-in failed")
			}
			for _, callback := range callbacks {
				callback(b)
			}
		})
		return
	}

	s.client.SetTokenExpirationCallback(func() {
		s.client.RefreshToken(creds.ClientID, creds.RefreshToken, func(result *discord.Discord_ClientResult, accessToken, refreshToken string, tokenType discord.Discord_AuthorizationTokenType, expiresIn int32, scopes string) {
			if !result.Successful() {
				slog.Warn("Failed to refresh Discord token", "error", result.ErrorCode())
				s.signInMu.Lock()
				s.loggedIn = false
				s.signInMu.Unlock()
				return
			}
			creds.AccessToken = accessToken
			creds.RefreshToken = refreshToken
			creds.TokenType = tokenType
			creds.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second).Unix()
			if err := s.saveCredentials(creds); err != nil {
				slog.Error("Failed to save updated Discord credentials", "error", err)
			}
			s.signInMu.Lock()
			s.loggedIn = true
			s.signInMu.Unlock()
		})
	})

	s.client.UpdateToken(creds.TokenType, creds.AccessToken, func(result *discord.Discord_ClientResult) {
		if !result.Successful() {
			slog.Warn("Failed to update Discord token", "error", result.ErrorCode())
			s.signInMu.Lock()
			s.loggedIn = false
			s.signInMu.Unlock()
			if connect {
				s.client.Connect()
			}
			for _, callback := range callbacks {
				callback(false)
			}
			return
		}
		s.signInMu.Lock()
		s.loggedIn = true
		s.signInMu.Unlock()
		slog.Info("Successfully logged in to Discord")
		if connect {
			s.client.Connect()
		}
		for _, callback := range callbacks {
			callback(true)
		}
	})
}

func (s *DiscordService) Logout() {
	creds, ok := s.loadCredentials()
	if !ok {
		slog.Info("No existing Discord credentials found, skipping logout")
		return
	}

	s.client.RevokeToken(creds.ClientID, creds.AccessToken, func(result *discord.Discord_ClientResult) {
		if !result.Successful() {
			slog.Warn("Failed to revoke Discord token", "error", result.ErrorCode())
		} else {
			slog.Info("Successfully revoked Discord token")
			s.clearCredentials()
		}
	})
	s.signInMu.Lock()
	s.loggedIn = false
	s.signInMu.Unlock()
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
