package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-boilerplate/internal/shared/logger"
)

// SecurityConfig holds configuration for security headers
type SecurityConfig struct {
	// Content Security Policy
	ContentSecurityPolicy string

	// Frame options
	XFrameOptions string

	// Content type options
	XContentTypeOptions string

	// XSS protection
	XXSSProtection string

	// HSTS (HTTP Strict Transport Security)
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool

	// Referrer policy
	ReferrerPolicy string

	// Permissions policy
	PermissionsPolicy string

	// Cross-Origin policies
	CrossOriginEmbedderPolicy string
	CrossOriginOpenerPolicy   string
	CrossOriginResourcePolicy string

	// Feature policy (deprecated, but still used)
	FeaturePolicy string

	// DNS prefetch control
	DNSPrefetchControl string

	// IE specific headers
	IENoOpen string

	// Environment-based settings
	IsDevelopment bool
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig(isDevelopment bool) *SecurityConfig {
	config := &SecurityConfig{
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'",
		XFrameOptions:             "DENY",
		XContentTypeOptions:       "nosniff",
		XXSSProtection:            "1; mode=block",
		HSTSMaxAge:                31536000, // 1 year
		HSTSIncludeSubdomains:     true,
		HSTSPreload:               false,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "camera=(), microphone=(), geolocation=()",
		CrossOriginEmbedderPolicy: "credentialless",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		FeaturePolicy:             "camera 'none'; microphone 'none'; geolocation 'none'",
		DNSPrefetchControl:        "on",
		IENoOpen:                  "noopen",
		IsDevelopment:             isDevelopment,
	}

	// Relax some policies for development
	if isDevelopment {
		config.ContentSecurityPolicy = "default-src 'self' 'unsafe-eval' 'unsafe-inline'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https: http:; font-src 'self' data:; connect-src 'self' ws: wss: http: https:"
		config.XFrameOptions = "SAMEORIGIN"
		config.HSTSIncludeSubdomains = false
	}

	return config
}

// SecurityMiddleware provides security headers middleware
type SecurityMiddleware struct {
	config *SecurityConfig
	logger *logger.Logger
}

// NewSecurityMiddleware creates a new security middleware with default configuration
func NewSecurityMiddleware(logger *logger.Logger, isDevelopment bool) *SecurityMiddleware {
	return &SecurityMiddleware{
		config: DefaultSecurityConfig(isDevelopment),
		logger: logger.Named("security"),
	}
}

// NewSecurityMiddlewareWithConfig creates a new security middleware with custom configuration
func NewSecurityMiddlewareWithConfig(config *SecurityConfig, logger *logger.Logger) *SecurityMiddleware {
	return &SecurityMiddleware{
		config: config,
		logger: logger.Named("security"),
	}
}

// GinSecurityHeaders provides Gin-compatible security headers middleware
func (sm *SecurityMiddleware) GinSecurityHeaders(c *gin.Context) {
	// Content Security Policy
	if sm.config.ContentSecurityPolicy != "" {
		c.Header("Content-Security-Policy", sm.config.ContentSecurityPolicy)
	}

	// X-Frame-Options
	if sm.config.XFrameOptions != "" {
		c.Header("X-Frame-Options", sm.config.XFrameOptions)
	}

	// X-Content-Type-Options
	if sm.config.XContentTypeOptions != "" {
		c.Header("X-Content-Type-Options", sm.config.XContentTypeOptions)
	}

	// X-XSS-Protection
	if sm.config.XXSSProtection != "" {
		c.Header("X-XSS-Protection", sm.config.XXSSProtection)
	}

	// Strict-Transport-Security (HSTS)
	if !sm.config.IsDevelopment && sm.config.HSTSMaxAge > 0 {
		hstsValue := fmt.Sprintf("max-age=%d", sm.config.HSTSMaxAge)
		if sm.config.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if sm.config.HSTSPreload {
			hstsValue += "; preload"
		}
		c.Header("Strict-Transport-Security", hstsValue)
	}

	// Referrer-Policy
	if sm.config.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", sm.config.ReferrerPolicy)
	}

	// Permissions-Policy
	if sm.config.PermissionsPolicy != "" {
		c.Header("Permissions-Policy", sm.config.PermissionsPolicy)
	}

	// Cross-Origin-Embedder-Policy
	if sm.config.CrossOriginEmbedderPolicy != "" {
		c.Header("Cross-Origin-Embedder-Policy", sm.config.CrossOriginEmbedderPolicy)
	}

	// Cross-Origin-Opener-Policy
	if sm.config.CrossOriginOpenerPolicy != "" {
		c.Header("Cross-Origin-Opener-Policy", sm.config.CrossOriginOpenerPolicy)
	}

	// Cross-Origin-Resource-Policy
	if sm.config.CrossOriginResourcePolicy != "" {
		c.Header("Cross-Origin-Resource-Policy", sm.config.CrossOriginResourcePolicy)
	}

	// Feature-Policy (deprecated but still supported)
	if sm.config.FeaturePolicy != "" {
		c.Header("Feature-Policy", sm.config.FeaturePolicy)
	}

	// X-DNS-Prefetch-Control
	if sm.config.DNSPrefetchControl != "" {
		c.Header("X-DNS-Prefetch-Control", sm.config.DNSPrefetchControl)
	}

	// X-Download-Options (IE specific)
	c.Header("X-Download-Options", "noopen")

	// X-Permitted-Cross-Domain-Policies
	c.Header("X-Permitted-Cross-Domain-Policies", "none")

	c.Next()
}

