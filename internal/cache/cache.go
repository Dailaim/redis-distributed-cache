package cache

import (
    "context"
    "time"
    "distributed-cache/pkg/models"
)

// Cache defines the interface for distributed cache operations
type Cache interface {
    // Basic operations
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Get(ctx context.Context, key string) (*models.CacheItem, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)

    // Batch operations
    SetMultiple(ctx context.Context, items map[string]*models.CacheItem) error
    GetMultiple(ctx context.Context, keys []string) (map[string]*models.CacheItem, error)
    DeleteMultiple(ctx context.Context, keys []string) error

    // Cleanup operations
    Clear(ctx context.Context) error
    Expire(ctx context.Context, key string, ttl time.Duration) error
    TTL(ctx context.Context, key string) (time.Duration, error)

    // Pattern operations
    Keys(ctx context.Context, pattern string) ([]string, error)
    FlushExpired(ctx context.Context) error

    // Statistics
    Size(ctx context.Context) (int64, error)
    Info(ctx context.Context) (map[string]interface{}, error)

    // Connection
    Ping(ctx context.Context) error
    Close() error
}

// CacheConfig configuration for the cache
type CacheConfig struct {
    Addresses    []string      `mapstructure:"addresses"`
    Password     string        `mapstructure:"password"`
    Database     int           `mapstructure:"database"`
    MaxRetries   int           `mapstructure:"max_retries"`
    PoolSize     int           `mapstructure:"pool_size"`
    MinIdleConns int           `mapstructure:"min_idle_conns"`
    DialTimeout  time.Duration `mapstructure:"dial_timeout"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
    PoolTimeout  time.Duration `mapstructure:"pool_timeout"`
}

// DefaultCacheConfig returns the default configuration
func DefaultCacheConfig() *CacheConfig {
    return &CacheConfig{
        Addresses:    []string{"localhost:6379"},
        Password:     "",
        Database:     0,
        MaxRetries:   3,
        PoolSize:     10,
        MinIdleConns: 5,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolTimeout:  4 * time.Second,
    }
}
