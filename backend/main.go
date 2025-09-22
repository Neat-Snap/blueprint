package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Neat-Snap/blueprint-backend/api"
	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/utils/email"
	"github.com/Neat-Snap/blueprint-backend/workos"
	"github.com/workos/workos-go/v4/pkg/usermanagement"
)

func main() {
	cfg := config.Load()
	log, err := logger.New("main.log")
	if err != nil || log == nil {
		fmt.Fprintln(os.Stderr, "failed to create logger:", err)
		os.Exit(1)
	}

	dbConn, err := db.Connect(&cfg, log)
	if err != nil {
		log.Error("failed to connect to the database", "error", err)
		os.Exit(1)
	}

	connectionObject := db.NewConnection(dbConn)

	emailClient := email.NewEmailClient(cfg, *log)

	workosClient := usermanagement.NewClient(cfg.WORKOS_API_KEY)
	workosValidator, err := workos.NewValidator(workos.ValidatorConfig{
		Client:   workosClient,
		ClientID: cfg.WORKOS_CLIENT_ID,
		Issuer:   cfg.WORKOS_ISSUER,
	})
	if err != nil {
		log.Error("failed to configure workos validator", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(api.RouterConfig{
		Env:             cfg.Env,
		DB:              dbConn,
		Logger:          *log,
		Connection:      connectionObject,
		EmailClient:     emailClient,
		RedisSecret:     cfg.REDIS_SECRET,
		Config:          cfg,
		WorkOSValidator: workosValidator,
		WorkOSClient:    workosClient,
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
		log.Error("server error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		log.Error("graceful shutdown failed", slog.Any("error", err))
	}
}
