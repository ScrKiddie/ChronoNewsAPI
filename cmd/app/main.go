package main

import (
	"chrononewsapi/internal/bootstrap"
	"chrononewsapi/internal/config"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	appConfig := config.NewConfig()
	db := config.NewDatabase(appConfig)
	chi := config.NewChi(appConfig)
	validator := config.NewValidator()
	client := config.NewClient()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bootstrap.Init(chi, db, appConfig, validator, client, ctx)

	server := &http.Server{
		Addr:    "0.0.0.0:" + appConfig.Web.Port,
		Handler: chi,
	}

	go func() {
		slog.Info("server starting", "port", appConfig.Web.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("could not start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	slog.Info("shutdown signal received, starting graceful shutdown")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	slog.Info("server exited properly")
}
