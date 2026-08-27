package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"

	"github.com/ikafly144/au_mod_installer/server/model"
	gormrepo "github.com/ikafly144/au_mod_installer/server/repository/gorm"
)

type DiscordConfig struct {
	Token               string
	GuildID             string
	ModRoleID           string
	ReviewChannelID     string
	ShowcaseForumID     string
	UpdatesChannelID    string
	ReportChannelID     string
	AuditLogChannelID   string
}

type DiscordBotService struct {
	client     *bot.Client
	cfg        DiscordConfig
	subService *SubmissionService
	modService *ModService
	httpClient *http.Client
	router     *handler.Mux
}

func NewDiscordBotService(cfg DiscordConfig, subService *SubmissionService, modService *ModService) *DiscordBotService {
	s := &DiscordBotService{
		cfg:        cfg,
		subService: subService,
		modService: modService,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		router:     handler.New(),
	}
	s.setupRoutes()
	return s
}

func (s *DiscordBotService) setupRoutes() {
	r := s.router

	// Middleware: Request logging
	r.Use(func(next handler.Handler) handler.Handler {
		return func(e *handler.InteractionEvent) error {
			slog.Debug("Discord interaction received", "type", e.Type(), "user", e.User().Username)
			return next(e)
		}
	})

	// Error Handler
	r.Error(func(e *handler.InteractionEvent, err error) {
		slog.Error("Error handling Discord interaction", "error", err)
		_, _ = e.CreateFollowupMessage(discord.MessageCreate{
			Content: fmt.Sprintf("❌ 処理中にエラーが発生しました: %v", err),
			Flags:   discord.MessageFlagEphemeral,
		})
	})

	// Standard Slash Commands
	r.SlashCommand("/mod/create", s.handleModCreateModal)
	r.SlashCommand("/mod/version-add", s.handleVersionAdd)
	r.SlashCommand("/mod/list", s.handleModList)
	r.SlashCommand("/mod/info", s.handleModInfo)
	r.SlashCommand("/mod/collaborator/add", s.handleCollaboratorAdd)
	r.SlashCommand("/mod/collaborator/remove", s.handleCollaboratorRemove)
	r.SlashCommand("/mod/transfer-owner", s.handleTransferOwner)
	r.SlashCommand("/mod/report", s.handleModReportCommand)

	// Moderation Slash Commands
	r.SlashCommand("/mod/ban", s.handleModBan)
	r.SlashCommand("/mod/unban", s.handleModUnban)
	r.SlashCommand("/mod/unpublish", s.handleModUnpublish)
	r.SlashCommand("/mod/publish", s.handleModPublish)
	r.SlashCommand("/mod/review-queue", s.handleReviewQueue)
	r.SlashCommand("/mod/audit-log", s.handleModAuditLog)

	// Modal Submit Routing
	r.Modal("/mod_create_modal", s.onModCreateModalSubmit)
	r.Modal("/reject_mod_modal/{id}", s.handleRejectModModalSubmit)
	r.Modal("/reject_ver_modal/{id}", s.handleRejectVersionModalSubmit)
	r.Modal("/report_mod_modal/{id}", s.handleReportModModalSubmit)

	// Button Component Routing
	r.ButtonComponent("/approve_mod/{id}", s.handleApproveModButton)
	r.ButtonComponent("/reject_mod/{id}", s.handleRejectModButton)
	r.ButtonComponent("/approve_ver/{id}", s.handleApproveVersionButton)
	r.ButtonComponent("/reject_ver/{id}", s.handleRejectVersionButton)
	r.ButtonComponent("/open_report_modal/{id}", s.handleOpenReportModalButton)
	r.ButtonComponent("/report_ban/{id}", s.handleReportBanButton)
	r.ButtonComponent("/report_resolve/{id}", s.handleReportResolveButton)
	r.ButtonComponent("/report_dismiss/{id}", s.handleReportDismissButton)
}

func (s *DiscordBotService) Start(ctx context.Context) error {
	if s.cfg.Token == "" {
		slog.InfoContext(ctx, "DISCORD_TOKEN not provided; Discord bot is disabled")
		return nil
	}

	client, err := disgo.New(s.cfg.Token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMessages,
				gateway.IntentDirectMessages,
			),
		),
		bot.WithEventListeners(s.router),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize Disgo client: %w", err)
	}

	s.client = client

	if err := client.OpenGateway(ctx); err != nil {
		return fmt.Errorf("failed to open Discord gateway: %w", err)
	}

	slog.InfoContext(ctx, "Discord bot connected successfully with moderation features")

	// Register Slash Commands
	if err := s.registerCommands(ctx); err != nil {
		slog.WarnContext(ctx, "Failed to register Discord slash commands", "error", err)
	}

	return nil
}

func (s *DiscordBotService) Stop(ctx context.Context) {
	if s.client != nil {
		s.client.Close(ctx)
		slog.InfoContext(ctx, "Discord bot disconnected")
	}
}

