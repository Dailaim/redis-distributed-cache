package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"

    "distributed-cache/internal/cache"
    "distributed-cache/internal/config"
    "distributed-cache/internal/handlers"
    "distributed-cache/internal/middleware"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        fmt.Printf("Failed to load config: %v\n", err)
        os.Exit(1)
    }

    // Configure logger
    logger, err := setupLogger(&cfg.Logger)
    if err != nil {
        fmt.Printf("Failed to setup logger: %v\n", err)
        os.Exit(1)
    }
    defer logger.Sync()

    logger.Info("Starting Distributed Cache Server",
        zap.String("version", "1.0.0"),
        zap.String("address", cfg.Server.GetAddress()),
    )

    // Initialize cache
    cacheInstance, err := cache.NewRedisCache(&cfg.Cache, logger)
    if err != nil {
        logger.Fatal("Failed to initialize cache", zap.Error(err))
    }
    defer cacheInstance.Close()

    // Check cache connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := cacheInstance.Ping(ctx); err != nil {
        logger.Fatal("Failed to connect to cache", zap.Error(err))
    }
    logger.Info("Cache connection established successfully")

    // Configure Gin
    if cfg.Logger.Level == "debug" {
        gin.SetMode(gin.DebugMode)
    } else {
        gin.SetMode(gin.ReleaseMode)
    }

    // Create router
    router := gin.New()

    // Middlewares
    router.Use(middleware.Recovery(logger))
    router.Use(middleware.Logger(logger))
    router.Use(middleware.CORS())
    router.Use(middleware.RequestID())
    router.Use(middleware.RateLimiter())

    // Initialize handlers
    cacheHandler := handlers.NewCacheHandler(cacheInstance, logger)

    // Health routes
    router.GET("/health", cacheHandler.Health)
    router.GET("/ping", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "pong"})
    })

    // Cache routes
    api := router.Group("/api/v1")
    {
        cache := api.Group("/cache")
        {
            // Individual operations
            cache.PUT("/:key", cacheHandler.SetItem)
            cache.GET("/:key", cacheHandler.GetItem)
            cache.DELETE("/:key", cacheHandler.DeleteItem)
            cache.HEAD("/:key", cacheHandler.ExistsItem)

            // TTL operations
            cache.PUT("/:key/expire", cacheHandler.SetExpiration)
            cache.GET("/:key/ttl", cacheHandler.GetTTL)

            // Batch operations
            cache.POST("/batch", cacheHandler.SetMultiple)
            cache.POST("/batch/get", cacheHandler.GetMultiple)
            cache.DELETE("/batch", cacheHandler.DeleteMultiple)

            // Management operations
            cache.DELETE("/", cacheHandler.Clear)
            cache.GET("/keys", cacheHandler.GetKeys)
            cache.GET("/stats", cacheHandler.GetStats)
        }
    }

    // Configure HTTP server
    server := &http.Server{
        Addr:         cfg.Server.GetAddress(),
        Handler:      router,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
        IdleTimeout:  cfg.Server.IdleTimeout,
    }

    // Start server in goroutine
    go func() {
        logger.Info("Server starting", zap.String("address", server.Addr))
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server", zap.Error(err))
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    logger.Info("Shutting down server...")

    // Graceful shutdown
    ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        logger.Error("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server exited")
}

// setupLogger configures the logger according to the configuration
func setupLogger(cfg *config.LoggerConfig) (*zap.Logger, error) {
    var level zapcore.Level
    switch cfg.Level {
    case "debug":
        level = zapcore.DebugLevel
    case "info":
        level = zapcore.InfoLevel
    case "warn":
        level = zapcore.WarnLevel
    case "error":
        level = zapcore.ErrorLevel
    default:
        level = zapcore.InfoLevel
    }

    config := zap.Config{
        Level:       zap.NewAtomicLevelAt(level),
        Development: false,
        Sampling: &zap.SamplingConfig{
            Initial:    100,
            Thereafter: 100,
        },
        Encoding: cfg.Format,
        EncoderConfig: zapcore.EncoderConfig{
            TimeKey:        "timestamp",
            LevelKey:       "level",
            NameKey:        "logger",
            CallerKey:      "caller",
            FunctionKey:    zapcore.OmitKey,
            MessageKey:     "message",
            StacktraceKey:  "stacktrace",
            LineEnding:     zapcore.DefaultLineEnding,
            EncodeLevel:    zapcore.LowercaseLevelEncoder,
            EncodeTime:     zapcore.ISO8601TimeEncoder,
            EncodeDuration: zapcore.SecondsDurationEncoder,
            EncodeCaller:   zapcore.ShortCallerEncoder,
        },
        OutputPaths:      []string{cfg.OutputPath},
        ErrorOutputPaths: []string{cfg.OutputPath},
    }

    return config.Build()
}
