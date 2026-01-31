package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/utils"
)

// AuthMiddleware provides JWT authentication functionality
type AuthMiddleware struct {
	logger          *logger.Logger
	responseHandler *utils.ResponseHandler
	jwtUtils        *utils.JWTUtils
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(log *logger.Logger) (*AuthMiddleware, error) {
	jwtUtils := utils.NewJWTUtils()

	return &AuthMiddleware{
		logger:          log.Named("auth-middleware"),
		responseHandler: utils.NewResponseHandler(log.Named("auth-middleware-responses")),
		jwtUtils:        jwtUtils,
	}, nil
}

// Authenticate validates JWT tokens from the Authorization header
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			requestID := utils.GetRequestIDFromContext(r.Context())
			sessionID := utils.GetSessionIDFromContext(r.Context())
			responseCtx := utils.NewResponseContext(requestID, sessionID)

			utils.RespondWithError(
				w, r.Context(), responseCtx,
				"Authentication required",
				http.StatusUnauthorized,
				m.logger,
			)
			return
		}

		// Check if the Authorization header has the right format
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			requestID := utils.GetRequestIDFromContext(r.Context())
			sessionID := utils.GetSessionIDFromContext(r.Context())
			responseCtx := utils.NewResponseContext(requestID, sessionID)

			utils.RespondWithError(
				w, r.Context(), responseCtx,
				"Invalid authorization format, expected 'Bearer {token}'",
				http.StatusUnauthorized,
				m.logger,
			)
			return
		}

		// Parse the token
		tokenStr := bearerToken[1]
		validToken, err := m.jwtUtils.DecryptClaims(tokenStr)

		if validToken == nil || err != nil {
			requestID := utils.GetRequestIDFromContext(r.Context())
			sessionID := utils.GetSessionIDFromContext(r.Context())
			responseCtx := utils.NewResponseContext(requestID, sessionID)

			utils.RespondWithError(
				w, r.Context(), responseCtx,
				"Invalid or expired token",
				http.StatusUnauthorized,
				m.logger,
			)
			return
		}

		userID, ok := validToken["user_id"].(string)
		if !ok {
			requestID := utils.GetRequestIDFromContext(r.Context())
			sessionID := utils.GetSessionIDFromContext(r.Context())
			responseCtx := utils.NewResponseContext(requestID, sessionID)

			utils.RespondWithError(
				w, r.Context(), responseCtx,
				"Invalid user ID in token",
				http.StatusUnauthorized,
				m.logger,
			)
			return
		}

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Call the next handler with the modified context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from the request context
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

// GinAuthenticate provides Gin-compatible JWT authentication middleware
func (m *AuthMiddleware) GinAuthenticate(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		m.responseHandler.GinUnauthorized(c, "Authentication required")
		c.Abort()
		return
	}

	// Check if the Authorization header has the right format
	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		m.responseHandler.GinUnauthorized(c, "Invalid authorization format, expected 'Bearer {token}'")
		c.Abort()
		return
	}

	// Parse the token
	tokenStr := bearerToken[1]

	validToken, err := m.jwtUtils.DecryptClaims(tokenStr)

	if err != nil || validToken == nil {
		m.responseHandler.GinUnauthorized(c, "Invalid or expired token")
		c.Abort()
		return
	}

	userID, ok := validToken["user_id"].(string)
	if !ok {
		m.responseHandler.GinUnauthorized(c, "Invalid user ID in token")
		c.Abort()
		return
	}

	// Add user ID to Gin context
	c.Set("user_id", userID)
	c.Next()
}

// GetUserIDFromGin extracts the user ID from the Gin context
func GetUserIDFromGin(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	uid, ok := userID.(string)
	return uid, ok
}
