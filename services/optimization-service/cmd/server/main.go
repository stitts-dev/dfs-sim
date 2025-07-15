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

	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/pkg/cache"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

func main() {
	// Load configuration with optimization service defaults
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Set service-specific configuration
	cfg.ServiceName = config.ServiceTypeOptimization
	if cfg.Port == "8080" { // Only override if using default
		cfg.Port = "8082"
	}

	// Initialize structured logger with service context
	structuredLogger := logger.InitLogger("info", cfg.IsDevelopment())
	logger.WithService("optimization-service").WithFields(logrus.Fields{
		"version":     "1.0.0",
		"environment": cfg.Env,
		"port":        cfg.Port,
	}).Info("Starting Optimization Service")

	// Setup Gin mode
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database with optimization service connection pool
	db, err := database.NewOptimizationServiceConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.WithService("optimization-service").Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis with optimization-specific DB
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.WithService("optimization-service").Fatalf("Failed to parse Redis URL: %v", err)
	}
	opt.DB = 1 // Use DB 1 for optimization service
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithService("optimization-service").Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize cache service for optimization results
	cacheService := cache.NewOptimizationCacheService(redisClient, structuredLogger)

	// Initialize WebSocket hub for progress updates
	wsHub := websocket.NewHub(structuredLogger)
	go wsHub.Run()

	// Initialize router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Initialize handlers
	optimizationHandler := handlers.NewOptimizationHandler(
		db,
		cacheService,
		wsHub,
		cfg,
		structuredLogger,
	)
	simulationHandler := handlers.NewSimulationHandler(
		db,
		cacheService,
		wsHub,
		cfg,
		structuredLogger,
	)
	healthHandler := handlers.NewHealthHandler(db, redisClient, structuredLogger)

	// Setup API routes for optimization service
	apiV1 := router.Group("/api/v1")
	{
		// Optimization endpoints
		apiV1.POST("/optimize", optimizationHandler.OptimizeLineups)
		apiV1.POST("/optimize/validate", optimizationHandler.ValidateOptimizationRequest)
		apiV1.GET("/optimize/cache-status", optimizationHandler.GetCacheStatus)
		
		// Simulation endpoints
		apiV1.POST("/simulate", simulationHandler.RunSimulation)
		apiV1.GET("/simulate/:id/status", simulationHandler.GetSimulationStatus)
		apiV1.GET("/simulate/:id/results", simulationHandler.GetSimulationResults)
	}

	// WebSocket endpoint for progress updates
	router.GET("/ws/optimization-progress/:user_id", wsHub.HandleWebSocket)

	// Health check endpoints
	router.GET("/health", healthHandler.GetHealth)
	router.GET("/ready", healthHandler.GetReady)
	router.GET("/metrics", healthHandler.GetMetrics)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.WithService("optimization-service").WithField("port", cfg.Port).Info("Optimization service started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithService("optimization-service").Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.WithService("optimization-service").Info("Shutting down optimization service...")

	// The server has 5 seconds to finish the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithService("optimization-service").Fatalf("Optimization service forced to shutdown: %v", err)
	}

	logger.WithService("optimization-service").Info("Optimization service exited")
}