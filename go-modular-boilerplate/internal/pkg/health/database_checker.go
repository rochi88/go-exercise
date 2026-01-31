package health

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/database"
	"go-boilerplate/internal/shared/logger"
)

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	db     *database.Database
	logger *logger.Logger
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(db *database.Database, logger *logger.Logger) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		db:     db,
		logger: logger.Named("db-health-checker"),
	}
}

// Name returns the name of this health checker
func (c *DatabaseHealthChecker) Name() string {
	return "database"
}

// Check performs the health check
func (c *DatabaseHealthChecker) Check() Check {
	check := Check{
		Status: "healthy",
		Time:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to ping the database
	if err := c.db.PingContext(ctx); err != nil {
		check.Status = "unhealthy"
		check.Message = "Database connection failed: " + err.Error()
		c.logger.Error("Database health check failed", zap.Error(err))
	} else {
		// Get connection pool statistics
		stats := c.db.GetStats()
		check.Message = fmt.Sprintf("Database connection is healthy. Pool stats: %d/%d open, %d in use, %d idle",
			stats.OpenConnections, stats.MaxOpenConnections, stats.InUse, stats.Idle)
	}

	return check
}

// ReadWriteDatabaseHealthChecker checks read/write database connectivity
type ReadWriteDatabaseHealthChecker struct {
	rwDB   *database.ReadWriteDatabase
	logger *logger.Logger
}

// NewReadWriteDatabaseHealthChecker creates a new read/write database health checker
func NewReadWriteDatabaseHealthChecker(rwDB *database.ReadWriteDatabase, logger *logger.Logger) *ReadWriteDatabaseHealthChecker {
	return &ReadWriteDatabaseHealthChecker{
		rwDB:   rwDB,
		logger: logger.Named("rw-db-health-checker"),
	}
}

// Name returns the name of this health checker
func (c *ReadWriteDatabaseHealthChecker) Name() string {
	return "read-write-database"
}

// Check performs the health check
func (c *ReadWriteDatabaseHealthChecker) Check() Check {
	check := Check{
		Status: "healthy",
		Time:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check read/write database health
	if err := c.rwDB.HealthCheck(ctx); err != nil {
		check.Status = "unhealthy"
		check.Message = "Read/write database health check failed: " + err.Error()
		c.logger.Error("Read/write database health check failed", zap.Error(err))
	} else {
		// Get connection pool statistics
		stats := c.rwDB.GetStats()
		writeStats := stats["write"].(database.Stats)
		check.Message = fmt.Sprintf("Read/write database healthy. Write pool: %d/%d open, %d in use",
			writeStats.OpenConnections, writeStats.MaxOpenConnections, writeStats.InUse)

		if readStatsInterface, ok := stats["read"]; ok {
			if readStats, ok := readStatsInterface.([]database.Stats); ok && len(readStats) > 0 {
				check.Message += fmt.Sprintf(". Read pools: %d configured", len(readStats))
			}
		}
	}

	return check
}
