package healthService

import (
	"context"
	"time"

	"go-boilerplate/internal/pkg/health"
	"go-boilerplate/internal/shared/logger"
)

// HealthService provides health check functionality
type HealthService struct {
	startTime time.Time
	version   string
	checkers  []health.HealthChecker
	logger    *logger.Logger
}

// NewHealthService creates a new health service
func NewHealthService(version string, logger *logger.Logger) *HealthService {
	return &HealthService{
		startTime: time.Now(),
		version:   version,
		checkers:  make([]health.HealthChecker, 0),
		logger:    logger.Named("health-service"),
	}
}

// AddChecker adds a health checker to the service
func (s *HealthService) AddChecker(checker health.HealthChecker) {
	s.checkers = append(s.checkers, checker)
}

// Health returns the overall health status
func (s *HealthService) Health(ctx context.Context) health.HealthStatus {
	status := health.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   s.version,
		Uptime:    time.Since(s.startTime),
		Services:  make(map[string]health.Check),
	}

	// Check all registered services
	for _, checker := range s.checkers {
		check := checker.Check()
		status.Services[checker.Name()] = check

		// If any service is unhealthy, mark overall status as unhealthy
		if check.Status != "healthy" {
			status.Status = "unhealthy"
		}
	}

	return status
}

// Ready returns the readiness status
func (s *HealthService) Ready(ctx context.Context) health.ReadinessStatus {
	status := health.ReadinessStatus{
		Status:    "ready",
		Timestamp: time.Now(),
		Services:  make(map[string]health.Check),
	}

	// Check all registered services
	for _, checker := range s.checkers {
		if readinessChecker, ok := checker.(health.ReadinessChecker); ok {
			check := readinessChecker.Check()
			status.Services[readinessChecker.Name()] = check

			// If any service is not ready, mark overall status as not_ready
			if check.Status != "healthy" {
				status.Status = "not_ready"
			}
		}
	}

	return status
}
