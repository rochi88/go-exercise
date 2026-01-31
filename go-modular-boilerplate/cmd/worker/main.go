package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	cron "github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/database"
	"go-boilerplate/internal/scheduler"
	"go-boilerplate/internal/shared/cache"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	appLogger := logger.New(cfg.Environment)
	defer appLogger.Sync()

	// Initialize metrics (if enabled)
	var metricsCollector *metrics.Metrics
	if cfg.MetricsEnabled {
		metricsCollector = metrics.New(appLogger)
	}

	// Initialize database connection
	rwDBConfig := database.NewReadWriteConfig(cfg)
	rwDB, err := database.NewReadWriteDatabase(rwDBConfig, appLogger, metricsCollector)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer rwDB.Close()

	// Initialize Redis connection
	redisConfig := cache.DefaultConfig(cfg)
	redisClient, err := cache.New(redisConfig, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Initialize cron scheduler
	cronScheduler := cron.New()

	// Initialize scheduler with dependencies
	scheduler := scheduler.NewScheduler(cronScheduler, rwDB, redisClient, appLogger, metricsCollector)

	// Register cron jobs
	if err := scheduler.RegisterJobs(); err != nil {
		appLogger.Fatal("Failed to register cron jobs", zap.Error(err))
	}

	// Start the cron scheduler
	cronScheduler.Start()
	appLogger.Info("Cron scheduler started")

	// Set up graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	appLogger.Info("Worker is running", zap.String("environment", cfg.Environment))

	// Wait for interrupt signal
	<-done
	appLogger.Info("Worker is shutting down...")

	// Stop the cron scheduler
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Create a channel to signal when the scheduler has stopped
	stopCh := make(chan struct{})
	go func() {
		<-cronScheduler.Stop().Done()
		close(stopCh)
	}()

	// Wait for scheduler to stop or timeout
	select {
	case <-stopCh:
		appLogger.Info("Cron scheduler stopped gracefully")
	case <-shutdownCtx.Done():
		appLogger.Warn("Cron scheduler shutdown timed out")
	}

	appLogger.Info("Worker gracefully stopped")
}
