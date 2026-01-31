package bootstrap

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/shared/logger"

	"go.uber.org/zap"
)

// HealthChecker defines interface for component health checks
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	container *Container
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(container *Container) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{container: container}
}

// Name returns the health checker name
func (h *DatabaseHealthChecker) Name() string {
	return "database"
}

// Check performs database health check
func (h *DatabaseHealthChecker) Check(ctx context.Context) error {
	if h.container.Database == nil {
		return fmt.Errorf("database not initialized")
	}
	return h.container.Database.WriteDB().PingContext(ctx)
}

// CacheHealthChecker checks cache connectivity
type CacheHealthChecker struct {
	container *Container
}

// NewCacheHealthChecker creates a new cache health checker
func NewCacheHealthChecker(container *Container) *CacheHealthChecker {
	return &CacheHealthChecker{container: container}
}

// Name returns the health checker name
func (h *CacheHealthChecker) Name() string {
	return "cache"
}

// Check performs cache health check
func (h *CacheHealthChecker) Check(ctx context.Context) error {
	if h.container.Cache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return h.container.Cache.Client.Ping(ctx).Err()
}

// HealthManager manages health checks for the container
type HealthManager struct {
	checkers []HealthChecker
	logger   *logger.Logger
}

// NewHealthManager creates a new health manager
func NewHealthManager(log *logger.Logger) *HealthManager {
	return &HealthManager{
		checkers: make([]HealthChecker, 0),
		logger:   log,
	}
}

// AddChecker adds a health checker
func (h *HealthManager) AddChecker(checker HealthChecker) {
	h.checkers = append(h.checkers, checker)
}

// CheckAll performs all health checks
func (h *HealthManager) CheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)

	for _, checker := range h.checkers {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := checker.Check(checkCtx)
		cancel()

		results[checker.Name()] = err

		if err != nil {
			h.logger.Error("Health check failed",
				zap.String("checker", checker.Name()),
				zap.Error(err))
		} else {
			h.logger.Debug("Health check passed",
				zap.String("checker", checker.Name()))
		}
	}

	return results
}

// AddHealthCheckers adds default health checkers to the container
func (c *Container) AddHealthCheckers() {
	healthManager := NewHealthManager(c.Logger)

	// Add database health checker
	healthManager.AddChecker(NewDatabaseHealthChecker(c))

	// Add cache health checker
	healthManager.AddChecker(NewCacheHealthChecker(c))

	// Store health manager in container (we could add this as a field)
	// For now, we'll use the existing Health method
}
