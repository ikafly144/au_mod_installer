package launcher

import (
	"fmt"
	"image/color"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/client/discord"
)

const (
	lobbyDrawerWidth     = float32(340)
	lobbyAvatarSize      = 28
	maxRenderChatHistory = 100
)

func (l *Launcher) setupLobbyDrawerUI() {
	l.lobbyHeaderTitle = widget.NewLabelWithStyle(
		lang.LocalizeKey("launcher.lobby.title", "Discord Lobby"),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	l.lobbyHeaderSubtitle = widget.NewLabel(
		lang.LocalizeKey("launcher.lobby.no_lobby", "No active lobby"),
	)
	l.lobbyHeaderSubtitle.Wrapping = fyne.TextTruncate

	l.lobbyCreateButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.create", "Create Lobby"),
		theme.ContentAddIcon(),
		l.handleCreateManualLobby,
	)
	l.lobbyCreateButton.Importance = widget.LowImportance

	l.lobbyLeaveButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.leave", "Leave Lobby"),
		theme.CancelIcon(),
		l.handleLeaveLobby,
	)
	l.lobbyLeaveButton.Importance = widget.LowImportance
	l.lobbyLeaveButton.Hide()

	l.lobbyInviteButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.invite", "Invite"),
		theme.MailComposeIcon(),
		l.showDiscordFriendsDialog,
	)
	l.lobbyInviteButton.Importance = widget.LowImportance
	l.lobbyInviteButton.Hide()

	l.lobbyActionBox = container.NewHBox(
		l.lobbyCreateButton,
		l.lobbyInviteButton,
		l.lobbyLeaveButton,
	)

	l.lobbyMemberListContainer = container.NewVBox()

	// Voice Controls
	l.lobbyVoiceStatusLabel = widget.NewLabel(
		lang.LocalizeKey("launcher.lobby.voice_disconnected", "Disconnected"),
	)
	l.lobbyVoiceJoinButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.voice_join", "Join Voice"),
		theme.VolumeUpIcon(),
		l.handleJoinVoice,
	)
	l.lobbyVoiceJoinButton.Importance = widget.MediumImportance

	l.lobbyVoiceLeaveButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.voice_leave", "Leave"),
		theme.CancelIcon(),
		l.handleLeaveVoice,
	)
	l.lobbyVoiceLeaveButton.Importance = widget.LowImportance
	l.lobbyVoiceLeaveButton.Hide()

	l.lobbyVoiceMuteButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.voice_mute", "Mute"),
		theme.MediaRecordIcon(),
		l.handleToggleMute,
	)
	l.lobbyVoiceMuteButton.Importance = widget.LowImportance
	l.lobbyVoiceMuteButton.Hide()

	l.lobbyVoiceDeafenButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.voice_deafen", "Deafen"),
		theme.VolumeMuteIcon(),
		l.handleToggleDeafen,
	)
	l.lobbyVoiceDeafenButton.Importance = widget.LowImportance
	l.lobbyVoiceDeafenButton.Hide()

	l.lobbyVoiceBar = container.NewVBox(
		container.NewBorder(
			nil, nil,
			widget.NewLabelWithStyle(lang.LocalizeKey("launcher.lobby.voice_title", "Voice Call"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			l.lobbyVoiceStatusLabel,
		),
		container.NewHBox(
			l.lobbyVoiceJoinButton,
			l.lobbyVoiceMuteButton,
			l.lobbyVoiceDeafenButton,
			l.lobbyVoiceLeaveButton,
		),
	)

	// Chat Section
	l.lobbyChatMessagesContainer = container.NewVBox()
	l.lobbyChatScroll = container.NewVScroll(l.lobbyChatMessagesContainer)
	l.lobbyChatScroll.SetMinSize(fyne.NewSize(lobbyDrawerWidth-30, 160))

	l.lobbyChatEntry = widget.NewEntry()
	l.lobbyChatEntry.SetPlaceHolder(lang.LocalizeKey("launcher.lobby.chat_placeholder", "Type a message..."))
	l.lobbyChatEntry.OnSubmitted = func(text string) {
		l.handleSendChatMessage()
	}

	l.lobbyChatSendButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.chat_send", "Send"),
		theme.MailSendIcon(),
		l.handleSendChatMessage,
	)
	l.lobbyChatSendButton.Importance = widget.LowImportance

	chatInputRow := container.NewBorder(nil, nil, nil, l.lobbyChatSendButton, l.lobbyChatEntry)

	chatSection := container.NewBorder(
		widget.NewLabelWithStyle(lang.LocalizeKey("launcher.lobby.chat_title", "Lobby Chat"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		chatInputRow,
		nil,
		nil,
		l.lobbyChatScroll,
	)

	// Background and sizing
	drawerBackground := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	drawerBackground.CornerRadius = theme.InputRadiusSize()
	drawerSizer := canvas.NewRectangle(color.Transparent)
	drawerSizer.SetMinSize(fyne.NewSize(lobbyDrawerWidth, 0))

	drawerContent := container.NewPadded(container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, l.lobbyHeaderTitle, l.lobbyActionBox),
			l.lobbyHeaderSubtitle,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			l.lobbyVoiceBar,
		),
		nil,
		nil,
		container.NewVScroll(container.NewVBox(
			l.lobbyMemberListContainer,
			widget.NewSeparator(),
			chatSection,
		)),
	))

	l.lobbyDrawerPanel = container.NewStack(
		drawerSizer,
		drawerBackground,
		drawerContent,
	)

	l.lobbyToggleDrawerButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.toggle_button", "Lobby"),
		theme.NavigateNextIcon(),
		func() {
			l.setLobbyDrawerExpanded(!l.lobbyDrawerExpanded)
		},
	)
	l.lobbyToggleDrawerButton.Importance = widget.LowImportance

	l.lobbyDrawerContainer = container.NewHBox(
		l.lobbyToggleDrawerButton,
		l.lobbyDrawerPanel,
	)

	l.setLobbyDrawerExpanded(false)

	// Register discord callbacks
	if l.state != nil && l.state.Core != nil && l.state.Core.DiscordService != nil {
		ds := l.state.Core.DiscordService
		ds.AddLobbyUpdatedCallback(func(info *discord.LobbyInfo) {
			fyne.Do(func() {
				l.refreshLobbyUI(info)
			})
		})
		ds.AddLobbyMessageCallback(func(msg discord.LobbyMessage) {
			fyne.Do(func() {
				l.appendLobbyMessage(msg)
			})
		})
		ds.AddVoiceStatusCallback(func(status discord.CallStatus) {
			fyne.Do(func() {
				l.refreshVoiceUI(status)
			})
		})
		ds.AddSpeakingCallback(func(speaking map[uint64]bool) {
			fyne.Do(func() {
				if info, ok := ds.GetActiveLobby(); ok {
					l.refreshLobbyUI(info)
				}
			})
		})
	}
}

