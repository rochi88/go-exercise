package reportScheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/database"
	"go-boilerplate/internal/shared/cache"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
)

// DatabaseHealthCheckJob represents a database health check job that runs periodically
type DatabaseHealthCheckJob struct {
	db         *database.ReadWriteDatabase
	redis      *cache.Redis
	logger     *logger.Logger
	metrics    *metrics.Metrics
	historyMgr *HistoricalDataManager
}

// HistoricalDataManager manages storage and retrieval of health check history
type HistoricalDataManager struct {
	filePath     string
	maxRecords   int
	rotationDays int
	mutex        sync.RWMutex
	logger       *zap.Logger
}

// HealthCheckResult represents a single health check execution result
type HealthCheckResult struct {
	Timestamp          time.Time          `json:"timestamp"`
	JobName            string             `json:"job_name"`
	Success            bool               `json:"success"`
	Duration           time.Duration      `json:"duration"`
	Error              string             `json:"error,omitempty"`
	DatabaseStats      DatabaseStats      `json:"database_stats,omitempty"`
	RedisStats         RedisStats         `json:"redis_stats,omitempty"`
	ConnectionPoolInfo ConnectionPoolInfo `json:"connection_pool_info,omitempty"`
}

// DatabaseStats holds database-specific statistics
type DatabaseStats struct {
	WriteConnections    int `json:"write_connections"`
	WriteMaxConnections int `json:"write_max_connections"`
	WriteIdle           int `json:"write_idle"`
	ReadConnections     int `json:"read_connections,omitempty"`
	ReadMaxConnections  int `json:"read_max_connections,omitempty"`
	ReadIdle            int `json:"read_idle,omitempty"`
}

// RedisStats holds Redis-specific statistics
type RedisStats struct {
	PingSuccess bool   `json:"ping_success"`
	Info        string `json:"info,omitempty"`
}

// ConnectionPoolInfo holds connection pool analysis
type ConnectionPoolInfo struct {
	WritePoolUtilization float64 `json:"write_pool_utilization"`
	ReadPoolUtilization  float64 `json:"read_pool_utilization,omitempty"`
	WritePoolStatus      string  `json:"write_pool_status"`
	ReadPoolStatus       string  `json:"read_pool_status,omitempty"`
}

// HealthSummary provides aggregated health check statistics
type HealthSummary struct {
	JobName          string        `json:"job_name"`
	TimeRange        string        `json:"time_range"`
	TotalChecks      int           `json:"total_checks"`
	SuccessfulChecks int           `json:"successful_checks"`
	FailedChecks     int           `json:"failed_checks"`
	SuccessRate      float64       `json:"success_rate"`
	AverageDuration  time.Duration `json:"average_duration"`
	MinDuration      time.Duration `json:"min_duration"`
	MaxDuration      time.Duration `json:"max_duration"`
}

// NewDatabaseHealthCheckJob creates a new database health check job
func NewDatabaseHealthCheckJob(db *database.ReadWriteDatabase, redis *cache.Redis, logger *logger.Logger, metrics *metrics.Metrics) *DatabaseHealthCheckJob {
	// Initialize historical data manager
	historyFile := "logs/db-health.json"
	historyMgr := NewHistoricalDataManager(historyFile, 1000, logger.Logger) // Keep last 1000 records

	return &DatabaseHealthCheckJob{
		db:         db,
		redis:      redis,
		logger:     logger.Named("db-health-check-job"),
		metrics:    metrics,
		historyMgr: historyMgr,
	}
}

// Name returns the name of the job
func (j *DatabaseHealthCheckJob) Name() string {
	return "database-health-check"
}

// Schedule returns the cron schedule expression (runs every minute)
func (j *DatabaseHealthCheckJob) Schedule() string {
	return "* * * * *" // Every minute
}

// Description returns a description of what the job does
func (j *DatabaseHealthCheckJob) Description() string {
	return "Performs periodic health checks on database connectivity and performance"
}

// Timeout returns the maximum time the job should run
func (j *DatabaseHealthCheckJob) Timeout() time.Duration {
	return 2 * time.Minute
}

