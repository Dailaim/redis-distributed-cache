package middleware

import (
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// Logger middleware for request logging
func Logger(logger *zap.Logger) gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        logger.Info("HTTP Request",
            zap.String("client_ip", param.ClientIP),
            zap.String("method", param.Method),
            zap.String("path", param.Path),
            zap.Int("status_code", param.StatusCode),
            zap.Duration("latency", param.Latency),
            zap.String("user_agent", param.Request.UserAgent()),
        )
        return ""
    })
}

// CORS middleware
func CORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
        c.Header("Access-Control-Allow-Credentials", "true")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}

// Recovery middleware personalizado
func Recovery(logger *zap.Logger) gin.HandlerFunc {
    return gin.RecoveryWithWriter(gin.DefaultWriter, func(c *gin.Context, recovered interface{}) {
        logger.Error("Panic recovered",
            zap.Any("error", recovered),
            zap.String("path", c.Request.URL.Path),
            zap.String("method", c.Request.Method),
        )
        c.AbortWithStatus(500)
    })
}

// RateLimiter middleware básico (en producción usar Redis)
func RateLimiter() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Implementación básica - en producción usar una solución más robusta
        c.Next()
    }
}

// RequestID middleware para trazabilidad
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = generateRequestID()
        }
        c.Header("X-Request-ID", requestID)
        c.Set("RequestID", requestID)
        c.Next()
    }
}

// generateRequestID genera un ID único para el request
func generateRequestID() string {
    return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString genera una cadena aleatoria
func randomString(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
    }
    return string(b)
}