func (s *DiscordBotService) registerCommands(ctx context.Context) error {
	commands := []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "mod",
			Description: "Mod of Us: Mod の作成・管理・モデレーションコマンド",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "create",
					Description: "新しい Mod を登録申請します",
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "version-add",
					Description: "Mod の新しいバージョンを登録申請します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "対象の Mod ID",
							Required:    true,
						},
						discord.ApplicationCommandOptionString{
							Name:        "version",
							Description: "バージョン番号 (例: 1.0.0)",
							Required:    true,
						},
						discord.ApplicationCommandOptionAttachment{
							Name:        "file",
							Description: "Mod ファイル (.zip または .dll)",
							Required:    false,
						},
						discord.ApplicationCommandOptionString{
							Name:        "download_url",
							Description: "外部ダウンロード URL (GitHub Release 等)",
							Required:    false,
						},
						discord.ApplicationCommandOptionString{
							Name:        "changelog",
							Description: "更新内容 / Changelog",
							Required:    false,
						},
						discord.ApplicationCommandOptionString{
							Name:        "platform",
							Description: "対象プラットフォーム",
							Required:    false,
							Choices: []discord.ApplicationCommandOptionChoiceString{
								{Name: "Any", Value: "any"},
								{Name: "x64 (Windows 64bit)", Value: "x64"},
								{Name: "x86 (Windows 32bit)", Value: "x86"},
								{Name: "AArch64", Value: "aarch64"},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "list",
					Description: "自分が管理している Mod 一覧を表示します",
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "info",
					Description: "Mod の詳細情報を確認します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "確認したい Mod ID",
							Required:    true,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "report",
					Description: "規約違反や危険な Mod をモデレーターに通報します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "通報対象の Mod ID",
							Required:    true,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "collaborator",
					Description: "共同開発者の管理",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "add",
							Description: "共同開発者を追加します",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "mod_id",
									Description: "対象の Mod ID",
									Required:    true,
								},
								discord.ApplicationCommandOptionUser{
									Name:        "user",
									Description: "追加する Discord ユーザー",
									Required:    true,
								},
							},
						},
						{
							Name:        "remove",
							Description: "共同開発者を解除します",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:        "mod_id",
									Description: "対象の Mod ID",
									Required:    true,
								},
								discord.ApplicationCommandOptionUser{
									Name:        "user",
									Description: "解除する Discord ユーザー",
									Required:    true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "transfer-owner",
					Description: "Mod のオーナー権限を譲渡します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "対象の Mod ID",
							Required:    true,
						},
						discord.ApplicationCommandOptionUser{
							Name:        "new_owner",
							Description: "新しいオーナーとなる Discord ユーザー",
							Required:    true,
						},
					},
				},
				// Moderation Commands
				discord.ApplicationCommandOptionSubCommand{
					Name:        "ban",
					Description: "[モデレーター専用] Mod を BAN (完全配信停止) します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "BAN する Mod ID",
							Required:    true,
						},
						discord.ApplicationCommandOptionString{
							Name:        "reason",
							Description: "BAN の理由",
							Required:    true,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "unban",
					Description: "[モデレーター専用] Mod の BAN を解除します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "BAN を解除する Mod ID",
							Required:    true,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "unpublish",
					Description: "[モデレーター専用] Mod を一時的に非公開にします",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "非公開にする Mod ID",
							Required:    true,
						},
						discord.ApplicationCommandOptionString{
							Name:        "reason",
							Description: "非公開の理由",
							Required:    true,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "publish",
					Description: "[モデレーター専用] 非公開 Mod を再公開します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "再公開する Mod ID",
							Required:    true,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "review-queue",
					Description: "[モデレーター専用] 審査待ちの Mod / バージョン一覧を表示します",
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "audit-log",
					Description: "[モデレーター専用] Mod のモデレーション履歴・監査ログを表示します",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name:        "mod_id",
							Description: "確認したい Mod ID",
							Required:    true,
						},
					},
				},
			},
		},
	}

	if s.cfg.GuildID != "" {
		guildID, err := snowflake.Parse(s.cfg.GuildID)
		if err != nil {
			return fmt.Errorf("invalid DISCORD_GUILD_ID: %w", err)
		}
		_, err = s.client.Rest.SetGuildCommands(s.client.ApplicationID, guildID, commands)
		return err
	}

	_, err := s.client.Rest.SetGlobalCommands(s.client.ApplicationID, commands)
	return err
}

func (s *DiscordBotService) isModerator(member *discord.ResolvedMember, user discord.User) bool {
	if member != nil {
		perms := member.Permissions
		if perms.Has(discord.PermissionAdministrator) || perms.Has(discord.PermissionManageGuild) || perms.Has(discord.PermissionModerateMembers) {
			return true
		}
		if s.cfg.ModRoleID != "" {
			for _, roleID := range member.RoleIDs {
				if roleID.String() == s.cfg.ModRoleID {
					return true
				}
			}
		}
	}
	return false
}

func (s *DiscordBotService) handleModCreateModal(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	modal := discord.NewModalCreate("/mod_create_modal", "新しい Mod の登録申請",
		discord.NewLabel("Mod ID (英数字・ドット・ハイフン)",
			discord.NewShortTextInput("mod_id").
				WithPlaceholder("example.my-mod").
				WithRequired(true).
				WithMinLength(3).
				WithMaxLength(64),
		),
		discord.NewLabel("Mod 名 (表示名)",
			discord.NewShortTextInput("name").
				WithPlaceholder("My Awesome Mod").
				WithRequired(true).
				WithMinLength(2).
				WithMaxLength(100),
		),
		discord.NewLabel("作者名 (クレジット)",
			discord.NewShortTextInput("author").
				WithPlaceholder(e.User().Username).
				WithValue(e.User().Username).
				WithRequired(true).
				WithMaxLength(100),
		),
		discord.NewLabel("Mod の説明",
			discord.NewParagraphTextInput("description").
				WithPlaceholder("この Mod の機能や特徴を入力してください").
				WithRequired(true).
				WithMaxLength(1000),
		),
		discord.NewLabel("サムネイル画像 URL (省略可)",
			discord.NewShortTextInput("thumbnail_url").
				WithPlaceholder("https://example.com/icon.png").
				WithRequired(false),
		),
	)

	return e.Modal(modal)
}

