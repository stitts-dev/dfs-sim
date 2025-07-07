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
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Setup logging
	if cfg.IsDevelopment() {
		logrus.SetLevel(logrus.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
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
	
	// Parse fetch interval
	fetchInterval, err := time.ParseDuration(cfg.DataFetchInterval)
	if err != nil {
		logrus.Warnf("Invalid fetch interval, using default 2h: %v", err)
		fetchInterval = 2 * time.Hour
	}
	
	// Initialize data fetcher
	dataFetcher := services.NewDataFetcherService(db, cacheService, aggregator, logrus.StandardLogger(), fetchInterval)
	if err := dataFetcher.Start(); err != nil {
		logrus.Errorf("Failed to start data fetcher: %v", err)
	}
	defer dataFetcher.Stop()

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.CorsOrigins))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	// Setup API routes under /api/v1
	apiV1 := router.Group("/api/v1")
	api.SetupRoutes(apiV1, db, cacheService, webSocketHub, cfg, aggregator, dataFetcher)

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
