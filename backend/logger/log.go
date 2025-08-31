package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type MultiLogger struct {
	logger zerolog.Logger
}

func New(logFile string) (*MultiLogger, error) {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006/01/02 15:04:05",
	}

	var writers []io.Writer
	writers = append(writers, consoleWriter)

	if logFile != "" {
		if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			writers = append(writers, f)
		}
	}

	zerolog.TimeFieldFormat = time.RFC3339

	multi := io.MultiWriter(writers...)

	logger := zerolog.New(multi).With().Timestamp().Caller().Logger()

	zerolog.DefaultContextLogger = &logger

	return &MultiLogger{logger: logger}, nil
}

func (l *MultiLogger) Info(msg string, fields ...any) {
	l.logger.Info().Fields(fields).Msg(msg)
}

func (l *MultiLogger) Warn(msg string, fields ...any) {
	l.logger.Warn().Fields(fields).Msg(msg)
}

func (l *MultiLogger) Error(msg string, fields ...any) {
	l.logger.Error().Fields(fields).Msg(msg)
}

func (l *MultiLogger) Debug(msg string, fields ...any) {
	l.logger.Debug().Fields(fields).Msg(msg)
}

func (l *MultiLogger) Fatal(msg string, fields ...any) {
	l.logger.Fatal().Fields(fields).Msg(msg)
}
