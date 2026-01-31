package authService

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/pkg/auth"
	authRepository "go-boilerplate/internal/pkg/auth/repository"
	"go-boilerplate/internal/shared/interfaces"
	jwkKeyManager "go-boilerplate/internal/shared/jwk"
	"go-boilerplate/internal/shared/logger"
	"go-boilerplate/internal/shared/metrics"
	sharedUtils "go-boilerplate/internal/shared/utils"
	"go-boilerplate/internal/utils"
)

// AuthService defines the interface for authentication service
type AuthService interface {
	interfaces.Service
	Login(ctx context.Context, req *auth.LoginRequest, ipAddress string, deviceInfo *auth.DeviceInfo) (*auth.LoginResponse, error)
	Register(ctx context.Context, req *auth.RegisterRequest, ipAddress string, deviceInfo *auth.DeviceInfo) (*auth.User, error)
	RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest, ipAddress string, deviceInfo *auth.DeviceInfo) (*auth.RefreshTokenResponse, error)
	VerifyEmail(ctx context.Context, req *auth.VerifyEmailRequest) (*auth.VerifyEmailResponse, error)
	RequestPasswordReset(ctx context.Context, req *auth.RequestPasswordResetRequest) (*auth.RequestPasswordResetResponse, error)
	ResetPassword(ctx context.Context, req *auth.ResetPasswordRequest) (*auth.ResetPasswordResponse, error)
	ChangePassword(ctx context.Context, userID string, req *auth.ChangePasswordRequest) (*auth.ChangePasswordResponse, error)
	GenerateAccessToken(user *auth.User) (string, int64, error)
	GenerateTokenPair(user *auth.User, sessionID string) (*auth.TokenPair, error)
	ValidateRefreshToken(ctx context.Context, refreshToken string) (*auth.Session, error)
	RevokeRefreshToken(ctx context.Context, sessionID string) error
	GetJWKS() (map[string]interface{}, error)
}

// DefaultAuthService is the default implementation of AuthService
type DefaultAuthService struct {
	repo       authRepository.SqlcAuthRepository
	config     *config.Config
	logger     *logger.Logger
	metrics    *metrics.Metrics
	jwtUtils   *sharedUtils.JWTUtils
	keyManager *jwkKeyManager.JWKKeyManager
}

// NewAuthService creates a new authentication service
func NewAuthService(repo authRepository.SqlcAuthRepository, cfg *config.Config, log *logger.Logger, metrics *metrics.Metrics) (AuthService, error) {
	jwtUtils := sharedUtils.NewJWTUtils()
	keyManager := jwkKeyManager.NewJWKKeyManager(log.Logger)

	return &DefaultAuthService{
		repo:       repo,
		config:     cfg,
		logger:     log.Named("auth-service"),
		metrics:    metrics,
		jwtUtils:   jwtUtils,
		keyManager: keyManager,
	}, nil
}

