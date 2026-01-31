package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
)

// Database wraps sqlx.DB to provide custom functionality
type Database struct {
	*sqlx.DB
	logger              *logger.Logger
	config              *Config
	lastHealthCheck     time.Time
	healthCheckInterval time.Duration
	metrics             *metrics.Metrics
}

// Stats represents database connection pool statistics
type Stats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxLifetimeClosed  int64
}

// Config holds the database configuration options
type Config struct {
	URL                string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetime    time.Duration
	ConnMaxIdleTime    time.Duration
	RetryAttempts      int
	RetryDelay         time.Duration
	SlowQueryThreshold time.Duration
}

// DefaultConfig returns the default database configuration
func DefaultConfig(cfg *config.Config) *Config {
	return &Config{
		URL:                cfg.DBURL,
		MaxOpenConns:       cfg.DBMaxOpenConns,
		MaxIdleConns:       cfg.DBMaxIdleConns,
		ConnMaxLifetime:    cfg.DBConnMaxLifetime,
		ConnMaxIdleTime:    cfg.DBConnMaxIdleTime,
		RetryAttempts:      cfg.DBRetryAttempts,
		RetryDelay:         cfg.DBRetryDelay,
		SlowQueryThreshold: cfg.DBSlowQueryThreshold,
	}
}

// New creates a new database connection with enhanced pooling and retry logic
func New(cfg *Config, log *logger.Logger, metrics *metrics.Metrics) (*Database, error) {
	log = log.Named("database")

	var db *sqlx.DB
	var err error

	// Retry connection with exponential backoff
	for attempt := 1; attempt <= cfg.RetryAttempts; attempt++ {
		db, err = sqlx.Connect("postgres", cfg.URL)
		if err == nil {
			break
		}

		if attempt < cfg.RetryAttempts {
			log.Warn("Database connection attempt failed",
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", cfg.RetryAttempts),
				zap.Error(err),
				zap.Duration("retrying_in", cfg.RetryDelay))

			time.Sleep(cfg.RetryDelay)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", cfg.RetryAttempts, err)
	}

	// Configure connection pool with enhanced settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Enable connection pool statistics
	db.DB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.DB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Log connection pool configuration
	log.Info("Successfully connected to the database")

	return &Database{
		DB:                  db,
		logger:              log,
		config:              cfg,
		lastHealthCheck:     time.Now(),
		healthCheckInterval: 30 * time.Second, // Health check every 30 seconds
		metrics:             metrics,
	}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	db.logger.Info("Closing database connection")
	return db.DB.Close()
}

// GetStats returns current database connection pool statistics
func (db *Database) GetStats() Stats {
	stats := db.DB.Stats()
	return Stats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

// HealthCheck performs a health check on the database connection
func (db *Database) HealthCheck(ctx context.Context) error {
	// Check if we need to perform a health check
	if time.Since(db.lastHealthCheck) < db.healthCheckInterval {
		return nil // Skip health check if recently performed
	}

	// Perform health check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.DB.PingContext(checkCtx); err != nil {
		db.logger.Error("Database health check failed", zap.Error(err))
		return fmt.Errorf("database health check failed: %w", err)
	}

	db.lastHealthCheck = time.Now()
	db.logger.Debug("Database health check passed")
	return nil
}

// LogStats logs the current database connection pool statistics
func (db *Database) LogStats() {
	stats := db.GetStats()
	db.logger.Info("Database connection pool statistics",
		zap.Int("max_open_conns", stats.MaxOpenConnections),
		zap.Int("open_conns", stats.OpenConnections),
		zap.Int("in_use", stats.InUse),
		zap.Int("idle", stats.Idle),
		zap.Int64("wait_count", stats.WaitCount),
		zap.Duration("wait_duration", stats.WaitDuration),
		zap.Int64("max_idle_closed", stats.MaxIdleClosed),
		zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed))

	// Update metrics if available
	if db.metrics != nil {
		db.metrics.UpdateDBConnections(stats.MaxOpenConnections, stats.InUse, stats.Idle)
	}
}

// IsHealthy checks if the database connection is healthy
func (db *Database) IsHealthy(ctx context.Context) bool {
	return db.HealthCheck(ctx) == nil
}

// StartHealthMonitoring starts periodic health monitoring in a goroutine
func (db *Database) StartHealthMonitoring(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(db.healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				db.logger.Info("Stopping database health monitoring")
				return
			case <-ticker.C:
				if err := db.HealthCheck(ctx); err != nil {
					db.logger.Error("Periodic health check failed", zap.Error(err))
				} else {
					// Log stats periodically
					db.LogStats()
				}
			}
		}
	}()

	db.logger.Info("Started database health monitoring",
		zap.Duration("interval", db.healthCheckInterval))
}

// QueryMetrics holds query execution metrics
type QueryMetrics struct {
	Query     string
	Duration  time.Duration
	Args      []interface{}
	IsSlow    bool
	Error     error
	Timestamp time.Time
}

// QueryMonitor handles query monitoring and slow query logging
type QueryMonitor struct {
	logger             *logger.Logger
	slowQueryThreshold time.Duration
}

