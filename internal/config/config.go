package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"distributed-cache/internal/cache"
)

// Config estructura de configuración principal
type Config struct {
	Server ServerConfig      `mapstructure:"server"`
	Cache  cache.CacheConfig `mapstructure:"cache"`
	Logger LoggerConfig      `mapstructure:"logger"`
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

	// Configurar mapeos específicos para variables de entorno
	viper.BindEnv("cache.addresses", "DC_CACHE_ADDRESSES")
	viper.BindEnv("cache.password", "DC_CACHE_PASSWORD")
	viper.BindEnv("cache.database", "DC_CACHE_DATABASE")
	viper.BindEnv("cache.max_retries", "DC_CACHE_MAX_RETRIES")
	viper.BindEnv("cache.pool_size", "DC_CACHE_POOL_SIZE")
	viper.BindEnv("cache.min_idle_conns", "DC_CACHE_MIN_IDLE_CONNS")
	viper.BindEnv("cache.dial_timeout", "DC_CACHE_DIAL_TIMEOUT")
	viper.BindEnv("cache.read_timeout", "DC_CACHE_READ_TIMEOUT")
	viper.BindEnv("cache.write_timeout", "DC_CACHE_WRITE_TIMEOUT")
	viper.BindEnv("cache.pool_timeout", "DC_CACHE_POOL_TIMEOUT")

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

	// Procesar la variable de entorno ADDRESSES si es una string
	if addressesStr := viper.GetString("cache.addresses"); addressesStr != "" {
		// Si la variable de entorno es una string, convertirla a slice
		addresses := strings.Split(addressesStr, ",")
		for i, addr := range addresses {
			addresses[i] = strings.TrimSpace(addr)
		}
		config.Cache.Addresses = addresses
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

	// Cache defaults - usar localhost para desarrollo local, redis para contenedores
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
