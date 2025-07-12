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

	"github.com/jstittsworth/dfs-optimizer/internal/api"
	"github.com/jstittsworth/dfs-optimizer/internal/api/handlers"
	"github.com/jstittsworth/dfs-optimizer/internal/api/middleware"
	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Initialize structured logger
	structuredLogger := logger.InitLogger()
	structuredLogger.WithFields(logrus.Fields{
		"version":      "1.0.0",
		"environment":  cfg.Env,
		"database_url": cfg.DatabaseURL,
		"redis_url":    cfg.RedisURL,
	}).Info("Starting DFS Optimizer")

	// Setup Gin mode
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	db, err := database.NewConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logrus.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logrus.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize services
	cacheService := services.NewCacheService(redisClient)
	webSocketHub := services.NewWebSocketHub()
	go webSocketHub.Run()

	// Initialize data providers
	aggregator := services.NewDataAggregator(db, cacheService, logrus.StandardLogger(), cfg.BallDontLieAPIKey)

	// Initialize golf provider for tournament sync
	var golfProvider interface {
		GetCurrentTournament() (*providers.GolfTournamentData, error)
		GetTournamentSchedule() ([]providers.GolfTournamentData, error)
		GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error)
		GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error)
		GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error)
	}
	if cfg.RapidAPIKey != "" {
		logrus.Info("Initializing RapidAPI Golf provider for tournament sync")
		golfProvider = providers.NewRapidAPIGolfClient(cfg.RapidAPIKey, cacheService, logrus.StandardLogger())
	} else {
		logrus.Info("Initializing ESPN Golf provider for tournament sync")
		golfProvider = providers.NewESPNGolfClient(cacheService, logrus.StandardLogger())
	}

	// Set golf provider in aggregator
	aggregator.SetGolfProvider(golfProvider)

	// Initialize golf tournament sync service
	golfSyncService := services.NewGolfTournamentSyncService(db, golfProvider, logrus.StandardLogger())

	// Parse fetch interval
	fetchInterval, err := time.ParseDuration(cfg.DataFetchInterval)
	if err != nil {
		logrus.Warnf("Invalid fetch interval, using default 2h: %v", err)
		fetchInterval = 2 * time.Hour
	}

	// Initialize data fetcher
	dataFetcher := services.NewDataFetcherService(db, cacheService, aggregator, logrus.StandardLogger(), fetchInterval)

	// Set golf sync service in data fetcher for scheduled syncs
	dataFetcher.SetGolfSyncService(golfSyncService)

	// Initialize circuit breaker service
	circuitBreakerService := services.NewCircuitBreakerService(
		cfg.CircuitBreakerThreshold,
		cfg.ExternalAPITimeout,
		logrus.StandardLogger(),
	)

	// Initialize startup manager for coordinated startup
	startupManager := services.NewStartupManager(
		cfg,
		logrus.StandardLogger(),
		dataFetcher,
		golfSyncService,
		circuitBreakerService,
	)

	// Start critical services (non-blocking)
	if err := startupManager.StartCriticalServices(); err != nil {
		logrus.Fatalf("Failed to start critical services: %v", err)
	}

	// Start background initialization (non-blocking)
	startupManager.StartBackgroundInitialization()

	// Defer cleanup
	defer dataFetcher.Stop()
	defer startupManager.Shutdown()

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.CorsOrigins))

	// Initialize health and admin handlers
	healthHandler := handlers.NewHealthHandler(startupManager)
	adminHandler := handlers.NewAdminHandler(startupManager)

	// Health check endpoints
	router.GET("/health", healthHandler.GetHealth)
	router.GET("/ready", healthHandler.GetReady)
	router.GET("/startup-status", healthHandler.GetStartupStatus)

	// Setup API routes under /api/v1
	apiV1 := router.Group("/api/v1")
	api.SetupRoutes(apiV1, db, cacheService, webSocketHub, cfg, aggregator, dataFetcher)

	// Admin endpoints under /api/v1/admin
	adminGroup := apiV1.Group("/admin")
	{
		adminGroup.GET("/status", adminHandler.GetSystemStatus)
		adminGroup.POST("/sync/golf", adminHandler.TriggerGolfSync)
		adminGroup.POST("/sync/data", adminHandler.TriggerDataFetch)
	}

	// Setup WebSocket endpoint at root level (not under /api/v1)
	wsHandler := handlers.NewWebSocketHandler(webSocketHub, cfg.JWTSecret)
	router.GET("/ws", middleware.OptionalAuth(cfg.JWTSecret), wsHandler.HandleWebSocket)

	// Log all registered routes
	logrus.Info("=== REGISTERED ROUTES ===")
	for _, route := range router.Routes() {
		logrus.Infof("%s %s", route.Method, route.Path)
	}
	logrus.Info("=========================")

	// Setup server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logrus.Infof("Starting server on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}
