package authHttp

import (
	"github.com/gin-gonic/gin"

	"go-boilerplate/internal/shared/middleware"
)

// RegisterGinRoutes registers auth routes on the given Gin router
func (h *AuthHandler) RegisterGinRoutes(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	// Public routes
	auth := r.Group("/auth")
	{

		auth.Use(rateLimitMiddleware.GinRateLimitWithOptions(middleware.RateLimitOptions{
			Window:    1,  // 1 minute
			Limit:     20, // 20 registrations per minute
			BurstSize: 5,  // Small burst
			KeyPrefix: "auth_routes_rate_limit",
		}))

		// POST /auth/login - Login endpoint with strict rate limiting
		auth.POST("/login", h.GinLogin)
		auth.POST("/register", h.GinRegister)
		auth.POST("/refresh-token", h.GinRefreshToken)
		auth.POST("/verify-email", h.GinVerifyEmail)
		auth.POST("/request-password-reset", h.GinRequestPasswordReset)
		auth.POST("/reset-password", h.GinResetPassword)
		auth.GET("/jwk-key", h.GinJWKKey)

		// Protected routes example
		protected := auth.Group("")
		protected.Use(authMiddleware.GinAuthenticate)
		{
			// protected routes
			protected.POST("/change-password", h.GinChangePassword)
			// Example: protected.GET("/refresh-token", h.GinRefreshToken)
		}
	}
}
