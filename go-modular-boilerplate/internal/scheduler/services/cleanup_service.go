package reportScheduler

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/database"
	"go-boilerplate/internal/shared/cache"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
)

// ExampleCleanupJob represents a cleanup job that runs periodically
type ExampleCleanupJob struct {
	db      *database.ReadWriteDatabase
	redis   *cache.Redis
	logger  *logger.Logger
	metrics *metrics.Metrics
}

// NewExampleCleanupJob creates a new example cleanup job
func NewExampleCleanupJob(db *database.ReadWriteDatabase, redis *cache.Redis, logger *logger.Logger, metrics *metrics.Metrics) *ExampleCleanupJob {
	return &ExampleCleanupJob{
		db:      db,
		redis:   redis,
		logger:  logger.Named("cleanup-job"),
		metrics: metrics,
	}
}

// Name returns the name of the job
func (j *ExampleCleanupJob) Name() string {
	return "example-cleanup"
}

// Schedule returns the cron schedule expression (runs every hour)
func (j *ExampleCleanupJob) Schedule() string {
	return "0 * * * *" // Every hour at minute 0
}

// Description returns a description of what the job does
func (j *ExampleCleanupJob) Description() string {
	return "Cleans up old data and temporary files"
}

// Timeout returns the maximum time the job should run
func (j *ExampleCleanupJob) Timeout() time.Duration {
	return 30 * time.Minute
}

// Run executes the cleanup job
func (j *ExampleCleanupJob) Run(ctx context.Context) error {
	j.logger.Info("Starting cleanup job")

	// Example: Clean up old sessions from database
	if err := j.cleanupOldSessions(ctx); err != nil {
		j.logger.Error("Failed to cleanup old sessions", zap.Error(err))
		return fmt.Errorf("cleanup old sessions: %w", err)
	}

	// Example: Clean up old cache keys
	if err := j.cleanupOldCacheKeys(ctx); err != nil {
		j.logger.Error("Failed to cleanup old cache keys", zap.Error(err))
		return fmt.Errorf("cleanup old cache keys: %w", err)
	}

	// Example: Clean up temporary files (if any)
	if err := j.cleanupTempFiles(ctx); err != nil {
		j.logger.Error("Failed to cleanup temp files", zap.Error(err))
		return fmt.Errorf("cleanup temp files: %w", err)
	}

	j.logger.Info("Cleanup job completed successfully")
	return nil
}

// cleanupOldSessions removes sessions older than 30 days
func (j *ExampleCleanupJob) cleanupOldSessions(ctx context.Context) error {
	query := `
		UPDATE sessions
		SET is_active = FALSE
		WHERE created_at < $1
	`

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	result, err := j.db.ExecContext(ctx, query, thirtyDaysAgo)
	if err != nil {
		return fmt.Errorf("delete old sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	j.logger.Info("Cleaned up old sessions",
		zap.Int64("rows_deleted", rowsAffected))

	return nil
}

// cleanupOldCacheKeys removes expired cache keys
func (j *ExampleCleanupJob) cleanupOldCacheKeys(ctx context.Context) error {

	// Delete keys that match a pattern and are expired
	pattern := "temp:*"
	keys, err := j.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("scan cache keys: %w", err)
	}

	var deletedCount int64
	for _, key := range keys {
		// Check if key is expired (TTL <= 0)
		ttl := j.redis.Client.TTL(ctx, key)
		if ttl.Val() <= 0 {
			if err := j.redis.Client.Del(ctx, key).Err(); err != nil {
				j.logger.Warn("Failed to delete expired cache key",
					zap.String("key", key), zap.Error(err))
				continue
			}
			deletedCount++
		}
	}

	j.logger.Info("Cleaned up expired cache keys",
		zap.Int64("keys_deleted", deletedCount),
		zap.String("pattern", pattern))

	return nil
}

// cleanupTempFiles removes temporary files older than 24 hours
func (j *ExampleCleanupJob) cleanupTempFiles(ctx context.Context) error {

	const tempPath = "./tmp"
	os.RemoveAll(tempPath)

	j.logger.Info("Temp file cleanup completed")
	return nil
}
