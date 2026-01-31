package utils

import (
	"context"
	"net/http"
	"time"

	"go-boilerplate/internal/shared/logger"

	"github.com/gin-gonic/gin"
)

// ResponseHandler provides centralized response handling for all APIs
type ResponseHandler struct {
	logger *logger.Logger
}

// NewResponseHandler creates a new response handler
func NewResponseHandler(log *logger.Logger) *ResponseHandler {
	return &ResponseHandler{
		logger: log.Named("response-handler"),
	}
}

// Success sends a standardized success response
func (rh *ResponseHandler) Success(
	w http.ResponseWriter,
	ctx context.Context,
	result interface{},
	message string,
) {
	requestID := GetRequestIDFromContext(ctx)
	sessionID := GetSessionIDFromContext(ctx)

	// Get the actual start time from context
	startTime, ok := GetStartTimeFromContext(ctx)
	if !ok {
		startTime = time.Now() // Fallback if start time not found
	}
	responseCtx := NewResponseContextWithStartTime(requestID, sessionID, startTime)

	RespondWithSuccess(w, ctx, responseCtx, result, message)
}

// Error sends a standardized error response
func (rh *ResponseHandler) Error(
	w http.ResponseWriter,
	ctx context.Context,
	message string,
	statusCode int,
) {
	requestID := GetRequestIDFromContext(ctx)
	sessionID := GetSessionIDFromContext(ctx)

	// Get the actual start time from context
	startTime, ok := GetStartTimeFromContext(ctx)
	if !ok {
		startTime = time.Now() // Fallback if start time not found
	}
	responseCtx := NewResponseContextWithStartTime(requestID, sessionID, startTime)

	RespondWithError(w, ctx, responseCtx, message, statusCode, rh.logger)
}

// InternalError sends a standardized internal server error response
func (rh *ResponseHandler) InternalError(
	w http.ResponseWriter,
	ctx context.Context,
	err error,
) {
	requestID := GetRequestIDFromContext(ctx)
	sessionID := GetSessionIDFromContext(ctx)

	// Get the actual start time from context
	startTime, ok := GetStartTimeFromContext(ctx)
	if !ok {
		startTime = time.Now() // Fallback if start time not found
	}
	responseCtx := NewResponseContextWithStartTime(requestID, sessionID, startTime)

	RespondWithInternalError(w, ctx, responseCtx, err, rh.logger)
}

// Unauthorized sends a standardized unauthorized response
func (rh *ResponseHandler) Unauthorized(
	w http.ResponseWriter,
	ctx context.Context,
	message string,
) {
	rh.Error(w, ctx, message, http.StatusUnauthorized)
}

// BadRequest sends a standardized bad request response
func (rh *ResponseHandler) BadRequest(
	w http.ResponseWriter,
	ctx context.Context,
	message string,
) {
	rh.Error(w, ctx, message, http.StatusBadRequest)
}

// NotFound sends a standardized not found response
func (rh *ResponseHandler) NotFound(
	w http.ResponseWriter,
	ctx context.Context,
	message string,
) {
	rh.Error(w, ctx, message, http.StatusNotFound)
}

// Forbidden sends a standardized forbidden response
func (rh *ResponseHandler) Forbidden(
	w http.ResponseWriter,
	ctx context.Context,
	message string,
) {
	rh.Error(w, ctx, message, http.StatusForbidden)
}

// Conflict sends a standardized conflict response
func (rh *ResponseHandler) Conflict(
	w http.ResponseWriter,
	ctx context.Context,
	message string,
) {
	rh.Error(w, ctx, message, http.StatusConflict)
}

// Created sends a standardized created response
func (rh *ResponseHandler) Created(
	w http.ResponseWriter,
	ctx context.Context,
	result interface{},
	message string,
) {
	requestID := GetRequestIDFromContext(ctx)
	sessionID := GetSessionIDFromContext(ctx)
	responseCtx := NewResponseContext(requestID, sessionID)

	RespondWithStandardFormat(w, ctx, http.StatusCreated, true, result, message, responseCtx)
}

// NoContent sends a standardized no content response
func (rh *ResponseHandler) NoContent(
	w http.ResponseWriter,
	ctx context.Context,
) {
	requestID := GetRequestIDFromContext(ctx)
	sessionID := GetSessionIDFromContext(ctx)
	responseCtx := NewResponseContext(requestID, sessionID)

	RespondWithStandardFormat(w, ctx, http.StatusNoContent, true, nil, "No content", responseCtx)
}

// GinSuccess sends a standardized success response for Gin handlers
func (rh *ResponseHandler) GinSuccess(
	c *gin.Context,
	result interface{},
	message string,
) {
	requestID := c.GetString("request_id")
	sessionID := c.GetString("session_id")

	// Get the actual start time from context
	startTime := c.GetTime("start_time")
	responseCtx := NewResponseContextWithStartTime(requestID, sessionID, startTime)

	RespondWithStandardFormat(c.Writer, c.Request.Context(), http.StatusOK, true, result, message, responseCtx)
}

// GinError sends a standardized error response for Gin handlers
func (rh *ResponseHandler) GinError(
	c *gin.Context,
	message string,
	statusCode int,
) {
	requestID := c.GetString("request_id")
	sessionID := c.GetString("session_id")

	// Get the actual start time from context
	startTime := c.GetTime("start_time")
	responseCtx := NewResponseContextWithStartTime(requestID, sessionID, startTime)

	RespondWithError(c.Writer, c.Request.Context(), responseCtx, message, statusCode, rh.logger)
	c.Abort()
}

// GinInternalError sends a standardized internal server error response for Gin handlers
func (rh *ResponseHandler) GinInternalError(
	c *gin.Context,
	err error,
) {
	requestID := c.GetString("request_id")
	sessionID := c.GetString("session_id")
	responseCtx := NewResponseContext(requestID, sessionID)

	RespondWithInternalError(c.Writer, c.Request.Context(), responseCtx, err, rh.logger)
	c.Abort()
}

// GinUnauthorized sends a standardized unauthorized response for Gin handlers
func (rh *ResponseHandler) GinUnauthorized(
	c *gin.Context,
	message string,
) {
	rh.GinError(c, message, http.StatusUnauthorized)
}

// GinBadRequest sends a standardized bad request response for Gin handlers
func (rh *ResponseHandler) GinBadRequest(
	c *gin.Context,
	message string,
) {
	rh.GinError(c, message, http.StatusBadRequest)
}

// GinNotFound sends a standardized not found response for Gin handlers
func (rh *ResponseHandler) GinNotFound(
	c *gin.Context,
	message string,
) {
	rh.GinError(c, message, http.StatusNotFound)
}

// GinForbidden sends a standardized forbidden response for Gin handlers
func (rh *ResponseHandler) GinForbidden(
	c *gin.Context,
	message string,
) {
	rh.GinError(c, message, http.StatusForbidden)
}

// GinConflict sends a standardized conflict response for Gin handlers
func (rh *ResponseHandler) GinConflict(
	c *gin.Context,
	message string,
) {
	rh.GinError(c, message, http.StatusConflict)
}

// GinCreated sends a standardized created response for Gin handlers
func (rh *ResponseHandler) GinCreated(
	c *gin.Context,
	result interface{},
	message string,
) {
	requestID := c.GetString("request_id")
	sessionID := c.GetString("session_id")

	// Get the actual start time from context
	startTime := c.GetTime("start_time")
	responseCtx := NewResponseContextWithStartTime(requestID, sessionID, startTime)

	RespondWithStandardFormat(c.Writer, c.Request.Context(), http.StatusCreated, true, result, message, responseCtx)
}
