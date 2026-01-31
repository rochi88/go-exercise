package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-boilerplate/internal/shared/logger"
)

// LoggingMiddleware provides request logging functionality
type LoggingMiddleware struct {
	logger *logger.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(log *logger.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: log.Named("http"),
	}
}

// LogRequest logs information about incoming HTTP requests
func (m *LoggingMiddleware) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer that captures the status code
		rwCapture := &responseWriterCapture{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default status code
		}

		// Extract request ID from header or generate one
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			// Generate a request ID if not provided
			bytes := make([]byte, 8)
			if _, err := rand.Read(bytes); err != nil {
				requestID = "req-" + time.Now().Format("150405")
			} else {
				requestID = "req-" + hex.EncodeToString(bytes)
			}
			// Set the request ID in the header for downstream handlers
			r.Header.Set(RequestIDHeader, requestID)
		}

		// Generate or extract session ID
		sessionID := getOrGenerateSessionID(r)

		// Set session ID cookie if it doesn't exist
		if cookie, err := r.Cookie("session_id"); err != nil || cookie == nil || cookie.Value == "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    sessionID,
				Path:     "/",
				MaxAge:   86400 * 30, // 30 days
				HttpOnly: true,
				Secure:   false, // Set to true in production with HTTPS
			})
		}

		// Set session ID in context for downstream handlers
		ctx := context.WithValue(r.Context(), SessionIDKey, sessionID)
		r = r.WithContext(ctx)

		// Log the request
		m.logger.Info(
			"Request started",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.String("request_id", requestID),
			zap.String("session_id", sessionID),
		)

		// Store the request start time in the request context
		ctx = contextWithStartTime(ctx, start)

		// Call the next handler
		next.ServeHTTP(rwCapture, r.WithContext(ctx))

		// Calculate duration
		duration := time.Since(start)

		// Log the response
		m.logger.Info(
			"Request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rwCapture.statusCode),
			zap.Duration("duration", duration),
			zap.String("request_id", requestID),
			zap.String("session_id", sessionID),
		)
	})
}

// GinLogRequest provides Gin-compatible request logging middleware
func (m *LoggingMiddleware) GinLogRequest(c *gin.Context) {
	start := time.Now()

	// Extract request ID from header or generate one
	requestID := c.GetHeader(RequestIDHeader)
	if requestID == "" {
		// Generate a request ID if not provided
		bytes := make([]byte, 8)
		if _, err := rand.Read(bytes); err != nil {
			requestID = "req-" + time.Now().Format("150405")
		} else {
			requestID = "req-" + hex.EncodeToString(bytes)
		}
		// Set the request ID in the header for downstream handlers
		c.Header(RequestIDHeader, requestID)
	}

	// Generate or extract session ID
	sessionID := getOrGenerateSessionIDGin(c)

	// Set session ID cookie if it doesn't exist
	if cookie, err := c.Cookie("session_id"); err != nil || cookie == "" {
		c.SetCookie("session_id", sessionID, 86400*30, "/", "", false, true) // 30 days, httpOnly
	}

	// Set session ID in Gin context and request context
	c.Set("session_id", sessionID)
	ctx := context.WithValue(c.Request.Context(), SessionIDKey, sessionID)
	c.Request = c.Request.WithContext(ctx)

	// Log the request
	m.logger.Info(
		"Request started",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("remote_addr", c.ClientIP()),
		zap.String("user_agent", c.GetHeader("User-Agent")),
		zap.String("request_id", requestID),
		zap.String("session_id", sessionID),
	)

	// Process request
	c.Next()

	// Calculate duration
	duration := time.Since(start)

	// Log the response
	m.logger.Info(
		"Request completed",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Int("status", c.Writer.Status()),
		zap.Duration("duration", duration),
		zap.String("request_id", requestID),
		zap.String("session_id", sessionID),
	)
}

// responseWriterCapture is a wrapper around http.ResponseWriter that captures the status code
type responseWriterCapture struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and writes it to the underlying ResponseWriter
func (rwc *responseWriterCapture) WriteHeader(statusCode int) {
	rwc.statusCode = statusCode
	rwc.ResponseWriter.WriteHeader(statusCode)
}

// startTimeKey is a key for storing request start time in context
type startTimeKey struct{}

// contextWithStartTime stores the start time in the context
func contextWithStartTime(ctx context.Context, start time.Time) context.Context {
	return context.WithValue(ctx, startTimeKey{}, start)
}

// GetStartTimeFromContext retrieves the request start time from the context
func GetStartTimeFromContext(ctx context.Context) (time.Time, bool) {
	start, ok := ctx.Value(startTimeKey{}).(time.Time)
	return start, ok
}

// Header constants
const (
	RequestIDHeader = "X-Request-ID"
	SessionIDHeader = "X-Session-ID"
	ServerIDHeader  = "X-Server-ID"
)

// GetServerID retrieves server ID from context (stub implementation)
func GetServerID(ctx context.Context) string {
	// TODO: Implement server ID retrieval from context
	return "server-1"
}

// Cache hit context key
type cacheHitKey struct{}

// MarkRequestCacheHit marks the request as a cache hit in the context
func MarkRequestCacheHit(ctx context.Context) context.Context {
	return context.WithValue(ctx, cacheHitKey{}, true)
}

// CacheHitFromContext checks if the request was a cache hit
func CacheHitFromContext(ctx context.Context) bool {
	hit, ok := ctx.Value(cacheHitKey{}).(bool)
	return ok && hit
}

// GetStartTime is an alias for GetStartTimeFromContext for backward compatibility
func GetStartTime(ctx context.Context) (time.Time, bool) {
	return GetStartTimeFromContext(ctx)
}

// getOrGenerateSessionID generates or extracts a session ID for the request
func getOrGenerateSessionID(r *http.Request) string {
	// First try to get from header
	if sessionID := r.Header.Get(SessionIDHeader); sessionID != "" {
		return sessionID
	}

	// Try to get from cookie
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Generate a new session ID
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback if random generation fails
		return "fallback-" + time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(bytes)
}

// getOrGenerateSessionIDGin generates or extracts a session ID for Gin requests
func getOrGenerateSessionIDGin(c *gin.Context) string {
	// First try to get from header
	if sessionID := c.GetHeader(SessionIDHeader); sessionID != "" {
		return sessionID
	}

	// Try to get from cookie
	if cookie, err := c.Cookie("session_id"); err == nil && cookie != "" {
		return cookie
	}

	// Generate a new session ID
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback if random generation fails
		return "fallback-" + time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(bytes)
}