// Run executes the database health check job
func (j *DatabaseHealthCheckJob) Run(ctx context.Context) error {
	startTime := time.Now()
	j.logger.Info("Starting database health check job")

	result := HealthCheckResult{
		Timestamp: startTime,
		JobName:   j.Name(),
		Success:   true,
	}

	// Perform database health check
	dbStats, dbErr := j.checkDatabaseHealth(ctx)
	if dbErr != nil {
		j.logger.Error("Database health check failed", zap.Error(dbErr))
		result.Success = false
		result.Error = dbErr.Error()
	} else {
		result.DatabaseStats = dbStats.DatabaseStats
		result.ConnectionPoolInfo = dbStats.ConnectionPoolInfo
	}

	// Perform Redis health check
	redisStats, redisErr := j.checkRedisHealth(ctx)
	if redisErr != nil {
		j.logger.Error("Redis health check failed", zap.Error(redisErr))
		result.Success = false
		if result.Error != "" {
			result.Error += "; " + redisErr.Error()
		} else {
			result.Error = redisErr.Error()
		}
	} else {

		//delete the info field to reduce size
		redisStats.Info = ""
		if redisStats.PingSuccess {
			redisStats.Info = "Ping successful"
		}

		result.RedisStats = redisStats
	}

	// Calculate duration
	result.Duration = time.Since(startTime)

	// Save result to history
	if err := j.historyMgr.SaveResult(result); err != nil {
		j.logger.Error("Failed to save health check result to history", zap.Error(err))
		// Don't fail the job if we can't save history
	}

	// Periodically cleanup old data (once per day)
	if startTime.Hour() == 2 && startTime.Minute() < 5 { // Run cleanup around 2 AM
		if err := j.historyMgr.CleanupOldData(7 * 24 * time.Hour); err != nil { // Keep 7 days of data
			j.logger.Warn("Failed to cleanup old health check data", zap.Error(err))
		} else {
			j.logger.Info("Cleaned up old health check data")
		}
	}

	// Log final result
	if result.Success {
		j.logger.Info("Database health check job completed successfully",
			zap.Duration("duration", result.Duration))
	} else {
		j.logger.Error("Database health check job failed",
			zap.Duration("duration", result.Duration),
			zap.String("error", result.Error))
	}

	return nil
}

// DatabaseHealthResult holds the result of database health check
type DatabaseHealthResult struct {
	DatabaseStats      DatabaseStats
	ConnectionPoolInfo ConnectionPoolInfo
}

