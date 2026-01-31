package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"

	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
)

// ReadWriteDatabase manages read/write database splitting
type ReadWriteDatabase struct {
	writeDB      *Database
	readDBs      []*Database
	logger       *logger.Logger
	loadBalancer LoadBalancer
	mu           sync.RWMutex
}

// LoadBalancer handles load balancing across read databases
type LoadBalancer interface {
	Next() *Database
}

// RoundRobinLoadBalancer implements round-robin load balancing
type RoundRobinLoadBalancer struct {
	dbs   []*Database
	index int
	mu    sync.Mutex
}

// Next returns the next database in round-robin fashion
func (lb *RoundRobinLoadBalancer) Next() *Database {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.dbs) == 0 {
		return nil
	}

	db := lb.dbs[lb.index]
	lb.index = (lb.index + 1) % len(lb.dbs)
	return db
}

// RandomLoadBalancer implements random load balancing
type RandomLoadBalancer struct {
	dbs []*Database
}

// Next returns a random database
func (lb *RandomLoadBalancer) Next() *Database {
	if len(lb.dbs) == 0 {
		return nil
	}
	return lb.dbs[rand.Intn(len(lb.dbs))]
}

// ReadWriteConfig holds configuration for read/write database setup
type ReadWriteConfig struct {
	WriteConfig *Config
	ReadConfigs []*Config
}

// NewReadWriteConfig creates configuration for read/write database setup
func NewReadWriteConfig(cfg *config.Config) *ReadWriteConfig {
	writeConfig := &Config{
		URL:                cfg.DBURL,
		MaxOpenConns:       cfg.DBMaxOpenConns,
		MaxIdleConns:       cfg.DBMaxIdleConns,
		ConnMaxLifetime:    cfg.DBConnMaxLifetime,
		ConnMaxIdleTime:    cfg.DBConnMaxIdleTime,
		RetryAttempts:      cfg.DBRetryAttempts,
		RetryDelay:         cfg.DBRetryDelay,
		SlowQueryThreshold: cfg.DBSlowQueryThreshold,
	}

	var readConfigs []*Config
	for _, url := range cfg.ReadDBURLs {
		readConfig := &Config{
			URL:                url,
			MaxOpenConns:       cfg.DBMaxOpenConns, // Use same settings as write DB
			MaxIdleConns:       cfg.DBMaxIdleConns,
			ConnMaxLifetime:    cfg.DBConnMaxLifetime,
			ConnMaxIdleTime:    cfg.DBConnMaxIdleTime,
			RetryAttempts:      cfg.DBRetryAttempts,
			RetryDelay:         cfg.DBRetryDelay,
			SlowQueryThreshold: cfg.DBSlowQueryThreshold,
		}
		readConfigs = append(readConfigs, readConfig)
	}

	return &ReadWriteConfig{
		WriteConfig: writeConfig,
		ReadConfigs: readConfigs,
	}
}

// NewReadWriteDatabase creates a new read/write database manager
func NewReadWriteDatabase(rwConfig *ReadWriteConfig, log *logger.Logger, metrics *metrics.Metrics) (*ReadWriteDatabase, error) {
	log = log.Named("read-write-db")

	// Initialize write database
	writeDB, err := New(rwConfig.WriteConfig, log.Named("write"), metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create write database: %w", err)
	}

	rwDB := &ReadWriteDatabase{
		writeDB: writeDB,
		logger:  log,
	}

	// Initialize read databases if configured
	if len(rwConfig.ReadConfigs) > 0 {
		var readDBs []*Database
		for i, readConfig := range rwConfig.ReadConfigs {
			readDB, err := New(readConfig, log.Named(fmt.Sprintf("read-%d", i)), metrics)
			if err != nil {
				log.Warn("Failed to create read database",
					zap.Int("index", i),
					zap.String("url", readConfig.URL),
					zap.Error(err))
				continue
			}
			readDBs = append(readDBs, readDB)
		}

		if len(readDBs) > 0 {
			rwDB.readDBs = readDBs
			// Use round-robin load balancer by default
			rwDB.loadBalancer = &RoundRobinLoadBalancer{dbs: readDBs}
			log.Info("Initialized read/write database splitting",
				zap.Int("read_databases", len(readDBs)))
		} else {
			log.Warn("No read databases available, falling back to write database for reads")
		}
	} else {
		log.Warn("No read databases configured, using write database for all operations")
	}

	return rwDB, nil
}

// ReadDB returns a database connection for read operations
func (rw *ReadWriteDatabase) ReadDB() *Database {
	rw.mu.RLock()
	defer rw.mu.RUnlock()

	if rw.loadBalancer != nil {
		if db := rw.loadBalancer.Next(); db != nil {
			return db
		}
	}

	// Fallback to write database if no read databases available
	return rw.writeDB
}

// WriteDB returns the database connection for write operations
func (rw *ReadWriteDatabase) WriteDB() *Database {
	return rw.writeDB
}

// Close closes all database connections
func (rw *ReadWriteDatabase) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	var errors []error

	// Close write database
	if err := rw.writeDB.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close write database: %w", err))
	}

	// Close read databases
	for i, db := range rw.readDBs {
		if err := db.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close read database %d: %w", i, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing databases: %v", errors)
	}

	rw.logger.Info("Closed all database connections")
	return nil
}

// HealthCheck performs health checks on all databases
func (rw *ReadWriteDatabase) HealthCheck(ctx context.Context) error {
	rw.mu.RLock()
	defer rw.mu.RUnlock()

	// Check write database
	if err := rw.writeDB.HealthCheck(ctx); err != nil {
		return fmt.Errorf("write database health check failed: %w", err)
	}

	// Check read databases
	for i, db := range rw.readDBs {
		if err := db.HealthCheck(ctx); err != nil {
			rw.logger.Warn("Read database health check failed",
				zap.Int("index", i),
				zap.Error(err))
			// Don't fail completely if read database is down
		}
	}

	return nil
}

// GetStats returns statistics for all databases
func (rw *ReadWriteDatabase) GetStats() map[string]interface{} {
	rw.mu.RLock()
	defer rw.mu.RUnlock()

	stats := make(map[string]interface{})

	// Write database stats
	stats["write"] = rw.writeDB.GetStats()

	// Read database stats
	readStats := make([]Stats, len(rw.readDBs))
	for i, db := range rw.readDBs {
		readStats[i] = db.GetStats()
	}
	stats["read"] = readStats

	return stats
}

// IsHealthy checks if the database setup is healthy
func (rw *ReadWriteDatabase) IsHealthy(ctx context.Context) bool {
	return rw.HealthCheck(ctx) == nil
}

// ExecContext executes a write query (INSERT, UPDATE, DELETE)
func (rw *ReadWriteDatabase) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return rw.WriteDB().ExecContext(ctx, query, args...)
}

// QueryContext executes a read query (SELECT) with load balancing
func (rw *ReadWriteDatabase) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return rw.ReadDB().QueryContext(ctx, query, args...)
}

// QueryRowContext executes a read query that returns at most one row
func (rw *ReadWriteDatabase) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return rw.ReadDB().QueryRowContext(ctx, query, args...)
}

// GetContext gets a single row from read database and scans it into dest
func (rw *ReadWriteDatabase) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return rw.ReadDB().GetContext(ctx, dest, query, args...)
}

// SelectContext gets multiple rows from read database and scans them into dest
func (rw *ReadWriteDatabase) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return rw.ReadDB().SelectContext(ctx, dest, query, args...)
}
