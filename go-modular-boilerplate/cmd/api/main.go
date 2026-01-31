package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/app/api"
	"go-boilerplate/internal/app/bootstrap"
)

func main() {
	// Initialize container with all dependencies
	container, err := bootstrap.NewContainer(bootstrap.ContainerOptions{
		ConfigPath: "./configs",
	})
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}

	// Ensure graceful cleanup
	defer func() {
		if err := container.Close(); err != nil {
			container.Logger.Error("Failed to close container gracefully", zap.Error(err))
		}
	}()

	// Initialize HTTP server with all dependencies from container
	serverOptions := &api.ServerOptions{
		Config:              container.Config,
		Logger:              container.Logger,
		AuthHandler:         container.AuthHandler,
		UserHandler:         container.UserHandler,
		HealthHandler:       container.HealthHandler,
		AuthMiddleware:      container.AuthMiddleware,
		LoggingMiddleware:   container.LoggingMiddleware,
		RecoveryMiddleware:  container.RecoveryMiddleware,
		SecurityMiddleware:  container.SecurityMiddleware,
		RateLimitMiddleware: container.RateLimitMiddleware,
		RequestIDMiddleware: container.RequestIDMiddleware,
		Metrics:             container.Metrics,
	}

	server := api.NewServer(serverOptions)

	// Set up graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			container.Logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	container.Logger.Info("Server is running",
		zap.Int("port", container.Config.ServerPort),
		zap.String("environment", container.Config.Environment))

	// Perform initial health check
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer healthCancel()

	if err := container.Health(healthCtx); err != nil {
		container.Logger.Warn("Initial health check failed", zap.Error(err))
	} else {
		container.Logger.Info("Initial health check passed")
	}

	// Wait for interrupt signal
	<-done
	container.Logger.Info("Server is shutting down...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Stop(ctx); err != nil {
		container.Logger.Fatal("Server shutdown failed", zap.Error(err))
	}

	container.Logger.Info("Server gracefully stopped")
}
