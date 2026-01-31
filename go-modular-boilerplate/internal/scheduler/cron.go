package scheduler

import (
	"context"
	"time"

	cron "github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"go-boilerplate/internal/database"
	services "go-boilerplate/internal/scheduler/services"
	"go-boilerplate/internal/shared/cache"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
)

// Job represents a cron job that can be scheduled
type Job interface {
	// Name returns the name of the job
	Name() string

	// Schedule returns the cron schedule expression
	Schedule() string

	// Run executes the job
	Run(ctx context.Context) error

	// Description returns a description of what the job does
	Description() string

	// Timeout returns the maximum time the job should run
	Timeout() time.Duration
}

// JobResult represents the result of a job execution
type JobResult struct {
	JobName   string
	Success   bool
	Duration  time.Duration
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

// Scheduler manages cron jobs
type Scheduler struct {
	cron    *cron.Cron
	db      *database.ReadWriteDatabase
	redis   *cache.Redis
	logger  *logger.Logger
	metrics *metrics.Metrics
	jobs    []Job
	jobIDs  map[string]cron.EntryID
}

// NewScheduler creates a new scheduler instance
func NewScheduler(c *cron.Cron, db *database.ReadWriteDatabase, redis *cache.Redis, logger *logger.Logger, metrics *metrics.Metrics) *Scheduler {
	return &Scheduler{
		cron:    c,
		db:      db,
		redis:   redis,
		logger:  logger.Named("scheduler"),
		metrics: metrics,
		jobs:    make([]Job, 0),
		jobIDs:  make(map[string]cron.EntryID),
	}
}

// RegisterJob registers a single job with the scheduler
func (s *Scheduler) RegisterJob(job Job) error {
	s.jobs = append(s.jobs, job)

	// Wrap the job execution with context and timeout
	wrappedJob := s.wrapJob(job)

	// Schedule the job
	id, err := s.cron.AddFunc(job.Schedule(), wrappedJob)
	if err != nil {
		s.logger.Error("Failed to schedule job",
			zap.String("job_name", job.Name()),
			zap.String("schedule", job.Schedule()),
			zap.Error(err))
		return err
	}

	s.jobIDs[job.Name()] = id
	s.logger.Info("Job registered successfully",
		zap.String("job_name", job.Name()),
		zap.String("schedule", job.Schedule()),
		zap.String("description", job.Description()))

	return nil
}

// RegisterJobs registers all predefined jobs
func (s *Scheduler) RegisterJobs() error {
	// Register example jobs - these would be replaced with actual job implementations
	jobs := []Job{
		services.NewDatabaseHealthCheckJob(s.db, s.redis, s.logger, s.metrics),
		services.NewExampleCleanupJob(s.db, s.redis, s.logger, s.metrics),
		services.NewJWKKeyRotationJob(s.logger),
	}

	for _, job := range jobs {
		if err := s.RegisterJob(job); err != nil {
			return err
		}
	}

	s.logger.Info("All jobs registered successfully", zap.Int("job_count", len(jobs)))
	return nil
}

// wrapJob wraps a job with context, timeout, and error handling
func (s *Scheduler) wrapJob(job Job) func() {
	return func() {
		startTime := time.Now()

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), job.Timeout())
		defer cancel()

		s.logger.Info("Starting job execution",
			zap.String("job_name", job.Name()),
			zap.Time("start_time", startTime))

		// Execute the job
		err := job.Run(ctx)

		endTime := time.Now()
		duration := endTime.Sub(startTime)

		result := JobResult{
			JobName:   job.Name(),
			Success:   err == nil,
			Duration:  duration,
			Error:     err,
			StartTime: startTime,
			EndTime:   endTime,
		}

		// Log the result
		if result.Success {
			s.logger.Info("Job completed successfully",
				zap.String("job_name", result.JobName),
				zap.Duration("duration", result.Duration))
		} else {
			s.logger.Error("Job failed",
				zap.String("job_name", result.JobName),
				zap.Duration("duration", result.Duration),
				zap.Error(result.Error))
		}
	}
}

// GetRegisteredJobs returns a list of all registered job names
func (s *Scheduler) GetRegisteredJobs() []string {
	names := make([]string, 0, len(s.jobs))
	for _, job := range s.jobs {
		names = append(names, job.Name())
	}
	return names
}

// Stop stops the scheduler and all running jobs
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping scheduler")
	s.cron.Stop()
}
