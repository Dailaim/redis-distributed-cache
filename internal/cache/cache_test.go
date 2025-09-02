package cache

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap/zaptest"

    "distributed-cache/pkg/models"
)

func setupTestCache(t *testing.T) Cache {
    logger := zaptest.NewLogger(t)
    config := DefaultCacheConfig()

    cache, err := NewRedisCache(config, logger)
    require.NoError(t, err)

    // Clear the cache for tests
    err = cache.Clear(context.Background())
    require.NoError(t, err)

    return cache
}

func TestRedisCache_SetAndGet(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    // Test basic set and get
    key := "test_key"
    value := "test_value"
    ttl := 1 * time.Hour

    err := cache.Set(ctx, key, value, ttl)
    assert.NoError(t, err)

    item, err := cache.Get(ctx, key)
    assert.NoError(t, err)
    assert.NotNil(t, item)
    assert.Equal(t, key, item.Key)
    assert.Equal(t, value, item.Value)
    assert.False(t, item.IsExpired())
}

func TestRedisCache_GetNonExistent(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    item, err := cache.Get(ctx, "non_existent_key")
    assert.NoError(t, err)
    assert.Nil(t, item)
}

func TestRedisCache_SetWithExpiration(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    key := "expiring_key"
    value := "expiring_value"
    ttl := 100 * time.Millisecond

    err := cache.Set(ctx, key, value, ttl)
    assert.NoError(t, err)

    // Verificar que existe inmediatamente
    item, err := cache.Get(ctx, key)
    assert.NoError(t, err)
    assert.NotNil(t, item)

    // Esperar a que expire
    time.Sleep(150 * time.Millisecond)

    // Verificar que ha expirado
    item, err = cache.Get(ctx, key)
    assert.NoError(t, err)
    assert.Nil(t, item)
}

func TestRedisCache_Delete(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    key := "delete_key"
    value := "delete_value"

    // Establecer valor
    err := cache.Set(ctx, key, value, 1*time.Hour)
    assert.NoError(t, err)

    // Verificar que existe
    exists, err := cache.Exists(ctx, key)
    assert.NoError(t, err)
    assert.True(t, exists)

    // Eliminar
    err = cache.Delete(ctx, key)
    assert.NoError(t, err)

    // Verificar que no existe
    exists, err = cache.Exists(ctx, key)
    assert.NoError(t, err)
    assert.False(t, exists)
}

func TestRedisCache_SetMultiple(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    items := map[string]*models.CacheItem{
        "key1": models.NewCacheItem("key1", "value1", 1*time.Hour),
        "key2": models.NewCacheItem("key2", "value2", 1*time.Hour),
        "key3": models.NewCacheItem("key3", "value3", 1*time.Hour),
    }

    err := cache.SetMultiple(ctx, items)
    assert.NoError(t, err)

    // Verificar que todos los elementos se establecieron
    for key := range items {
        item, err := cache.Get(ctx, key)
        assert.NoError(t, err)
        assert.NotNil(t, item)
        assert.Equal(t, key, item.Key)
    }
}

func TestRedisCache_GetMultiple(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    // Establecer algunos valores
    keys := []string{"multi1", "multi2", "multi3"}
    for i, key := range keys {
        err := cache.Set(ctx, key, fmt.Sprintf("value%d", i+1), 1*time.Hour)
        assert.NoError(t, err)
    }

    // Agregar una clave que no existe
    keys = append(keys, "non_existent")

    items, err := cache.GetMultiple(ctx, keys)
    assert.NoError(t, err)
    assert.Len(t, items, 3) // Solo las 3 claves existentes

    for i := 0; i < 3; i++ {
        key := fmt.Sprintf("multi%d", i+1)
        assert.Contains(t, items, key)
        assert.Equal(t, fmt.Sprintf("value%d", i+1), items[key].Value)
    }
}

func TestRedisCache_DeleteMultiple(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    // Establecer algunos valores
    keys := []string{"del1", "del2", "del3"}
    for i, key := range keys {
        err := cache.Set(ctx, key, fmt.Sprintf("value%d", i+1), 1*time.Hour)
        assert.NoError(t, err)
    }

    // Eliminar múltiples
    err := cache.DeleteMultiple(ctx, keys)
    assert.NoError(t, err)

    // Verificar que fueron eliminados
    for _, key := range keys {
        exists, err := cache.Exists(ctx, key)
        assert.NoError(t, err)
        assert.False(t, exists)
    }
}

