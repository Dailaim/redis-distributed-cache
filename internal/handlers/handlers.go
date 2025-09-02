package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "distributed-cache/internal/cache"
    "distributed-cache/pkg/models"
)

// CacheHandler handles HTTP cache operations
type CacheHandler struct {
    cache  cache.Cache
    logger *zap.Logger
}

// NewCacheHandler creates a new handler
func NewCacheHandler(cache cache.Cache, logger *zap.Logger) *CacheHandler {
    return &CacheHandler{
        cache:  cache,
        logger: logger,
    }
}

// SetItem handles PUT /cache/:key
func (h *CacheHandler) SetItem(c *gin.Context) {
    key := c.Param("key")
    if key == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
        return
    }

    var request struct {
        Value interface{} `json:"value"`
    TTL   string      `json:"ttl,omitempty"` // Duration in format "1h", "30m", "60s"
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        h.logger.Warn("invalid request body", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    // Parse TTL (default 1 hour)
    ttl := 1 * time.Hour
    if request.TTL != "" {
        parsedTTL, err := time.ParseDuration(request.TTL)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid TTL format"})
            return
        }
        ttl = parsedTTL
    }

    err := h.cache.Set(c.Request.Context(), key, request.Value, ttl)
    if err != nil {
        h.logger.Error("failed to set cache item", zap.Error(err), zap.String("key", key))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set cache item"})
        return
    }

    h.logger.Debug("cache item set via API", zap.String("key", key), zap.Duration("ttl", ttl))
    c.JSON(http.StatusOK, gin.H{"message": "item stored successfully"})
}

// GetItem maneja GET /cache/:key
func (h *CacheHandler) GetItem(c *gin.Context) {
    key := c.Param("key")
    if key == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
        return
    }

    item, err := h.cache.Get(c.Request.Context(), key)
    if err != nil {
        h.logger.Error("failed to get cache item", zap.Error(err), zap.String("key", key))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cache item"})
        return
    }

    if item == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
        return
    }

    response := gin.H{
        "key":         item.Key,
        "value":       item.Value,
        "created_at":  item.CreatedAt,
        "expires_at":  item.ExpiresAt,
        "remaining_ttl": item.RemainingTTL().String(),
    }

    c.JSON(http.StatusOK, response)
}

// DeleteItem maneja DELETE /cache/:key
func (h *CacheHandler) DeleteItem(c *gin.Context) {
    key := c.Param("key")
    if key == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
        return
    }

    err := h.cache.Delete(c.Request.Context(), key)
    if err != nil {
        h.logger.Error("failed to delete cache item", zap.Error(err), zap.String("key", key))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete cache item"})
        return
    }

    h.logger.Debug("cache item deleted via API", zap.String("key", key))
    c.JSON(http.StatusOK, gin.H{"message": "item deleted successfully"})
}

// ExistsItem maneja HEAD /cache/:key
func (h *CacheHandler) ExistsItem(c *gin.Context) {
    key := c.Param("key")
    if key == "" {
        c.Status(http.StatusBadRequest)
        return
    }

    exists, err := h.cache.Exists(c.Request.Context(), key)
    if err != nil {
        h.logger.Error("failed to check cache item existence", zap.Error(err), zap.String("key", key))
        c.Status(http.StatusInternalServerError)
        return
    }

    if exists {
        c.Status(http.StatusOK)
    } else {
        c.Status(http.StatusNotFound)
    }
}

