package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	ServerPort int `mapstructure:"server_port"`

	// Database configuration
	DBURL             string        `mapstructure:"db_url"`
	DBMaxOpenConns    int           `mapstructure:"db_max_open_conns"`
	DBMaxIdleConns    int           `mapstructure:"db_max_idle_conns"`
	DBConnMaxLifetime time.Duration `mapstructure:"db_conn_max_lifetime"`
	DBConnMaxIdleTime time.Duration `mapstructure:"db_conn_max_idle_time"`
	DBRetryAttempts   int           `mapstructure:"db_retry_attempts"`
	DBRetryDelay      time.Duration `mapstructure:"db_retry_delay"`

	// Query monitoring configuration
	DBSlowQueryThreshold time.Duration `mapstructure:"db_slow_query_threshold"`

	// Read database configuration (optional - uses same settings as write DB)
	ReadDBURLs []string `mapstructure:"read_db_urls"`

	// Redis configuration
	RedisURL string `mapstructure:"redis_url"`

	// Redis cluster configuration
	RedisClusterMode     bool     `mapstructure:"redis_cluster_mode"`
	RedisClusterAddrs    []string `mapstructure:"redis_cluster_addrs"`
	RedisClusterPassword string   `mapstructure:"redis_cluster_password"`
	RedisClusterDB       int      `mapstructure:"redis_cluster_db"`

	// JWT configuration
	RefreshTokenExpiryHour int    `mapstructure:"jwt_refresh_token_expiry_hours"`
	AccessTokenExpiryHour  int    `mapstructure:"jwt_access_token_expiry_hours"`
	JWTIssuer              string `mapstructure:"jwt_issuer"`

	// Security headers configuration
	SecurityHeadersEnabled  bool   `mapstructure:"security_headers_enabled"`
	SecurityDevelopmentMode bool   `mapstructure:"security_development_mode"`
	ContentSecurityPolicy   string `mapstructure:"content_security_policy"`

	// Metrics configuration
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	MetricsPath    string `mapstructure:"metrics_path"`

	// Environment (development, production, test)
	Environment string `mapstructure:"environment"`

	JWKRotationCron string `mapstructure:"jwk_rotation_cron"`
}

// LoadConfig reads configuration from file or environment variables
func LoadConfig(path string) (*Config, error) {
	// Set default values
	viper.SetDefault("server_port", 8080)

	viper.SetDefault("jwt_refresh_token_expiry_hours", 24*7)
	viper.SetDefault("jwt_access_token_expiry_hours", 1)
	viper.SetDefault("jwt_issuer", "go-boilerplate")

	viper.SetDefault("jwk_rotation_cron", "0 0 * * *") // Default: daily at midnight
	viper.SetDefault("environment", "development")

	// Database defaults
	viper.SetDefault("db_max_open_conns", 25)
	viper.SetDefault("db_max_idle_conns", 10)
	viper.SetDefault("db_conn_max_lifetime", "15m")
	viper.SetDefault("db_conn_max_idle_time", "5m")
	viper.SetDefault("db_retry_attempts", 3)
	viper.SetDefault("db_retry_delay", "2s")
	viper.SetDefault("db_slow_query_threshold", "1s")

	// Redis defaults
	viper.SetDefault("redis_cluster_mode", false)
	viper.SetDefault("redis_cluster_db", 0)

	// Security defaults
	viper.SetDefault("security_headers_enabled", true)
	viper.SetDefault("security_development_mode", true)

	// Metrics defaults
	viper.SetDefault("metrics_enabled", true)
	viper.SetDefault("metrics_path", "/metrics")

	// Set config file path
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Read the config file
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, continue with environment variables
	}

	// Override with environment variables if they exist
	// Convert format: SERVER_PORT -> server_port
	viper.SetEnvKeyReplacer(strings.NewReplacer("_", "."))
	viper.AutomaticEnv()

	// Unmarshal config into struct
	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate required configuration
	return &config, nil
}
