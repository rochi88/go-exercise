package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"go-boilerplate/internal/shared/cache"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/utils"
)

// RateLimitConfig holds the configuration for rate limiting
type RateLimitConfig struct {
	// MaxAttempts is the maximum number of requests allowed in the time window
	MaxAttempts int
	// Window is the time window for rate limiting (e.g., 1 * time.Minute)
	Window time.Duration
	// KeyPrefix is the prefix for Redis keys or in-memory map keys
	KeyPrefix string
	// SkipSuccessfulRequests determines if successful requests should be counted
	SkipSuccessfulRequests bool
	// SkipPaths contains paths that should be excluded from rate limiting
	SkipPaths []string
	// TrustedProxies contains IP addresses/ranges that should be trusted for X-Forwarded-For
	TrustedProxies []string
	// BurstSize allows for burst requests above the normal rate limit
	BurstSize int
}

// RateLimiterStorage defines the interface for rate limiting storage backends
type RateLimiterStorage interface {
	// Increment increments the counter for the given key and returns the new count
	Increment(ctx context.Context, key string, window time.Duration) (int, error)
	// Get returns the current count for the given key
	Get(ctx context.Context, key string) (int, error)
	// Reset resets the counter for the given key
	Reset(ctx context.Context, key string) error
	// IsAvailable returns true if the storage backend is available
	IsAvailable() bool
}

// RedisRateLimiter implements RateLimiterStorage using Redis
type RedisRateLimiter struct {
	client *cache.Redis
	logger *logger.Logger
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(redisClient *cache.Redis, logger *logger.Logger) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: redisClient,
		logger: logger.Named("redis-rate-limiter"),
	}
}

// Increment increments the counter for the given key
func (r *RedisRateLimiter) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	// Use Redis pipeline for atomic operations
	pipe := r.client.Client.TxPipeline()

	// Increment the counter
	incr := pipe.Incr(ctx, key)
	// Set expiration if this is the first time (only if key didn't exist)
	pipe.Expire(ctx, key, window)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Failed to increment rate limit counter", zap.String("key", key), zap.Error(err))
		return 0, err
	}

	count, err := incr.Result()
	if err != nil {
		r.logger.Error("Failed to get increment result", zap.String("key", key), zap.Error(err))
		return 0, err
	}

	return int(count), nil
}

// Get returns the current count for the given key
func (r *RedisRateLimiter) Get(ctx context.Context, key string) (int, error) {
	count, err := r.client.Client.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		r.logger.Error("Failed to get rate limit counter", zap.String("key", key), zap.Error(err))
		return 0, err
	}
	if err == redis.Nil {
		return 0, nil
	}
	return count, nil
}

// Reset resets the counter for the given key
func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	err := r.client.Client.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error("Failed to reset rate limit counter", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

// IsAvailable returns true if Redis is available
func (r *RedisRateLimiter) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := r.client.Client.Ping(ctx).Result()
	return err == nil
}

// InMemoryRateLimiter implements RateLimiterStorage using in-memory storage
type InMemoryRateLimiter struct {
	data   map[string]*rateLimitEntry
	mutex  sync.RWMutex
	logger *logger.Logger
}

