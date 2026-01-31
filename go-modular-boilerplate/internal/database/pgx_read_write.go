package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/database/sqlc"
	"go-boilerplate/internal/shared/logger"
)

// PgxReadWriteDB manages pgx connection pools for read/write splitting
type PgxReadWriteDB struct {
	writePool *pgxpool.Pool
	readPool  *pgxpool.Pool
	writeDB   *sqlx.DB
	readDB    *sqlx.DB
	queries   *sqlc.Queries
	logger    *logger.Logger
}

// NewPgxReadWriteDB creates a new pgx-based read/write database manager
func NewPgxReadWriteDB(cfg *config.Config, log *logger.Logger) (*PgxReadWriteDB, error) {
	ctx := context.Background()

	// Configure write pool
	writeConfig, err := pgxpool.ParseConfig(cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse write database URL: %w", err)
	}
	configurePool(writeConfig, cfg)

	writePool, err := pgxpool.NewWithConfig(ctx, writeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create write pool: %w", err)
	}

	// Configure read pool (use read URLs if available, otherwise fall back to write URL)
	var readPoolConfig *pgxpool.Config
	if len(cfg.ReadDBURLs) > 0 {
		readPoolConfig, err = pgxpool.ParseConfig(cfg.ReadDBURLs[0]) // Use first read DB
		if err != nil {
			writePool.Close()
			return nil, fmt.Errorf("failed to parse read database URL: %w", err)
		}
	} else {
		// Use write DB config for read operations
		readPoolConfig = writeConfig.Copy()
	}
	configurePool(readPoolConfig, cfg)

	readPool, err := pgxpool.NewWithConfig(ctx, readPoolConfig)
	if err != nil {
		writePool.Close()
		return nil, fmt.Errorf("failed to create read pool: %w", err)
	}

	// Create sqlx wrappers for backward compatibility
	writeDB := sqlx.NewDb(stdlib.OpenDBFromPool(writePool), "pgx")
	readDB := sqlx.NewDb(stdlib.OpenDBFromPool(readPool), "pgx")

	// Create sqlc queries instance
	queries := sqlc.New()

	db := &PgxReadWriteDB{
		writePool: writePool,
		readPool:  readPool,
		writeDB:   writeDB,
		readDB:    readDB,
		queries:   queries,
		logger:    log.Named("pgx-database"),
	}

	// Test connections
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping databases: %w", err)
	}

	db.logger.Info("pgx database connections established",
		zap.String("write_host", writeConfig.ConnConfig.Host),
		zap.String("read_host", readPoolConfig.ConnConfig.Host))

	return db, nil
}

func configurePool(config *pgxpool.Config, cfg *config.Config) {
	config.MaxConns = int32(cfg.DBMaxOpenConns)
	config.MinConns = int32(cfg.DBMaxIdleConns)
	config.MaxConnLifetime = cfg.DBConnMaxLifetime
	config.MaxConnIdleTime = cfg.DBConnMaxIdleTime

	// Configure connection settings
	config.ConnConfig.RuntimeParams = map[string]string{
		"application_name": "go-boilerplate",
		"timezone":         "UTC",
	}
}

// WritePool returns the write connection pool for direct pgx usage
func (db *PgxReadWriteDB) WritePool() *pgxpool.Pool {
	return db.writePool
}

// ReadPool returns the read connection pool for direct pgx usage
func (db *PgxReadWriteDB) ReadPool() *pgxpool.Pool {
	return db.readPool
}

// WriteDB returns the write database connection (sqlx wrapper for backward compatibility)
func (db *PgxReadWriteDB) WriteDB() *sqlx.DB {
	return db.writeDB
}

// ReadDB returns the read database connection (sqlx wrapper for backward compatibility)
func (db *PgxReadWriteDB) ReadDB() *sqlx.DB {
	return db.readDB
}

// Queries returns the sqlc queries instance
func (db *PgxReadWriteDB) Queries() *sqlc.Queries {
	return db.queries
}

// QueriesWithTx returns sqlc queries instance with transaction
func (db *PgxReadWriteDB) QueriesWithTx(tx pgx.Tx) *sqlc.Queries {
	return db.queries // sqlc queries work with any DBTX interface, including pgx.Tx
}

// BeginTx starts a new transaction on the write pool
func (db *PgxReadWriteDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.writePool.Begin(ctx)
}

// Ping tests both read and write connections
func (db *PgxReadWriteDB) Ping(ctx context.Context) error {
	if err := db.writePool.Ping(ctx); err != nil {
		return fmt.Errorf("write pool ping failed: %w", err)
	}

	if err := db.readPool.Ping(ctx); err != nil {
		return fmt.Errorf("read pool ping failed: %w", err)
	}

	return nil
}

// Close closes all database connections
func (db *PgxReadWriteDB) Close() {
	if db.writePool != nil {
		db.writePool.Close()
	}
	if db.readPool != nil {
		db.readPool.Close()
	}
	if db.writeDB != nil {
		db.writeDB.Close()
	}
	if db.readDB != nil {
		db.readDB.Close()
	}

	db.logger.Info("pgx database connections closed")
}

// Stats returns database pool statistics
func (db *PgxReadWriteDB) Stats() (writeStats, readStats *pgxpool.Stat) {
	return db.writePool.Stat(), db.readPool.Stat()
}

// Health checks the health of both database connections
func (db *PgxReadWriteDB) Health(ctx context.Context) map[string]interface{} {
	result := make(map[string]interface{})

	// Check write database
	writeCtx, writeCancel := context.WithTimeout(ctx, 5*time.Second)
	defer writeCancel()

	writeErr := db.writePool.Ping(writeCtx)
	writeStats := db.writePool.Stat()

	result["write"] = map[string]interface{}{
		"status":            writeErr == nil,
		"error":             writeErr,
		"total_connections": writeStats.TotalConns(),
		"idle_connections":  writeStats.IdleConns(),
		"used_connections":  writeStats.AcquiredConns(),
	}

	// Check read database
	readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
	defer readCancel()

	readErr := db.readPool.Ping(readCtx)
	readStats := db.readPool.Stat()

	result["read"] = map[string]interface{}{
		"status":            readErr == nil,
		"error":             readErr,
		"total_connections": readStats.TotalConns(),
		"idle_connections":  readStats.IdleConns(),
		"used_connections":  readStats.AcquiredConns(),
	}

	return result
}

// ExecInTx executes a function within a transaction
func (db *PgxReadWriteDB) ExecInTx(ctx context.Context, fn func(pgx.Tx, *sqlc.Queries) error) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := db.QueriesWithTx(tx)
	if err := fn(tx, queries); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
