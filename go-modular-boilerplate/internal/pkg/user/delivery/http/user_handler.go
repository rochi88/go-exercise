package userHttp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-boilerplate/internal/pkg/user"
	userService "go-boilerplate/internal/pkg/user/service"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/middleware"
	"go-boilerplate/internal/shared/utils"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	service         userService.UserService
	logger          *logger.Logger
	responseHandler *utils.ResponseHandler
}

// NewUserHandler creates a new user handler
func NewUserHandler(svc userService.UserService, log *logger.Logger) *UserHandler {
	return &UserHandler{
		service:         svc,
		logger:          log.Named("user-handler"),
		responseHandler: utils.NewResponseHandler(log.Named("user-responses")),
	}
}

// GetProfile handles the get profile request
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		h.responseHandler.Unauthorized(w, r.Context(), "Unauthorized")
		return
	}

	// Call service
	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		utils.HandleServiceError(w, r.Context(), err, h.responseHandler)
		return
	}

	h.responseHandler.Success(w, r.Context(),
		user.ProfileResponse{User: *profile},
		"Profile retrieved successfully")
}

// UpdateProfile handles the update profile request
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		h.responseHandler.Unauthorized(w, r.Context(), "Unauthorized")
		return
	}

	// Parse request
	var req user.UpdateProfileRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		h.responseHandler.BadRequest(w, r.Context(), "Invalid request payload")
		return
	}

	// Call service
	profile, err := h.service.UpdateProfile(r.Context(), userID, &req)
	if err != nil {
		utils.HandleServiceError(w, r.Context(), err, h.responseHandler)
		return
	}

	h.responseHandler.Success(w, r.Context(),
		user.ProfileResponse{User: *profile},
		"Profile updated successfully")
}

// GinGetProfile provides Gin-compatible get profile endpoint
func (h *UserHandler) GinGetProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responseHandler.GinUnauthorized(c, "Unauthorized")
		return
	}

	// Call service
	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		utils.GinHandleServiceError(c, err, h.responseHandler)
		return
	}

	h.responseHandler.GinSuccess(c,
		user.ProfileResponse{User: *profile},
		"Profile retrieved successfully")
}

// GinUpdateProfile provides Gin-compatible update profile endpoint
func (h *UserHandler) GinUpdateProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responseHandler.GinUnauthorized(c, "Unauthorized")
		return
	}

	// Parse request
	var req user.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.GinBadRequest(c, "Invalid request payload")
		return
	}

	// Call service
	profile, err := h.service.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		utils.GinHandleServiceError(c, err, h.responseHandler)
		return
	}

	h.responseHandler.GinSuccess(c,
		user.ProfileResponse{User: *profile},
		"Profile updated successfully")
}
