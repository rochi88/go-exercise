package health

import "time"

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string           `json:"status"` // "healthy" or "unhealthy"
	Timestamp time.Time        `json:"timestamp"`
	Version   string           `json:"version,omitempty"`
	Uptime    time.Duration    `json:"uptime,omitempty"`
	Services  map[string]Check `json:"services,omitempty"`
}

// Check represents the health check result for a service
type Check struct {
	Status  string    `json:"status"` // "healthy", "unhealthy", or "unknown"
	Message string    `json:"message,omitempty"`
	Time    time.Time `json:"time"`
}

// HealthChecker interface for checking service health
type HealthChecker interface {
	Check() Check
	Name() string
}

// ReadinessStatus represents the readiness status
type ReadinessStatus struct {
	Status    string           `json:"status"` // "ready" or "not_ready"
	Timestamp time.Time        `json:"timestamp"`
	Services  map[string]Check `json:"services,omitempty"`
}

// ReadinessChecker interface for checking service readiness
type ReadinessChecker interface {
	Check() Check
	Name() string
}