// rateLimitEntry holds the rate limit data for in-memory storage
type rateLimitEntry struct {
	count     int
	expiresAt time.Time
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(logger *logger.Logger) *InMemoryRateLimiter {
	limiter := &InMemoryRateLimiter{
		data:   make(map[string]*rateLimitEntry),
		logger: logger.Named("memory-rate-limiter"),
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Increment increments the counter for the given key
func (m *InMemoryRateLimiter) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	expiresAt := now.Add(window)

	entry, exists := m.data[key]
	if !exists || now.After(entry.expiresAt) {
		// Create new entry or reset expired entry
		m.data[key] = &rateLimitEntry{
			count:     1,
			expiresAt: expiresAt,
		}
		return 1, nil
	}

	// Increment existing entry
	entry.count++
	return entry.count, nil
}

// Get returns the current count for the given key
func (m *InMemoryRateLimiter) Get(ctx context.Context, key string) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entry, exists := m.data[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return 0, nil
	}

	return entry.count, nil
}

// Reset resets the counter for the given key
func (m *InMemoryRateLimiter) Reset(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	return nil
}

// IsAvailable returns true for in-memory storage (always available)
func (m *InMemoryRateLimiter) IsAvailable() bool {
	return true
}

// cleanup removes expired entries periodically
func (m *InMemoryRateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.Lock()
		now := time.Now()
		for key, entry := range m.data {
			if now.After(entry.expiresAt) {
				delete(m.data, key)
			}
		}
		m.mutex.Unlock()
	}
}

// RateLimitMiddleware provides rate limiting functionality
type RateLimitMiddleware struct {
	config  *RateLimitConfig
	storage RateLimiterStorage
	logger  *logger.Logger
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(config *RateLimitConfig, redisClient *cache.Redis, logger *logger.Logger) *RateLimitMiddleware {
	var storage RateLimiterStorage

	// Try Redis first, fallback to in-memory
	if redisClient != nil {
		redisLimiter := NewRedisRateLimiter(redisClient, logger)
		if redisLimiter.IsAvailable() {
			storage = redisLimiter
		} else {
			logger.Warn("Redis not available, falling back to in-memory storage for rate limiting")
			storage = NewInMemoryRateLimiter(logger)
		}
	} else {
		logger.Info("Redis not configured, using in-memory storage for rate limiting")
		storage = NewInMemoryRateLimiter(logger)
	}

	return &RateLimitMiddleware{
		config:  config,
		storage: storage,
		logger:  logger.Named("rate-limit"),
	}
}

// GinRateLimit returns a Gin middleware function for rate limiting
func (m *RateLimitMiddleware) GinRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path should be skipped
		if m.shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get client IP
		clientIP := m.getClientIP(c)

		// Create rate limit key
		key := m.createKey(clientIP, c.Request.URL.Path)

		// Check rate limit
		count, err := m.storage.Increment(c.Request.Context(), key, m.config.Window)
		if err != nil {
			m.logger.Error("Failed to check rate limit", zap.Error(err))
			// Allow request on storage error to avoid blocking legitimate traffic
			c.Next()
			return
		}

		// Calculate effective limit (base limit + burst)
		effectiveLimit := m.config.MaxAttempts
		if m.config.BurstSize > 0 {
			effectiveLimit += m.config.BurstSize
		}

		// Check if limit exceeded
		if count > effectiveLimit {
			m.handleRateLimitExceeded(c, clientIP, count, effectiveLimit)
			return
		}

		// Add rate limit headers
		m.addRateLimitHeaders(c, count, effectiveLimit)

		// Skip counting successful requests if configured
		if m.config.SkipSuccessfulRequests {
			c.Next()

			// Only count if request was not successful (5xx errors)
			if c.Writer.Status() >= 500 {
				return
			}

			// Reset the counter since we don't want to count successful requests
			m.storage.Reset(c.Request.Context(), key)
			return
		}

		c.Next()
	}
}

// RateLimitOptions holds dynamic configuration for rate limiting
type RateLimitOptions struct {
	// Window is the time window for rate limiting (e.g., 60 for 60 seconds)
	Window int `json:"window"`
	// Limit is the maximum number of requests allowed in the time window
	Limit int `json:"limit"`
	// BurstSize allows for burst requests above the normal rate limit
	BurstSize int `json:"burst_size,omitempty"`
	// KeyPrefix is the prefix for Redis keys or in-memory map keys
	KeyPrefix string `json:"key_prefix,omitempty"`
}