func (l *Launcher) setLobbyDrawerExpanded(expanded bool) {
	l.lobbyDrawerExpanded = expanded
	if l.lobbyDrawerPanel == nil || l.lobbyToggleDrawerButton == nil {
		return
	}
	if expanded {
		l.lobbyDrawerPanel.Show()
		l.lobbyToggleDrawerButton.SetIcon(theme.NavigateBackIcon())
		if l.lobbyChatScroll != nil {
			l.lobbyChatScroll.ScrollToBottom()
		}
	} else {
		l.lobbyDrawerPanel.Hide()
		l.lobbyToggleDrawerButton.SetIcon(theme.NavigateNextIcon())
	}
}

func (l *Launcher) refreshLobbyUI(info *discord.LobbyInfo) {
	if info == nil || info.ID == 0 {
		l.lobbyHeaderSubtitle.SetText(lang.LocalizeKey("launcher.lobby.no_lobby", "No active lobby"))
		l.lobbyCreateButton.Show()
		l.lobbyLeaveButton.Hide()
		l.lobbyInviteButton.Hide()
		l.lobbyMemberListContainer.Objects = nil
		l.lobbyMemberListContainer.Refresh()
		l.refreshVoiceUI(discord.CallStatusDisconnected)
		return
	}

	// Active lobby exists
	l.lobbyCreateButton.Hide()
	l.lobbyLeaveButton.Show()
	l.lobbyInviteButton.Show()

	roomCode := info.Metadata["room_code"]
	modName := info.Metadata["profile_name"]

	subtitleParts := make([]string, 0, 2)
	if roomCode != "" {
		subtitleParts = append(subtitleParts, lang.LocalizeKey("launcher.lobby.room", "Room: {{.Code}}", map[string]any{"Code": roomCode}))
	}
	if modName != "" {
		subtitleParts = append(subtitleParts, lang.LocalizeKey("launcher.lobby.mod", "Mod: {{.Name}}", map[string]any{"Name": modName}))
	}
	if len(subtitleParts) == 0 {
		subtitleParts = append(subtitleParts, fmt.Sprintf("ID: %d", info.ID))
	}
	l.lobbyHeaderSubtitle.SetText(strings.Join(subtitleParts, " | "))

	// Build member items
	l.lobbyMemberListContainer.Objects = nil
	memberCountLabel := widget.NewLabelWithStyle(
		lang.LocalizeKey("launcher.lobby.members", "Members ({{.Count}})", map[string]any{"Count": len(info.Members)}),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	l.lobbyMemberListContainer.Add(memberCountLabel)

	for _, member := range info.Members {
		card := l.buildMemberCard(member, info.Metadata)
		l.lobbyMemberListContainer.Add(card)
	}
	l.lobbyMemberListContainer.Refresh()

	// Automatically open drawer if it was closed and user just joined
	if !l.lobbyDrawerExpanded {
		l.setLobbyDrawerExpanded(true)
		l.loadLobbyMessagesHistory()
	}
}

func (l *Launcher) buildMemberCard(member discord.LobbyMember, lobbyMeta map[string]string) fyne.CanvasObject {
	avatar := canvas.NewImageFromResource(theme.AccountIcon())
	avatar.SetMinSize(fyne.NewSquareSize(lobbyAvatarSize))
	l.refreshDiscordFriendAvatarCanvas(avatar, member.UserID, lobbyAvatarSize)
	l.ensureDiscordFriendAvatarLoaded(member.UserID, member.AvatarURL, func() {
		l.refreshDiscordFriendAvatarCanvas(avatar, member.UserID, lobbyAvatarSize)
	})

	nameLabel := widget.NewLabelWithStyle(member.DisplayName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	nameLabel.Wrapping = fyne.TextTruncate

	badges := container.NewHBox()

	if member.IsHost {
		hostBadge := widget.NewLabel("👑 " + lang.LocalizeKey("launcher.lobby.host", "Host"))
		badges.Add(hostBadge)
	}

	if member.IsReady {
		readyBadge := widget.NewLabel("✓ " + lang.LocalizeKey("launcher.lobby.ready", "Ready"))
		badges.Add(readyBadge)
	}

	if member.IsSpeaking {
		speakingBadge := widget.NewIcon(theme.VolumeUpIcon())
		badges.Add(speakingBadge)
	} else if member.IsVoiceConnected {
		voiceBadge := widget.NewIcon(theme.VolumeMuteIcon())
		badges.Add(voiceBadge)
	}

	// Mod compatibility check
	if expectedHash, ok := lobbyMeta["profile_hash"]; ok && expectedHash != "" {
		memberHash := member.Metadata["profile_hash"]
		if memberHash != "" && memberHash != expectedHash {
			diffLabel := widget.NewLabel("⚠️ " + lang.LocalizeKey("launcher.lobby.mod_diff", "Mod Diff"))
			badges.Add(diffLabel)
		}
	}

	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	cardBackground.CornerRadius = 4

	cardContent := container.NewBorder(
		nil,
		nil,
		container.NewPadded(avatar),
		badges,
		container.NewVBox(nameLabel),
	)

	return container.NewStack(cardBackground, container.NewPadded(cardContent))
}

func (l *Launcher) refreshVoiceUI(status discord.CallStatus) {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService

	if status == discord.CallStatusConnected {
		l.lobbyVoiceStatusLabel.SetText(lang.LocalizeKey("launcher.lobby.voice_connected", "Connected"))
		l.lobbyVoiceJoinButton.Hide()
		l.lobbyVoiceLeaveButton.Show()
		l.lobbyVoiceMuteButton.Show()
		l.lobbyVoiceDeafenButton.Show()

		if ds.IsSelfMuted() {
			l.lobbyVoiceMuteButton.SetIcon(theme.MediaRecordIcon())
			l.lobbyVoiceMuteButton.SetText(lang.LocalizeKey("launcher.lobby.voice_unmute", "Unmute"))
		} else {
			l.lobbyVoiceMuteButton.SetIcon(theme.VolumeMuteIcon())
			l.lobbyVoiceMuteButton.SetText(lang.LocalizeKey("launcher.lobby.voice_mute", "Mute"))
		}

		if ds.IsSelfDeafened() {
			l.lobbyVoiceDeafenButton.SetIcon(theme.VolumeUpIcon())
			l.lobbyVoiceDeafenButton.SetText(lang.LocalizeKey("launcher.lobby.voice_undeafen", "Undeafen"))
		} else {
			l.lobbyVoiceDeafenButton.SetIcon(theme.VolumeMuteIcon())
			l.lobbyVoiceDeafenButton.SetText(lang.LocalizeKey("launcher.lobby.voice_deafen", "Deafen"))
		}
	} else {
		l.lobbyVoiceStatusLabel.SetText(lang.LocalizeKey("launcher.lobby.voice_disconnected", "Disconnected"))
		l.lobbyVoiceJoinButton.Show()
		l.lobbyVoiceLeaveButton.Hide()
		l.lobbyVoiceMuteButton.Hide()
		l.lobbyVoiceDeafenButton.Hide()
	}
	l.lobbyVoiceBar.Refresh()
}

func (l *Launcher) appendLobbyMessage(msg discord.LobbyMessage) {
	timeStr := msg.SentAt.Format("15:04")
	if msg.SentAt.IsZero() {
		timeStr = time.Now().Format("15:04")
	}

	authorHeader := widget.NewLabelWithStyle(
		fmt.Sprintf("%s (%s)", msg.AuthorName, timeStr),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	authorHeader.Wrapping = fyne.TextTruncate

	contentLabel := widget.NewLabel(msg.Content)
	contentLabel.Wrapping = fyne.TextWrapBreak

	msgBox := container.NewVBox(authorHeader, contentLabel)
	msgBg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	msgBg.CornerRadius = 4

	msgCard := container.NewStack(msgBg, container.NewPadded(msgBox))

	l.lobbyChatMessagesContainer.Add(msgCard)

	// Cap rendered items
	if len(l.lobbyChatMessagesContainer.Objects) > maxRenderChatHistory {
		l.lobbyChatMessagesContainer.Objects = l.lobbyChatMessagesContainer.Objects[len(l.lobbyChatMessagesContainer.Objects)-maxRenderChatHistory:]
	}
	l.lobbyChatMessagesContainer.Refresh()

	if l.lobbyChatScroll != nil {
		l.lobbyChatScroll.ScrollToBottom()
	}
}

func (l *Launcher) loadLobbyMessagesHistory() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService
	ds.GetLobbyMessages(50, func(err error, messages []discord.LobbyMessage) {
		if err != nil {
			slog.Debug("Failed to load lobby messages history", "error", err)
			return
		}
		fyne.Do(func() {
			l.lobbyChatMessagesContainer.Objects = nil
			for _, m := range messages {
				l.appendLobbyMessage(m)
			}
		})
	})
}

func (l *Launcher) handleCreateManualLobby() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	if !l.state.Core.DiscordService.IsLoggedIn() {
		l.showDiscordFriendsDialog()
		return
	}

	profileID := l.selectedProfileID
	if profileID == uuid.Nil && len(l.profiles) > 0 {
		profileID = l.profiles[0].ID
	}
	if profileID == uuid.Nil {
		l.state.ShowErrorDialog(fmt.Errorf("please select a profile first"))
		return
	}

	l.state.Core.CreateManualLobby(profileID, func(err error, lobbyID uint64) {
		if err != nil {
			fyne.Do(func() {
				l.state.ShowErrorDialog(err)
			})
		}
	})
}

func (l *Launcher) handleLeaveLobby() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	l.state.Core.DiscordService.LeaveLobby(func(err error) {
		if err != nil {
			slog.Warn("Failed to leave lobby", "error", err)
		}
	})
}

func (l *Launcher) handleJoinVoice() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	l.state.Core.DiscordService.ConnectVoice(func(err error) {
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, l.state.Window)
			})
		}
	})
}

func (l *Launcher) handleLeaveVoice() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	l.state.Core.DiscordService.DisconnectVoice(func(err error) {
		if err != nil {
			slog.Warn("Failed to disconnect voice", "error", err)
		}
	})
}

func (l *Launcher) handleToggleMute() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService
	ds.SetSelfMuted(!ds.IsSelfMuted())
	l.refreshVoiceUI(ds.GetVoiceStatus())
}

func (l *Launcher) handleToggleDeafen() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService
	ds.SetSelfDeafened(!ds.IsSelfDeafened())
	l.refreshVoiceUI(ds.GetVoiceStatus())
}

func (l *Launcher) handleSendChatMessage() {
	if l.lobbyChatEntry == nil {
		return
	}
	text := strings.TrimSpace(l.lobbyChatEntry.Text)
	if text == "" {
		return
	}
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService
	l.lobbyChatEntry.SetText("")
	ds.SendLobbyMessage(text, func(err error, messageID uint64) {
		if err != nil {
			slog.Warn("Failed to send lobby message", "error", err)
		}
	})
}
