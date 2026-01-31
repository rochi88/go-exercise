package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/utils"
)

// RecoveryMiddleware provides panic recovery functionality
type RecoveryMiddleware struct {
	logger *logger.Logger
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(log *logger.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: log.Named("recovery"),
	}
}

// Recover catches panics and returns a 500 Internal Server Error
func (m *RecoveryMiddleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Defer recovery function
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()

				// Get request context
				requestID := utils.GetRequestIDFromContext(r.Context())
				sessionID := utils.GetSessionIDFromContext(r.Context())
				responseCtx := utils.NewResponseContext(requestID, sessionID)

				// Log the panic
				m.logger.Error(
					"Panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(stack)),
					zap.String("request_id", requestID),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)

				// Respond with standardized error
				utils.RespondWithInternalError(
					w, r.Context(), responseCtx,
					fmt.Errorf("panic: %v", err),
					m.logger,
				)
			}
		}()

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// GinRecover provides Gin-compatible panic recovery middleware
func (m *RecoveryMiddleware) GinRecover(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			// Get stack trace
			stack := debug.Stack()

			// Get request context
			requestID := c.GetString("request_id")
			sessionID := c.GetString("session_id")
			responseCtx := utils.NewResponseContext(requestID, sessionID)

			// Log the panic
			m.logger.Error(
				"Panic recovered",
				zap.Any("error", err),
				zap.String("stack", string(stack)),
				zap.String("request_id", requestID),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
			)

			// Respond with standardized error
			utils.RespondWithStandardFormat(
				c.Writer, c.Request.Context(),
				http.StatusInternalServerError, false, nil,
				"Internal server error", responseCtx,
			)
			c.Abort()
		}
	}()

	c.Next()
}
