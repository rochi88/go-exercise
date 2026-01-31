package utils

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	jwkKeyManager "go-boilerplate/internal/shared/jwk"
)

// JWTUtils provides utilities for JWT token operations
type JWTUtils struct {
	keyManager *jwkKeyManager.JWKKeyManager
	logger     *zap.Logger
	issuer     string
}

type JWTClaims map[string]interface{}

// NewJWTUtils creates a new JWT utilities instance
func NewJWTUtils() *JWTUtils {
	cfg, err := config.LoadConfig("./configs")
	logger := zap.NewNop()
	keyManager := jwkKeyManager.NewJWKKeyManager(logger)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	return &JWTUtils{
		keyManager: keyManager,
		logger:     logger.Named("jwt-utils"),
		issuer:     cfg.JWTIssuer,
	}
}

// EncryptClaims creates a signed JWT token with the provided claims
func (j *JWTUtils) EncryptClaims(claims JWTClaims, expiresInHour int) (string, error) {
	if claims == nil {
		return "", errors.New("claims cannot be nil")
	}

	expiresAt := time.Now().Add(time.Duration(expiresInHour) * time.Hour).Unix()

	// Get the active key for signing
	activeKey, err := j.keyManager.GetActiveKey()
	if err != nil {
		j.logger.Error("Failed to get active key for JWT signing", zap.Error(err))
		return "", fmt.Errorf("failed to get signing key: %w", err)
	}

	// Create the JWT token with custom claims
	now := time.Now()
	jwtClaims := jwt.MapClaims{}

	// Copy custom claims
	for k, v := range claims {
		jwtClaims[k] = v
	}

	// Add standard JWT claims
	jwtClaims["iat"] = now.Unix()
	jwtClaims["exp"] = expiresAt
	jwtClaims["iss"] = j.issuer

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)

	// Set the key ID in the header
	token.Header["kid"] = activeKey.KeyID

	// Sign the token with the private key
	tokenString, err := token.SignedString(activeKey.PrivateKey)
	if err != nil {
		j.logger.Error("Failed to sign JWT token", zap.Error(err))
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	j.logger.Info("JWT token created successfully", zap.String("kid", activeKey.KeyID))

	return tokenString, nil
}

// DecryptClaims validates and parses a JWT token, returning the claims
func (j *JWTUtils) DecryptClaims(tokenString string) (JWTClaims, error) {
	if tokenString == "" {
		return nil, errors.New("token string cannot be empty")
	}

	// Parse the token without verification first to get the key ID
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing key ID in token header")
		}

		// Get the key by ID
		key, err := j.keyManager.GetKeyByID(kid)

		if err != nil {
			j.logger.Error("Failed to get key by ID", zap.String("kid", kid), zap.Error(err))
			return nil, fmt.Errorf("invalid key ID: %w", err)
		}

		return key.PublicKey, nil
	})

	if err != nil {
		j.logger.Error("Failed to parse JWT token", zap.Error(err))
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check if token is valid
	if !token.Valid {
		j.logger.Warn("Invalid JWT token provided")
		return nil, errors.New("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		j.logger.Error("Invalid token claims format")
		return nil, errors.New("invalid token claims")
	}

	// Validate expiration
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("missing expiration claim")
	}
	if time.Now().Unix() > int64(exp) {
		j.logger.Warn("JWT token has expired")
		return nil, errors.New("token has expired")
	}

	// Validate issuer
	iss, ok := claims["iss"].(string)
	if !ok || iss != j.issuer {
		j.logger.Warn("Invalid token issuer", zap.String("expected", j.issuer), zap.String("got", iss))
		return nil, errors.New("invalid token issuer")
	}

	// Convert to map[string]interface{} for flexibility
	result := make(map[string]interface{})
	for k, v := range claims {
		result[k] = v
	}

	j.logger.Info("JWT token decrypted successfully")

	return result, nil
}

// GetPublicKeyForToken extracts the public key that was used to sign a token
func (j *JWTUtils) GetPublicKeyForToken(tokenString string) (*rsa.PublicKey, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing key ID in token header")
		}

		key, err := j.keyManager.GetKeyByID(kid)
		if err != nil {
			return nil, fmt.Errorf("invalid key ID: %w", err)
		}

		return key.PublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("missing key ID in token header")
	}

	key, err := j.keyManager.GetKeyByID(kid)
	if err != nil {
		return nil, err
	}

	return key.PublicKey, nil
}
