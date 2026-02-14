package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/handler"
	"github.com/ikafly144/au_mod_installer/server/middleware"
	repo "github.com/ikafly144/au_mod_installer/server/repository/gorm"
	"github.com/ikafly144/au_mod_installer/server/service"
)

func main() {
	slog.Info("Starting Among Us Mod Installer server", "version", version, "revision", revision)
	var (
		addr                string
		databaseURL         string
		pathPrefix          string
		basePath            string
		disabledVersionsStr string
		jwtSecret           string
		discordClientID     string
		discordClientSecret string
		discordRedirectURI  string
		discordAdminIDsStr  string
	)

	defaultAddr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		defaultAddr = ":" + port
	}
	if envAddr := os.Getenv("ADDR"); envAddr != "" {
		defaultAddr = envAddr
	}

	flag.StringVar(&addr, "addr", defaultAddr, "HTTP server address")
	flag.StringVar(&databaseURL, "database-url", "", "PostgreSQL database URL (e.g., postgres://user:pass@localhost:5432/dbname). If empty, uses file-based storage")
	flag.StringVar(&pathPrefix, "path-prefix", "", "URL path prefix (e.g. /api)")
	flag.StringVar(&basePath, "base-path", "", "API version base path (e.g. /v1)")
	flag.StringVar(&disabledVersionsStr, "disabled-versions", "", "Comma-separated list of disabled versions")
	flag.StringVar(&jwtSecret, "jwt-secret", "", "JWT secret key")
	flag.StringVar(&discordClientID, "discord-client-id", "", "Discord OAuth client ID")
	flag.StringVar(&discordClientSecret, "discord-client-secret", "", "Discord OAuth client secret")
	flag.StringVar(&discordRedirectURI, "discord-redirect-uri", "", "Discord OAuth redirect URI")
	flag.StringVar(&discordAdminIDsStr, "discord-admin-ids", "", "Comma-separated list of Discord user IDs to grant admin")
	flag.Parse()

	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if jwtSecret == "" {
		jwtSecret = os.Getenv("JWT_SECRET")
	}
	if discordClientID == "" {
		discordClientID = os.Getenv("DISCORD_CLIENT_ID")
	}
	if discordClientSecret == "" {
		discordClientSecret = os.Getenv("DISCORD_CLIENT_SECRET")
	}
	if discordRedirectURI == "" {
		discordRedirectURI = os.Getenv("DISCORD_REDIRECT_URI")
	}
	if discordAdminIDsStr == "" {
		discordAdminIDsStr = os.Getenv("DISCORD_ADMIN_IDS")
	}

	if pathPrefix == "" {
		pathPrefix = os.Getenv("PATH_PREFIX")
	}
	if basePath == "" {
		basePath = os.Getenv("BASE_PATH")
	}

	if disabledVersionsStr == "" {
		disabledVersionsStr = os.Getenv("DISABLED_VERSIONS")
	}

	var disabledVersions []string
	if disabledVersionsStr != "" {
		parts := strings.Split(disabledVersionsStr, ",")
		for _, p := range parts {
			if v := strings.TrimSpace(p); v != "" {
				disabledVersions = append(disabledVersions, v)
			}
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create context that listens for SIGINT and SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var modService handler.ModServiceInterface
	var authService *service.AuthService

	if databaseURL != "" {
		// Use PostgreSQL-based storage (GORM)
		slog.Info("connecting to PostgreSQL via GORM", "url", databaseURL)

		r, err := repo.NewRepository(databaseURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer r.Close()

		slog.Info("database connected and migrations applied")

		modService = service.NewModServiceWithRepo(r)
		if jwtSecret != "" && discordClientID != "" && discordClientSecret != "" && discordRedirectURI != "" {
			var adminIDs []string
			if discordAdminIDsStr != "" {
				for _, id := range strings.Split(discordAdminIDsStr, ",") {
					if trimmed := strings.TrimSpace(id); trimmed != "" {
						adminIDs = append(adminIDs, trimmed)
					}
				}
				slog.Info("discord admin IDs configured", "count", len(adminIDs))
			}
			authService = service.NewAuthService(service.AuthServiceConfig{
				UserRepo:            r,
				JWTSecret:           jwtSecret,
				DiscordClientID:     discordClientID,
				DiscordClientSecret: discordClientSecret,
				DiscordRedirectURI:  discordRedirectURI,
				AdminDiscordIDs:     adminIDs,
			})
		} else {
			slog.Warn("Discord OAuth or JWT secret not configured, authentication will be disabled")
		}
		slog.Info("using PostgreSQL storage")
	} else {

		// Use file-based storage (backward compatibility)
		fileService, err := service.NewModService("mods.json")
		if err != nil {
			slog.Error("failed to create mod service", "error", err)
			os.Exit(1)
		}
		modService = &fileModServiceAdapter{fileService}
		slog.Info("using file-based storage", "file", "mods.json")
	}

	h := handler.NewHandler(modService, version, disabledVersions)
	if jwtSecret != "" {
		mw := middleware.NewAuthMiddleware(jwtSecret)
		h.SetAuthMiddleware(mw)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux, basePath)

	if authService != nil {

		authHandler := handler.NewAuthHandler(authService)
		authHandler.RegisterRoutes(mux, basePath)
	}

	var rootHandler http.Handler = mux

	if pathPrefix != "" {
		slog.Info("enabling path prefix", "prefix", pathPrefix)
		rootHandler = http.StripPrefix(pathPrefix, mux)
	}

	// Apply middlewares
	httpHandler := middleware.Chain(rootHandler, middleware.Logging, middleware.CORS)

	server := &http.Server{
		Addr:    addr,
		Handler: httpHandler,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("starting server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	stop()
	slog.Info("shutting down server...")

	// Create a deadline for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}

// fileModServiceAdapter adapts FileModService to ModServiceInterface
type fileModServiceAdapter struct {
	*service.FileModService
}

func (a *fileModServiceAdapter) GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error) {
	return a.FileModService.GetModList(ctx, limit, after, before)
}

func (a *fileModServiceAdapter) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	return a.FileModService.GetMod(ctx, modID)
}

func (a *fileModServiceAdapter) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	return a.FileModService.GetModVersions(ctx, modID, limit, after)
}

func (a *fileModServiceAdapter) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	return a.FileModService.GetModVersion(ctx, modID, versionID)
}

func (a *fileModServiceAdapter) CreateMod(ctx context.Context, mod modmgr.Mod) error {
	return errors.New("write operations not supported in file mode")
}

func (a *fileModServiceAdapter) UpdateMod(ctx context.Context, mod modmgr.Mod) error {
	return errors.New("write operations not supported in file mode")
}

func (a *fileModServiceAdapter) DeleteMod(ctx context.Context, modID string) error {
	return errors.New("write operations not supported in file mode")
}

func (a *fileModServiceAdapter) CreateModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	return errors.New("write operations not supported in file mode")
}

func (a *fileModServiceAdapter) DeleteModVersion(ctx context.Context, modID string, versionID string) error {
	return errors.New("write operations not supported in file mode")
}
