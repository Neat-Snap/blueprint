package logger

import (
	"log/slog"
	"os"
)

func New(env string) *slog.Logger {
	if env == "prod" || env == "dev" { // for now
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return nil
}
