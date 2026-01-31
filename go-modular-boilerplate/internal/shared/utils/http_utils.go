package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"go.uber.org/zap"

	"go-boilerplate/internal/shared/logger"
)

// StandardizedResponse represents the standardized API response format
type StandardizedResponse struct {
	Code         int         `json:"code"`          // HTTP status code
	Status       bool        `json:"status"`        // true for success, false for error
	Result       interface{} `json:"result"`        // actual response data (can be null)
	Message      string      `json:"message"`       // success/error message
	ResponseTime int64       `json:"response_time"` // response time in milliseconds
	RequestID    string      `json:"request_id"`    // unique request identifier
	SessionID    string      `json:"session_id"`    // session identifier (if available)
	ServerID     string      `json:"server_id"`     // server identifier
}

// ResponseContext holds context information for response generation
type ResponseContext struct {
	RequestID string
	SessionID string
	StartTime time.Time
	ServerID  string
}

// NewResponseContext creates a new response context
func NewResponseContext(requestID, sessionID string) *ResponseContext {
	serverID := os.Getenv("SERVER_ID")
	if serverID == "" {
		serverID = "default-server"
	}

	return &ResponseContext{
		RequestID: requestID,
		SessionID: sessionID,
		StartTime: time.Now(),
		ServerID:  serverID,
	}
}

// NewResponseContextWithStartTime creates a new response context with a specific start time
func NewResponseContextWithStartTime(requestID, sessionID string, startTime time.Time) *ResponseContext {
	serverID := os.Getenv("SERVER_ID")
	if serverID == "" {
		serverID = "default-server"
	}

	return &ResponseContext{
		RequestID: requestID,
		SessionID: sessionID,
		StartTime: startTime,
		ServerID:  serverID,
	}
}

// startTimeKey is a key for storing request start time in context
type startTimeKey struct{}

// GetStartTimeFromContext retrieves the request start time from the context
func GetStartTimeFromContext(ctx context.Context) (time.Time, bool) {
	start, ok := ctx.Value(startTimeKey{}).(time.Time)
	return start, ok
}

// GetRequestIDFromContext extracts request ID from context
func GetRequestIDFromContext(ctx context.Context) string {
	if reqID, ok := ctx.Value("request_id").(string); ok {
		return reqID
	}
	return ""
}

// GetSessionIDFromContext extracts session ID from context
func GetSessionIDFromContext(ctx context.Context) string {
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		return sessionID
	}
	return ""
}

// RespondWithStandardFormat sends a standardized response
func RespondWithStandardFormat(
	w http.ResponseWriter,
	ctx context.Context,
	statusCode int,
	success bool,
	result interface{},
	message string,
	responseCtx *ResponseContext,
) {
	responseTime := time.Since(responseCtx.StartTime).Milliseconds()

	response := &StandardizedResponse{
		Code:         statusCode,
		Status:       success,
		Result:       result,
		Message:      message,
		ResponseTime: responseTime,
		RequestID:    responseCtx.RequestID,
		SessionID:    responseCtx.SessionID,
		ServerID:     responseCtx.ServerID,
	}

	// Set Content-Type header
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Encode response to JSON
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If encoding fails, log the error and send a plain text response
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// RespondWithSuccess formats and writes a successful response in standardized format
func RespondWithSuccess(
	w http.ResponseWriter,
	ctx context.Context,
	responseCtx *ResponseContext,
	result interface{},
	message string,
) {
	RespondWithStandardFormat(w, ctx, http.StatusOK, true, result, message, responseCtx)
}

// RespondWithError formats and writes an error response in standardized format
func RespondWithError(
	w http.ResponseWriter,
	ctx context.Context,
	responseCtx *ResponseContext,
	message string,
	statusCode int,
	log *logger.Logger,
) {
	// Log the error
	log.Error(
		"HTTP error response",
		zap.String("request_id", responseCtx.RequestID),
		zap.String("session_id", responseCtx.SessionID),
		zap.Int("status_code", statusCode),
		zap.String("message", message),
		zap.Int64("response_time", time.Since(responseCtx.StartTime).Milliseconds()),
	)

	RespondWithStandardFormat(w, ctx, statusCode, false, nil, message, responseCtx)
}

// RespondWithInternalError handles unhandled errors (500) in standardized format
func RespondWithInternalError(
	w http.ResponseWriter,
	ctx context.Context,
	responseCtx *ResponseContext,
	err error,
	log *logger.Logger,
) {
	log.Error(
		"Internal server error",
		zap.String("request_id", responseCtx.RequestID),
		zap.String("session_id", responseCtx.SessionID),
		zap.Error(err),
		zap.Int64("response_time", time.Since(responseCtx.StartTime).Milliseconds()),
	)

	RespondWithStandardFormat(w, ctx, http.StatusInternalServerError, false, nil, "Internal server error", responseCtx)
}

// HTTPError is a struct for handling HTTP errors
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// StatusCode returns the HTTP status code
func (e *HTTPError) StatusCode() int {
	return e.Code
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(code int, message string, err error) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// DecodeJSON decodes JSON from an HTTP request
func DecodeJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return NewHTTPError(http.StatusBadRequest, "Invalid JSON payload", err)
	}
	return nil
}

// StatusCoder interface for errors that have HTTP status codes
type StatusCoder interface {
	StatusCode() int
}

// HandleServiceError handles service errors and returns appropriate HTTP response
func HandleServiceError(w http.ResponseWriter, ctx context.Context, err error, responseHandler *ResponseHandler) {
	if statusErr, ok := err.(StatusCoder); ok {
		// Error has a status code, use it
		statusCode := statusErr.StatusCode()
		switch statusCode {
		case 400:
			responseHandler.BadRequest(w, ctx, err.Error())
		case 401:
			responseHandler.Unauthorized(w, ctx, err.Error())
		case 403:
			responseHandler.Forbidden(w, ctx, err.Error())
		case 404:
			responseHandler.NotFound(w, ctx, err.Error())
		case 409:
			responseHandler.Conflict(w, ctx, err.Error())
		default:
			responseHandler.InternalError(w, ctx, err)
		}
	} else {
		// Generic error, treat as internal server error
		responseHandler.InternalError(w, ctx, err)
	}
}

// GinHandleServiceError handles service errors for Gin handlers
func GinHandleServiceError(c *gin.Context, err error, responseHandler *ResponseHandler) {
	if statusErr, ok := err.(StatusCoder); ok {
		// Error has a status code, use it
		statusCode := statusErr.StatusCode()
		switch statusCode {
		case 400:
			responseHandler.GinBadRequest(c, err.Error())
		case 401:
			responseHandler.GinUnauthorized(c, err.Error())
		case 403:
			responseHandler.GinForbidden(c, err.Error())
		case 404:
			responseHandler.GinNotFound(c, err.Error())
		case 409:
			responseHandler.GinConflict(c, err.Error())
		default:
			responseHandler.GinInternalError(c, err)
		}
	} else {
		// Generic error, treat as internal server error
		responseHandler.GinInternalError(c, err)
	}
}

// RespondWithJSON sends a raw JSON response (not wrapped in standard response format)
func RespondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If encoding fails, send a plain text error
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