func (s *DiscordBotService) handleVersionAdd(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)

	modID := data.String("mod_id")
	version := data.String("version")
	changelog, _ := data.OptString("changelog")
	downloadURL, _ := data.OptString("download_url")
	platformStr, ok := data.OptString("platform")
	if !ok || platformStr == "" {
		platformStr = "any"
	}

	var fileData []byte
	var filename string

	if attachment, ok := data.OptAttachment("file"); ok {
		filename = attachment.Filename
		resp, err := s.httpClient.Get(attachment.URL)
		if err != nil {
			_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
				Content: ptr(fmt.Sprintf("❌ 添付ファイルのダウンロードに失敗しました: %v", err)),
			})
			return nil
		}
		defer resp.Body.Close()
		fileData, err = io.ReadAll(resp.Body)
		if err != nil {
			_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
				Content: ptr(fmt.Sprintf("❌ 添付ファイルの読み込みに失敗しました: %v", err)),
			})
			return nil
		}
	}

	if len(fileData) == 0 && downloadURL == "" {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr("❌ ファイルの添付または `download_url` の指定のどちらかが必要です。"),
		})
		return nil
	}

	ctx := context.Background()
	req := SubmitVersionRequest{
		ModID:          modID,
		VersionID:      version,
		Changelog:      changelog,
		SubmitterID:    e.User().ID.String(),
		TargetPlatform: model.TargetPlatform(platformStr),
		Filename:       filename,
		FileData:       fileData,
		ExternalURL:    downloadURL,
	}

	verSub, inspection, err := s.subService.SubmitVersion(ctx, req)
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ バージョン申請エラー: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("✅ Mod `%s` バージョン `%s` の申請を受け付けました！モデレーターによる審査が行われます。", modID, version)),
	})

	s.postVersionReviewEmbed(verSub, inspection, e.User())
	s.postAuditLog(model.AuditActionVersionSubmitted, modID, e.User().Username, fmt.Sprintf("v%s submitted", version), nil)
	return nil
}

func (s *DiscordBotService) handleModList(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	userID := e.User().ID.String()

	repo, ok := s.modService.repo.(*gormrepo.GormRepository)
	if !ok {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr("❌ データベースリポジトリの取得に失敗しました。"),
		})
		return nil
	}

	mods, err := repo.GetModsByOwner(userID)
	if err != nil || len(mods) == 0 {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr("📦 あなたがオーナーの Mod は現在ありません。`/mod create` で新しい Mod を登録できます！"),
		})
		return nil
	}

	var sb strings.Builder
	sb.WriteString("📦 **あなたが管理している Mod 一覧:**\n\n")
	for _, m := range mods {
		latest := m.LatestVersionExternal
		if latest == "" {
			latest = "未リリース"
		}
		statusIcon := "🟢"
		if m.Status == model.ModStatusBanned {
			statusIcon = "🚫 (BAN)"
		} else if m.Status == model.ModStatusUnpublished {
			statusIcon = "🟡 (非公開)"
		}
		sb.WriteString(fmt.Sprintf("%s **%s** (`%s`) - 最新版: `%s`\n", statusIcon, m.Name, m.ID, latest))
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(sb.String()),
	})
	return nil
}

func (s *DiscordBotService) handleModInfo(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(false)
	modID := data.String("mod_id")

	mod, err := s.modService.GetModDetails(modID)
	if err != nil || mod == nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ Mod `%s` が見つかりませんでした。", modID)),
		})
		return nil
	}

	statusText := "公開中"
	color := 0x5865F2
	if mod.Status == model.ModStatusBanned {
		statusText = "🚫 BAN済み"
		color = 0xED4245
	} else if mod.Status == model.ModStatusUnpublished {
		statusText = "🟡 非公開中"
		color = 0xFEE75C
	}

	embed := discord.NewEmbed().
		WithTitle(mod.Name).
		WithDescription(mod.Description).
		AddField("Mod ID", fmt.Sprintf("`%s`", mod.ID), true).
		AddField("作者", mod.Author, true).
		AddField("最新バージョン", fmt.Sprintf("`%s`", mod.LatestVersionExternal), true).
		AddField("状態", statusText, true).
		WithColor(color).
		WithTimestamp(time.Now())

	if mod.ThumbnailURI != nil && *mod.ThumbnailURI != "" {
		embed = embed.WithThumbnail(*mod.ThumbnailURI)
	}

	actionRow := discord.NewActionRow(
		discord.NewDangerButton("🚨 通報する (Report)", fmt.Sprintf("/open_report_modal/%s", mod.ID)),
	)

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{embed},
		Components: &[]discord.LayoutComponent{actionRow},
	})
	return nil
}

func (s *DiscordBotService) handleCollaboratorAdd(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := data.String("mod_id")
	user := data.User("user")

	ctx := context.Background()
	err := s.subService.AddCollaborator(ctx, modID, e.User().ID.String(), user.ID.String())
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 共同開発者の追加に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("✅ %s を Mod `%s` の共同開発者に追加しました！", user.Mention(), modID)),
	})
	s.postAuditLog(model.AuditActionCollaboratorAdded, modID, e.User().Username, fmt.Sprintf("Added %s (%s)", user.Username, user.ID), nil)
	return nil
}

func (s *DiscordBotService) handleCollaboratorRemove(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := data.String("mod_id")
	user := data.User("user")

	ctx := context.Background()
	err := s.subService.RemoveCollaborator(ctx, modID, e.User().ID.String(), user.ID.String())
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 共同開発者の解除に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("✅ %s を Mod `%s` の共同開発者から解除しました。", user.Mention(), modID)),
	})
	s.postAuditLog(model.AuditActionCollaboratorRemoved, modID, e.User().Username, fmt.Sprintf("Removed %s (%s)", user.Username, user.ID), nil)
	return nil
}

