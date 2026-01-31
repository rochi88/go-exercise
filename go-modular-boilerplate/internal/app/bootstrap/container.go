package bootstrap

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/database"
	authHttp "go-boilerplate/internal/pkg/auth/delivery/http"
	authRepository "go-boilerplate/internal/pkg/auth/repository"
	authService "go-boilerplate/internal/pkg/auth/service"
	"go-boilerplate/internal/pkg/health"
	healthHttp "go-boilerplate/internal/pkg/health/delivery/http"
	healthService "go-boilerplate/internal/pkg/health/service"
	userHttp "go-boilerplate/internal/pkg/user/delivery/http"
	userRepository "go-boilerplate/internal/pkg/user/repository"
	userService "go-boilerplate/internal/pkg/user/service"
	"go-boilerplate/internal/shared/cache"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
	"go-boilerplate/internal/shared/middleware"
)

// Container holds all application dependencies
type Container struct {
	// Configuration and Infrastructure
	Config  *config.Config
	Logger  *logger.Logger
	Metrics *metrics.Metrics

	// Database and Cache
	Database    *database.ReadWriteDatabase
	PgxDatabase *database.PgxReadWriteDB
	Cache       *cache.Redis

	// Repositories
	AuthRepository authRepository.SqlcAuthRepository
	UserRepository userRepository.SqlcUserRepository

	// Services
	AuthService   authService.AuthService
	UserService   userService.UserService
	HealthService *healthService.HealthService

	// HTTP Handlers
	AuthHandler   *authHttp.AuthHandler
	UserHandler   *userHttp.UserHandler
	HealthHandler *healthHttp.HealthHandler

	// Middleware
	AuthMiddleware      *middleware.AuthMiddleware
	LoggingMiddleware   *middleware.LoggingMiddleware
	RecoveryMiddleware  *middleware.RecoveryMiddleware
	SecurityMiddleware  *middleware.SecurityMiddleware
	RateLimitMiddleware *middleware.RateLimitMiddleware
	RequestIDMiddleware *middleware.RequestIDMiddleware
}

// ContainerOptions defines configuration options for the container
type ContainerOptions struct {
	ConfigPath string
}

// NewContainer creates and initializes all application dependencies
func NewContainer(opts ContainerOptions) (*Container, error) {
	container := &Container{}

	// Load configuration first
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	container.Config = cfg

	// Initialize logger
	appLogger := logger.New(cfg.Environment)
	container.Logger = appLogger

	// Initialize metrics if enabled
	if cfg.MetricsEnabled {
		container.Metrics = metrics.New(appLogger)
	}

	// Initialize database
	if err := container.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize cache
	if err := container.initCache(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize repositories
	if err := container.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	// Initialize services
	if err := container.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize middleware
	if err := container.initMiddleware(); err != nil {
		return nil, fmt.Errorf("failed to initialize middleware: %w", err)
	}

	// Initialize handlers
	if err := container.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	// Validate container
	if err := container.validate(); err != nil {
		return nil, fmt.Errorf("container validation failed: %w", err)
	}

	container.Logger.Info("Container initialized successfully")
	return container, nil
}

// initDatabase initializes database connections
func (c *Container) initDatabase() error {
	rwDBConfig := database.NewReadWriteConfig(c.Config)
	rwDB, err := database.NewReadWriteDatabase(rwDBConfig, c.Logger, c.Metrics)
	if err != nil {
		return fmt.Errorf("failed to create read-write database: %w", err)
	}

	// Test database connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rwDB.WriteDB().PingContext(ctx); err != nil {
		return fmt.Errorf("database connectivity test failed: %w", err)
	}

	c.Database = rwDB

	// Initialize pgx database for sqlc
	pgxDB, err := database.NewPgxReadWriteDB(c.Config, c.Logger)
	if err != nil {
		return fmt.Errorf("failed to create pgx database: %w", err)
	}
	c.PgxDatabase = pgxDB

	c.Logger.Info("Database initialized successfully")
	return nil
}

// initCache initializes Redis cache
func (c *Container) initCache() error {
	redisConfig := cache.DefaultConfig(c.Config)
	redisClient, err := cache.New(redisConfig, c.Logger)
	if err != nil {
		return fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Test cache connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisClient.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache connectivity test failed: %w", err)
	}

	c.Cache = redisClient
	c.Logger.Info("Cache initialized successfully")
	return nil
}

