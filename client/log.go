package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Join(configDir, "au_mod_installer"), 0755); err != nil {
		panic(err)
	}
	fileLogger := &lumberjack.Logger{
		Filename:   filepath.Join(configDir, "au_mod_installer", "app.log"), // ログファイルのパス
		MaxSize:    10,                                                      // 1ファイルあたりの最大サイズ (MB)
		MaxBackups: 5,                                                       // 残す古いログファイルの最大数
		MaxAge:     30,                                                      // 古いログファイルを保持する最大日数
		Compress:   true,                                                    // 古いログを自動でgzip圧縮するかどうか
	}

	slog.SetDefault(slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		slog.NewJSONHandler(fileLogger, &slog.HandlerOptions{Level: slog.LevelDebug}),
	)))
}