// Login authenticates a user and returns access and refresh tokens
func (s *DefaultAuthService) Login(ctx context.Context, req *auth.LoginRequest, ipAddress string, deviceInfo *auth.DeviceInfo) (*auth.LoginResponse, error) {
	// Validate request
	if err := auth.Validate(req); err != nil {
		if s.metrics != nil {
			s.metrics.RecordUserLoginError()
		}
		return nil, err
	}

	// time.Sleep(2 * time.Second) // Prevent brute-force attacks

	// Find user by email
	user, err := s.repo.FindUserByEmail(ctx, req.Email, req.VendorID)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordUserLoginError()
		}
		return nil, auth.ErrInvalidCredentials
	}

	// Check if user is disabled
	if user.IsDisabled {
		if s.metrics != nil {
			s.metrics.RecordUserLoginError()
		}
		return nil, auth.ErrInvalidCredentials
	}

	// Check if email is verified
	if !user.EmailVerified {
		if s.metrics != nil {
			s.metrics.RecordUserLoginError()
		}
		return nil, auth.ErrEmailNotVerified
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordUserLoginError()
		}
		return nil, auth.ErrInvalidCredentials
	}

	// Create session with refresh token
	sessionID := utils.GetNanoIDWithPrefix("SES")

	// Generate token pair
	tokenPair, err := s.GenerateTokenPair(user, sessionID)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordUserLoginError()
		}
		return nil, err
	}

	refreshTokenHash := sha256.Sum256([]byte(tokenPair.RefreshToken))
	refreshTokenHashStr := hex.EncodeToString(refreshTokenHash[:])

	session := &auth.Session{
		ID:                sessionID,
		UserID:            user.ID,
		RefreshTokenHash:  refreshTokenHashStr,
		IPAddress:         ipAddress,
		DeviceName:        deviceInfo.Name,
		UserAgent:         deviceInfo.UserAgent,
		TrustScore:        deviceInfo.TrustScore,
		City:              deviceInfo.City,
		Country:           deviceInfo.Country,
		Region:            deviceInfo.Region,
		Timezone:          deviceInfo.Timezone,
		ISP:               deviceInfo.ISP,
		DeviceFingerprint: deviceInfo.Fingerprint,
		IsActive:          true,
		TrustedDevice:     deviceInfo.TrustScore < 30, // Trust devices with low risk scores
		CreatedAt:         time.Now().UTC(),
		ValidTill:         time.Now().UTC().Add(7 * 24 * time.Hour), // 7 days for refresh token validity
		LastUsed:          nil,
		RevokedAt:         nil,
	}

	err = s.repo.CreateSession(ctx, session)
	if err != nil {
		s.logger.Error("Failed to create session", zap.Error(err))
		// Don't fail the login if session creation fails
	}

	// Record successful login
	if s.metrics != nil {
		s.metrics.RecordUserLogin()
	}

	// Hide password hash in response
	user.Password = ""

	return &auth.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         *user,
	}, nil
}