// SecurityHeaders provides standard http.Handler compatible security headers middleware
func (sm *SecurityMiddleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		if sm.config.ContentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", sm.config.ContentSecurityPolicy)
		}

		// X-Frame-Options
		if sm.config.XFrameOptions != "" {
			w.Header().Set("X-Frame-Options", sm.config.XFrameOptions)
		}

		// X-Content-Type-Options
		if sm.config.XContentTypeOptions != "" {
			w.Header().Set("X-Content-Type-Options", sm.config.XContentTypeOptions)
		}

		// X-XSS-Protection
		if sm.config.XXSSProtection != "" {
			w.Header().Set("X-XSS-Protection", sm.config.XXSSProtection)
		}

		// Strict-Transport-Security (HSTS)
		if !sm.config.IsDevelopment && sm.config.HSTSMaxAge > 0 {
			hstsValue := fmt.Sprintf("max-age=%d", sm.config.HSTSMaxAge)
			if sm.config.HSTSIncludeSubdomains {
				hstsValue += "; includeSubDomains"
			}
			if sm.config.HSTSPreload {
				hstsValue += "; preload"
			}
			w.Header().Set("Strict-Transport-Security", hstsValue)
		}

		// Referrer-Policy
		if sm.config.ReferrerPolicy != "" {
			w.Header().Set("Referrer-Policy", sm.config.ReferrerPolicy)
		}

		// Permissions-Policy
		if sm.config.PermissionsPolicy != "" {
			w.Header().Set("Permissions-Policy", sm.config.PermissionsPolicy)
		}

		// Cross-Origin-Embedder-Policy
		if sm.config.CrossOriginEmbedderPolicy != "" {
			w.Header().Set("Cross-Origin-Embedder-Policy", sm.config.CrossOriginEmbedderPolicy)
		}

		// Cross-Origin-Opener-Policy
		if sm.config.CrossOriginOpenerPolicy != "" {
			w.Header().Set("Cross-Origin-Opener-Policy", sm.config.CrossOriginOpenerPolicy)
		}

		// Cross-Origin-Resource-Policy
		if sm.config.CrossOriginResourcePolicy != "" {
			w.Header().Set("Cross-Origin-Resource-Policy", sm.config.CrossOriginResourcePolicy)
		}

		// Feature-Policy (deprecated but still supported)
		if sm.config.FeaturePolicy != "" {
			w.Header().Set("Feature-Policy", sm.config.FeaturePolicy)
		}

		// X-DNS-Prefetch-Control
		if sm.config.DNSPrefetchControl != "" {
			w.Header().Set("X-DNS-Prefetch-Control", sm.config.DNSPrefetchControl)
		}

		// X-Download-Options (IE specific)
		w.Header().Set("X-Download-Options", "noopen")

		// X-Permitted-Cross-Domain-Policies
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		next.ServeHTTP(w, r)
	})
}

// GetSecurityHeaders returns a map of all security headers that would be set
func (sm *SecurityMiddleware) GetSecurityHeaders() map[string]string {
	headers := make(map[string]string)

	if sm.config.ContentSecurityPolicy != "" {
		headers["Content-Security-Policy"] = sm.config.ContentSecurityPolicy
	}

	if sm.config.XFrameOptions != "" {
		headers["X-Frame-Options"] = sm.config.XFrameOptions
	}

	if sm.config.XContentTypeOptions != "" {
		headers["X-Content-Type-Options"] = sm.config.XContentTypeOptions
	}

	if sm.config.XXSSProtection != "" {
		headers["X-XSS-Protection"] = sm.config.XXSSProtection
	}

	if !sm.config.IsDevelopment && sm.config.HSTSMaxAge > 0 {
		hstsValue := fmt.Sprintf("max-age=%d", sm.config.HSTSMaxAge)
		if sm.config.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if sm.config.HSTSPreload {
			hstsValue += "; preload"
		}
		headers["Strict-Transport-Security"] = hstsValue
	}

	if sm.config.ReferrerPolicy != "" {
		headers["Referrer-Policy"] = sm.config.ReferrerPolicy
	}

	if sm.config.PermissionsPolicy != "" {
		headers["Permissions-Policy"] = sm.config.PermissionsPolicy
	}

	if sm.config.CrossOriginEmbedderPolicy != "" {
		headers["Cross-Origin-Embedder-Policy"] = sm.config.CrossOriginEmbedderPolicy
	}

	if sm.config.CrossOriginOpenerPolicy != "" {
		headers["Cross-Origin-Opener-Policy"] = sm.config.CrossOriginOpenerPolicy
	}

	if sm.config.CrossOriginResourcePolicy != "" {
		headers["Cross-Origin-Resource-Policy"] = sm.config.CrossOriginResourcePolicy
	}

	if sm.config.FeaturePolicy != "" {
		headers["Feature-Policy"] = sm.config.FeaturePolicy
	}

	if sm.config.DNSPrefetchControl != "" {
		headers["X-DNS-Prefetch-Control"] = sm.config.DNSPrefetchControl
	}

	headers["X-Download-Options"] = "noopen"
	headers["X-Permitted-Cross-Domain-Policies"] = "none"

	return headers
}
