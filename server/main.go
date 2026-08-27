package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	gormrepo "github.com/ikafly144/au_mod_installer/server/repository/gorm"
	"github.com/ikafly144/au_mod_installer/server/service"
)

func main() {
	zl := log.Logger
	handler := zerolog.NewSlogHandler(zl)
	slog.SetDefault(slog.New(handler))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := realMain(ctx); err != nil {
		slog.ErrorContext(ctx, err.Error())
	}
}

func realMain(ctx context.Context) error {
	var addr = flag.String("addr", ":8080", "Address to listen on")
	var pathPrefix = flag.String("path-prefix", "/api", "Path prefix for API endpoints")
	var basePath = flag.String("base-path", "/v1", "Base path for API endpoints")
	flag.Parse()

	// read from environment variables
	if envAddr := os.Getenv("ADDR"); envAddr != "" {
		*addr = envAddr
	}
	if envPathPrefix := os.Getenv("PATH_PREFIX"); envPathPrefix != "" {
		*pathPrefix = envPathPrefix
	}
	if envBasePath := os.Getenv("BASE_PATH"); envBasePath != "" {
		*basePath = envBasePath
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/au_mod_installer?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(databaseURL))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	gormRepo := gormrepo.NewGormRepository(db)
	if err := gormRepo.Migrate(); err != nil {
		slog.WarnContext(ctx, "Database migration warning", "error", err)
	}

	// Initialize S3-compatible Storage (MinIO / Cloudflare R2 / AWS S3)
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "au-mods"
	}
	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		s3Region = "us-east-1"
	}
	usePathStyle := true
	if val := os.Getenv("S3_USE_PATH_STYLE"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			usePathStyle = parsed
		}
	}

	storageCfg := service.StorageConfig{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		PublicEndpoint:  os.Getenv("S3_PUBLIC_ENDPOINT"),
		Region:          s3Region,
		Bucket:          s3Bucket,
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		UsePathStyle:    usePathStyle,
	}

	storageSvc, err := service.NewS3StorageService(ctx, storageCfg)
	if err != nil {
		slog.WarnContext(ctx, "Failed to initialize S3 storage service", "error", err)
	} else {
		if err := storageSvc.EnsureBucket(ctx); err != nil {
			slog.WarnContext(ctx, "Failed to ensure S3 bucket exists", "error", err, "bucket", s3Bucket)
		}
	}

	// Initialize Security & File Scanning Service
	scanSvc := service.NewScanService(os.Getenv("VIRUSTOTAL_API_KEY"))

	// Initialize Submission & Moderation Service
	submissionSvc := service.NewSubmissionService(gormRepo, gormRepo, gormRepo, storageSvc, scanSvc)

	// Initialize Mod Service
	modSrv := service.NewModService(gormRepo)

	// Initialize Version Info Provider
	versionInfoTTL := time.Duration(0)
	if rawTTL := os.Getenv("VERSION_INFO_TTL"); rawTTL != "" {
		parsedTTL, err := time.ParseDuration(rawTTL)
		if err != nil {
			slog.WarnContext(ctx, "Invalid VERSION_INFO_TTL; using default", "value", rawTTL, "error", err)
		} else {
			versionInfoTTL = parsedTTL
		}
	}
	versionSvc := service.NewVersionInfoService(service.VersionInfoOptions{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Token:      os.Getenv("GITHUB_TOKEN"),
		TTL:        versionInfoTTL,
	})

	// Initialize Discord Bot Service
	discordCfg := service.DiscordConfig{
		Token:             os.Getenv("DISCORD_TOKEN"),
		GuildID:           os.Getenv("DISCORD_GUILD_ID"),
		ModRoleID:         os.Getenv("DISCORD_MOD_ROLE_ID"),
		ReviewChannelID:   os.Getenv("DISCORD_REVIEW_CHANNEL_ID"),
		ShowcaseForumID:   os.Getenv("DISCORD_SHOWCASE_FORUM_ID"),
		UpdatesChannelID:  os.Getenv("DISCORD_UPDATES_CHANNEL_ID"),
		ReportChannelID:   os.Getenv("DISCORD_REPORT_CHANNEL_ID"),
		AuditLogChannelID: os.Getenv("DISCORD_AUDIT_LOG_CHANNEL_ID"),
	}

	discordBot := service.NewDiscordBotService(discordCfg, submissionSvc, modSrv)
	if err := discordBot.Start(ctx); err != nil {
		slog.WarnContext(ctx, "Failed to start Discord bot", "error", err)
	}
	defer discordBot.Stop(context.Background())

	srv := &http.Server{
		Addr:    *addr,
		Handler: router(modSrv, versionSvc, *pathPrefix, *basePath),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "listen: %s\n", "err", err)
		}
	}()

	slog.InfoContext(ctx, "Server started", "addr", *addr)

	<-ctx.Done()

	slog.InfoContext(ctx, "Server shutting down")
	if err := srv.Shutdown(context.Background()); err != nil {
		slog.ErrorContext(ctx, "Server forced to shutdown: %s\n", "err", err)
	}

	slog.InfoContext(ctx, "Server exiting")

	return nil
}
