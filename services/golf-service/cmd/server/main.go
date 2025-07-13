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
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/golf-service/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/golf-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/services/golf-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
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
	logger.WithService("golf-service").WithFields(logrus.Fields{
		"version":     "1.0.0",
		"environment": cfg.Env,
		"port":        cfg.Port,
	}).Info("Starting Golf Service")

	// Setup Gin mode
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database with golf service connection pool
	db, err := database.NewGolfServiceConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.WithService("golf-service").Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.WithService("golf-service").Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithService("golf-service").Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize services
	cacheService := services.NewCacheService(redisClient, structuredLogger)
	circuitBreakerService := services.NewCircuitBreakerService(
		cfg.CircuitBreakerThreshold,
		cfg.ExternalAPITimeout,
		structuredLogger,
	)

	// Initialize golf providers with rate limiting
	var rapidAPIGolf *providers.RapidAPIGolfClient
	var espnGolf *providers.ESPNGolfClient

	if cfg.RapidAPIKey != "" {
		logger.WithService("golf-service").Info("Initializing RapidAPI Golf provider with rate limiting")
		rapidAPIGolf = providers.NewRapidAPIGolfClient(cfg.RapidAPIKey, cacheService, structuredLogger)
	}

	logger.WithService("golf-service").Info("Initializing ESPN Golf fallback provider")
	espnGolf = providers.NewESPNGolfClient(cacheService, structuredLogger)

	// Initialize golf business services
	golfProjectionService := services.NewGolfProjectionService(db, structuredLogger)
	golfSyncService := services.NewGolfTournamentSyncService(
		db, rapidAPIGolf, espnGolf, cacheService, structuredLogger,
	)

	// Initialize startup manager for golf service
	startupManager := services.NewStartupManager(cfg, structuredLogger, golfSyncService, circuitBreakerService)

	// Initialize router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Initialize handlers
	golfHandler := handlers.NewGolfHandler(
		db, 
		cacheService, 
		golfProjectionService,
		golfSyncService, 
		rapidAPIGolf, 
		espnGolf,
		structuredLogger,
	)
	healthHandler := handlers.NewHealthHandler(db, redisClient, startupManager, structuredLogger)

	// Setup API routes for golf service only
	apiV1 := router.Group("/api/v1")
	{
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

	// Health check endpoints
	router.GET("/health", healthHandler.GetHealth)
	router.GET("/ready", healthHandler.GetReady)
	router.GET("/metrics", healthHandler.GetMetrics)

	// Start critical services
	logger.WithService("golf-service").Info("Starting critical services")
	startupManager.StartCriticalServices()

	// Start background initialization in separate goroutine
	go func() {
		logger.WithService("golf-service").Info("Starting background initialization")
		startupManager.StartBackgroundInitialization()
	}()

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.WithService("golf-service").WithField("port", cfg.Port).Info("Golf service started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithService("golf-service").Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.WithService("golf-service").Info("Shutting down golf service...")

	// The server has 5 seconds to finish the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithService("golf-service").Fatalf("Golf service forced to shutdown: %v", err)
	}

	logger.WithService("golf-service").Info("Golf service exited")
}