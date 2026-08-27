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
	Token            string
	GuildID          string
	ReviewChannelID  string
	ShowcaseForumID  string
	UpdatesChannelID string
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

	// Slash Commands Routing
	r.SlashCommand("/mod/create", s.handleModCreateModal)
	r.SlashCommand("/mod/version-add", s.handleVersionAdd)
	r.SlashCommand("/mod/list", s.handleModList)
	r.SlashCommand("/mod/info", s.handleModInfo)
	r.SlashCommand("/mod/collaborator/add", s.handleCollaboratorAdd)
	r.SlashCommand("/mod/collaborator/remove", s.handleCollaboratorRemove)
	r.SlashCommand("/mod/transfer-owner", s.handleTransferOwner)

	// Modal Submit Routing
	r.Modal("/mod_create_modal", s.onModCreateModalSubmit)
	r.Modal("/reject_mod_modal/{id}", s.handleRejectModModalSubmit)
	r.Modal("/reject_ver_modal/{id}", s.handleRejectVersionModalSubmit)

	// Button Component Routing
	r.ButtonComponent("/approve_mod/{id}", s.handleApproveModButton)
	r.ButtonComponent("/reject_mod/{id}", s.handleRejectModButton)
	r.ButtonComponent("/approve_ver/{id}", s.handleApproveVersionButton)
	r.ButtonComponent("/reject_ver/{id}", s.handleRejectVersionButton)
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

	slog.InfoContext(ctx, "Discord bot connected successfully with handler package")

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
			Description: "Mod of Us: Mod の作成・管理コマンド",
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

	// Notify Submitter
	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: ptr(fmt.Sprintf("✅ Mod `%s` バージョン `%s` の申請を受け付けました！モデレーターによる審査が行われます。", modID, version)),
	})

	// Post Review Embed to Staff Channel
	s.postVersionReviewEmbed(verSub, inspection, e.User())
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
		sb.WriteString(fmt.Sprintf("• **%s** (`%s`) - 最新版: `%s`\n", m.Name, m.ID, latest))
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

	embed := discord.NewEmbed().
		WithTitle(mod.Name).
		WithDescription(mod.Description).
		AddField("Mod ID", fmt.Sprintf("`%s`", mod.ID), true).
		AddField("作者", mod.Author, true).
		AddField("最新バージョン", fmt.Sprintf("`%s`", mod.LatestVersionExternal), true).
		WithColor(0x5865F2).
		WithTimestamp(time.Now())

	if mod.ThumbnailURI != nil && *mod.ThumbnailURI != "" {
		embed = embed.WithThumbnail(*mod.ThumbnailURI)
	}

	_, _ = e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds: &[]discord.Embed{embed},
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
	return nil
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