// initRepositories initializes all repositories
func (c *Container) initRepositories() error {
	c.AuthRepository = authRepository.NewSqlcAuthRepository(c.PgxDatabase, c.Logger)
	c.UserRepository = userRepository.NewSqlcUserRepository(c.PgxDatabase, c.Logger)

	c.Logger.Info("Repositories initialized successfully")
	return nil
}

// initServices initializes all services
func (c *Container) initServices() error {
	var err error

	// Initialize auth service
	c.AuthService, err = authService.NewAuthService(
		c.AuthRepository,
		c.Config,
		c.Logger,
		c.Metrics,
	)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %w", err)
	}

	// Initialize user service
	c.UserService = userService.NewUserService(c.UserRepository, c.Logger)

	// Initialize health service
	c.HealthService = healthService.NewHealthService("1.0.0", c.Logger)
	c.HealthService.AddChecker(health.NewRedisHealthChecker(c.Cache, c.Logger))

	c.Logger.Info("Services initialized successfully")
	return nil
}

// initMiddleware initializes all middleware
func (c *Container) initMiddleware() error {
	var err error

	// Initialize auth middleware
	c.AuthMiddleware, err = middleware.NewAuthMiddleware(c.Logger)
	if err != nil {
		return fmt.Errorf("failed to create auth middleware: %w", err)
	}

	// Initialize other middleware
	c.LoggingMiddleware = middleware.NewLoggingMiddleware(c.Logger)
	c.RecoveryMiddleware = middleware.NewRecoveryMiddleware(c.Logger)
	c.SecurityMiddleware = middleware.NewSecurityMiddleware(c.Logger, c.Config.Environment == "development")
	c.RequestIDMiddleware = middleware.NewRequestIDMiddleware(c.Logger)

	// Initialize rate limiting middleware
	rateLimitConfig := middleware.DefaultRateLimitConfig()
	c.RateLimitMiddleware = middleware.NewRateLimitMiddleware(rateLimitConfig, c.Cache, c.Logger)

	c.Logger.Info("Middleware initialized successfully")
	return nil
}

// initHandlers initializes all HTTP handlers
func (c *Container) initHandlers() error {
	c.AuthHandler = authHttp.NewAuthHandler(c.AuthService, c.Cache, c.Logger)
	c.UserHandler = userHttp.NewUserHandler(c.UserService, c.Logger)
	c.HealthHandler = healthHttp.NewHealthHandler(c.HealthService, c.Logger)

	c.Logger.Info("Handlers initialized successfully")
	return nil
}

// validate performs validation on the container dependencies
func (c *Container) validate() error {
	if c.Config == nil {
		return fmt.Errorf("config is nil")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger is nil")
	}
	if c.Database == nil {
		return fmt.Errorf("database is nil")
	}
	if c.Cache == nil {
		return fmt.Errorf("cache is nil")
	}
	if c.AuthRepository == nil {
		return fmt.Errorf("auth repository is nil")
	}
	if c.UserRepository == nil {
		return fmt.Errorf("user repository is nil")
	}
	if c.AuthService == nil {
		return fmt.Errorf("auth service is nil")
	}
	if c.UserService == nil {
		return fmt.Errorf("user service is nil")
	}
	if c.HealthService == nil {
		return fmt.Errorf("health service is nil")
	}
	if c.AuthHandler == nil {
		return fmt.Errorf("auth handler is nil")
	}
	if c.UserHandler == nil {
		return fmt.Errorf("user handler is nil")
	}
	if c.HealthHandler == nil {
		return fmt.Errorf("health handler is nil")
	}

	return nil
}

// Close gracefully shuts down all container dependencies
func (c *Container) Close() error {
	c.Logger.Info("Shutting down container...")

	var errs []error

	// Close database connections
	if c.Database != nil {
		if err := c.Database.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database: %w", err))
		}
	}

	// Close pgx database connections
	if c.PgxDatabase != nil {
		c.PgxDatabase.Close()
	}

	// Close cache connections
	if c.Cache != nil {
		if err := c.Cache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close cache: %w", err))
		}
	}

	// Sync logger
	if c.Logger != nil {
		if err := c.Logger.Sync(); err != nil {
			// Don't add sync errors to the error list as they're often harmless
			c.Logger.Warn("Failed to sync logger", zap.Error(err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// Health performs a health check on all container dependencies
func (c *Container) Health(ctx context.Context) error {
	// Check database health
	if err := c.Database.WriteDB().PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check cache health
	if err := c.Cache.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache health check failed: %w", err)
	}

	return nil
}
