package main

import (
	jwkKeyManager "go-boilerplate/internal/shared/jwk"
	"go-boilerplate/internal/shared/logger"

	"go.uber.org/zap"
)

func main() {
	// Create a new JWK key
	keyManager := jwkKeyManager.NewJWKKeyManager(logger.New("development").Logger)
	if err := keyManager.RotateKey(); err != nil {
		logger.New("development").Fatal("Failed to create initial key", zap.Error(err))
	}
}
