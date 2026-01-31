package auth

import (
	"time"

	validator "github.com/go-playground/validator/v10"
)

// User represents a user in the authentication system
type User struct {
	ID                string    `json:"id" db:"id"`
	Email             string    `json:"email" db:"email"`
	Name              string    `json:"name" db:"name"`
	Password          string    `json:"-" db:"password_hash"` // Password hash, not returned in JSON
	EmailVerified     bool      `json:"email_verified" db:"email_verified"`
	VendorID          string    `json:"vendor_id" db:"vendor_id"`
	Country           *string   `json:"country" db:"country"`
	City              *string   `json:"city" db:"city"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	IsDisabled        bool      `json:"is_disabled" db:"is_disabled"`
	EnableSocialLogin bool      `json:"enable_social_login" db:"enable_social_login"`
	SignupSource      *string   `json:"signup_source" db:"signup_source"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// Session represents a user session in the authentication system
type Session struct {
	ID                string     `json:"id" db:"id"`
	UserID            string     `json:"user_id" db:"user_id"`
	RefreshTokenHash  string     `json:"-" db:"refresh_token_hash"` // Token hash, not returned in JSON
	IPAddress         string     `json:"ip_address" db:"ip_address"`
	DeviceName        string     `json:"device_name" db:"device_name"`
	UserAgent         string     `json:"user_agent" db:"user_agent"`
	TrustScore        int        `json:"trust_score" db:"trust_score"`
	City              string     `json:"city" db:"city"`
	Country           string     `json:"country" db:"country"`
	Region            string     `json:"region" db:"region"`
	Timezone          string     `json:"timezone" db:"timezone"`
	ISP               string     `json:"isp" db:"isp"`
	DeviceFingerprint string     `json:"device_fingerprint" db:"device_fingerprint"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	TrustedDevice     bool       `json:"trusted_device" db:"trusted_device"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	ValidTill         time.Time  `json:"valid_till" db:"valid_till"`
	LastUsed          *time.Time `json:"last_used" db:"last_used"`
	RevokedAt         *time.Time `json:"revoked_at" db:"revoked_at"`
}

// DeviceInfo represents simplified device information for session tracking
type DeviceInfo struct {
	Name        string `json:"device_name"` // Format: "Browser (OS) - OS_Version"
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"device_fingerprint"`
	TrustScore  int    `json:"trust_score"` // Risk score from 0-100
	City        string `json:"city,omitempty"`
	Country     string `json:"country,omitempty"`
	Region      string `json:"region,omitempty"`
	ISP         string `json:"isp,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	VendorID string `json:"vendor_id" validate:"required"`
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email        string  `json:"email" validate:"required,email"`
	Password     string  `json:"password" validate:"required,min=6"`
	VendorID     string  `json:"vendor_id" validate:"required"`
	SignupSource *string `json:"signup_source,omitempty"` // Optional signup source
}

// LoginResponse represents the login response payload
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // Access token expiration in seconds
	User         User   `json:"user"`
}

// RefreshTokenRequest represents the refresh token request payload
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshTokenResponse represents the refresh token response payload
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // Access token expiration in seconds
}

// VerifyEmailRequest represents the verify email request payload
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyEmailResponse represents the verify email response payload
type VerifyEmailResponse struct {
	Message string `json:"message"`
	User    User   `json:"user"`
}

// RequestPasswordResetRequest represents the request password reset payload
type RequestPasswordResetRequest struct {
	Email    string `json:"email" validate:"required,email"`
	VendorID string `json:"vendor_id" validate:"required"`
}

// RequestPasswordResetResponse represents the request password reset response payload
type RequestPasswordResetResponse struct {
	Message string `json:"message"`
}

// ResetPasswordRequest represents the reset password request payload
type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

// ResetPasswordResponse represents the reset password response payload
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}

// ChangePasswordResponse represents the change password response payload
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

// TokenPair represents both access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// Validate validates a struct using the validator package
func Validate(s interface{}) error {
	validate := validator.New()
	return validate.Struct(s)
}

// Domain errors for authentication with appropriate HTTP status codes
var (
	ErrInvalidCredentials  = NewAuthErrorWithCode("invalid email or password", 401)
	ErrUserAlreadyExists   = NewAuthErrorWithCode("user with this email already exists", 409)
	ErrUserNotFound        = NewAuthErrorWithCode("user not found", 404)
	ErrEmailNotVerified    = NewAuthErrorWithCode("email not verified", 403)
	ErrInvalidRefreshToken = NewAuthErrorWithCode("invalid refresh token", 401)
	ErrRefreshTokenExpired = NewAuthErrorWithCode("refresh token expired", 401)
	ErrSessionNotFound     = NewAuthErrorWithCode("session not found", 404)
	ErrSessionRevoked      = NewAuthErrorWithCode("session revoked", 401)
)

// AuthError represents an authentication error with HTTP status code
type AuthError struct {
	Message string
	Code    int // HTTP status code
}

// NewAuthError creates a new AuthError with default 500 status
func NewAuthError(message string) *AuthError {
	return &AuthError{
		Message: message,
		Code:    500, // Default to internal server error
	}
}

// NewAuthErrorWithCode creates a new AuthError with specific HTTP status code
func NewAuthErrorWithCode(message string, code int) *AuthError {
	return &AuthError{
		Message: message,
		Code:    code,
	}
}

func (e *AuthError) Error() string {
	return e.Message
}

// StatusCode returns the HTTP status code for this error
func (e *AuthError) StatusCode() int {
	return e.Code
}