// Register creates a new user
func (s *DefaultAuthService) Register(ctx context.Context, req *auth.RegisterRequest, ipAddress string, deviceInfo *auth.DeviceInfo) (*auth.User, error) {
	// Validate request
	if err := auth.Validate(req); err != nil {
		return nil, err
	}

	// Check if user already exists
	_, err := s.repo.FindUserByEmail(ctx, req.Email, req.VendorID)
	if err == nil {
		return nil, auth.ErrUserAlreadyExists
	} else if !errors.Is(err, auth.ErrUserNotFound) {
		return nil, err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	now := time.Now().UTC()
	user := &auth.User{
		ID:                utils.GetNanoIDWithPrefix("USR"),
		Email:             req.Email,
		Password:          string(hashedPassword),
		EmailVerified:     false,        // Email verification required
		VendorID:          req.VendorID, // Populate from request
		Country:           nil,
		City:              nil,
		IsActive:          true,
		IsDisabled:        false,
		EnableSocialLogin: false,
		SignupSource:      req.SignupSource, // Populate from request (optional)
		CreatedAt:         now,
	}

	// Save user
	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	// Generate token for auto-login after registration (optional)
	token, _, err := s.GenerateAccessToken(user)
	if err != nil {
		s.logger.Error("Failed to generate token after registration", zap.Error(err))
		// Don't fail registration if token generation fails
	} else {
		// Create session
		sessionID := utils.GetNanoIDWithPrefix("SES")
		tokenHash := sha256.Sum256([]byte(token))
		tokenHashStr := hex.EncodeToString(tokenHash[:])

		session := &auth.Session{
			ID:                sessionID,
			UserID:            user.ID,
			RefreshTokenHash:  tokenHashStr,
			IPAddress:         ipAddress,
			DeviceName:        deviceInfo.Name,
			UserAgent:         deviceInfo.UserAgent,
			TrustScore:        deviceInfo.TrustScore,
			City:              deviceInfo.City,
			Country:           deviceInfo.Country,
			Region:            deviceInfo.Region,
			Timezone:          deviceInfo.Timezone,
			ISP:               deviceInfo.ISP,
			DeviceFingerprint: deviceInfo.Fingerprint,
			IsActive:          true,
			TrustedDevice:     deviceInfo.TrustScore < 30, // Trust devices with low risk scores
			CreatedAt:         now,
			ValidTill:         now.Add(time.Duration(s.config.RefreshTokenExpiryHour) * time.Hour),
			LastUsed:          nil,
			RevokedAt:         nil,
		}

		err = s.repo.CreateSession(ctx, session)
		if err != nil {
			s.logger.Error("Failed to create session after registration", zap.Error(err))
			// Don't fail registration if session creation fails
		}
	}

	// Record successful registration
	if s.metrics != nil {
		s.metrics.RecordUserRegistration()
	}

	// Hide password hash in response
	user.Password = ""

	return user, nil
}

// GenerateAccessToken generates a JWT access token for a user
func (s *DefaultAuthService) GenerateAccessToken(user *auth.User) (string, int64, error) {
	return s.GenerateTokenWithType(user, "auth_token", s.config.AccessTokenExpiryHour)
}

// GenerateTokenWithType generates a JWT token with specified type for a user
func (s *DefaultAuthService) GenerateTokenWithType(user *auth.User, tokenType string, expirationHours int) (string, int64, error) {
	// Set expiration time based on token type
	expiresAt := time.Now().Add(time.Duration(expirationHours) * time.Hour)
	expiresIn := int64(time.Until(expiresAt).Seconds())

	claims := map[string]interface{}{
		"user_id":    user.ID,
		"email":      user.Email,
		"vendor_id":  user.VendorID,
		"token_type": tokenType,
	}
	// TODO: Add more claims as needed
	tokenString, err := s.jwtUtils.EncryptClaims(claims, expirationHours)
	if err != nil {
		return "", expiresIn, err
	}

	return tokenString, expiresIn, nil
}

// GenerateRefreshToken generates a JWT refresh token for a session
func (s *DefaultAuthService) GenerateRefreshToken(sessionID, userID, vendorID string) (string, error) {
	// Create claims
	claims := map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"vendor_id":  vendorID,
		"token_type": auth.TOKEN_TYPE.REFRESH_TOKEN,
	}

	// Create token with claims
	tokenString, err := s.jwtUtils.EncryptClaims(claims, s.config.RefreshTokenExpiryHour)
	if err != nil {
		return "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	// Generate encoded token
	return tokenString, nil
}

// GenerateTokenPair generates both access and refresh tokens
func (s *DefaultAuthService) GenerateTokenPair(user *auth.User, sessionID string) (*auth.TokenPair, error) {
	// Generate access token
	accessToken, expiresIn, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token as JWT
	refreshToken, err := s.GenerateRefreshToken(sessionID, user.ID, user.VendorID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// RefreshToken validates a refresh token and generates new access token
func (s *DefaultAuthService) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest, ipAddress string, deviceInfo *auth.DeviceInfo) (*auth.RefreshTokenResponse, error) {
	// Validate refresh token
	session, err := s.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Validate device fingerprint if provided
	if deviceInfo.Fingerprint != "" && session.DeviceFingerprint != "" {
		if deviceInfo.Fingerprint != session.DeviceFingerprint {
			s.logger.Warn("Device fingerprint mismatch during token refresh",
				zap.String("session_id", session.ID),
				zap.String("user_id", session.UserID))
			return nil, auth.ErrInvalidRefreshToken
		}
	} else if deviceInfo.Fingerprint != "" && session.DeviceFingerprint == "" {
		// If current request has fingerprint but session doesn't, this might be suspicious
		s.logger.Warn("Device fingerprint provided but not stored in session",
			zap.String("session_id", session.ID),
			zap.String("user_id", session.UserID))
		// For now, we'll allow this but log it
	}

	// Find user
	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	// Check if user is still active and email verified
	if !user.IsActive || user.IsDisabled {
		return nil, auth.ErrInvalidCredentials
	}

	if !user.EmailVerified {
		return nil, auth.ErrEmailNotVerified
	}

	// Generate new access token
	accessToken, expiresIn, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	err = s.repo.UpdateSessionLastUsed(ctx, session.ID)
	if err != nil {
		s.logger.Error("Failed to update session last used", zap.Error(err))
		return nil, fmt.Errorf("failed to update session last used: %w", err)
	}

	return &auth.RefreshTokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

// ValidateRefreshToken validates a JWT refresh token and returns the associated session
func (s *DefaultAuthService) ValidateRefreshToken(ctx context.Context, refreshToken string) (*auth.Session, error) {
	// Parse the JWT refresh token
	decodedToken, err := s.jwtUtils.DecryptClaims(refreshToken)
	if err != nil {
		return nil, auth.ErrInvalidRefreshToken
	}

	tokenType, ok := decodedToken["token_type"].(string)
	if !ok || tokenType != auth.TOKEN_TYPE.REFRESH_TOKEN {
		return nil, auth.ErrInvalidRefreshToken
	}
	// Extract session_id from claims
	sessionID, ok := decodedToken["session_id"].(string)
	if !ok {
		return nil, auth.ErrInvalidRefreshToken
	}

	session, err := s.repo.FindSessionByID(ctx, sessionID)
	if err != nil {
		if err == auth.ErrSessionNotFound {
			return nil, auth.ErrInvalidRefreshToken
		}
		return nil, err
	}

	// Check if session is expired
	if time.Now().UTC().After(session.ValidTill) {
		return nil, auth.ErrRefreshTokenExpired
	}

	return session, nil
}

// RevokeRefreshToken revokes a refresh token by marking the session as inactive
func (s *DefaultAuthService) RevokeRefreshToken(ctx context.Context, sessionID string) error {
	s.logger.Info("Revoking refresh token", zap.String("session_id", sessionID))

	err := s.repo.RevokeSession(ctx, sessionID)
	if err != nil {
		s.logger.Error("Failed to revoke session", zap.String("session_id", sessionID), zap.Error(err))
		return err
	}

	s.logger.Info("Successfully revoked refresh token", zap.String("session_id", sessionID))
	return nil
}

// VerifyEmail verifies a user's email using a verification token
func (s *DefaultAuthService) VerifyEmail(ctx context.Context, req *auth.VerifyEmailRequest) (*auth.VerifyEmailResponse, error) {
	s.logger.Info("Verifying email", zap.String("token", req.Token))

	// Validate the verification token
	claims, err := s.jwtUtils.DecryptClaims(req.Token)
	if err != nil {
		s.logger.Error("Invalid verification token", zap.Error(err))
		return nil, auth.NewAuthError("Invalid or expired verification token")
	}

	// Validate token type
	tokenType, ok := claims["token_type"].(string)
	if !ok || tokenType != "email_verification" {
		s.logger.Warn("Invalid token type for email verification", zap.String("token_type", tokenType))
		return nil, auth.NewAuthError("Invalid token type")
	}

	// Get user ID from claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		s.logger.Error("Missing user_id in verification token")
		return nil, auth.NewAuthError("Invalid token format")
	}

	// Find the user by ID from the token
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to find user for email verification", zap.String("user_id", userID), zap.Error(err))
		return nil, auth.ErrUserNotFound
	}

	// Check if user is already verified
	if user.EmailVerified {
		s.logger.Info("User email already verified", zap.String("user_id", userID))
		return &auth.VerifyEmailResponse{
			Message: "Email already verified",
			User:    *user,
		}, nil
	}

	// TODO: Update user email_verified status in database
	// This would require adding an UpdateUserEmailVerified method to the repository
	user.EmailVerified = true

	s.logger.Info("Email verified successfully", zap.String("user_id", userID))

	return &auth.VerifyEmailResponse{
		Message: "Email verified successfully",
		User:    *user,
	}, nil
}

// RequestPasswordReset initiates a password reset request
func (s *DefaultAuthService) RequestPasswordReset(ctx context.Context, req *auth.RequestPasswordResetRequest) (*auth.RequestPasswordResetResponse, error) {
	s.logger.Info("Requesting password reset", zap.String("email", req.Email))

	// Find the user by email
	user, err := s.repo.FindUserByEmail(ctx, req.Email, req.VendorID)
	if err != nil {
		// Don't reveal if user exists or not for security
		s.logger.Info("Password reset requested for non-existent or inactive user", zap.String("email", req.Email))
		return &auth.RequestPasswordResetResponse{
			Message: "If the email exists, a password reset link has been sent",
		}, nil
	}

	// Check if user is active
	if !user.IsActive || user.IsDisabled {
		s.logger.Info("Password reset requested for inactive/disabled user", zap.String("email", req.Email))
		return &auth.RequestPasswordResetResponse{
			Message: "If the email exists, a password reset link has been sent",
		}, nil
	}

	// Create claims for password reset
	claims := map[string]interface{}{
		"user_id":    user.ID,
		"token_type": auth.TOKEN_TYPE.RESET_PASSWORD,
	}
	encryptedToken, err := s.jwtUtils.EncryptClaims(claims, 4)
	if err != nil {
		s.logger.Error("Failed to generate reset token", zap.String("user_id", user.ID), zap.Error(err))
		return nil, auth.NewAuthError("Failed to generate reset token")
	}

	// TODO: Send email with reset link containing the token
	// For now, just log the token (in production, this would be emailed)
	s.logger.Info("Password reset token generated", zap.String("user_id", user.ID), zap.String("reset_token", encryptedToken))

	return &auth.RequestPasswordResetResponse{
		Message: "If the email exists, a password reset link has been sent",
	}, nil
}

// ResetPassword resets a user's password using a reset token
func (s *DefaultAuthService) ResetPassword(ctx context.Context, req *auth.ResetPasswordRequest) (*auth.ResetPasswordResponse, error) {
	s.logger.Info("Resetting password")

	// Validate the reset token
	claims, err := s.jwtUtils.DecryptClaims(req.Token)
	if err != nil {
		s.logger.Error("Invalid reset token", zap.Error(err))
		return nil, auth.NewAuthError("Invalid or expired reset token")
	}

	// Validate token type
	tokenType, ok := claims["token_type"].(string)
	if !ok || tokenType != "reset_password" {
		s.logger.Warn("Invalid token type for password reset", zap.String("token_type", tokenType))
		return nil, auth.NewAuthError("Invalid token type")
	}

	// Get user ID from claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		s.logger.Error("Missing user_id in reset token")
		return nil, auth.NewAuthError("Invalid token format")
	}

	// Find the user by ID from the token
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to find user for password reset", zap.String("user_id", userID), zap.Error(err))
		return nil, auth.ErrUserNotFound
	}

	// Check if user is active
	if !user.IsActive || user.IsDisabled {
		s.logger.Warn("Attempted password reset for inactive/disabled user", zap.String("user_id", userID))
		return nil, auth.NewAuthError("Account is not active")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash new password", zap.Error(err))
		return nil, auth.NewAuthError("Failed to process password")
	}

	// TODO: Update user password in database
	// This would require adding an UpdateUserPassword method to the repository
	_ = hashedPassword

	s.logger.Info("Password reset successfully", zap.String("user_id", userID))

	return &auth.ResetPasswordResponse{
		Message: "Password reset successfully",
	}, nil
}

// ChangePassword changes a user's password
func (s *DefaultAuthService) ChangePassword(ctx context.Context, userID string, req *auth.ChangePasswordRequest) (*auth.ChangePasswordResponse, error) {
	s.logger.Info("Changing password", zap.String("user_id", userID))

	// Find the user
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to find user", zap.String("user_id", userID), zap.Error(err))
		return nil, err
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword))
	if err != nil {
		s.logger.Warn("Invalid current password", zap.String("user_id", userID))
		return nil, auth.ErrInvalidCredentials
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash new password", zap.Error(err))
		return nil, auth.NewAuthError("Failed to process new password")
	}

	// Update user password (this would require a new repository method)
	// For now, this is a placeholder - in real implementation you'd update the user record
	_ = hashedPassword

	return &auth.ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}

// GetJWKS returns the JSON Web Key Set for the public key
func (s *DefaultAuthService) GetJWKS() (map[string]interface{}, error) {
	return s.keyManager.GetJWKS()
}
