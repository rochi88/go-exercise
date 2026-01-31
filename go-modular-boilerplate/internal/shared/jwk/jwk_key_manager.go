package jwkKeyManager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// Constants for key management
const (
	MaxValidJWKKeys  = 5
	KeyFileExtension = ".pem"
	KeysDir          = "jwk-keys"
	RSAKeySize       = 2048
)

// JWKKey represents a JSON Web Key with metadata
type JWKKey struct {
	KeyID      string          `json:"kid"`
	PrivateKey *rsa.PrivateKey `json:"-"`
	PublicKey  *rsa.PublicKey  `json:"-"`
	CreatedAt  time.Time       `json:"created_at"`
	IsActive   bool            `json:"is_active"`
}

// JWKKeyManager manages JWK keys with rotation and file watching
type JWKKeyManager struct {
	keysDir string
	logger  *zap.Logger
	keys    map[string]*JWKKey
	watcher *fsnotify.Watcher
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewJWKKeyManager creates a new JWK key manager with file watching
func NewJWKKeyManager(logger *zap.Logger) *JWKKeyManager {
	keys := loadExistingKeys(logger, KeysDir)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Failed to create file watcher", zap.Error(err))
		return &JWKKeyManager{
			keysDir: KeysDir,
			logger:  logger.Named("jwk-key-manager"),
			keys:    keys,
			stopCh:  make(chan struct{}),
		}
	}

	if err := watcher.Add(KeysDir); err != nil {
		logger.Error("Failed to watch keys directory", zap.String("dir", KeysDir), zap.Error(err))
		watcher.Close()
		return &JWKKeyManager{
			keysDir: KeysDir,
			logger:  logger.Named("jwk-key-manager"),
			keys:    keys,
			stopCh:  make(chan struct{}),
		}
	}

	km := &JWKKeyManager{
		keysDir: KeysDir,
		logger:  logger.Named("jwk-key-manager"),
		keys:    keys,
		watcher: watcher,
		stopCh:  make(chan struct{}),
	}

	go km.watchFiles()

	return km
}

// Close gracefully shuts down the key manager
func (km *JWKKeyManager) Close() error {
	close(km.stopCh)
	if km.watcher != nil {
		return km.watcher.Close()
	}
	return nil
}

// loadExistingKeys loads and parses existing JWK keys from the directory
func loadExistingKeys(logger *zap.Logger, keysDir string) map[string]*JWKKey {
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		logger.Error("Failed to create keys directory", zap.String("path", keysDir), zap.Error(err))
		return make(map[string]*JWKKey)
	}

	files, err := os.ReadDir(keysDir)
	if err != nil {
		logger.Error("Failed to read keys directory", zap.String("path", keysDir), zap.Error(err))
		return make(map[string]*JWKKey)
	}

	// Sort files by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := files[i].Info()
		infoJ, _ := files[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	keys := make(map[string]*JWKKey)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), KeyFileExtension) {
			continue
		}

		keyID := strings.TrimSuffix(file.Name(), KeyFileExtension)
		keyPath := filepath.Join(keysDir, file.Name())

		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			logger.Error("Failed to read key file", zap.String("path", keyPath), zap.Error(err))
			continue
		}

		privateKey, err := parsePrivateKey(keyData)
		if err != nil {
			logger.Error("Failed to parse private key", zap.String("path", keyPath), zap.Error(err))
			continue
		}

		info, err := file.Info()
		createdAt := time.Now()
		if err == nil {
			createdAt = info.ModTime()
		}

		isActive := len(keys) == 0 // First (newest) key is active

		jwkKey := &JWKKey{
			KeyID:      keyID,
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
			CreatedAt:  createdAt,
			IsActive:   isActive,
		}

		keys[keyID] = jwkKey
	}

	if len(keys) == 0 {
		logger.Warn("No JWK keys found; run key generation script to create keys")
	}

	return keys
}

// parsePrivateKey parses a PEM-encoded private key
func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}

	// Try PKCS#1 first
	if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	// Try PKCS#8
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}

	return rsaKey, nil
}

