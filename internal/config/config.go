package config

import (
    "fmt"
    "time"

    "github.com/spf13/viper"

    "distributed-cache/internal/cache"
)

// Config estructura de configuración principal
type Config struct {
    Server ServerConfig `mapstructure:"server"`
    Cache  cache.CacheConfig `mapstructure:"cache"`
    Logger LoggerConfig `mapstructure:"logger"`
}

// ServerConfig configuración del servidor HTTP
type ServerConfig struct {
    Host         string        `mapstructure:"host"`
    Port         int           `mapstructure:"port"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
    IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// LoggerConfig configuración del logger
type LoggerConfig struct {
    Level      string `mapstructure:"level"`
    Format     string `mapstructure:"format"`
    OutputPath string `mapstructure:"output_path"`
}

// LoadConfig carga la configuración desde archivos de configuración y variables de entorno
func LoadConfig() (*Config, error) {
    // Configurar Viper
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")
    viper.AddConfigPath("/etc/distributed-cache")

    // Variables de entorno
    viper.AutomaticEnv()
    viper.SetEnvPrefix("DC") // Distributed Cache

    // Configuración por defecto
    setDefaults()

    // Leer archivo de configuración
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("error reading config file: %w", err)
        }
        // Si no se encuentra el archivo, usar valores por defecto
    }

    // Unmarshall a struct
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("error unmarshalling config: %w", err)
    }

    return &config, nil
}

// setDefaults establece los valores por defecto
func setDefaults() {
    // Server defaults
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("server.read_timeout", "30s")
    viper.SetDefault("server.write_timeout", "30s")
    viper.SetDefault("server.idle_timeout", "120s")

    // Cache defaults
    viper.SetDefault("cache.addresses", []string{"localhost:6379"})
    viper.SetDefault("cache.password", "")
    viper.SetDefault("cache.database", 0)
    viper.SetDefault("cache.max_retries", 3)
    viper.SetDefault("cache.pool_size", 10)
    viper.SetDefault("cache.min_idle_conns", 5)
    viper.SetDefault("cache.dial_timeout", "5s")
    viper.SetDefault("cache.read_timeout", "3s")
    viper.SetDefault("cache.write_timeout", "3s")
    viper.SetDefault("cache.pool_timeout", "4s")

    // Logger defaults
    viper.SetDefault("logger.level", "info")
    viper.SetDefault("logger.format", "json")
    viper.SetDefault("logger.output_path", "stdout")
}

// GetAddress devuelve la dirección completa del servidor
func (sc *ServerConfig) GetAddress() string {
    return fmt.Sprintf("%s:%d", sc.Host, sc.Port)
}
