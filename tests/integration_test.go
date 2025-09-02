package tests

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap/zaptest"

    "distributed-cache/internal/cache"
    "distributed-cache/internal/handlers"
)

func setupTestServer(t *testing.T) (*gin.Engine, cache.Cache) {
    logger := zaptest.NewLogger(t)
    config := cache.DefaultCacheConfig()

    cacheInstance, err := cache.NewRedisCache(config, logger)
    require.NoError(t, err)

    // Clear the cache
    err = cacheInstance.Clear(context.Background())
    require.NoError(t, err)

    gin.SetMode(gin.TestMode)
    router := gin.New()

    cacheHandler := handlers.NewCacheHandler(cacheInstance, logger)

    api := router.Group("/api/v1")
    cache := api.Group("/cache")
    {
        cache.PUT("/:key", cacheHandler.SetItem)
        cache.GET("/:key", cacheHandler.GetItem)
        cache.DELETE("/:key", cacheHandler.DeleteItem)
        cache.HEAD("/:key", cacheHandler.ExistsItem)
        cache.POST("/batch", cacheHandler.SetMultiple)
        cache.POST("/batch/get", cacheHandler.GetMultiple)
        cache.DELETE("/batch", cacheHandler.DeleteMultiple)
        cache.DELETE("/", cacheHandler.Clear)
        cache.GET("/keys", cacheHandler.GetKeys)
        cache.GET("/stats", cacheHandler.GetStats)
        cache.PUT("/:key/expire", cacheHandler.SetExpiration)
        cache.GET("/:key/ttl", cacheHandler.GetTTL)
    }

    router.GET("/health", cacheHandler.Health)

    return router, cacheInstance
}

func TestAPI_SetAndGetItem(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    // Test SET
    setPayload := map[string]interface{}{
        "value": "test_value",
        "ttl":   "1h",
    }

    body, _ := json.Marshal(setPayload)
    req := httptest.NewRequest("PUT", "/api/v1/cache/test_key", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    // Test GET
    req = httptest.NewRequest("GET", "/api/v1/cache/test_key", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, "test_key", response["key"])
    assert.Equal(t, "test_value", response["value"])
}

func TestAPI_GetNonExistentItem(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    req := httptest.NewRequest("GET", "/api/v1/cache/non_existent", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_DeleteItem(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    // Primero establecer un elemento
    setPayload := map[string]interface{}{
        "value": "delete_test",
    }

    body, _ := json.Marshal(setPayload)
    req := httptest.NewRequest("PUT", "/api/v1/cache/delete_key", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Verificar que existe
    req = httptest.NewRequest("HEAD", "/api/v1/cache/delete_key", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Eliminar
    req = httptest.NewRequest("DELETE", "/api/v1/cache/delete_key", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Verificar que no existe
    req = httptest.NewRequest("HEAD", "/api/v1/cache/delete_key", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_BatchOperations(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    // Test batch SET
    batchSetPayload := map[string]interface{}{
        "items": map[string]interface{}{
            "batch1": map[string]interface{}{"value": "value1", "ttl": "1h"},
            "batch2": map[string]interface{}{"value": "value2", "ttl": "30m"},
            "batch3": map[string]interface{}{"value": "value3"},
        },
    }

    body, _ := json.Marshal(batchSetPayload)
    req := httptest.NewRequest("POST", "/api/v1/cache/batch", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Test batch GET
    batchGetPayload := map[string]interface{}{
        "keys": []string{"batch1", "batch2", "batch3", "non_existent"},
    }

    body, _ = json.Marshal(batchGetPayload)
    req = httptest.NewRequest("POST", "/api/v1/cache/batch/get", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)

    items := response["items"].(map[string]interface{})
    assert.Len(t, items, 3) // Solo las 3 claves existentes
    assert.Contains(t, items, "batch1")
    assert.Contains(t, items, "batch2")
    assert.Contains(t, items, "batch3")

    // Test batch DELETE
    batchDeletePayload := map[string]interface{}{
        "keys": []string{"batch1", "batch2"},
    }

    body, _ = json.Marshal(batchDeletePayload)
    req = httptest.NewRequest("DELETE", "/api/v1/cache/batch", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Verificar que fueron eliminadas
    req = httptest.NewRequest("GET", "/api/v1/cache/batch1", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_TTLOperations(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    // Establecer un elemento
    setPayload := map[string]interface{}{
        "value": "ttl_test",
        "ttl":   "1h",
    }

    body, _ := json.Marshal(setPayload)
    req := httptest.NewRequest("PUT", "/api/v1/cache/ttl_key", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Obtener TTL
    req = httptest.NewRequest("GET", "/api/v1/cache/ttl_key/ttl", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    var ttlResponse map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &ttlResponse)
    assert.NoError(t, err)
    assert.Equal(t, "ttl_key", ttlResponse["key"])
    assert.NotEmpty(t, ttlResponse["ttl"])

    // Cambiar TTL
    expirePayload := map[string]interface{}{
        "ttl": "30m",
    }

    body, _ = json.Marshal(expirePayload)
    req = httptest.NewRequest("PUT", "/api/v1/cache/ttl_key/expire", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPI_KeysAndStats(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    // Establecer algunos elementos
    for i := 0; i < 5; i++ {
        setPayload := map[string]interface{}{
            "value": fmt.Sprintf("value%d", i),
        }

        body, _ := json.Marshal(setPayload)
        req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/cache/key%d", i), bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        assert.Equal(t, http.StatusOK, w.Code)
    }

    // Test obtener claves
    req := httptest.NewRequest("GET", "/api/v1/cache/keys?pattern=key*", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    var keysResponse map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &keysResponse)
    assert.NoError(t, err)
    assert.Equal(t, float64(5), keysResponse["count"])

    // Test estadísticas
    req = httptest.NewRequest("GET", "/api/v1/cache/stats", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    var statsResponse map[string]interface{}
    err = json.Unmarshal(w.Body.Bytes(), &statsResponse)
    assert.NoError(t, err)
    assert.NotZero(t, statsResponse["size"])
}

func TestAPI_Clear(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    // Establecer algunos elementos
    for i := 0; i < 3; i++ {
        setPayload := map[string]interface{}{
            "value": fmt.Sprintf("clear_value%d", i),
        }

        body, _ := json.Marshal(setPayload)
        req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/cache/clear_key%d", i), bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        assert.Equal(t, http.StatusOK, w.Code)
    }

    // Limpiar todo
    req := httptest.NewRequest("DELETE", "/api/v1/cache/", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Verificar que se limpió
    req = httptest.NewRequest("GET", "/api/v1/cache/stats", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    var statsResponse map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &statsResponse)
    assert.NoError(t, err)
    assert.Equal(t, float64(0), statsResponse["size"])
}

func TestAPI_Health(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, "healthy", response["status"])
}

func TestAPI_ExpirationBehavior(t *testing.T) {
    router, cacheInstance := setupTestServer(t)
    defer cacheInstance.Close()

    // Establecer elemento con TTL corto
    setPayload := map[string]interface{}{
        "value": "expiring_value",
        "ttl":   "100ms",
    }

    body, _ := json.Marshal(setPayload)
    req := httptest.NewRequest("PUT", "/api/v1/cache/expiring_key", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Verificar que existe inmediatamente
    req = httptest.NewRequest("GET", "/api/v1/cache/expiring_key", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Esperar a que expire
    time.Sleep(150 * time.Millisecond)

    // Verificar que expiró
    req = httptest.NewRequest("GET", "/api/v1/cache/expiring_key", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
}
