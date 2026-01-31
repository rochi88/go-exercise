package userRepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"go-boilerplate/internal/database"
	"go-boilerplate/internal/database/sqlc"
	"go-boilerplate/internal/pkg/auth"
	"go-boilerplate/internal/pkg/user"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/utils"
)

// SqlcUserRepository defines the interface for user operations using sqlc
type SqlcUserRepository interface {
	GetUserByID(ctx context.Context, id string) (*user.Profile, error)
	GetUserByEmail(ctx context.Context, email, vendorID string) (*user.User, error)
	CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.User, error)
	UpdateUser(ctx context.Context, id string, req *user.UpdateUserRequest) (*user.User, error)
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error
	DeactivateUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, vendorID string, limit, offset int32) ([]*user.User, error)
	CountUsers(ctx context.Context, vendorID string) (int64, error)
}

// sqlcUserRepository is a sqlc-based implementation of SqlcUserRepository
type sqlcUserRepository struct {
	db     *database.PgxReadWriteDB
	logger *logger.Logger
}

// NewSqlcUserRepository creates a new sqlc-based user repository
func NewSqlcUserRepository(db *database.PgxReadWriteDB, log *logger.Logger) SqlcUserRepository {
	return &sqlcUserRepository{
		db:     db,
		logger: log.Named("sqlc-user-repo"),
	}
}