// checkDatabaseHealth performs health checks on the database
func (j *DatabaseHealthCheckJob) checkDatabaseHealth(ctx context.Context) (DatabaseHealthResult, error) {
	result := DatabaseHealthResult{}

	// Check database connectivity
	if err := j.db.HealthCheck(ctx); err != nil {
		return result, fmt.Errorf("database connectivity check failed: %w", err)
	}

	// Get connection pool statistics
	stats := j.db.GetStats()
	writeStats := stats["write"].(database.Stats)

	// Populate database stats
	result.DatabaseStats = DatabaseStats{
		WriteConnections:    writeStats.OpenConnections,
		WriteMaxConnections: writeStats.MaxOpenConnections,
		WriteIdle:           writeStats.Idle,
	}

	// Calculate write pool utilization
	if writeStats.MaxOpenConnections > 0 {
		result.ConnectionPoolInfo.WritePoolUtilization = float64(writeStats.OpenConnections) / float64(writeStats.MaxOpenConnections) * 100
		if result.ConnectionPoolInfo.WritePoolUtilization >= 90 {
			result.ConnectionPoolInfo.WritePoolStatus = "high"
		} else if result.ConnectionPoolInfo.WritePoolUtilization >= 70 {
			result.ConnectionPoolInfo.WritePoolStatus = "moderate"
		} else {
			result.ConnectionPoolInfo.WritePoolStatus = "normal"
		}
	}

	j.logger.Info("Database health check passed",
		zap.Int("write_open_connections", writeStats.OpenConnections),
		zap.Int("write_max_open_connections", writeStats.MaxOpenConnections),
		zap.Int("write_in_use", writeStats.InUse),
		zap.Int("write_idle", writeStats.Idle),
		zap.Float64("write_pool_utilization", result.ConnectionPoolInfo.WritePoolUtilization))

	// Check for connection pool issues
	if writeStats.OpenConnections >= writeStats.MaxOpenConnections {
		j.logger.Warn("Database connection pool is at maximum capacity",
			zap.Int("open_connections", writeStats.OpenConnections),
			zap.Int("max_connections", writeStats.MaxOpenConnections))
	}

	// Check read connections if available
	if readStatsInterface, ok := stats["read"]; ok {
		if readStats, ok := readStatsInterface.([]database.Stats); ok && len(readStats) > 0 {
			result.DatabaseStats.ReadConnections = readStats[0].OpenConnections
			result.DatabaseStats.ReadMaxConnections = readStats[0].MaxOpenConnections
			result.DatabaseStats.ReadIdle = readStats[0].Idle

			// Calculate read pool utilization
			if readStats[0].MaxOpenConnections > 0 {
				result.ConnectionPoolInfo.ReadPoolUtilization = float64(readStats[0].OpenConnections) / float64(readStats[0].MaxOpenConnections) * 100
				if result.ConnectionPoolInfo.ReadPoolUtilization >= 90 {
					result.ConnectionPoolInfo.ReadPoolStatus = "high"
				} else if result.ConnectionPoolInfo.ReadPoolUtilization >= 70 {
					result.ConnectionPoolInfo.ReadPoolStatus = "moderate"
				} else {
					result.ConnectionPoolInfo.ReadPoolStatus = "normal"
				}
			}

			j.logger.Info("Read database pools health check",
				zap.Int("read_pools_count", len(readStats)),
				zap.Float64("read_pool_utilization", result.ConnectionPoolInfo.ReadPoolUtilization))

			for i, readStat := range readStats {
				j.logger.Info("Read pool statistics",
					zap.Int("pool_index", i),
					zap.Int("open_connections", readStat.OpenConnections),
					zap.Int("max_open_connections", readStat.MaxOpenConnections),
					zap.Int("in_use", readStat.InUse),
					zap.Int("idle", readStat.Idle))
			}
		}
	}

	return result, nil
}

// checkRedisHealth performs health checks on Redis
func (j *DatabaseHealthCheckJob) checkRedisHealth(ctx context.Context) (RedisStats, error) {
	stats := RedisStats{PingSuccess: false}

	// Ping Redis
	if err := j.redis.Client.Ping(ctx).Err(); err != nil {
		return stats, fmt.Errorf("redis ping failed: %w", err)
	}

	stats.PingSuccess = true

	// Get Redis info
	info, err := j.redis.Client.Info(ctx).Result()
	if err != nil {
		j.logger.Warn("Failed to get Redis info", zap.Error(err))
		stats.Info = "Failed to retrieve Redis info"
	} else {
		stats.Info = info
		j.logger.Info("Redis health check passed")
	}

	return stats, nil
}

// NewHistoricalDataManager creates a new historical data manager
func NewHistoricalDataManager(filePath string, maxRecords int, logger *zap.Logger) *HistoricalDataManager {
	return &HistoricalDataManager{
		filePath:     filePath,
		maxRecords:   maxRecords,
		rotationDays: 7, // Rotate every 7 days
		logger:       logger.Named("historical-data-manager"),
	}
}

// SaveResult saves a health check result to the historical data file
func (h *HistoricalDataManager) SaveResult(result HealthCheckResult) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Check if file rotation is needed
	if err := h.checkAndRotateFile(); err != nil {
		h.logger.Error("Failed to check/rotate file", zap.Error(err))
		// Continue with saving even if rotation fails
	}

	// Ensure directory exists
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Load existing results
	results, err := h.loadResults()
	if err != nil {
		h.logger.Warn("Failed to load existing results, starting fresh", zap.Error(err))
		results = []HealthCheckResult{}
	}

	// Add new result
	results = append(results, result)

	// Keep only the most recent records
	if len(results) > h.maxRecords {
		results = results[len(results)-h.maxRecords:]
	}

	// Save to file
	if err := h.saveResults(results); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	h.logger.Debug("Saved health check result",
		zap.String("job_name", result.JobName),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration))

	return nil
}