func (s *DiscordBotService) handleTransferOwner(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := data.String("mod_id")
	newOwner := data.User("new_owner")

	ctx := context.Background()
	err := s.subService.TransferOwnership(ctx, modID, e.User().ID.String(), newOwner.ID.String())
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ オーナー権限の譲渡に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("👑 Mod `%s` のオーナー権限を %s に譲渡しました！", modID, newOwner.Mention())),
	})
	s.postAuditLog(model.AuditActionOwnershipTransferred, modID, e.User().Username, fmt.Sprintf("Transferred to %s (%s)", newOwner.Username, newOwner.ID), nil)
	return nil
}

// Moderation Commands

func (s *DiscordBotService) handleModBan(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	if member, ok := data.OptMember("user"); ok && !s.isModerator(&member, e.User()) {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr("❌ この操作を実行するモデレーター権限がありません。"),
		})
		return nil
	}

	modID := data.String("mod_id")
	reason := data.String("reason")

	ctx := context.Background()
	mod, err := s.modService.GetModDetails(modID)
	if err != nil || mod == nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ Mod `%s` が見つかりませんでした。", modID)),
		})
		return nil
	}

	if err := s.subService.BanMod(ctx, modID, e.User().ID.String(), reason); err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ BAN処理に失敗しました: %v", err)),
		})
		return nil
	}

	// Lock Forum Thread if exists
	if mod.DiscordThreadID != "" {
		if threadID, err := snowflake.Parse(mod.DiscordThreadID); err == nil {
			_, _ = s.client.Rest.CreateMessage(threadID, discord.MessageCreate{
				Content: fmt.Sprintf("🚨 **この Mod はモデレーターにより利用規約違反のため BAN されました。**\n理由: %s", reason),
			})
		}
	}

	// DM Author
	if mod.OwnerDiscordID != "" {
		if authorID, err := snowflake.Parse(mod.OwnerDiscordID); err == nil {
			if dmCh, err := s.client.Rest.CreateDMChannel(authorID); err == nil && dmCh != nil {
				_, _ = s.client.Rest.CreateMessage(dmCh.ID(), discord.MessageCreate{
					Content: fmt.Sprintf("🚨 あなたの Mod **%s** (`%s`) は以下の理由により BAN（配信停止）されました:\n\n> %s", mod.Name, mod.ID, reason),
				})
			}
		}
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("🚫 Mod **%s** (`%s`) を BAN しました。API から即座に非公開化されました。", mod.Name, modID)),
	})

	s.postAuditLog(model.AuditActionModBanned, modID, e.User().Username, reason, nil)
	return nil
}

func (s *DiscordBotService) handleModUnban(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := data.String("mod_id")

	ctx := context.Background()
	if err := s.subService.UnbanMod(ctx, modID, e.User().ID.String()); err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ BAN解除に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("✅ Mod `%s` の BAN を解除し、公開状態に戻しました。", modID)),
	})

	s.postAuditLog(model.AuditActionModUnbanned, modID, e.User().Username, "Unbanned by moderator", nil)
	return nil
}

func (s *DiscordBotService) handleModUnpublish(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := data.String("mod_id")
	reason := data.String("reason")

	ctx := context.Background()
	if err := s.subService.UnpublishMod(ctx, modID, e.User().ID.String(), reason); err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 非公開処理に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("🟡 Mod `%s` を非公開に設定しました。", modID)),
	})

	s.postAuditLog(model.AuditActionModUnpublished, modID, e.User().Username, reason, nil)
	return nil
}

func (s *DiscordBotService) handleModPublish(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := data.String("mod_id")

	ctx := context.Background()
	if err := s.subService.PublishMod(ctx, modID, e.User().ID.String()); err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 再公開処理に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("✅ Mod `%s` を再公開しました。", modID)),
	})

	s.postAuditLog(model.AuditActionModPublished, modID, e.User().Username, "Republished by moderator", nil)
	return nil
}

func (s *DiscordBotService) handleReviewQueue(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	ctx := context.Background()
	modSubs, verSubs, err := s.subService.GetPendingSubmissions(ctx)
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 審査キューの取得に失敗しました: %v", err)),
		})
		return nil
	}

	var sb strings.Builder
	sb.WriteString("📋 **現在の審査キュー一覧:**\n\n")

	sb.WriteString(fmt.Sprintf("**新規 Mod 申請 (%d 件):**\n", len(modSubs)))
	if len(modSubs) == 0 {
		sb.WriteString("（審査待ちの新規 Mod はありません）\n")
	} else {
		for _, sub := range modSubs {
			sb.WriteString(fmt.Sprintf("• **%s** (`%s`) - 申請者: <@%s>\n", sub.Name, sub.ModID, sub.SubmitterID))
		}
	}

	sb.WriteString(fmt.Sprintf("\n**新バージョン申請 (%d 件):**\n", len(verSubs)))
	if len(verSubs) == 0 {
		sb.WriteString("（審査待ちの新バージョンはありません）\n")
	} else {
		for _, sub := range verSubs {
			sb.WriteString(fmt.Sprintf("• Mod: `%s` (v%s) - 申請者: <@%s>\n", sub.ModID, sub.VersionID, sub.SubmitterID))
		}
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(sb.String()),
	})
	return nil
}

