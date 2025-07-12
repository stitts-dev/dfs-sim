package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	// Server
	Port string `mapstructure:"PORT"`
	Env  string `mapstructure:"ENV"`

	// Database
	DatabaseURL string `mapstructure:"DATABASE_URL"`

	// Redis
	RedisURL string `mapstructure:"REDIS_URL"`

	// JWT
	JWTSecret string `mapstructure:"JWT_SECRET"`

	// CORS
	CorsOrigins []string `mapstructure:"CORS_ORIGINS"`

	// Optimization
	MaxLineups          int `mapstructure:"MAX_LINEUPS"`
	OptimizationTimeout int `mapstructure:"OPTIMIZATION_TIMEOUT"`

	// Simulation
	MaxSimulations    int `mapstructure:"MAX_SIMULATIONS"`
	SimulationWorkers int `mapstructure:"SIMULATION_WORKERS"`

	// External APIs
	ESPNRateLimit     int    `mapstructure:"ESPN_RATE_LIMIT"`
	BallDontLieAPIKey string `mapstructure:"BALLDONTLIE_API_KEY"`
	TheSportsDBAPIKey string `mapstructure:"THESPORTSDB_API_KEY"`
	RapidAPIKey       string `mapstructure:"RAPIDAPI_KEY"`
	DataFetchInterval string `mapstructure:"DATA_FETCH_INTERVAL"`

	// AI Integration
	AnthropicAPIKey   string `mapstructure:"ANTHROPIC_API_KEY"`
	AIRateLimit       int    `mapstructure:"AI_RATE_LIMIT"`
	AICacheExpiration int    `mapstructure:"AI_CACHE_EXPIRATION"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")

	// Set defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/dfs_optimizer?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("JWT_SECRET", "your-secret-key")
	viper.SetDefault("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")
	viper.SetDefault("MAX_LINEUPS", 150)
	viper.SetDefault("OPTIMIZATION_TIMEOUT", 30)
	viper.SetDefault("MAX_SIMULATIONS", 10000)
	viper.SetDefault("SIMULATION_WORKERS", 4)
	viper.SetDefault("ESPN_RATE_LIMIT", 10)
	viper.SetDefault("BALLDONTLIE_API_KEY", "")
	viper.SetDefault("THESPORTSDB_API_KEY", "4191544") // Free tier
	viper.SetDefault("RAPIDAPI_KEY", "")
	viper.SetDefault("DATA_FETCH_INTERVAL", "2h")
	viper.SetDefault("ANTHROPIC_API_KEY", "")
	viper.SetDefault("AI_RATE_LIMIT", 5)          // requests per minute
	viper.SetDefault("AI_CACHE_EXPIRATION", 3600) // 1 hour in seconds

	// Read from environment
	viper.AutomaticEnv()

	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Parse CORS origins from comma-separated string
	if corsStr := viper.GetString("CORS_ORIGINS"); corsStr != "" {
		config.CorsOrigins = strings.Split(corsStr, ",")
	}

	return &config, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
