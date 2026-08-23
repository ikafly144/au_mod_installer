package launcher

import (
	"errors"
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
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	discordsdk "github.com/ikafly144/discord_social_sdk"
)

const (
	lobbyDrawerWidth     = float32(360)
	lobbyAvatarSize      = 28
	maxRenderChatHistory = 100
)

func (l *Launcher) setupLobbyDrawerUI() {
	l.lobbyDrawerCurrentTab = "lobby"

	// 1. Tab buttons in top header
	l.lobbyTabButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.tab_lobby", "Lobby"),
		theme.MailAttachmentIcon(),
		func() { l.switchLobbyDrawerTab("lobby") },
	)
	l.lobbyTabButton.Importance = widget.MediumImportance

	l.friendsTabButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.tab_friends", "Friends"),
		theme.AccountIcon(),
		func() { l.switchLobbyDrawerTab("friends") },
	)
	l.friendsTabButton.Importance = widget.LowImportance

	l.lobbyDrawerCloseButton = widget.NewButtonWithIcon(
		"",
		theme.CancelIcon(),
		func() { l.setLobbyDrawerExpanded(false) },
	)
	l.lobbyDrawerCloseButton.Importance = widget.LowImportance

	tabHeader := container.NewBorder(
		nil, nil,
		container.NewHBox(l.lobbyTabButton, l.friendsTabButton),
		l.lobbyDrawerCloseButton,
	)

	// 2. Lobby Tab Components
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
		lang.LocalizeKey("launcher.lobby.invite", "Invite Friends"),
		theme.MailComposeIcon(),
		func() { l.switchLobbyDrawerTab("friends") },
	)
	l.lobbyInviteButton.Importance = widget.LowImportance
	l.lobbyInviteButton.Hide()

	l.lobbyActionBox = container.NewHBox(
		l.lobbyCreateButton,
		l.lobbyInviteButton,
		l.lobbyLeaveButton,
	)

	// Lobby URL Share Section
	l.lobbyShareButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.share_url", "Share Lobby URL"),
		theme.ContentCopyIcon(),
		l.handleShareLobby,
	)
	l.lobbyShareButton.Importance = widget.MediumImportance

	l.lobbyShareURLEntry = widget.NewEntry()
	l.lobbyShareURLEntry.Wrapping = fyne.TextTruncate
	l.lobbyShareURLEntry.Disable()

	l.lobbyShareCopyButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.copy_url", "Copy"),
		theme.ContentCopyIcon(),
		l.handleCopyLobbyURL,
	)
	l.lobbyShareCopyButton.Importance = widget.LowImportance

	l.lobbyShareStopButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.stop_share", "Stop"),
		theme.CancelIcon(),
		l.handleStopSharingLobby,
	)
	l.lobbyShareStopButton.Importance = widget.LowImportance

	l.lobbyShareCard = container.NewVBox(
		l.lobbyShareButton,
	)

	// Connected Channel Section
	l.lobbyChannelButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.link_channel", "Link Discord Channel"),
		theme.MediaRecordIcon(),
		l.showLinkChannelDialog,
	)
	l.lobbyChannelButton.Importance = widget.LowImportance

	l.lobbyChannelNameLabel = widget.NewLabel("")
	l.lobbyChannelNameLabel.Wrapping = fyne.TextTruncate

	l.lobbyChannelUnlinkButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.unlink_channel", "Unlink"),
		theme.CancelIcon(),
		l.handleUnlinkChannel,
	)
	l.lobbyChannelUnlinkButton.Importance = widget.LowImportance

	l.lobbyChannelCard = container.NewVBox()

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
	l.lobbyChatScroll.SetMinSize(fyne.NewSize(lobbyDrawerWidth-30, 140))

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

	l.lobbyTabContent = container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, l.lobbyHeaderTitle, l.lobbyActionBox),
			l.lobbyHeaderSubtitle,
			widget.NewSeparator(),
			l.lobbyShareCard,
			l.lobbyChannelCard,
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
	)

	// 3. Friends Tab Components
	l.drawerFriendsSearchEntry = widget.NewEntry()
	l.drawerFriendsSearchEntry.SetPlaceHolder(lang.LocalizeKey("launcher.discord_friends.search_placeholder", "Search friends..."))
	l.drawerFriendsSearchEntry.OnChanged = func(query string) {
		l.refreshDrawerFriendsList(query)
	}

	l.drawerFriendsLoading = widget.NewProgressBarInfinite()
	l.drawerFriendsLoading.Hide()

	l.drawerFriendsListContainer = container.NewVBox()
	friendsScroll := container.NewVScroll(l.drawerFriendsListContainer)

	l.friendsTabContent = container.NewBorder(
		container.NewVBox(
			l.drawerFriendsSearchEntry,
			l.drawerFriendsLoading,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		friendsScroll,
	)
	l.friendsTabContent.Hide()

	// Background and sizing
	drawerBackground := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	drawerBackground.CornerRadius = theme.InputRadiusSize()
	drawerBackground.StrokeColor = theme.Color(theme.ColorNameButton)
	drawerBackground.StrokeWidth = 1
	drawerSizer := canvas.NewRectangle(color.Transparent)
	drawerSizer.SetMinSize(fyne.NewSize(lobbyDrawerWidth, 0))

	drawerContent := container.NewPadded(container.NewBorder(
		tabHeader,
		nil,
		nil,
		nil,
		container.NewStack(
			l.lobbyTabContent,
			l.friendsTabContent,
		),
	))

	rawDrawerPanel := container.NewStack(
		drawerSizer,
		drawerBackground,
		drawerContent,
	)

	l.lobbyDrawerPanel = container.NewStack(
		uicommon.NewEventCatcherContainer(rawDrawerPanel),
	)

	l.lobbyToggleDrawerButton = widget.NewButtonWithIcon(
		lang.LocalizeKey("launcher.lobby.toggle_button", "Lobby"),
		theme.MailAttachmentIcon(),
		func() {
			l.setLobbyDrawerExpanded(!l.lobbyDrawerExpanded)
		},
	)
	l.lobbyToggleDrawerButton.Importance = widget.LowImportance

	topBar := container.NewBorder(
		nil, nil, nil,
		container.NewPadded(l.lobbyToggleDrawerButton),
	)

	l.lobbyDrawerOverlay = container.NewBorder(
		topBar,
		nil,
		nil,
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

func (l *Launcher) switchLobbyDrawerTab(tab string) {
	l.lobbyDrawerCurrentTab = tab
	if tab == "friends" {
		l.lobbyTabButton.Importance = widget.LowImportance
		l.friendsTabButton.Importance = widget.MediumImportance
		l.lobbyTabContent.Hide()
		l.friendsTabContent.Show()
		l.refreshDrawerFriendsList(l.drawerFriendsSearchEntry.Text)
	} else {
		l.lobbyTabButton.Importance = widget.MediumImportance
		l.friendsTabButton.Importance = widget.LowImportance
		l.friendsTabContent.Hide()
		l.lobbyTabContent.Show()
		if l.state != nil && l.state.Core != nil && l.state.Core.DiscordService != nil {
			if info, ok := l.state.Core.DiscordService.GetActiveLobby(); ok {
				l.refreshLobbyUI(info)
			}
		}
	}
	l.lobbyTabButton.Refresh()
	l.friendsTabButton.Refresh()
}

func (l *Launcher) openDrawerTab(tab string) {
	l.setLobbyDrawerExpanded(true)
	l.switchLobbyDrawerTab(tab)
}

func (l *Launcher) setLobbyDrawerExpanded(expanded bool) {
	l.lobbyDrawerExpanded = expanded
	if l.lobbyDrawerPanel == nil || l.lobbyToggleDrawerButton == nil {
		return
	}
	if expanded {
		l.lobbyDrawerPanel.Show()
		l.lobbyToggleDrawerButton.SetIcon(theme.CancelIcon())
		if l.lobbyDrawerCurrentTab == "friends" {
			l.refreshDrawerFriendsList(l.drawerFriendsSearchEntry.Text)
		} else if l.lobbyChatScroll != nil {
			l.lobbyChatScroll.ScrollToBottom()
		}
	} else {
		l.lobbyDrawerPanel.Hide()
		l.lobbyToggleDrawerButton.SetIcon(theme.MailAttachmentIcon())
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
		l.refreshLobbyShareUI()
		l.refreshConnectedChannelUI(nil, false)
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

	// Check if current user is host
	isHost := false
	if l.state != nil && l.state.Core != nil && l.state.Core.DiscordService != nil {
		if u, ok := l.state.Core.DiscordService.UserInfo(); ok {
			for _, m := range info.Members {
				if m.UserID == u.Id() && m.IsHost {
					isHost = true
					break
				}
			}
		}
	}

	// Refresh Share UI & Channel UI
	l.refreshLobbyShareUI()
	l.refreshConnectedChannelUI(info.LinkedChannel, isHost)

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

func (l *Launcher) refreshLobbyShareUI() {
	if l.state == nil || l.state.Core == nil {
		return
	}
	shared := l.state.Core.GetSharedLobby()
	l.lobbyShareCard.Objects = nil

	if shared.SessionID != "" && shared.URL != "" {
		l.lobbyShareURLEntry.SetText(shared.URL)
		l.lobbyShareCard.Add(container.NewBorder(
			nil, nil,
			widget.NewLabelWithStyle(lang.LocalizeKey("launcher.lobby.shared_url_title", "Lobby Link:"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(l.lobbyShareCopyButton, l.lobbyShareStopButton),
			l.lobbyShareURLEntry,
		))
	} else {
		l.lobbyShareCard.Add(l.lobbyShareButton)
	}
	l.lobbyShareCard.Refresh()
}

func (l *Launcher) refreshConnectedChannelUI(linked *discord.LinkedChannelInfo, isHost bool) {
	l.lobbyChannelCard.Objects = nil
	if linked != nil && linked.ID != 0 {
		name := linked.Name
		l.lobbyChannelNameLabel.SetText("🔗 " + name)
		var right fyne.CanvasObject = nil
		if isHost {
			right = l.lobbyChannelUnlinkButton
		}
		l.lobbyChannelCard.Add(container.NewBorder(
			nil, nil,
			widget.NewLabelWithStyle(lang.LocalizeKey("launcher.lobby.channel_title", "Linked Channel:"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			right,
			l.lobbyChannelNameLabel,
		))
	} else if isHost {
		l.lobbyChannelCard.Add(l.lobbyChannelButton)
	}
	l.lobbyChannelCard.Refresh()
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
		l.openDrawerTab("friends")
		return
	}

	profileID := l.selectedProfileID
	if profileID == uuid.Nil && len(l.profiles) > 0 {
		profileID = l.profiles[0].ID
	}
	if profileID == uuid.Nil {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.lobby.error_select_profile", "Please select a profile first.")))
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
	if l.state == nil || l.state.Core == nil {
		return
	}
	shared := l.state.Core.GetSharedLobby()
	if shared.SessionID != "" {
		go func(sess, hostKey string) {
			if hostKey != "" {
				_ = l.state.Core.Rest.DeleteSharedLobby(sess, hostKey)
			} else if l.state.Core.DiscordService != nil {
				if user, ok := l.state.Core.DiscordService.UserInfo(); ok && user != nil {
					_ = l.state.Core.Rest.RemoveLobbyMember(sess, user.Id())
				}
			}
		}(shared.SessionID, shared.HostKey)
	}
	l.state.Core.InvalidateCachedLobbyShareAsync()
	if l.state.Core.DiscordService != nil {
		l.state.Core.DiscordService.LeaveLobby(func(err error) {
			if err != nil {
				slog.Warn("Failed to leave lobby", "error", err)
			}
		})
	}
	l.refreshLobbyShareUI()
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

func (l *Launcher) handleShareLobby() {
	if l.state == nil || l.state.Core == nil {
		return
	}
	profileID := l.selectedProfileID
	if profileID == uuid.Nil && len(l.profiles) > 0 {
		profileID = l.profiles[0].ID
	}
	if profileID == uuid.Nil {
		l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.lobby.error_select_profile", "Please select a profile first.")))
		return
	}

	l.lobbyShareButton.Disable()
	go func() {
		defer fyne.Do(func() {
			l.lobbyShareButton.Enable()
		})
		link, err := l.state.Core.ShareCurrentLobby(profileID)
		if err != nil {
			fyne.Do(func() {
				l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.lobby.error_share_failed", "Failed to share lobby: {{.Error}}", map[string]any{"Error": err.Error()})))
			})
			return
		}
		fyne.Do(func() {
			if l.state.Window != nil {
				l.state.Window.Clipboard().SetContent(link.URL)
			}
			l.refreshLobbyShareUI()
			l.state.ShowInfoDialog(
				lang.LocalizeKey("launcher.lobby.share_success_title", "Lobby URL Copied"),
				lang.LocalizeKey("launcher.lobby.share_success_message", "Lobby URL has been created and copied to clipboard!"),
			)
		})
	}()
}

func (l *Launcher) handleCopyLobbyURL() {
	if l.state == nil || l.state.Core == nil {
		return
	}
	shared := l.state.Core.GetSharedLobby()
	if shared.URL != "" && l.state.Window != nil {
		l.state.Window.Clipboard().SetContent(shared.URL)
		l.state.ShowInfoDialog(
			lang.LocalizeKey("launcher.lobby.copied_title", "Copied"),
			lang.LocalizeKey("launcher.lobby.copied_message", "Lobby URL copied to clipboard!"),
		)
	}
}

func (l *Launcher) handleStopSharingLobby() {
	if l.state == nil || l.state.Core == nil {
		return
	}
	l.state.Core.InvalidateCachedLobbyShareAsync()
	l.refreshLobbyShareUI()
}

func (l *Launcher) showLinkChannelDialog() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService
	if !ds.IsLoggedIn() {
		l.openDrawerTab("friends")
		return
	}

	loadingDialog := dialog.NewCustom(
		lang.LocalizeKey("launcher.lobby.loading_guilds", "Loading Discord Servers..."),
		lang.LocalizeKey("common.cancel", "Cancel"),
		widget.NewProgressBarInfinite(),
		l.state.Window,
	)
	loadingDialog.Show()

	ds.GetUserGuilds(func(err error, guilds []discord.GuildInfo) {
		fyne.Do(func() {
			loadingDialog.Hide()
			if err != nil {
				l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.lobby.error_fetch_servers_failed", "Failed to fetch Discord servers: {{.Error}}", map[string]any{"Error": err.Error()})))
				return
			}
			if len(guilds) == 0 {
				l.state.ShowInfoDialog(
					lang.LocalizeKey("launcher.lobby.no_guilds_title", "No Servers Found"),
					lang.LocalizeKey("launcher.lobby.no_guilds_message", "No Discord servers available for linking."),
				)
				return
			}

			guildNames := make([]string, len(guilds))
			guildMap := make(map[string]discord.GuildInfo)
			for i, g := range guilds {
				guildNames[i] = g.Name
				guildMap[g.Name] = g
			}

			channelSelect := widget.NewSelect([]string{}, nil)
			channelSelect.Disable()
			var currentChannels []discord.GuildChannelInfo

			guildSelect := widget.NewSelect(guildNames, func(selected string) {
				g, ok := guildMap[selected]
				if !ok {
					return
				}
				loadingText := lang.LocalizeKey("launcher.lobby.loading_channels", "Loading channels...")
				channelSelect.Options = []string{loadingText}
				channelSelect.SetSelected(loadingText)
				channelSelect.Disable()

				ds.GetGuildChannels(g.ID, func(chErr error, channels []discord.GuildChannelInfo) {
					fyne.Do(func() {
						if chErr != nil {
							channelSelect.Options = []string{lang.LocalizeKey("launcher.lobby.failed_load_channels", "Failed to load channels")}
							channelSelect.Refresh()
							return
						}
						currentChannels = nil
						var linkableNames []string
						for _, ch := range channels {
							if ch.IsLinkable {
								currentChannels = append(currentChannels, ch)
								linkableNames = append(linkableNames, "# "+ch.Name)
							}
						}
						if len(linkableNames) == 0 {
							noChannelsText := lang.LocalizeKey("launcher.lobby.no_linkable_channels", "No linkable text channels")
							channelSelect.Options = []string{noChannelsText}
							channelSelect.SetSelected(noChannelsText)
							channelSelect.Disable()
						} else {
							channelSelect.Options = linkableNames
							channelSelect.SetSelected(linkableNames[0])
							channelSelect.Enable()
						}
						channelSelect.Refresh()
					})
				})
			})

			guildSelect.SetSelected(guildNames[0])

			form := container.NewVBox(
				widget.NewLabelWithStyle(lang.LocalizeKey("launcher.lobby.select_guild", "Select Discord Server:"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				guildSelect,
				widget.NewLabelWithStyle(lang.LocalizeKey("launcher.lobby.select_channel", "Select Text Channel:"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				channelSelect,
			)

			d := dialog.NewCustomConfirm(
				lang.LocalizeKey("launcher.lobby.link_channel_title", "Link Discord Channel"),
				lang.LocalizeKey("launcher.lobby.link_button", "Link"),
				lang.LocalizeKey("common.cancel", "Cancel"),
				form,
				func(confirm bool) {
					if !confirm || channelSelect.Selected == "" {
						return
					}
					var selectedChID uint64
					for _, ch := range currentChannels {
						if "# "+ch.Name == channelSelect.Selected {
							selectedChID = ch.ID
							break
						}
					}
					if selectedChID == 0 {
						return
					}
					ds.LinkChannelToLobby(selectedChID, func(linkErr error) {
						fyne.Do(func() {
							if linkErr != nil {
								l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.lobby.error_link_channel_failed", "Failed to link channel: {{.Error}}", map[string]any{"Error": linkErr.Error()})))
							} else {
								l.state.ShowInfoDialog(
									lang.LocalizeKey("launcher.lobby.link_success_title", "Channel Linked"),
									lang.LocalizeKey("launcher.lobby.link_success_message", "Discord channel successfully linked to lobby!"),
								)
								if info, ok := ds.GetActiveLobby(); ok {
									l.refreshLobbyUI(info)
								}
							}
						})
					})
				},
				l.state.Window,
			)
			d.Resize(fyne.NewSize(380, 220))
			d.Show()
		})
	})
}

func (l *Launcher) handleUnlinkChannel() {
	if l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService
	ds.UnlinkChannelFromLobby(func(err error) {
		fyne.Do(func() {
			if err != nil {
				l.state.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.lobby.error_unlink_channel_failed", "Failed to unlink channel: {{.Error}}", map[string]any{"Error": err.Error()})))
			} else {
				if info, ok := ds.GetActiveLobby(); ok {
					l.refreshLobbyUI(info)
				}
			}
		})
	})
}

// ---------------- Friend List in Drawer ----------------

func (l *Launcher) refreshDrawerFriendsList(query string) {
	if l.drawerFriendsListContainer == nil || l.state == nil || l.state.Core == nil || l.state.Core.DiscordService == nil {
		return
	}
	ds := l.state.Core.DiscordService
	if !ds.IsLoggedIn() {
		l.drawerFriendsListContainer.Objects = []fyne.CanvasObject{
			widget.NewLabel(lang.LocalizeKey("launcher.discord_friends.not_logged_in", "Please log in to Discord via Settings.")),
		}
		l.drawerFriendsListContainer.Refresh()
		return
	}

	l.drawerFriendsLoading.Show()
	go func() {
		userHandles, err := ds.GetFriends()
		if err != nil {
			fyne.Do(func() {
				l.drawerFriendsLoading.Hide()
			})
			return
		}
		var filtered []discordFriend
		queryLower := strings.ToLower(strings.TrimSpace(query))

		for _, u := range userHandles {
			name := strings.TrimSpace(u.DisplayName())
			if name == "" {
				name = strings.TrimSpace(u.Username())
			}
			if queryLower != "" && !strings.Contains(strings.ToLower(name), queryLower) {
				continue
			}
			avatarURL := u.AvatarUrl(discordsdk.UserHandleAvatarTypeGif, discordsdk.UserHandleAvatarTypePng)
			status := u.Status()
			df := discordFriend{
				id:        u.Id(),
				name:      name,
				avatarURL: avatarURL,
				status:    status,
			}
			filtered = append(filtered, df)
		}

		fyne.Do(func() {
			l.drawerFriendsLoading.Hide()
			l.drawerFriendsListContainer.Objects = nil

			if len(filtered) == 0 {
				l.drawerFriendsListContainer.Add(widget.NewLabel(lang.LocalizeKey("launcher.discord_friends.no_friends", "No friends found.")))
				l.drawerFriendsListContainer.Refresh()
				return
			}

			for _, f := range filtered {
				card := l.buildDrawerFriendCard(f)
				l.drawerFriendsListContainer.Add(card)
			}
			l.drawerFriendsListContainer.Refresh()
		})
	}()
}

func (l *Launcher) buildDrawerFriendCard(friend discordFriend) fyne.CanvasObject {
	avatarSize := float32(32)
	avatarBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	avatarBg.CornerRadius = 6
	avatarBg.SetMinSize(fyne.NewSquareSize(avatarSize))

	avatar := canvas.NewImageFromImage(placeholderProfileIcon(int(avatarSize)))
	avatar.CornerRadius = 6
	avatar.SetMinSize(fyne.NewSquareSize(avatarSize))
	avatar.FillMode = canvas.ImageFillContain

	statusDot := canvas.NewCircle(discordStatusColor(friend.status, friend.playingModOfUs))
	statusDot.StrokeColor = theme.Color(theme.ColorNameBackground)
	statusDot.StrokeWidth = 1.5

	avatarContainer := container.New(&discordFriendAvatarLayout{
		statusSize: 10,
		inset:      1,
	}, avatarBg, avatar, statusDot)

	l.refreshDiscordFriendAvatarCanvas(avatar, friend.id, int(avatarSize))
	l.ensureDiscordFriendAvatarLoaded(friend.id, friend.avatarURL, func() {
		l.refreshDiscordFriendAvatarCanvas(avatar, friend.id, int(avatarSize))
	})

	nameLabel := widget.NewLabelWithStyle(friend.name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	nameLabel.Wrapping = fyne.TextTruncate

	inviteBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		if l.state != nil && l.state.Core != nil && l.state.Core.DiscordService != nil {
			shared := l.state.Core.GetSharedLobby()
			url := shared.URL
			if url == "" {
				url = l.state.Core.GetSharedRoom().URL
			}
			if url == "" {
				profileID := l.selectedProfileID
				if profileID == uuid.Nil && len(l.profiles) > 0 {
					profileID = l.profiles[0].ID
				}
				if profileID != uuid.Nil {
					link, err := l.state.Core.ShareCurrentLobby(profileID)
					if err == nil && link != nil {
						url = link.URL
						l.refreshLobbyShareUI()
					}
				}
			}
			if url != "" {
				l.state.Core.DiscordService.SendInvite(friend.id, url, func(err error) {
					fyne.Do(func() {
						if err != nil {
							l.state.ShowErrorDialog(err)
						} else {
							l.state.ShowInfoDialog(
								lang.LocalizeKey("launcher.discord_friends.invite_sent_title", "Invite Sent"),
								lang.LocalizeKey("launcher.discord_friends.invite_sent_message", "Sent invite to {{.Name}}.", map[string]any{"Name": friend.name}),
							)
						}
					})
				})
			} else {
				l.state.ShowInfoDialog(
					lang.LocalizeKey("launcher.discord_friends.invite_unavailable_title", "Invite Unavailable"),
					lang.LocalizeKey("launcher.discord_friends.invite_unavailable_message", "Please share a lobby URL first."),
				)
			}
		}
	})
	inviteBtn.Importance = widget.LowImportance

	content := container.NewBorder(
		nil, nil,
		avatarContainer,
		inviteBtn,
		container.NewVBox(nameLabel),
	)

	cardBg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	cardBg.CornerRadius = 4

	return container.NewStack(cardBg, container.NewPadded(content))
}
