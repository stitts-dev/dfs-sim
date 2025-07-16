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
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

func main() {
	// Load configuration with golf service defaults
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Set service-specific configuration
	cfg.ServiceName = config.ServiceTypeGolf
	if cfg.Port == "8080" { // Only override if using default
		cfg.Port = "8081"
	}

	// Initialize structured logger with service context
	structuredLogger := logger.InitLogger("info", cfg.IsDevelopment())
	logger.WithService("sports-data-service").WithFields(logrus.Fields{
		"version":     "1.0.0",
		"environment": cfg.Env,
		"port":        cfg.Port,
	}).Info("Starting Sports Data Service")

	// Setup Gin mode
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database with sports data service connection pool
	db, err := database.NewGolfServiceConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.WithService("sports-data-service").Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.WithService("sports-data-service").Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithService("sports-data-service").Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize services
	cacheService := services.NewCacheService(redisClient)
	circuitBreakerService := services.NewCircuitBreakerService(
		cfg.CircuitBreakerThreshold,
		cfg.ExternalAPITimeout,
		structuredLogger,
	)

	// Initialize DataGolf provider (only provider now)
	var dataGolf *providers.DataGolfClient

	// Initialize DataGolf provider if enabled and configured
	if cfg.DataGolfEnabled && cfg.DataGolfAPIKey != "" {
		logger.WithService("sports-data-service").Info("Initializing DataGolf provider with advanced analytics")
		dataGolf = providers.NewDataGolfClient(cfg.DataGolfAPIKey, db.DB, cacheService, structuredLogger)
	} else {
		logger.WithService("sports-data-service").Fatal("DataGolf not configured - DataGolf key missing or disabled (required for operation)")
	}

	// Initialize golf business services
	golfProjectionService := services.NewGolfProjectionService(db, cacheService, structuredLogger, cfg.DataGolfAPIKey)

	// Use DataGolf as the only provider
	var primaryGolfProvider interface {
		GetCurrentTournament() (*providers.GolfTournamentData, error)
		GetTournamentSchedule() ([]providers.GolfTournamentData, error)
		GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error)
	}

	if dataGolf != nil {
		primaryGolfProvider = dataGolf
		logger.WithService("sports-data-service").Info("Using DataGolf as the only golf provider (advanced analytics enabled)")
	} else {
		logger.WithService("sports-data-service").Fatal("DataGolf provider is required but not available")
	}

	golfSyncService := services.NewGolfTournamentSyncService(
		db, primaryGolfProvider, structuredLogger,
	)

	// Initialize data fetcher service with DataGolf provider
	dataFetcherService := services.NewDataFetcherService(
		db,
		structuredLogger,
		golfSyncService,
		dataGolf,
		circuitBreakerService,
		cacheService,
	)

	// Initialize startup manager for sports data service
	startupManager := services.NewStartupManager(cfg, structuredLogger, dataFetcherService, golfSyncService, circuitBreakerService)

	// Initialize router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Initialize handlers
	golfHandler := handlers.NewGolfHandler(
		db,
		cacheService,
		golfProjectionService,
		golfSyncService,
		dataGolf,
		structuredLogger,
	)
	healthHandler := handlers.NewHealthHandler(db, redisClient, startupManager, structuredLogger)

	// Setup API routes for golf service only
	apiV1 := router.Group("/api/v1")
	{
		// Sports configuration endpoints (golf service only supports golf)
		apiV1.GET("/sports/available", golfHandler.GetAvailableSports)

		// Contest endpoints (golf tournaments as contests)
		apiV1.GET("/contests", golfHandler.ListContests)
		apiV1.GET("/contests/:id", golfHandler.GetContest)

		// Golf tournament endpoints
		apiV1.GET("/golf/tournaments", golfHandler.ListTournaments)
		apiV1.GET("/golf/tournaments/:id", golfHandler.GetTournament)
		apiV1.GET("/golf/tournaments/:id/leaderboard", golfHandler.GetTournamentLeaderboard)
		apiV1.GET("/golf/tournaments/:id/players", golfHandler.GetTournamentPlayers)
		apiV1.POST("/golf/tournaments/sync", golfHandler.SyncTournamentData)

		// Golf player endpoints
		apiV1.GET("/golf/players/:id", golfHandler.GetGolfPlayer)
		apiV1.GET("/golf/players/:id/projections", golfHandler.GetPlayerProjections)
		apiV1.GET("/golf/players/:id/history", golfHandler.GetPlayerCourseHistory)

		// Golf data sync endpoints
		apiV1.POST("/golf/sync/current", golfHandler.SyncCurrentTournament)
		apiV1.POST("/golf/sync/schedule", golfHandler.SyncTournamentSchedule)
	}

	// Health check endpoints (support both GET and HEAD)
	router.GET("/health", healthHandler.GetHealth)
	router.HEAD("/health", healthHandler.GetHealth)
	router.GET("/ready", healthHandler.GetReady)
	router.HEAD("/ready", healthHandler.GetReady)
	router.GET("/metrics", healthHandler.GetMetrics)
	router.HEAD("/metrics", healthHandler.GetMetrics)

	// Start critical services
	logger.WithService("sports-data-service").Info("Starting critical services")
	startupManager.StartCriticalServices()

	// Start background initialization in separate goroutine
	go func() {
		logger.WithService("sports-data-service").Info("Starting background initialization")
		startupManager.StartBackgroundInitialization()
	}()

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.WithService("sports-data-service").WithField("port", cfg.Port).Info("Sports data service started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithService("sports-data-service").Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.WithService("sports-data-service").Info("Shutting down sports data service...")

	// The server has 5 seconds to finish the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithService("sports-data-service").Fatalf("Sports data service forced to shutdown: %v", err)
	}

	logger.WithService("sports-data-service").Info("Sports data service exited")
}