func (s *DiscordBotService) handleModAuditLog(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := data.String("mod_id")

	ctx := context.Background()
	logs, err := s.subService.GetAuditLogs(ctx, modID, 15)
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 監査ログの取得に失敗しました: %v", err)),
		})
		return nil
	}

	if len(logs) == 0 {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("📜 Mod `%s` の監査ログはありません。", modID)),
		})
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📜 **Mod `%s` の監査ログ履歴 (最新15件):**\n\n", modID))
	for _, l := range logs {
		sb.WriteString(fmt.Sprintf("• `[%s]` **%s** by <@%s> - %s\n", l.CreatedAt.Format("2006-01-02 15:04"), l.Action, l.ActorID, l.Reason))
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(sb.String()),
	})
	return nil
}

// User Report Handlers

func (s *DiscordBotService) handleModReportCommand(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	modID := data.String("mod_id")
	return s.showReportModal(e, modID)
}

func (s *DiscordBotService) handleOpenReportModalButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	modID := e.Vars["id"]
	modal := discord.NewModalCreate(fmt.Sprintf("/report_mod_modal/%s", modID), "Mod の通報",
		discord.NewLabel("通報理由・カテゴリ (malware/crash/nsfw/copyright/other)",
			discord.NewShortTextInput("category").
				WithPlaceholder("malware / crash / copyright / other").
				WithRequired(true).
				WithMaxLength(32),
		),
		discord.NewLabel("詳細な理由・問題の説明",
			discord.NewParagraphTextInput("reason").
				WithPlaceholder("発生した問題や不正な挙動の詳細を記述してください").
				WithRequired(true).
				WithMaxLength(500),
		),
	)
	return e.Modal(modal)
}

func (s *DiscordBotService) showReportModal(e *handler.CommandEvent, modID string) error {
	modal := discord.NewModalCreate(fmt.Sprintf("/report_mod_modal/%s", modID), "Mod の通報",
		discord.NewLabel("通報カテゴリ (malware/crash/nsfw/copyright/other)",
			discord.NewShortTextInput("category").
				WithPlaceholder("malware / crash / copyright / other").
				WithRequired(true).
				WithMaxLength(32),
		),
		discord.NewLabel("詳細な理由・問題の説明",
			discord.NewParagraphTextInput("reason").
				WithPlaceholder("発生した問題や不正な挙動の詳細を記述してください").
				WithRequired(true).
				WithMaxLength(500),
		),
	)
	return e.Modal(modal)
}

func (s *DiscordBotService) handleReportModModalSubmit(e *handler.ModalEvent) error {
	_ = e.DeferCreateMessage(true)
	modID := e.Vars["id"]
	categoryStr := strings.ToLower(strings.TrimSpace(e.Data.Text("category")))
	reason := e.Data.Text("reason")

	category := model.ReportCategoryOther
	switch categoryStr {
	case "malware", "virus":
		category = model.ReportCategoryMalware
	case "crash", "bug":
		category = model.ReportCategoryCrash
	case "nsfw":
		category = model.ReportCategoryNSFW
	case "copyright":
		category = model.ReportCategoryCopyright
	case "spam":
		category = model.ReportCategorySpam
	}

	ctx := context.Background()
	report, err := s.subService.CreateReport(ctx, modID, e.User().ID.String(), category, reason)
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 通報の送信に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr("✅ 通報を受け付けました。モデレーターチームが確認・調査を行います。ご協力ありがとうございます。"),
	})

	s.postReportTicketEmbed(report, e.User())
	s.postAuditLog(model.AuditActionModReported, modID, e.User().Username, fmt.Sprintf("Reported: %s", reason), nil)
	return nil
}

func (s *DiscordBotService) postReportTicketEmbed(report *model.ModReport, reporter discord.User) {
	if s.cfg.ReportChannelID == "" || s.client == nil {
		return
	}
	chID, err := snowflake.Parse(s.cfg.ReportChannelID)
	if err != nil {
		return
	}

	embed := discord.NewEmbed().
		WithTitle("🚨 [Mod 通報チケット] " + report.ModID).
		AddField("通報対象 Mod", fmt.Sprintf("`%s`", report.ModID), true).
		AddField("カテゴリ", string(report.Category), true).
		AddField("通報者", reporter.Mention(), true).
		AddField("通報理由", report.Reason, false).
		WithColor(0xED4245).
		WithTimestamp(time.Now())

	actionRow := discord.NewActionRow(
		discord.NewDangerButton("🚫 即時 BAN", fmt.Sprintf("/report_ban/%s", report.ID)),
		discord.NewSuccessButton("✅ 解決済みにする", fmt.Sprintf("/report_resolve/%s", report.ID)),
		discord.NewSecondaryButton("❌ 却下 / 誤報", fmt.Sprintf("/report_dismiss/%s", report.ID)),
	)

	_, _ = s.client.Rest.CreateMessage(chID, discord.MessageCreate{
		Embeds:     []discord.Embed{embed},
		Components: []discord.LayoutComponent{actionRow},
	})
}

func (s *DiscordBotService) handleReportBanButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	reportID := e.Vars["id"]
	_ = e.DeferUpdateMessage()

	ctx := context.Background()
	report, err := s.subService.modrRepo.GetReport(reportID)
	if err != nil || report == nil {
		return nil
	}

	_ = s.subService.BanMod(ctx, report.ModID, e.User().ID.String(), "Banned via user report: "+report.Reason)
	_, _ = s.subService.ResolveReport(ctx, reportID, e.User().ID.String(), "Mod banned")

	resolvedEmbed := discord.NewEmbed().
		WithTitle(fmt.Sprintf("🚫 [通報対応完了: BAN] %s", report.ModID)).
		AddField("対象 Mod", fmt.Sprintf("`%s`", report.ModID), true).
		AddField("対応者", e.User().Mention(), true).
		AddField("結果", "Mod を BAN しました", false).
		WithColor(0xED4245).
		WithTimestamp(time.Now())

	_ = e.UpdateMessage(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{resolvedEmbed},
		Components: &[]discord.LayoutComponent{},
	})
	return nil
}

