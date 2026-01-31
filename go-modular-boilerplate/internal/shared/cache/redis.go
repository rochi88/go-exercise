package cache

import (
	"context"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"go-boilerplate/internal/app/config"
	"go-boilerplate/internal/shared/logger"
)

// Redis wraps redis.Client or redis.ClusterClient to provide custom functionality
type Redis struct {
	Client        redis.Cmdable
	logger        *logger.Logger
	isCluster     bool
	singleClient  *redis.Client
	clusterClient *redis.ClusterClient
}

// Config holds the Redis configuration options
type Config struct {
	URL             string
	PoolSize        int
	MinIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// Cluster configuration
	ClusterMode     bool
	ClusterAddrs    []string
	ClusterPassword string
	ClusterDB       int
}

// DefaultConfig returns the default Redis configuration
func DefaultConfig(cfg *config.Config) *Config {
	config := &Config{
		URL:             cfg.RedisURL,
		PoolSize:        10,
		MinIdleConns:    5,
		ConnMaxLifetime: 15 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		ClusterMode:     cfg.RedisClusterMode,
		ClusterAddrs:    cfg.RedisClusterAddrs,
		ClusterPassword: cfg.RedisClusterPassword,
		ClusterDB:       cfg.RedisClusterDB,
	}

	// If cluster mode is enabled but no addresses provided, fall back to single node
	if config.ClusterMode && len(config.ClusterAddrs) == 0 {
		config.ClusterMode = false
	}

	return config
}

// New creates a new Redis connection (single node or cluster)
func New(cfg *Config, log *logger.Logger) (*Redis, error) {
	log = log.Named("redis")

	var client redis.Cmdable
	var singleClient *redis.Client
	var clusterClient *redis.ClusterClient
	isCluster := cfg.ClusterMode

	if cfg.ClusterMode {
		// Initialize cluster client
		clusterOptions := &redis.ClusterOptions{
			Addrs:    cfg.ClusterAddrs,
			Password: cfg.ClusterPassword,
			PoolSize: cfg.PoolSize,
		}

		clusterClient = redis.NewClusterClient(clusterOptions)
		client = clusterClient

		log.Info("Initializing Redis cluster client", zap.Strings("addresses", cfg.ClusterAddrs))
	} else {
		// Initialize single node client
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
		}

		// Configure connection pool
		opt.PoolSize = cfg.PoolSize
		opt.MinIdleConns = cfg.MinIdleConns

		singleClient = redis.NewClient(opt)
		client = singleClient

		// log.Info("Initializing Redis single node client", zap.String("url", cfg.URL))
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	log.Info("Successfully connected to Redis", zap.Bool("cluster_mode", isCluster))

	return &Redis{
		Client:        client,
		logger:        log,
		isCluster:     isCluster,
		singleClient:  singleClient,
		clusterClient: clusterClient,
	}, nil
}

// IsCluster returns true if this Redis client is connected to a cluster
func (r *Redis) IsCluster() bool {
	return r.isCluster
}

// GetClusterNodes returns cluster node information (only works with cluster client)
func (r *Redis) GetClusterNodes(ctx context.Context) (string, error) {
	if !r.isCluster || r.clusterClient == nil {
		return "", fmt.Errorf("not a cluster client")
	}
	return r.clusterClient.ClusterNodes(ctx).Result()
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	r.logger.Info("Closing Redis connection")

	if r.isCluster && r.clusterClient != nil {
		return r.clusterClient.Close()
	} else if r.singleClient != nil {
		return r.singleClient.Close()
	}

	return nil
}
