package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Neat-Snap/blueprint-backend/api"
	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
)

func main() {
	cfg := config.Load()
	log, err := logger.New("main.log")
	if err != nil {
		log.Error("failed to create logger", "error", err)
		os.Exit(1)
	}

	dbConn, err := db.Connect(&cfg, log)
	if err != nil {
		log.Error("failed to connect to the database", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(api.RouterConfig{
		Env:    cfg.Env,
		DB:     dbConn,
		Logger: *log,
	})

	server := api.NewServer(cfg, log, router)

	errCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-quit:
		log.Info("received signal", "signal", sig)
	case err := <-errCh:
		log.Error("server error", slog.Any("error", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		log.Error("graceful shutdown failed", slog.Any("error", err))
	}
}