func (s *DiscordBotService) handleReportResolveButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	reportID := e.Vars["id"]
	_ = e.DeferUpdateMessage()

	ctx := context.Background()
	report, err := s.subService.ResolveReport(ctx, reportID, e.User().ID.String(), "Resolved by moderator")
	if err != nil {
		return nil
	}

	resolvedEmbed := discord.NewEmbed().
		WithTitle(fmt.Sprintf("✅ [通報対応完了: 解決] %s", report.ModID)).
		AddField("対象 Mod", fmt.Sprintf("`%s`", report.ModID), true).
		AddField("対応者", e.User().Mention(), true).
		AddField("結果", "解決済みに設定", false).
		WithColor(0x57F287).
		WithTimestamp(time.Now())

	_ = e.UpdateMessage(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{resolvedEmbed},
		Components: &[]discord.LayoutComponent{},
	})
	return nil
}

func (s *DiscordBotService) handleReportDismissButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	reportID := e.Vars["id"]
	_ = e.DeferUpdateMessage()

	ctx := context.Background()
	report, err := s.subService.DismissReport(ctx, reportID, e.User().ID.String())
	if err != nil {
		return nil
	}

	dismissedEmbed := discord.NewEmbed().
		WithTitle(fmt.Sprintf("❌ [通報対応完了: 却下] %s", report.ModID)).
		AddField("対象 Mod", fmt.Sprintf("`%s`", report.ModID), true).
		AddField("対応者", e.User().Mention(), true).
		AddField("結果", "却下 / 誤報としてクローズ", false).
		WithColor(0x95A5A6).
		WithTimestamp(time.Now())

	_ = e.UpdateMessage(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{dismissedEmbed},
		Components: &[]discord.LayoutComponent{},
	})
	return nil
}

func (s *DiscordBotService) postAuditLog(action model.ModAuditAction, modID, actor, reason string, details model.StringMap) {
	if s.cfg.AuditLogChannelID == "" || s.client == nil {
		return
	}
	chID, err := snowflake.Parse(s.cfg.AuditLogChannelID)
	if err != nil {
		return
	}

	color := 0x5865F2
	if strings.Contains(string(action), "BAN") || strings.Contains(string(action), "REJECT") {
		color = 0xED4245
	} else if strings.Contains(string(action), "APPROVE") {
		color = 0x57F287
	}

	embed := discord.NewEmbed().
		WithTitle(fmt.Sprintf("📜 [監査ログ] %s", action)).
		AddField("Mod ID", fmt.Sprintf("`%s`", modID), true).
		AddField("実行者", actor, true).
		AddField("内容/理由", reason, false).
		WithColor(color).
		WithTimestamp(time.Now())

	_, _ = s.client.Rest.CreateMessage(chID, discord.MessageCreate{
		Embeds: []discord.Embed{embed},
	})
}

func (s *DiscordBotService) onModCreateModalSubmit(e *handler.ModalEvent) error {
	_ = e.DeferCreateMessage(true)
	data := e.Data
	modID := data.Text("mod_id")
	name := data.Text("name")
	author := data.Text("author")
	description := data.Text("description")
	thumbnailURL := data.Text("thumbnail_url")

	ctx := context.Background()
	sub, err := s.subService.SubmitMod(ctx, SubmitModRequest{
		ModID:        modID,
		Name:         name,
		Description:  description,
		AuthorName:   author,
		SubmitterID:  e.User().ID.String(),
		ThumbnailURL: thumbnailURL,
	})
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ Mod 登録申請に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("🎉 Mod **%s** (`%s`) の作成申請を提出しました！モデレーターによる承認をお待ちください。", name, modID)),
	})

	s.postModReviewEmbed(sub, e.User())
	s.postAuditLog(model.AuditActionModSubmitted, modID, e.User().Username, "Mod created and submitted for review", nil)
	return nil
}

