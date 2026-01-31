package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go-boilerplate/internal/shared/logger"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec

	// Database metrics
	dbConnectionsTotal  prometheus.Gauge
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbQueryDuration     *prometheus.HistogramVec
	dbQueryErrorsTotal  *prometheus.CounterVec

	// Redis metrics
	redisConnectionsTotal  prometheus.Gauge
	redisOperationsTotal   *prometheus.CounterVec
	redisOperationDuration *prometheus.HistogramVec
	redisErrorsTotal       *prometheus.CounterVec

	// Business logic metrics
	userRegistrationsTotal prometheus.Counter
	userLoginsTotal        prometheus.Counter
	userLoginErrorsTotal   prometheus.Counter

	// System metrics
	uptime prometheus.Gauge

	logger *logger.Logger
}

// New creates a new metrics instance
func New(logger *logger.Logger) *Metrics {
	m := &Metrics{
		logger: logger.Named("metrics"),
	}

	m.initHTTPMetrics()
	m.initDatabaseMetrics()
	m.initRedisMetrics()
	m.initBusinessMetrics()
	m.initSystemMetrics()

	m.logger.Info("Metrics initialized")

	return m
}

// initHTTPMetrics initializes HTTP-related metrics
func (m *Metrics) initHTTPMetrics() {
	m.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	m.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"method", "endpoint"},
	)
}

// initDatabaseMetrics initializes database-related metrics
func (m *Metrics) initDatabaseMetrics() {
	m.dbConnectionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_total",
			Help: "Total number of database connections",
		},
	)

	m.dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	m.dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	m.dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	m.dbQueryErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors",
		},
		[]string{"operation", "table"},
	)
}

// initRedisMetrics initializes Redis-related metrics
func (m *Metrics) initRedisMetrics() {
	m.redisConnectionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_connections_total",
			Help: "Total number of Redis connections",
		},
	)

	m.redisOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "key"},
	)

	m.redisOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Redis operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	m.redisErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_errors_total",
			Help: "Total number of Redis errors",
		},
		[]string{"operation"},
	)
}

// initBusinessMetrics initializes business logic metrics
func (m *Metrics) initBusinessMetrics() {
	m.userRegistrationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "user_registrations_total",
			Help: "Total number of user registrations",
		},
	)

	m.userLoginsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "user_logins_total",
			Help: "Total number of user logins",
		},
	)

	m.userLoginErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "user_login_errors_total",
			Help: "Total number of user login errors",
		},
	)
}

// initSystemMetrics initializes system metrics
func (m *Metrics) initSystemMetrics() {
	m.uptime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)
}

// HTTP Metrics Methods

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, endpoint string, status int, duration time.Duration) {
	statusStr := strconv.Itoa(status)

	m.httpRequestsTotal.WithLabelValues(method, endpoint, statusStr).Inc()
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// IncrementHTTPRequestsInFlight increments the number of in-flight HTTP requests
func (m *Metrics) IncrementHTTPRequestsInFlight(method, endpoint string) {
	m.httpRequestsInFlight.WithLabelValues(method, endpoint).Inc()
}

// DecrementHTTPRequestsInFlight decrements the number of in-flight HTTP requests
func (m *Metrics) DecrementHTTPRequestsInFlight(method, endpoint string) {
	m.httpRequestsInFlight.WithLabelValues(method, endpoint).Dec()
}

// Database Metrics Methods

// UpdateDBConnections updates database connection metrics
func (m *Metrics) UpdateDBConnections(total, active, idle int) {
	m.dbConnectionsTotal.Set(float64(total))
	m.dbConnectionsActive.Set(float64(active))
	m.dbConnectionsIdle.Set(float64(idle))
}

// RecordDBQuery records a database query
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())

	if err != nil {
		m.dbQueryErrorsTotal.WithLabelValues(operation, table).Inc()
	}
}

// Redis Metrics Methods

// UpdateRedisConnections updates Redis connection metrics
func (m *Metrics) UpdateRedisConnections(total int) {
	m.redisConnectionsTotal.Set(float64(total))
}

// RecordRedisOperation records a Redis operation
func (m *Metrics) RecordRedisOperation(operation, key string, duration time.Duration, err error) {
	m.redisOperationsTotal.WithLabelValues(operation, key).Inc()
	m.redisOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())

	if err != nil {
		m.redisErrorsTotal.WithLabelValues(operation).Inc()
	}
}

// Business Metrics Methods

// RecordUserRegistration records a user registration
func (m *Metrics) RecordUserRegistration() {
	m.userRegistrationsTotal.Inc()
}

// RecordUserLogin records a user login
func (m *Metrics) RecordUserLogin() {
	m.userLoginsTotal.Inc()
}

// RecordUserLoginError records a user login error
func (m *Metrics) RecordUserLoginError() {
	m.userLoginErrorsTotal.Inc()
}

// System Metrics Methods

// RecordUptime records the application uptime
func (m *Metrics) RecordUptime(uptime time.Duration) {
	m.uptime.Set(uptime.Seconds())
}

// GinMiddleware returns a Gin middleware for collecting HTTP metrics
func (m *Metrics) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path

		// Increment in-flight requests
		m.IncrementHTTPRequestsInFlight(method, path)
		defer m.DecrementHTTPRequestsInFlight(method, path)

		// Process request
		c.Next()

		// Record metrics
		status := c.Writer.Status()
		duration := time.Since(start)

		m.RecordHTTPRequest(method, path, status, duration)
	}
}

// GinMetricsHandler returns a Gin handler for the /metrics endpoint
func (m *Metrics) GinMetricsHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}
