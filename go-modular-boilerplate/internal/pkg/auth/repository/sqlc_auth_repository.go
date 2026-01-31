package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"go-boilerplate/internal/database"
	"go-boilerplate/internal/database/sqlc"
	"go-boilerplate/internal/pkg/auth"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/utils"
)

// SqlcAuthRepository defines the interface for auth operations using sqlc
type SqlcAuthRepository interface {
	FindUserByEmail(ctx context.Context, email string, vendorId string) (*auth.User, error)
	FindUserByID(ctx context.Context, userID string) (*auth.User, error)
	CreateUser(ctx context.Context, user *auth.User) error
	CreateSession(ctx context.Context, session *auth.Session) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (*auth.Session, error)
	FindSessionByID(ctx context.Context, sessionID string) (*auth.Session, error)
	UpdateSessionLastUsed(ctx context.Context, sessionID string) error
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeUserSessions(ctx context.Context, userID string) error
}

// sqlcAuthRepository is a sqlc-based implementation of SqlcAuthRepository
type sqlcAuthRepository struct {
	db     *database.PgxReadWriteDB
	logger *logger.Logger
}

// NewSqlcAuthRepository creates a new sqlc-based auth repository
func NewSqlcAuthRepository(db *database.PgxReadWriteDB, log *logger.Logger) SqlcAuthRepository {
	return &sqlcAuthRepository{
		db:     db,
		logger: log.Named("sqlc-auth-repo"),
	}
}

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

func ptrToBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func boolToPtr(b bool) *bool {
	return &b
}

// Convert FindUserByEmailRow to auth.User
func (r *sqlcAuthRepository) convertFindUserByEmailRow(row *sqlc.FindUserByEmailRow) *auth.User {
	return &auth.User{
		ID:                row.ID,
		Email:             row.Email,
		Name:              ptrToString(row.Name),
		Password:          row.PasswordHash,
		EmailVerified:     ptrToBool(row.EmailVerified),
		VendorID:          row.VendorID,
		Country:           row.Country,
		City:              row.City,
		IsActive:          ptrToBool(row.IsActive),
		IsDisabled:        ptrToBool(row.IsDisabled),
		EnableSocialLogin: ptrToBool(row.EnableSocialLogin),
		SignupSource:      row.SignupSource,
		CreatedAt:         row.CreatedAt.Time,
	}
}

// Convert FindUserByIDRow to auth.User
func (r *sqlcAuthRepository) convertFindUserByIDRow(row *sqlc.FindUserByIDRow) *auth.User {
	return &auth.User{
		ID:                row.ID,
		Email:             row.Email,
		Name:              ptrToString(row.Name),
		Password:          row.PasswordHash,
		EmailVerified:     ptrToBool(row.EmailVerified),
		VendorID:          row.VendorID,
		Country:           row.Country,
		City:              row.City,
		IsActive:          ptrToBool(row.IsActive),
		IsDisabled:        ptrToBool(row.IsDisabled),
		EnableSocialLogin: ptrToBool(row.EnableSocialLogin),
		SignupSource:      row.SignupSource,
		CreatedAt:         row.CreatedAt.Time,
	}
}

// Convert AuthSession to auth.Session (simplified mapping due to schema differences)
func (r *sqlcAuthRepository) convertSqlcSession(sqlcSession *sqlc.AuthSession) *auth.Session {
	return &auth.Session{
		ID:               sqlcSession.ID,
		UserID:           sqlcSession.UserID,
		RefreshTokenHash: sqlcSession.RefreshTokenHash,
		IPAddress:        sqlcSession.IpAddress,
		DeviceName:       ptrToString(sqlcSession.DeviceName),
		UserAgent:        ptrToString(sqlcSession.UserAgent),
		IsActive:         ptrToBool(sqlcSession.IsActive),
		CreatedAt:        sqlcSession.CreatedAt.Time,
		ValidTill:        sqlcSession.ExpiresAt.Time,
	}
}

