package reportScheduler

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	jwkKeyManager "go-boilerplate/internal/shared/jwk"
	"go-boilerplate/internal/shared/logger"
)

// JWKKeyRotationJob represents a job that rotates JSON Web Keys (JWKS) periodically
type JWKKeyRotationJob struct {
	logger *logger.Logger
}

// NewJWKKeyRotationJob creates a new JWK key rotation job
func NewJWKKeyRotationJob(logger *logger.Logger) *JWKKeyRotationJob {
	return &JWKKeyRotationJob{
		logger: logger.Named("jwk-key-rotation-job"),
	}
}

// Name returns the name of the job
func (j *JWKKeyRotationJob) Name() string {
	return "jwk-key-rotation"
}

func (j *JWKKeyRotationJob) Schedule() string {
	cfg, err := config.LoadConfig("./configs")
	if err != nil {
		j.logger.Fatal("Failed to load config", zap.Error(err))
	}

	return cfg.JWKRotationCron
}

func (j *JWKKeyRotationJob) Description() string {
	return "Rotates JSON Web Keys (JWKS) periodically"
}

func (j *JWKKeyRotationJob) Timeout() time.Duration {
	return 2 * time.Minute
}

func (j *JWKKeyRotationJob) Run(ctx context.Context) error {
	j.logger.Info("Starting key rotation job")

	if err := j.RotateKey(ctx); err != nil {
		j.logger.Error("Failed to rotate key", zap.Error(err))
		return fmt.Errorf("rotate key: %w", err)
	}

	j.logger.Info("Key rotation completed successfully")
	return nil
}

// RotateKey rotates the JSON Web Key (JWKS) used for signing tokens
func (j *JWKKeyRotationJob) RotateKey(ctx context.Context) error {
	keyManager := jwkKeyManager.NewJWKKeyManager(j.logger.Logger)

	// Rotate key
	if err := keyManager.RotateKey(); err != nil {
		j.logger.Fatal("Failed to rotate key", zap.Error(err))
	}

	if err := keyManager.CleanupExpiredKeys(); err != nil {
		j.logger.Fatal("Failed to cleanup expired keys", zap.Error(err))
	}
	return nil
}