// GetUserByID retrieves a user profile by ID
func (r *sqlcUserRepository) GetUserByID(ctx context.Context, id string) (*user.Profile, error) {
	queries := r.db.Queries()

	sqlcUser, err := queries.GetUserByID(ctx, r.db.ReadPool(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		r.logger.Error("failed to get user by ID",
			zap.Error(err),
			zap.String("user_id", id))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	profile := r.mapSqlcUserToProfile(sqlcUser)
	return profile, nil
}

// GetUserByEmail retrieves a user by email and vendor ID (for authentication)
func (r *sqlcUserRepository) GetUserByEmail(ctx context.Context, email, vendorID string) (*user.User, error) {
	queries := r.db.Queries()

	params := &sqlc.GetUserByEmailParams{
		Email:    email,
		VendorID: vendorID,
	}

	sqlcUser, err := queries.GetUserByEmail(ctx, r.db.ReadPool(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		r.logger.Error("failed to get user by email",
			zap.Error(err),
			zap.String("email", email),
			zap.String("vendor_id", vendorID))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userModel := r.mapSqlcUserToUser(sqlcUser)
	return userModel, nil
}

// CreateUser creates a new user
func (r *sqlcUserRepository) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	queries := r.db.Queries()

	params := &sqlc.CreateUserParams{
		ID:           utils.GenerateUUID(),
		Name:         stringToPtr(req.Name),
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		VendorID:     req.VendorID,
		Country:      stringToPtr(req.Country),
		City:         stringToPtr(req.City),
		SignupSource: stringToPtr(req.SignupSource),
	}

	sqlcUser, err := queries.CreateUser(ctx, r.db.WritePool(), params)
	if err != nil {
		r.logger.Error("failed to create user",
			zap.Error(err),
			zap.String("email", req.Email),
			zap.String("vendor_id", req.VendorID))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Info("user created successfully",
		zap.String("user_id", sqlcUser.ID),
		zap.String("email", sqlcUser.Email))

	userModel := r.mapCreateUserToUser(sqlcUser)
	return userModel, nil
}

// UpdateUser updates user information
func (r *sqlcUserRepository) UpdateUser(ctx context.Context, id string, req *user.UpdateUserRequest) (*user.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	queries := r.db.Queries()

	// For update, we need to handle the fields properly
	params := &sqlc.UpdateUserParams{
		ID:                id,
		Name:              stringToPtr(req.Name),
		Email:             req.Email, // Email is required field in UpdateUserParams
		Country:           stringToPtr(req.Country),
		City:              stringToPtr(req.City),
		EmailVerified:     &req.EmailVerified,
		EnableSocialLogin: &req.EnableSocialLogin,
	}

	sqlcUser, err := queries.UpdateUser(ctx, r.db.WritePool(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		r.logger.Error("failed to update user",
			zap.Error(err),
			zap.String("user_id", id))
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	r.logger.Info("user updated successfully",
		zap.String("user_id", id))

	userModel := r.mapUpdateUserToUser(sqlcUser)
	return userModel, nil
}

// UpdateUserPassword updates user password
func (r *sqlcUserRepository) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	queries := r.db.Queries()

	params := &sqlc.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	}

	err := queries.UpdateUserPassword(ctx, r.db.WritePool(), params)
	if err != nil {
		r.logger.Error("failed to update user password",
			zap.Error(err),
			zap.String("user_id", id))
		return fmt.Errorf("failed to update password: %w", err)
	}

	r.logger.Info("user password updated successfully",
		zap.String("user_id", id))

	return nil
}

// DeactivateUser deactivates a user
func (r *sqlcUserRepository) DeactivateUser(ctx context.Context, id string) error {
	queries := r.db.Queries()

	err := queries.DeactivateUser(ctx, r.db.WritePool(), id)
	if err != nil {
		r.logger.Error("failed to deactivate user",
			zap.Error(err),
			zap.String("user_id", id))
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	r.logger.Info("user deactivated successfully",
		zap.String("user_id", id))

	return nil
}

// ListUsers lists users with pagination
func (r *sqlcUserRepository) ListUsers(ctx context.Context, vendorID string, limit, offset int32) ([]*user.User, error) {
	queries := r.db.Queries()

	params := &sqlc.ListUsersParams{
		VendorID: vendorID,
		Limit:    limit,
		Offset:   offset,
	}

	sqlcUsers, err := queries.ListUsers(ctx, r.db.ReadPool(), params)
	if err != nil {
		r.logger.Error("failed to list users",
			zap.Error(err),
			zap.String("vendor_id", vendorID))
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*user.User, len(sqlcUsers))
	for i, sqlcUser := range sqlcUsers {
		users[i] = r.mapListUserToUser(sqlcUser)
	}

	return users, nil
}

// CountUsers counts total users for a vendor
func (r *sqlcUserRepository) CountUsers(ctx context.Context, vendorID string) (int64, error) {
	queries := r.db.Queries()

	count, err := queries.CountUsers(ctx, r.db.ReadPool(), vendorID)
	if err != nil {
		r.logger.Error("failed to count users",
			zap.Error(err),
			zap.String("vendor_id", vendorID))
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// Helper functions to map sqlc models to domain models
func (r *sqlcUserRepository) mapSqlcUserToProfile(sqlcUser *sqlc.GetUserByIDRow) *user.Profile {
	return &user.Profile{
		User: auth.User{
			ID:                sqlcUser.ID,
			Name:              ptrToString(sqlcUser.Name),
			Email:             sqlcUser.Email,
			EmailVerified:     ptrToBool(sqlcUser.EmailVerified),
			VendorID:          sqlcUser.VendorID,
			Country:           sqlcUser.Country,
			City:              sqlcUser.City,
			IsActive:          ptrToBool(sqlcUser.IsActive),
			IsDisabled:        ptrToBool(sqlcUser.IsDisabled),
			EnableSocialLogin: ptrToBool(sqlcUser.EnableSocialLogin),
			SignupSource:      sqlcUser.SignupSource,
			CreatedAt:         sqlcUser.CreatedAt.Time,
		},
	}
}

func (r *sqlcUserRepository) mapSqlcUserToUser(sqlcUser *sqlc.User) *user.User {
	return &user.User{
		ID:                sqlcUser.ID,
		Name:              ptrToString(sqlcUser.Name),
		Email:             sqlcUser.Email,
		PasswordHash:      sqlcUser.PasswordHash,
		EmailVerified:     ptrToBool(sqlcUser.EmailVerified),
		VendorID:          sqlcUser.VendorID,
		Country:           ptrToString(sqlcUser.Country),
		City:              ptrToString(sqlcUser.City),
		IsActive:          ptrToBool(sqlcUser.IsActive),
		IsDisabled:        ptrToBool(sqlcUser.IsDisabled),
		EnableSocialLogin: ptrToBool(sqlcUser.EnableSocialLogin),
		SignupSource:      ptrToString(sqlcUser.SignupSource),
		CreatedAt:         sqlcUser.CreatedAt.Time,
	}
}

func (r *sqlcUserRepository) mapCreateUserToUser(sqlcUser *sqlc.CreateUserRow) *user.User {
	return &user.User{
		ID:                sqlcUser.ID,
		Name:              ptrToString(sqlcUser.Name),
		Email:             sqlcUser.Email,
		EmailVerified:     ptrToBool(sqlcUser.EmailVerified),
		VendorID:          sqlcUser.VendorID,
		Country:           ptrToString(sqlcUser.Country),
		City:              ptrToString(sqlcUser.City),
		IsActive:          ptrToBool(sqlcUser.IsActive),
		IsDisabled:        ptrToBool(sqlcUser.IsDisabled),
		EnableSocialLogin: ptrToBool(sqlcUser.EnableSocialLogin),
		SignupSource:      ptrToString(sqlcUser.SignupSource),
		CreatedAt:         sqlcUser.CreatedAt.Time,
	}
}

func (r *sqlcUserRepository) mapUpdateUserToUser(sqlcUser *sqlc.UpdateUserRow) *user.User {
	return &user.User{
		ID:                sqlcUser.ID,
		Name:              ptrToString(sqlcUser.Name),
		Email:             sqlcUser.Email,
		EmailVerified:     ptrToBool(sqlcUser.EmailVerified),
		VendorID:          sqlcUser.VendorID,
		Country:           ptrToString(sqlcUser.Country),
		City:              ptrToString(sqlcUser.City),
		IsActive:          ptrToBool(sqlcUser.IsActive),
		IsDisabled:        ptrToBool(sqlcUser.IsDisabled),
		EnableSocialLogin: ptrToBool(sqlcUser.EnableSocialLogin),
		SignupSource:      ptrToString(sqlcUser.SignupSource),
		CreatedAt:         sqlcUser.CreatedAt.Time,
	}
}

func (r *sqlcUserRepository) mapListUserToUser(sqlcUser *sqlc.ListUsersRow) *user.User {
	return &user.User{
		ID:                sqlcUser.ID,
		Name:              ptrToString(sqlcUser.Name),
		Email:             sqlcUser.Email,
		EmailVerified:     ptrToBool(sqlcUser.EmailVerified),
		VendorID:          sqlcUser.VendorID,
		Country:           ptrToString(sqlcUser.Country),
		City:              ptrToString(sqlcUser.City),
		IsActive:          ptrToBool(sqlcUser.IsActive),
		IsDisabled:        ptrToBool(sqlcUser.IsDisabled),
		EnableSocialLogin: ptrToBool(sqlcUser.EnableSocialLogin),
		SignupSource:      ptrToString(sqlcUser.SignupSource),
		CreatedAt:         sqlcUser.CreatedAt.Time,
	}
}

// Helper functions for type conversions
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func ptrToBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}