func (s *DiscordBotService) handleApproveModButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	subID := e.Vars["id"]
	_ = e.DeferUpdateMessage()

	ctx := context.Background()
	mod, err := s.subService.ApproveModSubmission(ctx, subID, e.User().ID.String())
	if err != nil {
		_, _ = e.CreateFollowupMessage(discord.MessageCreate{
			Content: fmt.Sprintf("❌ 承認処理に失敗しました: %v", err),
			Flags:   discord.MessageFlagEphemeral,
		})
		return nil
	}

	// Update Moderator Embed
	approvedEmbed := discord.NewEmbed().
		WithTitle("✅ [承認完了] " + mod.Name).
		WithDescription(mod.Description).
		AddField("Mod ID", fmt.Sprintf("`%s`", mod.ID), true).
		AddField("承認者", e.User().Mention(), true).
		WithColor(0x57F287).
		WithTimestamp(time.Now())

	_ = e.UpdateMessage(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{approvedEmbed},
		Components: &[]discord.LayoutComponent{},
	})

	// Create Forum Thread if Forum Channel is configured
	if s.cfg.ShowcaseForumID != "" {
		forumID, err := snowflake.Parse(s.cfg.ShowcaseForumID)
		if err == nil {
			forumEmbed := discord.NewEmbed().
				WithTitle(mod.Name).
				WithDescription(mod.Description).
				AddField("Mod ID", fmt.Sprintf("`%s`", mod.ID), true).
				AddField("作者", mod.Author, true).
				WithColor(0x5865F2)

			post, err := s.client.Rest.CreatePostInThreadChannel(forumID, discord.ThreadChannelPostCreate{
				Name: mod.Name,
				Message: discord.MessageCreate{
					Embeds: []discord.Embed{forumEmbed},
				},
			})
			if err == nil && post != nil {
				_ = s.subService.SetDiscordThreadID(ctx, mod.ID, post.ID().String())
			}
		}
	}

	// DM Submitter
	if sub, err := s.subService.GetModSubmission(subID); err == nil {
		if submitterSnowflake, err := snowflake.Parse(sub.SubmitterID); err == nil {
			dmCh, err := s.client.Rest.CreateDMChannel(submitterSnowflake)
			if err == nil && dmCh != nil {
				_, _ = s.client.Rest.CreateMessage(dmCh.ID(), discord.MessageCreate{
					Content: fmt.Sprintf("🎉 おめでとうございます！あなたが申請した Mod **%s** (`%s`) が承認されました！", mod.Name, mod.ID),
				})
			}
		}
	}

	s.postAuditLog(model.AuditActionModApproved, mod.ID, e.User().Username, "Mod approved and published", nil)
	return nil
}

func (s *DiscordBotService) handleRejectModButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	subID := e.Vars["id"]
	modal := discord.NewModalCreate(fmt.Sprintf("/reject_mod_modal/%s", subID), "Mod 申請の却下理由",
		discord.NewLabel("却下理由",
			discord.NewParagraphTextInput("reason").
				WithPlaceholder("修正が必要な点や却下理由を入力してください").
				WithRequired(true).
				WithMaxLength(500),
		),
	)
	return e.Modal(modal)
}

func (s *DiscordBotService) handleRejectModModalSubmit(e *handler.ModalEvent) error {
	_ = e.DeferCreateMessage(true)
	subID := e.Vars["id"]
	reason := e.Data.Text("reason")

	ctx := context.Background()
	sub, err := s.subService.RejectModSubmission(ctx, subID, e.User().ID.String(), reason)
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ 却下処理に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr("✅ 申請を却下し、申請者に理由を送信しました。"),
	})

	// DM Submitter
	if submitterSnowflake, err := snowflake.Parse(sub.SubmitterID); err == nil {
		dmCh, err := s.client.Rest.CreateDMChannel(submitterSnowflake)
		if err == nil && dmCh != nil {
			_, _ = s.client.Rest.CreateMessage(dmCh.ID(), discord.MessageCreate{
				Content: fmt.Sprintf("⚠️ 申し訳ありません。申請された Mod **%s** (`%s`) は以下の理由により却下されました:\n\n> %s", sub.Name, sub.ModID, reason),
			})
		}
	}

	s.postAuditLog(model.AuditActionModRejected, sub.ModID, e.User().Username, reason, nil)
	return nil
}

func (s *DiscordBotService) handleApproveVersionButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	verSubID := e.Vars["id"]
	_ = e.DeferUpdateMessage()

	ctx := context.Background()
	ver, err := s.subService.ApproveVersionSubmission(ctx, verSubID, e.User().ID.String())
	if err != nil {
		_, _ = e.CreateFollowupMessage(discord.MessageCreate{
			Content: fmt.Sprintf("❌ バージョン承認処理に失敗しました: %v", err),
			Flags:   discord.MessageFlagEphemeral,
		})
		return nil
	}

	approvedEmbed := discord.NewEmbed().
		WithTitle(fmt.Sprintf("✅ [バージョン承認・公開] %s (v%s)", ver.ModID, ver.VersionID)).
		AddField("Mod ID", fmt.Sprintf("`%s`", ver.ModID), true).
		AddField("バージョン", fmt.Sprintf("`%s`", ver.VersionID), true).
		AddField("承認者", e.User().Mention(), true).
		WithColor(0x57F287).
		WithTimestamp(time.Now())

	_ = e.UpdateMessage(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{approvedEmbed},
		Components: &[]discord.LayoutComponent{},
	})

	// Post Announcement to Updates Channel
	if s.cfg.UpdatesChannelID != "" {
		if updatesChID, err := snowflake.Parse(s.cfg.UpdatesChannelID); err == nil {
			_, _ = s.client.Rest.CreateMessage(updatesChID, discord.MessageCreate{
				Content: fmt.Sprintf("🎉 **新バージョン公開!** Mod `%s` のバージョン `%s` がリリースされました！", ver.ModID, ver.VersionID),
			})
		}
	}

	// DM Submitter
	if verSub, err := s.subService.GetVersionSubmission(verSubID); err == nil {
		if submitterSnowflake, err := snowflake.Parse(verSub.SubmitterID); err == nil {
			dmCh, err := s.client.Rest.CreateDMChannel(submitterSnowflake)
			if err == nil && dmCh != nil {
				_, _ = s.client.Rest.CreateMessage(dmCh.ID(), discord.MessageCreate{
					Content: fmt.Sprintf("🎉 あなたが提出した Mod `%s` のバージョン `%s` が承認・公開されました！", ver.ModID, ver.VersionID),
				})
			}
		}
	}

	s.postAuditLog(model.AuditActionVersionApproved, ver.ModID, e.User().Username, fmt.Sprintf("v%s approved", ver.VersionID), nil)
	return nil
}

