package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"

    "distributed-cache/pkg/models"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
    client redis.UniversalClient
    logger *zap.Logger
    config *CacheConfig
}

// NewRedisCache creates a new instance of RedisCache
func NewRedisCache(config *CacheConfig, logger *zap.Logger) (*RedisCache, error) {
    if config == nil {
        config = DefaultCacheConfig()
    }

    // Configure the Redis client
    options := &redis.UniversalOptions{
        Addrs:        config.Addresses,
        Password:     config.Password,
        DB:           config.Database,
        MaxRetries:   config.MaxRetries,
        PoolSize:     config.PoolSize,
        MinIdleConns: config.MinIdleConns,
        DialTimeout:  config.DialTimeout,
        ReadTimeout:  config.ReadTimeout,
        WriteTimeout: config.WriteTimeout,
        PoolTimeout:  config.PoolTimeout,
    }

    client := redis.NewUniversalClient(options)

    // Check connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    return &RedisCache{
        client: client,
        logger: logger,
        config: config,
    }, nil
}

// Set stores an item in the cache
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    cacheItem := models.NewCacheItem(key, value, ttl)

    data, err := json.Marshal(cacheItem)
    if err != nil {
        rc.logger.Error("failed to marshal cache item", zap.Error(err), zap.String("key", key))
        return fmt.Errorf("failed to marshal cache item: %w", err)
    }

    err = rc.client.Set(ctx, key, data, ttl).Err()
    if err != nil {
        rc.logger.Error("failed to set cache item", zap.Error(err), zap.String("key", key))
        return fmt.Errorf("failed to set cache item: %w", err)
    }

    rc.logger.Debug("cache item set successfully", 
        zap.String("key", key), 
        zap.Duration("ttl", ttl))

    return nil
}

// Get retrieves an item from the cache
func (rc *RedisCache) Get(ctx context.Context, key string) (*models.CacheItem, error) {
    data, err := rc.client.Get(ctx, key).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, nil // Cache miss
        }
        rc.logger.Error("failed to get cache item", zap.Error(err), zap.String("key", key))
        return nil, fmt.Errorf("failed to get cache item: %w", err)
    }

    var cacheItem models.CacheItem
    if err := json.Unmarshal([]byte(data), &cacheItem); err != nil {
        rc.logger.Error("failed to unmarshal cache item", zap.Error(err), zap.String("key", key))
        return nil, fmt.Errorf("failed to unmarshal cache item: %w", err)
    }

    // Check if expired (double check)
    if cacheItem.IsExpired() {
        rc.logger.Debug("cache item expired, removing", zap.String("key", key))
    _ = rc.Delete(ctx, key) // Clean up expired item
        return nil, nil
    }

    rc.logger.Debug("cache item retrieved successfully", zap.String("key", key))
    return &cacheItem, nil
}

// Delete removes an item from the cache
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
    err := rc.client.Del(ctx, key).Err()
    if err != nil {
        rc.logger.Error("failed to delete cache item", zap.Error(err), zap.String("key", key))
        return fmt.Errorf("failed to delete cache item: %w", err)
    }

    rc.logger.Debug("cache item deleted successfully", zap.String("key", key))
    return nil
}

// Exists checks if a key exists in the cache
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
    count, err := rc.client.Exists(ctx, key).Result()
    if err != nil {
        rc.logger.Error("failed to check cache item existence", zap.Error(err), zap.String("key", key))
        return false, fmt.Errorf("failed to check cache item existence: %w", err)
    }

    return count > 0, nil
}

// SetMultiple stores multiple items
func (rc *RedisCache) SetMultiple(ctx context.Context, items map[string]*models.CacheItem) error {
    pipe := rc.client.Pipeline()

    for key, item := range items {
        data, err := json.Marshal(item)
        if err != nil {
            rc.logger.Error("failed to marshal cache item", zap.Error(err), zap.String("key", key))
            continue
        }
        pipe.Set(ctx, key, data, item.TTL)
    }

    _, err := pipe.Exec(ctx)
    if err != nil {
        rc.logger.Error("failed to set multiple cache items", zap.Error(err))
        return fmt.Errorf("failed to set multiple cache items: %w", err)
    }

    rc.logger.Debug("multiple cache items set successfully", zap.Int("count", len(items)))
    return nil
}

