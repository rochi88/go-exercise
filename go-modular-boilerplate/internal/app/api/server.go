package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-boilerplate/internal/app/config"
	authHttp "go-boilerplate/internal/pkg/auth/delivery/http"
	healthHttp "go-boilerplate/internal/pkg/health/delivery/http"
	userHttp "go-boilerplate/internal/pkg/user/delivery/http"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
	customMiddleware "go-boilerplate/internal/shared/middleware"
)

// Server represents the HTTP server
type Server struct {
	server *http.Server
	router *gin.Engine
	config *config.Config
	logger *logger.Logger
}

// ServerOptions holds the server dependencies
type ServerOptions struct {
	Config              *config.Config
	Logger              *logger.Logger
	AuthHandler         *authHttp.AuthHandler
	UserHandler         *userHttp.UserHandler
	HealthHandler       *healthHttp.HealthHandler
	AuthMiddleware      *customMiddleware.AuthMiddleware
	LoggingMiddleware   *customMiddleware.LoggingMiddleware
	RecoveryMiddleware  *customMiddleware.RecoveryMiddleware
	SecurityMiddleware  *customMiddleware.SecurityMiddleware
	RateLimitMiddleware *customMiddleware.RateLimitMiddleware
	RequestIDMiddleware *customMiddleware.RequestIDMiddleware
	Metrics             *metrics.Metrics
}

// NewServer creates a new HTTP server
func NewServer(opts *ServerOptions) *Server {
	// Set Gin mode based on environment
	if opts.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	// Add custom middleware
	if opts.RequestIDMiddleware != nil {
		r.Use(opts.RequestIDMiddleware.Middleware())
	}
	r.Use(opts.LoggingMiddleware.GinLogRequest)
	r.Use(opts.RecoveryMiddleware.GinRecover)
	r.Use(opts.SecurityMiddleware.GinSecurityHeaders)

	// Add rate limiting middleware if available
	if opts.RateLimitMiddleware != nil {
		r.Use(opts.RateLimitMiddleware.GinRateLimit())
	}

	// Add metrics middleware if available
	if opts.Metrics != nil {
		r.Use(opts.Metrics.GinMiddleware())
	}

	// Set up routes
	setupRoutes(r, opts)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Config.ServerPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		server: srv,
		router: r,
		config: opts.Config,
		logger: opts.Logger.Named("server"),
	}
}

// setupRoutes configures all the routes for the application
func setupRoutes(r *gin.Engine, opts *ServerOptions) {
	// API routes under /api/v1
	apiV1 := r.Group("/api/v1")
	{
		// Register auth routes
		opts.AuthHandler.RegisterGinRoutes(apiV1, opts.AuthMiddleware, opts.RateLimitMiddleware)

		// Register user routes
		opts.UserHandler.RegisterGinRoutes(apiV1, opts.AuthMiddleware)
	}

	// Health check routes (with rate limiting)
	healthGroup := r.Group("/")
	if opts.RateLimitMiddleware != nil {
		healthGroup.Use(opts.RateLimitMiddleware.GinRateLimitWithOptions(customMiddleware.RateLimitOptions{
			Window:    10, // 60 seconds
			Limit:     1,  // 1 request per minute
			BurstSize: 2,  // Allow 2 burst requests
			KeyPrefix: "health",
		}))
	}
	{
		healthGroup.GET("/health", opts.HealthHandler.GinHealth)
		healthGroup.GET("/ready", opts.HealthHandler.GinReady)
	}

	// Metrics endpoint (if metrics are enabled)
	if opts.Metrics != nil {
		r.GET(opts.Config.MetricsPath, opts.Metrics.GinMetricsHandler())
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info(fmt.Sprintf("Starting HTTP server on port %d", s.config.ServerPort))
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}
