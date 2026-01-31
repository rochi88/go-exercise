package health

import (
	"context"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/shared/cache"
	"go-boilerplate/internal/shared/logger"
)

// RedisHealthChecker checks Redis connectivity
type RedisHealthChecker struct {
	client *cache.Redis
	logger *logger.Logger
}

// NewRedisHealthChecker creates a new Redis health checker
func NewRedisHealthChecker(client *cache.Redis, logger *logger.Logger) *RedisHealthChecker {
	return &RedisHealthChecker{
		client: client,
		logger: logger.Named("redis-health-checker"),
	}
}

// Name returns the name of this health checker
func (c *RedisHealthChecker) Name() string {
	return "redis"
}

// Check performs the health check
func (c *RedisHealthChecker) Check() Check {
	check := Check{
		Status: "healthy",
		Time:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to ping Redis
	err := c.client.Client.Ping(ctx).Err()
	if err != nil {
		check.Status = "unhealthy"
		check.Message = "Redis connection failed: " + err.Error()
		c.logger.Error("Redis health check failed", zap.Error(err))
	} else {
		check.Message = "Redis connection is healthy"
	}

	return check
}
