package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/handler"
	"github.com/ikafly144/au_mod_installer/server/middleware"
	valkeyrepo "github.com/ikafly144/au_mod_installer/server/repository/valkey"
	"github.com/ikafly144/au_mod_installer/server/service"
	"github.com/valkey-io/valkey-go"
)

func main() {
	var (
		addr       string
		modsFile   string
		valkeyAddr string
		rootPath   string
	)

	flag.StringVar(&addr, "addr", ":8080", "HTTP server address")
	flag.StringVar(&modsFile, "mods", "mods.json", "Path to mods.json file")
	flag.StringVar(&valkeyAddr, "valkey", "", "Valkey server address (e.g., localhost:6379). If empty, uses file-based storage")
	flag.StringVar(&rootPath, "root", "/", "Root path for the server")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create context that listens for SIGINT and SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var modService handler.ModServiceInterface

	if valkeyAddr != "" {
		// Use Valkey-based storage
		slog.Info("connecting to Valkey", "addr", valkeyAddr)

		client, err := valkey.NewClient(valkey.ClientOption{
			InitAddress: []string{valkeyAddr},
		})
		if err != nil {
			slog.Error("failed to connect to Valkey", "error", err)
			os.Exit(1)
		}
		defer client.Close()

		repo := valkeyrepo.NewRepository(client)

		// Load initial data from file if it exists
		if modsFile != "" {
			if _, err := os.Stat(modsFile); err == nil {
				slog.Info("loading mods from file into Valkey", "file", modsFile)
				if err := valkeyrepo.LoadModsFromFile(ctx, repo, modsFile); err != nil {
					slog.Error("failed to load mods from file", "error", err)
					os.Exit(1)
				}
			}
		}

		modService = service.NewModServiceWithRepo(repo)
		slog.Info("using Valkey storage")
	} else {
		// Use file-based storage (backward compatibility)
		fileService, err := service.NewModService(modsFile)
		if err != nil {
			slog.Error("failed to create mod service", "error", err)
			os.Exit(1)
		}
		modService = &fileModServiceAdapter{fileService}
		slog.Info("using file-based storage", "file", modsFile)
	}

	h := handler.NewHandler(modService)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Apply middlewares
	httpHandler := middleware.Chain(mux, middleware.Logging, middleware.CORS)

	root := http.NewServeMux()
	root.Handle(rootPath, httpHandler)
	server := &http.Server{
		Addr:    addr,
		Handler: root,
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
	return a.FileModService.GetModList(limit, after, before)
}

func (a *fileModServiceAdapter) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	return a.FileModService.GetMod(modID)
}

func (a *fileModServiceAdapter) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	return a.FileModService.GetModVersions(modID, limit, after)
}

func (a *fileModServiceAdapter) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	return a.FileModService.GetModVersion(modID, versionID)
}