// FindUserByEmail finds a user by email and vendor ID
func (r *sqlcAuthRepository) FindUserByEmail(ctx context.Context, email string, vendorId string) (*auth.User, error) {
	queries := r.db.Queries()

	params := &sqlc.FindUserByEmailParams{
		Email:    email,
		VendorID: vendorId,
	}

	sqlcUser, err := queries.FindUserByEmail(ctx, r.db.ReadPool(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		r.logger.Error("Failed to find user by email", zap.Error(err), zap.String("email", email))
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return r.convertFindUserByEmailRow(sqlcUser), nil
}

// FindUserByID finds a user by ID
func (r *sqlcAuthRepository) FindUserByID(ctx context.Context, userID string) (*auth.User, error) {
	queries := r.db.Queries()

	sqlcUser, err := queries.FindUserByID(ctx, r.db.ReadPool(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		r.logger.Error("Failed to find user by ID", zap.Error(err), zap.String("userID", userID))
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return r.convertFindUserByIDRow(sqlcUser), nil
}

// CreateUser creates a new user
func (r *sqlcAuthRepository) CreateUser(ctx context.Context, user *auth.User) error {
	queries := r.db.Queries()

	// Generate ID if not provided
	if user.ID == "" {
		user.ID = utils.GenerateUUID()
	}

	params := &sqlc.CreateAuthUserParams{
		ID:                user.ID,
		Email:             user.Email,
		Name:              stringToPtr(user.Name),
		PasswordHash:      user.Password,
		EmailVerified:     boolToPtr(user.EmailVerified),
		VendorID:          user.VendorID,
		Country:           user.Country,
		City:              user.City,
		IsActive:          boolToPtr(user.IsActive),
		IsDisabled:        boolToPtr(user.IsDisabled),
		EnableSocialLogin: boolToPtr(user.EnableSocialLogin),
		SignupSource:      user.SignupSource,
	}

	createdUser, err := queries.CreateAuthUser(ctx, r.db.WritePool(), params)
	if err != nil {
		r.logger.Error("Failed to create user", zap.Error(err), zap.String("email", user.Email))
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Update the user with created data
	user.CreatedAt = createdUser.CreatedAt.Time

	r.logger.Info("User created successfully", zap.String("userID", user.ID), zap.String("email", user.Email))
	return nil
}

// CreateSession creates a new authentication session
func (r *sqlcAuthRepository) CreateSession(ctx context.Context, session *auth.Session) error {
	queries := r.db.Queries()

	// Generate ID if not provided
	if session.ID == "" {
		session.ID = utils.GenerateUUID()
	}

	params := &sqlc.CreateAuthSessionParams{
		ID:               session.ID,
		UserID:           session.UserID,
		RefreshTokenHash: session.RefreshTokenHash,
		IpAddress:        session.IPAddress,
		DeviceName:       stringToPtr(session.DeviceName),
		UserAgent:        stringToPtr(session.UserAgent),
		OsInfo:           nil, // Not available in simplified session
		ExpiresAt:        pgtype.Timestamptz{Time: session.ValidTill, Valid: true},
	}

	createdSession, err := queries.CreateAuthSession(ctx, r.db.WritePool(), params)
	if err != nil {
		r.logger.Error("Failed to create session", zap.Error(err), zap.String("userID", session.UserID))
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Update the session with created data
	session.CreatedAt = createdSession.CreatedAt.Time

	r.logger.Info("Session created successfully", zap.String("sessionID", session.ID), zap.String("userID", session.UserID))
	return nil
}

// FindSessionByTokenHash finds a session by refresh token hash
func (r *sqlcAuthRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*auth.Session, error) {
	queries := r.db.Queries()

	sqlcSession, err := queries.GetAuthSessionByToken(ctx, r.db.ReadPool(), tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("session not found with token hash")
		}
		r.logger.Error("Failed to find session by token", zap.Error(err))
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	return r.convertSqlcSession(sqlcSession), nil
}

// FindSessionByID finds a session by ID
func (r *sqlcAuthRepository) FindSessionByID(ctx context.Context, sessionID string) (*auth.Session, error) {
	queries := r.db.Queries()

	sqlcSession, err := queries.GetAuthSessionByID(ctx, r.db.ReadPool(), sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("session not found with ID %s", sessionID)
		}
		r.logger.Error("Failed to find session by ID", zap.Error(err), zap.String("sessionID", sessionID))
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	return r.convertSqlcSession(sqlcSession), nil
}

// UpdateSessionLastUsed updates the last used timestamp of a session
func (r *sqlcAuthRepository) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	queries := r.db.Queries()

	err := queries.UpdateAuthSessionLastUsed(ctx, r.db.WritePool(), sessionID)
	if err != nil {
		r.logger.Error("Failed to update session last used", zap.Error(err), zap.String("sessionID", sessionID))
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// RevokeSession revokes a specific session
func (r *sqlcAuthRepository) RevokeSession(ctx context.Context, sessionID string) error {
	queries := r.db.Queries()

	err := queries.DeactivateAuthSession(ctx, r.db.WritePool(), sessionID)
	if err != nil {
		r.logger.Error("Failed to revoke session", zap.Error(err), zap.String("sessionID", sessionID))
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	r.logger.Info("Session revoked successfully", zap.String("sessionID", sessionID))
	return nil
}

// RevokeUserSessions revokes all sessions for a user
func (r *sqlcAuthRepository) RevokeUserSessions(ctx context.Context, userID string) error {
	queries := r.db.Queries()

	err := queries.DeactivateUserSessions(ctx, r.db.WritePool(), userID)
	if err != nil {
		r.logger.Error("Failed to revoke user sessions", zap.Error(err), zap.String("userID", userID))
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	r.logger.Info("User sessions revoked successfully", zap.String("userID", userID))
	return nil
}
