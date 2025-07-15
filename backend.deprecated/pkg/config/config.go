package config

import (
	"fmt"
	"strings"
	"time"

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

	// SMS Configuration
	SMSProvider string `mapstructure:"SMS_PROVIDER"` // "supabase", "twilio", "mock"
	
	// Supabase Configuration
	SupabaseURL        string `mapstructure:"SUPABASE_URL"`
	SupabaseServiceKey string `mapstructure:"SUPABASE_SERVICE_KEY"`
	SupabaseAnonKey    string `mapstructure:"SUPABASE_ANON_KEY"`
	
	// Twilio Configuration
	TwilioAccountSID string `mapstructure:"TWILIO_ACCOUNT_SID"`
	TwilioAuthToken  string `mapstructure:"TWILIO_AUTH_TOKEN"`
	TwilioFromNumber string `mapstructure:"TWILIO_FROM_NUMBER"`

	// Startup Configuration
	SkipInitialGolfSync         bool          `mapstructure:"SKIP_INITIAL_GOLF_SYNC"`
	SkipInitialDataFetch        bool          `mapstructure:"SKIP_INITIAL_DATA_FETCH"`
	SkipInitialContestDiscovery bool          `mapstructure:"SKIP_INITIAL_CONTEST_DISCOVERY"`
	StartupDelaySeconds         int           `mapstructure:"STARTUP_DELAY_SECONDS"`
	ExternalAPITimeout          time.Duration `mapstructure:"EXTERNAL_API_TIMEOUT"`
	CircuitBreakerThreshold     int           `mapstructure:"CIRCUIT_BREAKER_THRESHOLD"`

	// Feature Flags
	EnableAutoPlayerFetch  bool     `mapstructure:"ENABLE_AUTO_PLAYER_FETCH"`
	EnableBackgroundJobs   bool     `mapstructure:"ENABLE_BACKGROUND_JOBS"`
	GolfOnlyMode          bool     `mapstructure:"GOLF_ONLY_MODE"`
	SupportedSports       []string `mapstructure:"SUPPORTED_SPORTS"`
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

	// SMS defaults
	viper.SetDefault("SMS_PROVIDER", "mock") // Default to mock for development
	viper.SetDefault("SUPABASE_URL", "")
	viper.SetDefault("SUPABASE_SERVICE_KEY", "")
	viper.SetDefault("SUPABASE_ANON_KEY", "")
	viper.SetDefault("TWILIO_ACCOUNT_SID", "")
	viper.SetDefault("TWILIO_AUTH_TOKEN", "")
	viper.SetDefault("TWILIO_FROM_NUMBER", "")

	// Startup optimization defaults - maintain backward compatibility
	viper.SetDefault("SKIP_INITIAL_GOLF_SYNC", false)         // Keep current behavior by default
	viper.SetDefault("SKIP_INITIAL_DATA_FETCH", false)        // Keep current behavior by default
	viper.SetDefault("SKIP_INITIAL_CONTEST_DISCOVERY", false) // Keep current behavior by default
	viper.SetDefault("STARTUP_DELAY_SECONDS", 0)              // No delay by default
	viper.SetDefault("EXTERNAL_API_TIMEOUT", "10s")           // Conservative timeout
	viper.SetDefault("CIRCUIT_BREAKER_THRESHOLD", 5)          // Fail after 5 consecutive failures

	// Feature flag defaults - golf-only mode by default for better stability
	viper.SetDefault("ENABLE_AUTO_PLAYER_FETCH", false) // Disable automatic player fetching by default
	viper.SetDefault("ENABLE_BACKGROUND_JOBS", false)   // Disable background jobs by default
	viper.SetDefault("GOLF_ONLY_MODE", true)            // Enable golf-only mode by default
	viper.SetDefault("SUPPORTED_SPORTS", "golf")        // Only support golf by default

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

	// Parse supported sports from comma-separated string
	if sportsStr := viper.GetString("SUPPORTED_SPORTS"); sportsStr != "" {
		config.SupportedSports = strings.Split(sportsStr, ",")
	}

	return &config, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
