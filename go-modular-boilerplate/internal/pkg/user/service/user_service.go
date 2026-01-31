package userService

import (
	"context"
	"fmt"

	"go-boilerplate/internal/pkg/auth"
	"go-boilerplate/internal/pkg/user"
	userRepository "go-boilerplate/internal/pkg/user/repository"
	"go-boilerplate/internal/shared/interfaces"
	"go-boilerplate/internal/shared/logger"
	passwordUtils "go-boilerplate/internal/utils/password"

	"go.uber.org/zap"
)

// Helper functions for pointer conversions
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// UserService defines the interface for user service operations
type UserService interface {
	interfaces.Service
	GetProfile(ctx context.Context, userID string) (*user.Profile, error)
	UpdateProfile(ctx context.Context, userID string, req *user.UpdateProfileRequest) (*user.Profile, error)
	GetUserByID(ctx context.Context, userID string) (*user.User, error)
	GetUserByEmail(ctx context.Context, email, vendorID string) (*user.User, error)
	CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.User, error)
	UpdateUser(ctx context.Context, userID string, req *user.UpdateUserRequest) (*user.User, error)
	UpdateUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error
	DeactivateUser(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, vendorID string, page, pageSize int) (*user.ListUsersResponse, error)
}

// DefaultUserService is the default implementation of UserService
type DefaultUserService struct {
	repo   userRepository.SqlcUserRepository
	logger *logger.Logger
}

// NewUserService creates a new user service
func NewUserService(repo userRepository.SqlcUserRepository, log *logger.Logger) UserService {
	return &DefaultUserService{
		repo:   repo,
		logger: log.Named("user-service"),
	}
}

// GetProfile retrieves a user's profile
func (s *DefaultUserService) GetProfile(ctx context.Context, userID string) (*user.Profile, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// UpdateProfile updates a user's profile
func (s *DefaultUserService) UpdateProfile(ctx context.Context, userID string, req *user.UpdateProfileRequest) (*user.Profile, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	updateReq := &user.UpdateUserRequest{
		Name: req.FirstName + " " + req.LastName,
	}

	updatedUser, err := s.repo.UpdateUser(ctx, userID, updateReq)
	if err != nil {
		return nil, err
	}

	authUser := auth.User{
		ID:                updatedUser.ID,
		Name:              updatedUser.Name,
		Email:             updatedUser.Email,
		EmailVerified:     updatedUser.EmailVerified,
		VendorID:          updatedUser.VendorID,
		Country:           stringToPtr(updatedUser.Country),
		City:              stringToPtr(updatedUser.City),
		IsActive:          updatedUser.IsActive,
		IsDisabled:        updatedUser.IsDisabled,
		EnableSocialLogin: updatedUser.EnableSocialLogin,
		SignupSource:      stringToPtr(updatedUser.SignupSource),
		CreatedAt:         updatedUser.CreatedAt,
	}

	return &user.Profile{User: authUser}, nil
}

// GetUserByID retrieves a user by ID
func (s *DefaultUserService) GetUserByID(ctx context.Context, userID string) (*user.User, error) {
	profile, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &user.User{
		ID:                profile.User.ID,
		Name:              profile.User.Name,
		Email:             profile.User.Email,
		EmailVerified:     profile.User.EmailVerified,
		VendorID:          profile.User.VendorID,
		Country:           ptrToString(profile.User.Country),
		City:              ptrToString(profile.User.City),
		IsActive:          profile.User.IsActive,
		IsDisabled:        profile.User.IsDisabled,
		EnableSocialLogin: profile.User.EnableSocialLogin,
		SignupSource:      ptrToString(profile.User.SignupSource),
		CreatedAt:         profile.User.CreatedAt,
	}, nil
}

// GetUserByEmail retrieves a user by email and vendor ID
func (s *DefaultUserService) GetUserByEmail(ctx context.Context, email, vendorID string) (*user.User, error) {
	return s.repo.GetUserByEmail(ctx, email, vendorID)
}

// CreateUser creates a new user
func (s *DefaultUserService) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return s.repo.CreateUser(ctx, req)
}

// UpdateUser updates a user
func (s *DefaultUserService) UpdateUser(ctx context.Context, userID string, req *user.UpdateUserRequest) (*user.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return s.repo.UpdateUser(ctx, userID, req)
}

// UpdateUserPassword updates a user's password
func (s *DefaultUserService) UpdateUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	currentUser, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	matches, err := passwordUtils.Matches(currentPassword, currentUser.PasswordHash)
	if err != nil {
		s.logger.Error("Failed to verify current password", zap.Error(err))
		return fmt.Errorf("failed to verify password: %w", err)
	}
	if !matches {
		return fmt.Errorf("current password is incorrect")
	}

	newPasswordHash, err := passwordUtils.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	return s.repo.UpdateUserPassword(ctx, userID, newPasswordHash)
}

// DeactivateUser deactivates a user account
func (s *DefaultUserService) DeactivateUser(ctx context.Context, userID string) error {
	return s.repo.DeactivateUser(ctx, userID)
}

// ListUsers lists users with pagination
func (s *DefaultUserService) ListUsers(ctx context.Context, vendorID string, page, pageSize int) (*user.ListUsersResponse, error) {
	offset := int32((page - 1) * pageSize)
	limit := int32(pageSize)

	users, err := s.repo.ListUsers(ctx, vendorID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountUsers(ctx, vendorID)
	if err != nil {
		return nil, err
	}

	return &user.ListUsersResponse{
		Users:      users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
}