// GinRateLimitWithOptions returns a Gin middleware function with dynamic configuration
func (m *RateLimitMiddleware) GinRateLimitWithOptions(opts RateLimitOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path should be skipped
		if m.shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get client IP
		clientIP := m.getClientIP(c)

		// Create rate limit key with custom prefix if provided
		keyPrefix := m.config.KeyPrefix
		if opts.KeyPrefix != "" {
			keyPrefix = opts.KeyPrefix
		}
		key := fmt.Sprintf("%s:%s:%s", keyPrefix, clientIP, c.Request.URL.Path)

		// Use dynamic window or fallback to middleware config
		window := m.config.Window
		if opts.Window > 0 {
			window = time.Duration(opts.Window) * time.Second
		}

		// Check rate limit
		count, err := m.storage.Increment(c.Request.Context(), key, window)
		if err != nil {
			m.logger.Error("Failed to check rate limit", zap.Error(err))
			// Allow request on storage error to avoid blocking legitimate traffic
			c.Next()
			return
		}

		// Calculate effective limit (dynamic limit + burst)
		effectiveLimit := opts.Limit
		if opts.BurstSize > 0 {
			effectiveLimit += opts.BurstSize
		} else if m.config.BurstSize > 0 {
			// Fallback to middleware burst size if not specified in options
			effectiveLimit += m.config.BurstSize
		}

		// Check if limit exceeded
		if count > effectiveLimit {
			m.handleRateLimitExceeded(c, clientIP, count, effectiveLimit)
			return
		}

		// Add rate limit headers
		m.addRateLimitHeaders(c, count, effectiveLimit)

		// Skip counting successful requests if configured
		if m.config.SkipSuccessfulRequests {
			c.Next()

			// Only count if request was not successful (5xx errors)
			if c.Writer.Status() >= 500 {
				return
			}

			// Reset the counter since we don't want to count successful requests
			m.storage.Reset(c.Request.Context(), key)
			return
		}

		c.Next()
	}
}
func (m *RateLimitMiddleware) GinRateLimitWithConfig(config map[string]interface{}) gin.HandlerFunc {
	opts := RateLimitOptions{}

	// Extract window from config
	if window, ok := config["window"].(int); ok {
		opts.Window = window
	}

	// Extract limit from config
	if limit, ok := config["limit"].(int); ok {
		opts.Limit = limit
	}

	// Extract burst_size from config
	if burstSize, ok := config["burst_size"].(int); ok {
		opts.BurstSize = burstSize
	}

	// Extract key_prefix from config
	if keyPrefix, ok := config["key_prefix"].(string); ok {
		opts.KeyPrefix = keyPrefix
	}

	return m.GinRateLimitWithOptions(opts)
}
func (m *RateLimitMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// getClientIP extracts the real client IP from the request
func (m *RateLimitMiddleware) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (for proxies)
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(ips[0])

		// Validate if it's a trusted proxy
		if m.isTrustedProxy(clientIP) {
			return clientIP
		}
	}

	// Check X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

// isTrustedProxy checks if the IP is from a trusted proxy
func (m *RateLimitMiddleware) isTrustedProxy(ip string) bool {
	for _, trusted := range m.config.TrustedProxies {
		if ip == trusted {
			return true
		}
	}
	return false
}

// createKey creates a unique key for rate limiting
func (m *RateLimitMiddleware) createKey(clientIP, path string) string {
	return fmt.Sprintf("%s:%s:%s", m.config.KeyPrefix, clientIP, path)
}

