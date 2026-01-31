package middleware

// contextKey is a type used for context keys to avoid string key collisions
type contextKey string

// Context keys for middleware
const (
	UserIDKey    contextKey = "user_id"
	RequestIDKey contextKey = "request_id"
	SessionIDKey contextKey = "session_id"
	StartTimeKey contextKey = "start_time"
)
