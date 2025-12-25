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

	"github.com/valkey-io/valkey-go"

	"github.com/ikafly144/au_mod_installer/server-admin/handler"
	"github.com/ikafly144/au_mod_installer/server-admin/repository"
	"github.com/ikafly144/au_mod_installer/server-admin/templates"
)

func main() {
	var (
		addr       string
		valkeyAddr string
	)

	flag.StringVar(&addr, "addr", ":8081", "HTTP server address")
	flag.StringVar(&valkeyAddr, "valkey", "localhost:6379", "Valkey server address")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create context that listens for SIGINT and SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect to Valkey
	slog.Info("connecting to Valkey", "addr", valkeyAddr)
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{valkeyAddr},
	})
	if err != nil {
		slog.Error("failed to connect to Valkey", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Create repository and handler
	repo := repository.NewValkeyRepository(client)
	tmpl := templates.New()
	h := handler.New(repo, tmpl)

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", templates.StaticHandler())

	// Page routes
	mux.HandleFunc("GET /{$}", h.HandleList)
	mux.HandleFunc("GET /mods/{modID}/versions/{$}", h.HandleVersionsPage)
	mux.HandleFunc("GET /mods/{modID}/versions/new", h.HandleVersionNew)
	mux.HandleFunc("GET /mods/{modID}/versions/{versionID}/edit", h.HandleVersionEdit)

	// API routes
	mux.HandleFunc("GET /api/mods", h.HandleGetMods)
	mux.HandleFunc("POST /api/mods", h.HandleCreateMod)
	mux.HandleFunc("GET /api/mods/{modID}", h.HandleGetMod)
	mux.HandleFunc("PUT /api/mods/{modID}", h.HandleUpdateMod)
	mux.HandleFunc("DELETE /api/mods/{modID}", h.HandleDeleteMod)

	mux.HandleFunc("GET /api/mods/{modID}/versions", h.HandleGetVersions)
	mux.HandleFunc("POST /api/mods/{modID}/versions", h.HandleCreateVersion)
	mux.HandleFunc("GET /api/mods/{modID}/versions/{versionID}", h.HandleGetVersion)
	mux.HandleFunc("PUT /api/mods/{modID}/versions/{versionID}", h.HandleUpdateVersion)
	mux.HandleFunc("DELETE /api/mods/{modID}/versions/{versionID}", h.HandleDeleteVersion)
	mux.HandleFunc("POST /api/mods/{modID}/versions/{versionID}/latest", h.HandleSetLatestVersion)

	mux.HandleFunc("GET /api/github/releases", h.HandleListGitHubReleases)
	mux.HandleFunc("GET /api/github/releases/info", h.HandleGetGitHubRelease)
	mux.HandleFunc("GET /api/github/releases/latest", h.HandleGetGitHubRelease)

	mux.HandleFunc("POST /api/import", h.HandleImport)
	mux.HandleFunc("GET /api/export", h.HandleExport)

	server := &http.Server{
		Addr:    addr,
		Handler: loggingMiddleware(corsMiddleware(mux)),
	}

	// Start server in a goroutine
	go func() {
		slog.Info("starting admin server", "addr", addr)
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"size", rw.size,
			"duration", time.Since(start),
			"remote", r.RemoteAddr,
		)
	})
}
