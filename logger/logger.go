package logger

import (
	"log/slog"
	"os"
)

func Init() {
	var handler slog.Handler
	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	if os.Getenv("ENV") == "development" {
		handler = slog.NewTextHandler(os.Stdout, options)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}
	slog.SetDefault(slog.New(handler))
}
