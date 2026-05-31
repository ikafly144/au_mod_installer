package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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

	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	modSrv := service.NewModService(gormrepo.NewGormRepository(db))
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

	srv := &http.Server{
		Addr:    *addr,
		Handler: router(modSrv, versionSvc, *pathPrefix, *basePath),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "listen: %s\n", "err", err)
		}
	}()

	slog.InfoContext(ctx, "Server started")

	<-ctx.Done()

	slog.InfoContext(ctx, "Server shutting down")
	if err := srv.Shutdown(context.Background()); err != nil {
		slog.ErrorContext(ctx, "Server forced to shutdown: %s\n", "err", err)
	}

	slog.InfoContext(ctx, "Server exiting")

	return nil
}
