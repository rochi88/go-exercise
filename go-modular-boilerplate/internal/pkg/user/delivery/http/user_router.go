package userHttp

import (
	"github.com/gin-gonic/gin"

	"go-boilerplate/internal/shared/middleware"
)

// RegisterGinRoutes registers user routes on the given Gin router
func (h *UserHandler) RegisterGinRoutes(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// All user routes are protected by authentication
	users := r.Group("/users")
	users.Use(authMiddleware.GinAuthenticate)
	{
		// GET /users/profile - Get user profile
		users.GET("/profile", h.GinGetProfile)

		// PUT /users/profile - Update user profile
		users.PUT("/profile", h.GinUpdateProfile)

		// Add more user routes as needed
	}
}
