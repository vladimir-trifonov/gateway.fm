package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

func InitLogger(_ context.Context, level string) {
	glogger := log.NewGlogHandler(log.NewTerminalHandler(os.Stdout, true))
	glogger.Verbosity(parseLogLevel(level))
	log.SetDefault(log.NewLogger(glogger))
}

func parseLogLevel(level string) slog.Level {
	level = strings.ToUpper(level)
	switch {
	case level == "DEBUG":
		return slog.LevelDebug
	case level == "INFO":
		return slog.LevelInfo
	case level == "WARN":
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}
