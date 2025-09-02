package models

import (
    "time"
)

// CacheItem represents an item stored in the cache
type CacheItem struct {
    Key       string      `json:"key"`
    Value     interface{} `json:"value"`
    TTL       time.Duration `json:"ttl"`
    CreatedAt time.Time   `json:"created_at"`
    ExpiresAt time.Time   `json:"expires_at"`
}

// NewCacheItem creates a new cache item
func NewCacheItem(key string, value interface{}, ttl time.Duration) *CacheItem {
    now := time.Now()
    return &CacheItem{
        Key:       key,
        Value:     value,
        TTL:       ttl,
        CreatedAt: now,
        ExpiresAt: now.Add(ttl),
    }
}

// IsExpired checks if the item has expired
func (ci *CacheItem) IsExpired() bool {
    return time.Now().After(ci.ExpiresAt)
}

// RemainingTTL returns the remaining time until expiration
func (ci *CacheItem) RemainingTTL() time.Duration {
    if ci.IsExpired() {
        return 0
    }
    return time.Until(ci.ExpiresAt)
}