// checkAndRotateFile checks if file rotation is needed and performs rotation
func (h *HistoricalDataManager) checkAndRotateFile() error {
	// Check if file exists
	fileInfo, err := os.Stat(h.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist, no rotation needed
	}
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if file is older than rotation period
	fileAge := time.Since(fileInfo.ModTime())
	if fileAge < time.Duration(h.rotationDays)*24*time.Hour {
		return nil // File is not old enough for rotation
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := h.filePath + "." + timestamp + ".backup"

	// Move current file to backup
	if err := os.Rename(h.filePath, backupPath); err != nil {
		return fmt.Errorf("failed to rotate file: %w", err)
	}

	h.logger.Info("Rotated health check history file",
		zap.String("original", h.filePath),
		zap.String("backup", backupPath),
		zap.Duration("file_age", fileAge))

	return nil
}

// GetResults retrieves historical health check results
func (h *HistoricalDataManager) GetResults(jobName string, limit int) ([]HealthCheckResult, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	results, err := h.loadResults()
	if err != nil {
		return nil, fmt.Errorf("failed to load results: %w", err)
	}

	// Filter by job name if specified
	if jobName != "" {
		filtered := []HealthCheckResult{}
		for _, result := range results {
			if result.JobName == jobName {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[len(results)-limit:]
	}

	return results, nil
}

// GetLatestResult gets the most recent result for a specific job
func (h *HistoricalDataManager) GetLatestResult(jobName string) (*HealthCheckResult, error) {
	results, err := h.GetResults(jobName, 1)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for job %s", jobName)
	}

	return &results[0], nil
}

// GetHealthSummary provides a summary of health check performance
func (h *HistoricalDataManager) GetHealthSummary(jobName string, hours int) (HealthSummary, error) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)

	results, err := h.GetResults(jobName, 0)
	if err != nil {
		return HealthSummary{}, err
	}

	summary := HealthSummary{
		JobName:          jobName,
		TimeRange:        fmt.Sprintf("Last %d hours", hours),
		TotalChecks:      0,
		SuccessfulChecks: 0,
		FailedChecks:     0,
		AverageDuration:  0,
		MinDuration:      0,
		MaxDuration:      0,
	}

	var totalDuration time.Duration
	var durations []time.Duration

	for _, result := range results {
		if result.Timestamp.Before(cutoff) {
			continue
		}

		summary.TotalChecks++
		if result.Success {
			summary.SuccessfulChecks++
		} else {
			summary.FailedChecks++
		}

		totalDuration += result.Duration
		durations = append(durations, result.Duration)
	}

	if summary.TotalChecks > 0 {
		summary.AverageDuration = totalDuration / time.Duration(summary.TotalChecks)
		summary.SuccessRate = float64(summary.SuccessfulChecks) / float64(summary.TotalChecks) * 100

		// Find min/max durations
		minDuration := durations[0]
		maxDuration := durations[0]
		for _, d := range durations {
			if d < minDuration {
				minDuration = d
			}
			if d > maxDuration {
				maxDuration = d
			}
		}
		summary.MinDuration = minDuration
		summary.MaxDuration = maxDuration
	}

	return summary, nil
}

// loadResults loads historical results from file
func (h *HistoricalDataManager) loadResults() ([]HealthCheckResult, error) {
	if _, err := os.Stat(h.filePath); os.IsNotExist(err) {
		return []HealthCheckResult{}, nil
	}

	data, err := os.ReadFile(h.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		return []HealthCheckResult{}, nil
	}

	var results []HealthCheckResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return results, nil
}

// saveResults saves results to file
func (h *HistoricalDataManager) saveResults(results []HealthCheckResult) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(h.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// CleanupOldData removes data older than specified duration
func (h *HistoricalDataManager) CleanupOldData(maxAge time.Duration) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	results, err := h.loadResults()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	filtered := []HealthCheckResult{}

	for _, result := range results {
		if result.Timestamp.After(cutoff) {
			filtered = append(filtered, result)
		}
	}

	if len(filtered) != len(results) {
		h.logger.Info("Cleaned up old health check data",
			zap.Int("removed", len(results)-len(filtered)),
			zap.Int("remaining", len(filtered)))

		return h.saveResults(filtered)
	}

	return nil
}