// GetMultiple retrieves multiple items
func (rc *RedisCache) GetMultiple(ctx context.Context, keys []string) (map[string]*models.CacheItem, error) {
    if len(keys) == 0 {
        return make(map[string]*models.CacheItem), nil
    }

    results, err := rc.client.MGet(ctx, keys...).Result()
    if err != nil {
        rc.logger.Error("failed to get multiple cache items", zap.Error(err))
        return nil, fmt.Errorf("failed to get multiple cache items: %w", err)
    }

    items := make(map[string]*models.CacheItem)
    for i, result := range results {
        if result == nil {
            continue // Cache miss
        }

        var cacheItem models.CacheItem
        data, ok := result.(string)
        if !ok {
            rc.logger.Warn("unexpected data type in cache", zap.String("key", keys[i]))
            continue
        }

        if err := json.Unmarshal([]byte(data), &cacheItem); err != nil {
            rc.logger.Error("failed to unmarshal cache item", zap.Error(err), zap.String("key", keys[i]))
            continue
        }

        if !cacheItem.IsExpired() {
            items[keys[i]] = &cacheItem
        } else {
            // Clean up expired items asynchronously
            go func(key string) {
                _ = rc.Delete(context.Background(), key)
            }(keys[i])
        }
    }

    rc.logger.Debug("multiple cache items retrieved", 
        zap.Int("requested", len(keys)), 
        zap.Int("found", len(items)))

    return items, nil
}

// DeleteMultiple removes multiple items
func (rc *RedisCache) DeleteMultiple(ctx context.Context, keys []string) error {
    if len(keys) == 0 {
        return nil
    }

    err := rc.client.Del(ctx, keys...).Err()
    if err != nil {
        rc.logger.Error("failed to delete multiple cache items", zap.Error(err))
        return fmt.Errorf("failed to delete multiple cache items: %w", err)
    }

    rc.logger.Debug("multiple cache items deleted successfully", zap.Int("count", len(keys)))
    return nil
}

// Clear wipes the entire cache
func (rc *RedisCache) Clear(ctx context.Context) error {
    err := rc.client.FlushDB(ctx).Err()
    if err != nil {
        rc.logger.Error("failed to clear cache", zap.Error(err))
        return fmt.Errorf("failed to clear cache: %w", err)
    }

    rc.logger.Info("cache cleared successfully")
    return nil
}

// Expire sets a new TTL for a key
func (rc *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
    success, err := rc.client.Expire(ctx, key, ttl).Result()
    if err != nil {
        rc.logger.Error("failed to set expiration", zap.Error(err), zap.String("key", key))
        return fmt.Errorf("failed to set expiration: %w", err)
    }

    if !success {
        return fmt.Errorf("key does not exist: %s", key)
    }

    rc.logger.Debug("expiration set successfully", zap.String("key", key), zap.Duration("ttl", ttl))
    return nil
}

// TTL gets the remaining lifetime of a key
func (rc *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
    ttl, err := rc.client.TTL(ctx, key).Result()
    if err != nil {
        rc.logger.Error("failed to get TTL", zap.Error(err), zap.String("key", key))
        return 0, fmt.Errorf("failed to get TTL: %w", err)
    }

    return ttl, nil
}

// Keys returns keys matching a pattern
func (rc *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
    keys, err := rc.client.Keys(ctx, pattern).Result()
    if err != nil {
        rc.logger.Error("failed to get keys", zap.Error(err), zap.String("pattern", pattern))
        return nil, fmt.Errorf("failed to get keys: %w", err)
    }

    return keys, nil
}

// FlushExpired removes expired items (in Redis this is done automatically)
func (rc *RedisCache) FlushExpired(ctx context.Context) error {
    rc.logger.Debug("flush expired called (Redis handles expiration automatically)")
    return nil
}

// Size devuelve el número de claves en el caché
func (rc *RedisCache) Size(ctx context.Context) (int64, error) {
    size, err := rc.client.DBSize(ctx).Result()
    if err != nil {
        rc.logger.Error("failed to get cache size", zap.Error(err))
        return 0, fmt.Errorf("failed to get cache size: %w", err)
    }

    return size, nil
}

// Info devuelve información del caché
func (rc *RedisCache) Info(ctx context.Context) (map[string]interface{}, error) {
    info, err := rc.client.Info(ctx).Result()
    if err != nil {
        rc.logger.Error("failed to get cache info", zap.Error(err))
        return nil, fmt.Errorf("failed to get cache info: %w", err)
    }

    // Parsear información básica
    result := make(map[string]interface{})
    lines := strings.Split(info, "\n")

    for _, line := range lines {
        if strings.Contains(line, ":") {
            parts := strings.SplitN(line, ":", 2)
            if len(parts) == 2 {
                result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
            }
        }
    }

    return result, nil
}

// Ping verifica la conexión con Redis
func (rc *RedisCache) Ping(ctx context.Context) error {
    err := rc.client.Ping(ctx).Err()
    if err != nil {
        rc.logger.Error("ping failed", zap.Error(err))
        return fmt.Errorf("ping failed: %w", err)
    }

    return nil
}

// Close cierra la conexión con Redis
func (rc *RedisCache) Close() error {
    err := rc.client.Close()
    if err != nil {
        rc.logger.Error("failed to close Redis connection", zap.Error(err))
        return fmt.Errorf("failed to close Redis connection: %w", err)
    }

    rc.logger.Info("Redis connection closed successfully")
    return nil
}
