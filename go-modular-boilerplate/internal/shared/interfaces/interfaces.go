package interfaces

import (
	"context"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
)

// Container defines the interface for dependency injection container
type Container interface {
	// Configuration and core services
	GetConfig() *config.Config
	GetLogger() *logger.Logger
	GetMetrics() *metrics.Metrics

	// Health and lifecycle
	Health(ctx context.Context) error
	Close() error
}

// Repository defines common repository operations
type Repository interface {
}

// Service defines common service operations
type Service interface {
}

// Handler defines common HTTP handler operations
type Handler interface {
}