// watchFiles monitors the keys directory for changes and reloads keys
func (km *JWKKeyManager) watchFiles() {
	defer func() {
		if km.watcher != nil {
			km.watcher.Close()
		}
	}()

	for {
		select {
		case event, ok := <-km.watcher.Events:
			if !ok {
				return
			}

			if (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) &&
				strings.HasSuffix(event.Name, KeyFileExtension) {
				km.logger.Info("Key file changed, reloading keys", zap.String("file", event.Name))
				km.mu.Lock()
				km.keys = loadExistingKeys(km.logger, km.keysDir)
				km.mu.Unlock()
			}

		case err, ok := <-km.watcher.Errors:
			if !ok {
				return
			}
			km.logger.Error("File watcher error", zap.Error(err))

		case <-km.stopCh:
			return
		}
	}
}

// GetActiveKey returns the currently active key for signing
func (km *JWKKeyManager) GetActiveKey() (*JWKKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	for _, key := range km.keys {
		if key.IsActive {
			return key, nil
		}
	}

	return nil, fmt.Errorf("no active key found")
}

// GetKeyByID returns a key by its ID
func (km *JWKKeyManager) GetKeyByID(keyID string) (*JWKKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	key, exists := km.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return key, nil
}

// GetValidKeys returns all valid keys
func (km *JWKKeyManager) GetValidKeys() []*JWKKey {
	km.mu.RLock()
	defer km.mu.RUnlock()

	validKeys := make([]*JWKKey, 0, len(km.keys))
	for _, key := range km.keys {
		validKeys = append(validKeys, key)
	}

	return validKeys
}

// RotateKey generates a new key and makes it active
func (km *JWKKeyManager) RotateKey() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, RSAKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	newKeyID := time.Now().Format("20060102150405")
	keyPath := filepath.Join(km.keysDir, newKeyID+KeyFileExtension)

	keyData := x509.MarshalPKCS1PrivateKey(privateKey)
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyData,
	})

	if err := os.WriteFile(keyPath, pemData, 0600); err != nil {
		return fmt.Errorf("failed to save new key: %w", err)
	}

	jwkKey := &JWKKey{
		KeyID:      newKeyID,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		CreatedAt:  time.Now(),
		IsActive:   true,
	}

	km.mu.Lock()
	// Deactivate previous active key
	for _, key := range km.keys {
		key.IsActive = false
	}
	km.keys[newKeyID] = jwkKey
	km.mu.Unlock()

	if err := km.CleanupExpiredKeys(); err != nil {
		km.logger.Error("Failed to cleanup old keys", zap.Error(err))
	}

	km.logger.Info("JWK key rotated", zap.String("key_id", newKeyID))
	return nil
}

// CleanupExpiredKeys removes old keys beyond the maximum allowed
func (km *JWKKeyManager) CleanupExpiredKeys() error {
	files, err := os.ReadDir(km.keysDir)
	if err != nil {
		return fmt.Errorf("failed to read keys directory: %w", err)
	}

	if len(files) <= MaxValidJWKKeys {
		return nil
	}

	// Sort by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := files[i].Info()
		infoJ, _ := files[j].Info()
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	filesToRemove := files[:len(files)-MaxValidJWKKeys]

	km.mu.Lock()
	defer km.mu.Unlock()

	for _, file := range filesToRemove {
		if !strings.HasSuffix(file.Name(), KeyFileExtension) {
			continue
		}

		keyID := strings.TrimSuffix(file.Name(), KeyFileExtension)
		delete(km.keys, keyID)

		keyPath := filepath.Join(km.keysDir, file.Name())
		if err := os.Remove(keyPath); err != nil {
			km.logger.Error("Failed to remove old key file", zap.String("path", keyPath), zap.Error(err))
		} else {
			km.logger.Info("Cleaned up old key", zap.String("key_id", keyID))
		}
	}

	return nil
}

// GetJWKS returns the JSON Web Key Set for all valid keys
func (km *JWKKeyManager) GetJWKS() (map[string]interface{}, error) {
	validKeys := km.GetValidKeys()
	jwks := make([]map[string]interface{}, 0, len(validKeys))

	for _, key := range validKeys {
		jwk := map[string]interface{}{
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"kid": key.KeyID,
			"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
		}
		jwks = append(jwks, jwk)
	}

	return map[string]interface{}{
		"keys": jwks,
	}, nil
}
