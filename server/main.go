package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

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
	flag.Parse()

	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	modSrv := service.NewModService(gormrepo.NewGormRepository(db))

	srv := &http.Server{
		Addr:    *addr,
		Handler: router(modSrv),
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
