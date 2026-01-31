package healthHttp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	healthService "go-boilerplate/internal/pkg/health/service"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/utils"
)

// HealthHandler handles HTTP requests for health checks
type HealthHandler struct {
	service         healthService.HealthService
	logger          *logger.Logger
	responseHandler *utils.ResponseHandler
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(service *healthService.HealthService, logger *logger.Logger) *HealthHandler {
	return &HealthHandler{
		service:         *service,
		logger:          logger.Named("health-handler"),
		responseHandler: utils.NewResponseHandler(logger.Named("health-responses")),
	}
}

// Health responds with the health status
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Get health status
	healthStatus := h.service.Health(r.Context())

	// Determine HTTP status code
	statusCode := http.StatusOK
	if healthStatus.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	// Log the request
	h.logger.Info("Health check requested",
		zap.String("status", healthStatus.Status),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", time.Since(start)),
	)

	// Use standardized response format
	if healthStatus.Status == "healthy" {
		h.responseHandler.Success(w, r.Context(), healthStatus, "Health check passed")
	} else {
		h.responseHandler.Error(w, r.Context(), "Health check failed", statusCode)
	}
}

// Ready responds with the readiness status
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Get readiness status
	readinessStatus := h.service.Ready(r.Context())

	// Determine HTTP status code
	statusCode := http.StatusOK
	if readinessStatus.Status != "ready" {
		statusCode = http.StatusServiceUnavailable
	}

	// Log the request
	h.logger.Info("Readiness check requested",
		zap.String("status", readinessStatus.Status),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", time.Since(start)),
	)

	// Use standardized response format
	if readinessStatus.Status == "ready" {
		h.responseHandler.Success(w, r.Context(), readinessStatus, "Readiness check passed")
	} else {
		h.responseHandler.Error(w, r.Context(), "Readiness check failed", statusCode)
	}
}

// GinHealth provides Gin-compatible health check endpoint
func (h *HealthHandler) GinHealth(c *gin.Context) {
	start := time.Now()

	// Get health status
	healthStatus := h.service.Health(c.Request.Context())

	// Determine HTTP status code
	statusCode := http.StatusOK
	if healthStatus.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	// Log the request
	h.logger.Info("Health check requested",
		zap.String("status", healthStatus.Status),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", time.Since(start)),
	)

	// Use standardized response format
	if healthStatus.Status == "healthy" {
		h.responseHandler.GinSuccess(c, healthStatus, "Health check passed")
	} else {
		h.responseHandler.GinError(c, "Health check failed", statusCode)
	}
}

// GinReady provides Gin-compatible readiness check endpoint
func (h *HealthHandler) GinReady(c *gin.Context) {
	start := time.Now()

	// Get readiness status
	readinessStatus := h.service.Ready(c.Request.Context())

	// Determine HTTP status code
	statusCode := http.StatusOK
	if readinessStatus.Status != "ready" {
		statusCode = http.StatusServiceUnavailable
	}

	// Log the request
	h.logger.Info("Readiness check requested",
		zap.String("status", readinessStatus.Status),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", time.Since(start)),
	)

	// Use standardized response format
	if readinessStatus.Status == "ready" {
		h.responseHandler.GinSuccess(c, readinessStatus, "Readiness check passed")
	} else {
		h.responseHandler.GinError(c, "Readiness check failed", statusCode)
	}
}