// handleRateLimitExceeded handles requests that exceed the rate limit
func (m *RateLimitMiddleware) handleRateLimitExceeded(c *gin.Context, clientIP string, count, limit int) {
	requestID := c.GetString("request_id")
	sessionID := c.GetString("session_id")

	m.logger.Warn("Rate limit exceeded",
		zap.String("client_ip", clientIP),
		zap.String("path", c.Request.URL.Path),
		zap.String("request_id", requestID),
		zap.String("session_id", sessionID),
		zap.Int("count", count),
		zap.Int("limit", limit),
		zap.String("user_agent", c.Request.UserAgent()),
	)

	// Add rate limit headers
	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", "0")
	c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(m.config.Window).Unix(), 10))
	c.Header("Retry-After", strconv.Itoa(int(m.config.Window.Seconds())))

	// Create response context for standardized response
	responseCtx := utils.NewResponseContext(requestID, sessionID)

	// Return 429 Too Many Requests with standardized format
	utils.RespondWithStandardFormat(
		c.Writer, c.Request.Context(),
		http.StatusTooManyRequests, false, nil,
		"Rate limit exceeded. Please try again later.",
		responseCtx,
	)
	c.Abort()
}

// addRateLimitHeaders adds rate limit information to response headers
func (m *RateLimitMiddleware) addRateLimitHeaders(c *gin.Context, count, limit int) {
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(m.config.Window).Unix(), 10))
}

// GetStats returns current rate limiting statistics
func (m *RateLimitMiddleware) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"max_attempts":             m.config.MaxAttempts,
		"window_seconds":           m.config.Window.Seconds(),
		"burst_size":               m.config.BurstSize,
		"skip_successful_requests": m.config.SkipSuccessfulRequests,
		"skip_paths":               m.config.SkipPaths,
		"storage_type":             m.getStorageType(),
		"storage_available":        m.storage.IsAvailable(),
	}
}

// getStorageType returns the type of storage backend being used
func (m *RateLimitMiddleware) getStorageType() string {
	switch m.storage.(type) {
	case *RedisRateLimiter:
		return "redis"
	case *InMemoryRateLimiter:
		return "memory"
	default:
		return "unknown"
	}
}

// DefaultRateLimitConfig returns a default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		MaxAttempts:            100,
		Window:                 1 * time.Minute,
		KeyPrefix:              "rate_limit",
		SkipSuccessfulRequests: false,
		// SkipPaths:              []string{"/health", "/ready", "/metrics"},
		TrustedProxies: []string{},
		BurstSize:      10,
	}
}

// AuthRateLimitConfig returns a strict rate limiting configuration for authentication endpoints
func AuthRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		MaxAttempts:            5,               // Only 5 attempts per minute for auth endpoints
		Window:                 1 * time.Minute, // Per minute window
		KeyPrefix:              "auth_rate_limit",
		SkipSuccessfulRequests: false,
		SkipPaths:              []string{}, // No paths skipped for auth
		TrustedProxies:         []string{},
		BurstSize:              2, // Small burst allowance
	}
}

// LoginRateLimitConfig returns rate limiting configuration specifically for login endpoints
func LoginRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		MaxAttempts:            3,               // Only 3 login attempts per 5 minutes
		Window:                 5 * time.Minute, // 5 minute window
		KeyPrefix:              "login_rate_limit",
		SkipSuccessfulRequests: true, // Don't count successful logins
		SkipPaths:              []string{},
		TrustedProxies:         []string{},
		BurstSize:              1, // Minimal burst
	}
}

// RefreshTokenRateLimitConfig returns rate limiting configuration for token refresh endpoints
func RefreshTokenRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		MaxAttempts:            10,              // 10 refresh attempts per minute
		Window:                 1 * time.Minute, // Per minute window
		KeyPrefix:              "refresh_rate_limit",
		SkipSuccessfulRequests: false,
		SkipPaths:              []string{},
		TrustedProxies:         []string{},
		BurstSize:              3, // Small burst allowance
	}
}

// CreateRateLimitConfig creates a custom rate limiting configuration
func CreateRateLimitConfig(maxAttempts int, window time.Duration, keyPrefix string) *RateLimitConfig {
	config := DefaultRateLimitConfig()
	config.MaxAttempts = maxAttempts
	config.Window = window
	config.KeyPrefix = keyPrefix
	return config
}