// NewQueryMonitor creates a new query monitor
func NewQueryMonitor(logger *logger.Logger, slowQueryThreshold time.Duration) *QueryMonitor {
	return &QueryMonitor{
		logger:             logger.Named("query-monitor"),
		slowQueryThreshold: slowQueryThreshold,
	}
}

// MonitorQuery monitors a query execution and logs if it's slow
func (qm *QueryMonitor) MonitorQuery(query string, args []interface{}, start time.Time, err error) {
	duration := time.Since(start)
	metrics := QueryMetrics{
		Query:     query,
		Duration:  duration,
		Args:      args,
		IsSlow:    duration >= qm.slowQueryThreshold,
		Error:     err,
		Timestamp: start,
	}

	// Log slow queries
	if metrics.IsSlow {
		qm.logger.Warn("Slow query detected",
			zap.String("query", metrics.Query),
			zap.Duration("duration", metrics.Duration),
			zap.Duration("threshold", qm.slowQueryThreshold),
			zap.Time("timestamp", metrics.Timestamp),
			zap.Error(metrics.Error))
	}

	// Log query metrics (debug level for all queries)
	qm.logger.Debug("Query executed",
		zap.String("query", metrics.Query),
		zap.Duration("duration", metrics.Duration),
		zap.Bool("is_slow", metrics.IsSlow),
		zap.Error(metrics.Error))
}

// ExecContext executes a query without returning rows (INSERT, UPDATE, DELETE)
func (db *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := db.DB.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if db.config.SlowQueryThreshold > 0 {
		monitor := NewQueryMonitor(db.logger, db.config.SlowQueryThreshold)
		monitor.MonitorQuery(query, args, start, err)
	}

	// Record metrics
	if db.metrics != nil {
		db.metrics.RecordDBQuery("exec", extractTableName(query), duration, err)
	}

	return result, err
}

// extractTableName extracts table name from SQL query for metrics
func extractTableName(query string) string {
	// Simple extraction - can be enhanced for more complex queries
	if len(query) == 0 {
		return "unknown"
	}

	// Convert to lowercase for easier matching
	query = strings.ToLower(strings.TrimSpace(query))

	// Handle common SQL patterns
	if strings.HasPrefix(query, "insert into ") {
		return extractTableFromInsert(query)
	} else if strings.HasPrefix(query, "update ") {
		return extractTableFromUpdate(query)
	} else if strings.HasPrefix(query, "delete from ") {
		return extractTableFromDelete(query)
	} else if strings.HasPrefix(query, "select ") {
		return extractTableFromSelect(query)
	}

	return "unknown"
}

func extractTableFromInsert(query string) string {
	parts := strings.Split(query, " ")
	if len(parts) >= 3 {
		return strings.Trim(parts[2], "`\"")
	}
	return "unknown"
}

func extractTableFromUpdate(query string) string {
	parts := strings.Split(query, " ")
	if len(parts) >= 2 {
		return strings.Trim(parts[1], "`\"")
	}
	return "unknown"
}

func extractTableFromDelete(query string) string {
	parts := strings.Split(query, " ")
	if len(parts) >= 3 {
		return strings.Trim(parts[2], "`\"")
	}
	return "unknown"
}

func extractTableFromSelect(query string) string {
	// Find FROM keyword
	fromIndex := strings.Index(query, " from ")
	if fromIndex == -1 {
		return "unknown"
	}

	fromPart := query[fromIndex+6:]
	parts := strings.Fields(fromPart)
	if len(parts) > 0 {
		return strings.Trim(parts[0], "`\"")
	}
	return "unknown"
}

// QueryContext executes a query that returns rows (SELECT)
func (db *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.DB.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if db.config.SlowQueryThreshold > 0 {
		monitor := NewQueryMonitor(db.logger, db.config.SlowQueryThreshold)
		monitor.MonitorQuery(query, args, start, err)
	}

	// Record metrics
	if db.metrics != nil {
		db.metrics.RecordDBQuery("query", extractTableName(query), duration, err)
	}

	return rows, err
}

// QueryRowContext executes a query that returns at most one row
func (db *Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := db.DB.QueryRowContext(ctx, query, args...)

	// For QueryRowContext, we can't easily monitor the error since it's deferred
	// We'll monitor the execution time only
	if db.config.SlowQueryThreshold > 0 {
		duration := time.Since(start)
		if duration >= db.config.SlowQueryThreshold {
			db.logger.Warn("Slow query detected (QueryRowContext)",
				zap.String("query", query),
				zap.Duration("duration", duration),
				zap.Duration("threshold", db.config.SlowQueryThreshold),
				zap.Time("timestamp", start))
		}
	}

	return row
}

// GetContext gets a single row and scans it into dest
func (db *Database) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := db.DB.GetContext(ctx, dest, query, args...)

	if db.config.SlowQueryThreshold > 0 {
		monitor := NewQueryMonitor(db.logger, db.config.SlowQueryThreshold)
		monitor.MonitorQuery(query, args, start, err)
	}

	return err
}

// SelectContext gets multiple rows and scans them into dest
func (db *Database) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := db.DB.SelectContext(ctx, dest, query, args...)

	if db.config.SlowQueryThreshold > 0 {
		monitor := NewQueryMonitor(db.logger, db.config.SlowQueryThreshold)
		monitor.MonitorQuery(query, args, start, err)
	}

	return err
}