func (s *DiscordBotService) handleRejectVersionButton(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	verSubID := e.Vars["id"]
	modal := discord.NewModalCreate(fmt.Sprintf("/reject_ver_modal/%s", verSubID), "バージョン申請の却下理由",
		discord.NewLabel("却下理由",
			discord.NewParagraphTextInput("reason").
				WithPlaceholder("修正が必要な点や却下理由を入力してください").
				WithRequired(true).
				WithMaxLength(500),
		),
	)
	return e.Modal(modal)
}

func (s *DiscordBotService) handleRejectVersionModalSubmit(e *handler.ModalEvent) error {
	_ = e.DeferCreateMessage(true)
	verSubID := e.Vars["id"]
	reason := e.Data.Text("reason")

	ctx := context.Background()
	verSub, err := s.subService.RejectVersionSubmission(ctx, verSubID, e.User().ID.String(), reason)
	if err != nil {
		_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
			Content: ptr(fmt.Sprintf("❌ バージョン却下処理に失敗しました: %v", err)),
		})
		return nil
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr("✅ バージョン申請を却下し、申請者に理由を送信しました。"),
	})

	// DM Submitter
	if submitterSnowflake, err := snowflake.Parse(verSub.SubmitterID); err == nil {
		dmCh, err := s.client.Rest.CreateDMChannel(submitterSnowflake)
		if err == nil && dmCh != nil {
			_, _ = s.client.Rest.CreateMessage(dmCh.ID(), discord.MessageCreate{
				Content: fmt.Sprintf("⚠️ 申し訳ありません。提出された Mod `%s` (v%s) のバージョン申請は以下の理由により却下されました:\n\n> %s", verSub.ModID, verSub.VersionID, reason),
			})
		}
	}

	s.postAuditLog(model.AuditActionVersionRejected, verSub.ModID, e.User().Username, reason, nil)
	return nil
}

func (s *DiscordBotService) postModReviewEmbed(sub *model.ModSubmission, user discord.User) {
	if s.cfg.ReviewChannelID == "" || s.client == nil {
		return
	}
	chID, err := snowflake.Parse(s.cfg.ReviewChannelID)
	if err != nil {
		return
	}

	embed := discord.NewEmbed().
		WithTitle("📝 [新規 Mod 申請] " + sub.Name).
		WithDescription(sub.Description).
		AddField("Mod ID", fmt.Sprintf("`%s`", sub.ModID), true).
		AddField("作者名", sub.AuthorName, true).
		AddField("申請者", user.Mention(), true).
		WithColor(0xFEE75C).
		WithTimestamp(time.Now())

	if sub.ThumbnailURL != "" {
		embed = embed.WithThumbnail(sub.ThumbnailURL)
	}

	actionRow := discord.NewActionRow(
		discord.NewSuccessButton("承認する (Approve)", fmt.Sprintf("/approve_mod/%s", sub.ID)),
		discord.NewDangerButton("却下する (Reject)", fmt.Sprintf("/reject_mod/%s", sub.ID)),
	)

	_, _ = s.client.Rest.CreateMessage(chID, discord.MessageCreate{
		Embeds:     []discord.Embed{embed},
		Components: []discord.LayoutComponent{actionRow},
	})
}

func (s *DiscordBotService) postVersionReviewEmbed(sub *model.VersionSubmission, insp *InspectionResult, user discord.User) {
	if s.cfg.ReviewChannelID == "" || s.client == nil {
		return
	}
	chID, err := snowflake.Parse(s.cfg.ReviewChannelID)
	if err != nil {
		return
	}

	vtDisplay := "未スキャン (API未設定)"
	if sub.VirusTotalStatus != "unscanned" {
		vtDisplay = fmt.Sprintf("%s (検知数: %d)", sub.VirusTotalStatus, sub.VirusTotalScore)
		if sub.VirusTotalURL != "" {
			vtDisplay = fmt.Sprintf("[%s](%s)", vtDisplay, sub.VirusTotalURL)
		}
	}

	embed := discord.NewEmbed().
		WithTitle(fmt.Sprintf("🚀 [新バージョン申請] %s (v%s)", sub.ModID, sub.VersionID)).
		AddField("Mod ID", fmt.Sprintf("`%s`", sub.ModID), true).
		AddField("バージョン", fmt.Sprintf("`%s`", sub.VersionID), true).
		AddField("プラットフォーム", string(sub.TargetPlatform), true).
		AddField("申請者", user.Mention(), true).
		AddField("VirusTotal", vtDisplay, true).
		AddField("ファイルサイズ", fmt.Sprintf("%.2f KB", float64(sub.FileSize)/1024), true).
		WithColor(0x57F287).
		WithTimestamp(time.Now())

	if sub.Changelog != "" {
		embed = embed.AddField("Changelog", sub.Changelog, false)
	}

	if insp != nil {
		if insp.ContainsDLL {
			embed = embed.AddField("検出 DLL", fmt.Sprintf("`%s`", strings.Join(insp.DLLFiles, ", ")), false)
		}
		if len(insp.Warnings) > 0 {
			embed = embed.AddField("⚠️ 警告", strings.Join(insp.Warnings, "\n"), false)
		}
	}

	actionRow := discord.NewActionRow(
		discord.NewSuccessButton("承認して公開 (Approve)", fmt.Sprintf("/approve_ver/%s", sub.ID)),
		discord.NewDangerButton("却下する (Reject)", fmt.Sprintf("/reject_ver/%s", sub.ID)),
	)

	_, _ = s.client.Rest.CreateMessage(chID, discord.MessageCreate{
		Embeds:     []discord.Embed{embed},
		Components: []discord.LayoutComponent{actionRow},
	})
}

func ptr[T any](v T) *T {
	return &v
}
