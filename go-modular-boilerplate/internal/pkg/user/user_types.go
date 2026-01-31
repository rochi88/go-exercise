package user

import (
	"time"

	validator "github.com/go-playground/validator/v10"

	"go-boilerplate/internal/pkg/auth"
)

// User represents a user domain model
type User struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	PasswordHash      string    `json:"-"` // Never expose in JSON
	EmailVerified     bool      `json:"email_verified"`
	VendorID          string    `json:"vendor_id"`
	Country           string    `json:"country"`
	City              string    `json:"city"`
	IsActive          bool      `json:"is_active"`
	IsDisabled        bool      `json:"is_disabled"`
	EnableSocialLogin bool      `json:"enable_social_login"`
	SignupSource      string    `json:"signup_source"`
	CreatedAt         time.Time `json:"created_at"`
}

// Profile extends the auth.User to include additional user-specific fields
type Profile struct {
	auth.User
	// Add additional user profile fields here, for example:
	// Bio        string    `json:"bio" db:"bio"`
	// AvatarURL  string    `json:"avatar_url" db:"avatar_url"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Name         string `json:"name" validate:"required,min=1,max=100"`
	Email        string `json:"email" validate:"required,email,max=255"`
	PasswordHash string `json:"-" validate:"required"` // Set internally, not from JSON
	VendorID     string `json:"vendor_id" validate:"required,max=50"`
	Country      string `json:"country" validate:"omitempty,len=2"`
	City         string `json:"city" validate:"omitempty,max=50"`
	SignupSource string `json:"signup_source" validate:"omitempty,max=25"`
}

// UpdateUserRequest represents the request to update user information
type UpdateUserRequest struct {
	Name              string `json:"name" validate:"omitempty,min=1,max=100"`
	Email             string `json:"email" validate:"omitempty,email,max=255"`
	Country           string `json:"country" validate:"omitempty,len=2"`
	City              string `json:"city" validate:"omitempty,max=50"`
	EmailVerified     bool   `json:"email_verified"`
	EnableSocialLogin bool   `json:"enable_social_login"`
}

// ProfileResponse represents the user profile response
type ProfileResponse struct {
	User Profile `json:"user"`
}

// UpdateProfileRequest represents the request to update a user profile
type UpdateProfileRequest struct {
	FirstName string `json:"first_name" validate:"omitempty"`
	LastName  string `json:"last_name" validate:"omitempty"`
	// Add additional fields that can be updated, for example:
	// Bio       string `json:"bio" validate:"omitempty"`
	// AvatarURL string `json:"avatar_url" validate:"omitempty,url"`
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users      []*User `json:"users"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// Validate methods for requests
func (r *CreateUserRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

func (r *UpdateUserRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

func (r *UpdateProfileRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

// Validate validates a struct using the validator package
func Validate(s interface{}) error {
	validate := validator.New()
	return validate.Struct(s)
}

// Domain errors for user module with appropriate HTTP status codes
var (
	ErrUserNotFound = NewUserErrorWithCode("user not found", 404)
)

// UserError represents a user-related error with HTTP status code
type UserError struct {
	Message string
	Code    int // HTTP status code
}

// NewUserErrorWithCode creates a new UserError with specific HTTP status code
func NewUserErrorWithCode(message string, code int) *UserError {
	return &UserError{
		Message: message,
		Code:    code,
	}
}

// Error implements the error interface
func (e *UserError) Error() string {
	return e.Message
}

// HTTPStatusCode returns the HTTP status code for this error
func (e *UserError) HTTPStatusCode() int {
	return e.Code
}
