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

	"github.com/stitts-dev/dfs-sim/services/api-gateway/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/api-gateway/internal/middleware"
	"github.com/stitts-dev/dfs-sim/services/api-gateway/internal/proxy"
	"github.com/stitts-dev/dfs-sim/services/api-gateway/internal/websocket"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

func main() {
	// Load configuration with gateway service defaults
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Set service-specific configuration
	cfg.ServiceName = config.ServiceTypeGateway
	// Gateway uses default port 8080

	// Initialize structured logger with service context
	structuredLogger := logger.InitLogger("info", cfg.IsDevelopment())
	logger.WithService("api-gateway").WithFields(logrus.Fields{
		"version":     "1.0.0",
		"environment": cfg.Env,
		"port":        cfg.Port,
	}).Info("Starting API Gateway")

	// Setup Gin mode
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database with gateway service connection pool
	db, err := database.NewGatewayServiceConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.WithService("api-gateway").Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.WithService("api-gateway").Fatalf("Failed to parse Redis URL: %v", err)
	}
	opt.DB = 2 // Use DB 2 for gateway service
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithService("api-gateway").Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize service proxy for inter-service communication
	serviceProxy := proxy.NewServiceProxy(cfg, structuredLogger)

	// Initialize WebSocket hub for optimization progress
	wsHub := websocket.NewGatewayHub(structuredLogger)
	go wsHub.Run()

	// Initialize router
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.CORS(cfg.CorsOrigins))
	router.Use(middleware.RequestLogger(structuredLogger))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg, structuredLogger)
	lineupHandler := handlers.NewLineupHandler(db, structuredLogger)
	userHandler := handlers.NewUserHandler(db, structuredLogger)
	healthHandler := handlers.NewHealthHandler(db, redisClient, serviceProxy, structuredLogger)

	// Setup API routes
	apiV1 := router.Group("/api/v1")
	{
		// Authentication endpoints (handled locally)
		auth := apiV1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}

		// User endpoints (handled locally)
		users := apiV1.Group("/users")
		users.Use(middleware.AuthRequired(cfg.JWTSecret))
		{
			users.GET("/profile", userHandler.GetProfile)
			users.PUT("/profile", userHandler.UpdateProfile)
			users.GET("/preferences", userHandler.GetPreferences)
			users.PUT("/preferences", userHandler.UpdatePreferences)
		}

		// Lineup endpoints (handled locally)
		lineups := apiV1.Group("/lineups")
		lineups.Use(middleware.AuthRequired(cfg.JWTSecret))
		{
			lineups.GET("", lineupHandler.GetUserLineups)
			lineups.POST("", lineupHandler.CreateLineup)
			lineups.GET("/:id", lineupHandler.GetLineup)
			lineups.PUT("/:id", lineupHandler.UpdateLineup)
			lineups.DELETE("/:id", lineupHandler.DeleteLineup)
			lineups.POST("/:id/export", lineupHandler.ExportLineup)
		}

		// Golf endpoints (proxied to golf service)
		golf := apiV1.Group("/golf")
		{
			golf.Any("/*path", serviceProxy.ProxyGolfRequest)
		}

		// Optimization endpoints (proxied to optimization service)
		optimization := apiV1.Group("/optimize")
		optimization.Use(middleware.AuthRequired(cfg.JWTSecret))
		{
			optimization.Any("", serviceProxy.ProxyOptimizationRequest)
			optimization.Any("/*path", serviceProxy.ProxyOptimizationRequest)
		}

		// Simulation endpoints (proxied to optimization service)
		simulation := apiV1.Group("/simulate")
		simulation.Use(middleware.AuthRequired(cfg.JWTSecret))
		{
			simulation.Any("", serviceProxy.ProxyOptimizationRequest)
			simulation.Any("/*path", serviceProxy.ProxyOptimizationRequest)
		}
	}

	// WebSocket endpoint for optimization progress (proxied to optimization service)
	router.GET("/ws/optimization-progress/:user_id", wsHub.HandleOptimizationProgress)

	// Health check endpoints
	router.GET("/health", healthHandler.GetHealth)
	router.GET("/ready", healthHandler.GetReady)
	router.GET("/metrics", healthHandler.GetMetrics)

	// Service status endpoints
	router.GET("/status/services", healthHandler.GetServiceStatus)
	router.GET("/status/circuit-breakers", healthHandler.GetCircuitBreakerStatus)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.WithService("api-gateway").WithField("port", cfg.Port).Info("API Gateway started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithService("api-gateway").Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.WithService("api-gateway").Info("Shutting down API Gateway...")

	// The server has 10 seconds to finish the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithService("api-gateway").Fatalf("API Gateway forced to shutdown: %v", err)
	}

	logger.WithService("api-gateway").Info("API Gateway exited")
}