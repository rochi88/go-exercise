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

// RequestIDMiddleware provides request ID generation and tracking
type RequestIDMiddleware struct {
	logger *logger.Logger
}

// NewRequestIDMiddleware creates a new request ID middleware
func NewRequestIDMiddleware(log *logger.Logger) *RequestIDMiddleware {
	return &RequestIDMiddleware{
		logger: log.Named("request-id"),
	}
}

// GenerateRequestID generates a unique request ID
func (m *RequestIDMiddleware) GenerateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto rand fails
		return "req-" + time.Now().Format("150405000000")
	}
	return "req-" + hex.EncodeToString(bytes)
}

// Middleware returns the Gin middleware function for request ID handling
func (m *RequestIDMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is already provided in header
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = m.GenerateRequestID()
		}

		// Set request ID in header and context
		c.Header(RequestIDHeader, requestID)
		c.Set("request_id", requestID)

		// Add request ID to context for use in handlers
		ctx := context.WithValue(c.Request.Context(), RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Log request start
		m.logger.Info("Request started",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("client_ip", c.ClientIP()),
		)

		// Store start time for response time calculation
		c.Set("start_time", time.Now())

		c.Next()

		// Log request completion
		status := c.Writer.Status()
		latency := time.Since(c.GetTime("start_time"))

		m.logger.Info("Request completed",
			zap.String("request_id", requestID),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int("response_size", c.Writer.Size()),
		)
	}
}

// Handler returns the standard http.Handler middleware for request ID handling
func (m *RequestIDMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID is already provided in header
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = m.GenerateRequestID()
		}

		// Set request ID in header and context
		w.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		r = r.WithContext(ctx)

		// Log request start
		m.logger.Info("Request started",
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("user_agent", r.UserAgent()),
			zap.String("remote_addr", r.RemoteAddr),
		)

		// Store start time in context
		ctx = context.WithValue(ctx, StartTimeKey, time.Now())
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