// SetMultiple maneja POST /cache/batch
func (h *CacheHandler) SetMultiple(c *gin.Context) {
    var request struct {
        Items map[string]struct {
            Value interface{} `json:"value"`
            TTL   string      `json:"ttl,omitempty"`
        } `json:"items"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        h.logger.Warn("invalid request body", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    items := make(map[string]*models.CacheItem)
    for key, item := range request.Items {
        ttl := 1 * time.Hour
        if item.TTL != "" {
            parsedTTL, err := time.ParseDuration(item.TTL)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": "invalid TTL format for key: " + key,
                })
                return
            }
            ttl = parsedTTL
        }
        items[key] = models.NewCacheItem(key, item.Value, ttl)
    }

    err := h.cache.SetMultiple(c.Request.Context(), items)
    if err != nil {
        h.logger.Error("failed to set multiple cache items", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set multiple items"})
        return
    }

    h.logger.Debug("multiple cache items set via API", zap.Int("count", len(items)))
    c.JSON(http.StatusOK, gin.H{
        "message": "items stored successfully",
        "count":   len(items),
    })
}

// GetMultiple maneja POST /cache/batch/get
func (h *CacheHandler) GetMultiple(c *gin.Context) {
    var request struct {
        Keys []string `json:"keys"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        h.logger.Warn("invalid request body", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    items, err := h.cache.GetMultiple(c.Request.Context(), request.Keys)
    if err != nil {
        h.logger.Error("failed to get multiple cache items", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get multiple items"})
        return
    }

    response := make(map[string]gin.H)
    for key, item := range items {
        response[key] = gin.H{
            "value":        item.Value,
            "created_at":   item.CreatedAt,
            "expires_at":   item.ExpiresAt,
            "remaining_ttl": item.RemainingTTL().String(),
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "items": response,
        "count": len(response),
    })
}

// DeleteMultiple maneja DELETE /cache/batch
func (h *CacheHandler) DeleteMultiple(c *gin.Context) {
    var request struct {
        Keys []string `json:"keys"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        h.logger.Warn("invalid request body", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    err := h.cache.DeleteMultiple(c.Request.Context(), request.Keys)
    if err != nil {
        h.logger.Error("failed to delete multiple cache items", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete multiple items"})
        return
    }

    h.logger.Debug("multiple cache items deleted via API", zap.Int("count", len(request.Keys)))
    c.JSON(http.StatusOK, gin.H{
        "message": "items deleted successfully",
        "count":   len(request.Keys),
    })
}

// Clear maneja DELETE /cache
func (h *CacheHandler) Clear(c *gin.Context) {
    err := h.cache.Clear(c.Request.Context())
    if err != nil {
        h.logger.Error("failed to clear cache", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear cache"})
        return
    }

    h.logger.Info("cache cleared via API")
    c.JSON(http.StatusOK, gin.H{"message": "cache cleared successfully"})
}

// SetExpiration maneja PUT /cache/:key/expire
func (h *CacheHandler) SetExpiration(c *gin.Context) {
    key := c.Param("key")
    if key == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
        return
    }

    var request struct {
        TTL string `json:"ttl"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        h.logger.Warn("invalid request body", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    ttl, err := time.ParseDuration(request.TTL)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid TTL format"})
        return
    }

    err = h.cache.Expire(c.Request.Context(), key, ttl)
    if err != nil {
        h.logger.Error("failed to set expiration", zap.Error(err), zap.String("key", key))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set expiration"})
        return
    }

    h.logger.Debug("expiration set via API", zap.String("key", key), zap.Duration("ttl", ttl))
    c.JSON(http.StatusOK, gin.H{"message": "expiration set successfully"})
}

// GetTTL maneja GET /cache/:key/ttl
func (h *CacheHandler) GetTTL(c *gin.Context) {
    key := c.Param("key")
    if key == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
        return
    }

    ttl, err := h.cache.TTL(c.Request.Context(), key)
    if err != nil {
        h.logger.Error("failed to get TTL", zap.Error(err), zap.String("key", key))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get TTL"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "key": key,
        "ttl": ttl.String(),
    })
}

// GetKeys maneja GET /cache/keys
func (h *CacheHandler) GetKeys(c *gin.Context) {
    pattern := c.DefaultQuery("pattern", "*")

    keys, err := h.cache.Keys(c.Request.Context(), pattern)
    if err != nil {
        h.logger.Error("failed to get keys", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get keys"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "keys":    keys,
        "count":   len(keys),
        "pattern": pattern,
    })
}

// GetStats maneja GET /cache/stats
func (h *CacheHandler) GetStats(c *gin.Context) {
    size, err := h.cache.Size(c.Request.Context())
    if err != nil {
        h.logger.Error("failed to get cache size", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cache stats"})
        return
    }

    info, err := h.cache.Info(c.Request.Context())
    if err != nil {
        h.logger.Warn("failed to get cache info", zap.Error(err))
        info = make(map[string]interface{})
    }

    stats := gin.H{
        "size": size,
        "info": info,
    }

    c.JSON(http.StatusOK, stats)
}

// Health maneja GET /health
func (h *CacheHandler) Health(c *gin.Context) {
    err := h.cache.Ping(c.Request.Context())
    if err != nil {
        h.logger.Error("health check failed", zap.Error(err))
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "status": "unhealthy",
            "error":  err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "healthy",
        "timestamp": time.Now(),
    })
}