func TestRedisCache_Expire(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    key := "expire_key"
    value := "expire_value"

    // Establecer con TTL largo
    err := cache.Set(ctx, key, value, 1*time.Hour)
    assert.NoError(t, err)

    // Cambiar TTL a corto
    err = cache.Expire(ctx, key, 100*time.Millisecond)
    assert.NoError(t, err)

    // Verificar TTL
    ttl, err := cache.TTL(ctx, key)
    assert.NoError(t, err)
    assert.True(t, ttl > 0 && ttl <= 100*time.Millisecond)

    // Esperar a que expire
    time.Sleep(150 * time.Millisecond)

    // Verificar que expiró
    exists, err := cache.Exists(ctx, key)
    assert.NoError(t, err)
    assert.False(t, exists)
}

func TestRedisCache_Keys(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    // Establecer algunos valores con patrón
    testKeys := []string{"pattern:key1", "pattern:key2", "other:key"}
    for _, key := range testKeys {
        err := cache.Set(ctx, key, "value", 1*time.Hour)
        assert.NoError(t, err)
    }

    // Buscar claves con patrón
    keys, err := cache.Keys(ctx, "pattern:*")
    assert.NoError(t, err)
    assert.Len(t, keys, 2)

    for _, key := range keys {
        assert.Contains(t, []string{"pattern:key1", "pattern:key2"}, key)
    }
}

func TestRedisCache_Size(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    // Verificar tamaño inicial
    size, err := cache.Size(ctx)
    assert.NoError(t, err)
    assert.Equal(t, int64(0), size)

    // Agregar algunos elementos
    for i := 0; i < 5; i++ {
        err := cache.Set(ctx, fmt.Sprintf("size_key_%d", i), "value", 1*time.Hour)
        assert.NoError(t, err)
    }

    // Verificar nuevo tamaño
    size, err = cache.Size(ctx)
    assert.NoError(t, err)
    assert.Equal(t, int64(5), size)
}

func TestRedisCache_Ping(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    err := cache.Ping(ctx)
    assert.NoError(t, err)
}

func TestRedisCache_Info(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    info, err := cache.Info(ctx)
    assert.NoError(t, err)
    assert.NotEmpty(t, info)

    // Verificar que contiene información típica de Redis
    assert.Contains(t, info, "redis_version")
}

func TestRedisCache_Clear(t *testing.T) {
    cache := setupTestCache(t)
    defer cache.Close()

    ctx := context.Background()

    // Agregar algunos elementos
    for i := 0; i < 3; i++ {
        err := cache.Set(ctx, fmt.Sprintf("clear_key_%d", i), "value", 1*time.Hour)
        assert.NoError(t, err)
    }

    // Verificar que existen
    size, err := cache.Size(ctx)
    assert.NoError(t, err)
    assert.Equal(t, int64(3), size)

    // Limpiar todo
    err = cache.Clear(ctx)
    assert.NoError(t, err)

    // Verificar que se limpió
    size, err = cache.Size(ctx)
    assert.NoError(t, err)
    assert.Equal(t, int64(0), size)
}

func BenchmarkRedisCache_Set(b *testing.B) {
    logger := zaptest.NewLogger(b)
    config := DefaultCacheConfig()

    cache, err := NewRedisCache(config, logger)
    require.NoError(b, err)
    defer cache.Close()

    ctx := context.Background()

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            key := fmt.Sprintf("bench_key_%d", i)
            err := cache.Set(ctx, key, "benchmark_value", 1*time.Hour)
            require.NoError(b, err)
            i++
        }
    })
}

func BenchmarkRedisCache_Get(b *testing.B) {
    logger := zaptest.NewLogger(b)
    config := DefaultCacheConfig()

    cache, err := NewRedisCache(config, logger)
    require.NoError(b, err)
    defer cache.Close()

    ctx := context.Background()

    // Precargar algunos datos
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("bench_get_key_%d", i)
        err := cache.Set(ctx, key, "benchmark_value", 1*time.Hour)
        require.NoError(b, err)
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            key := fmt.Sprintf("bench_get_key_%d", i%1000)
            _, err := cache.Get(ctx, key)
            require.NoError(b, err)
            i++
        }
    })
}
