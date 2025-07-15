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
	healthHandler := handlers.NewHealthHandler(db, redisClient, serviceProxy, structuredLogger)

	// Setup API routes
	apiV1 := router.Group("/api/v1")
	{
		// Authentication endpoints (proxied to user-service for phone auth)
		auth := apiV1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/verify", authHandler.VerifyOTP)
			auth.POST("/resend", authHandler.ResendOTP)
			auth.GET("/me", authHandler.GetCurrentUser)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}

		// User endpoints (proxied to user service)
		users := apiV1.Group("/users")
		users.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			users.Any("/*path", serviceProxy.ProxyUserRequest)
		}

		// Lineup endpoints (handled locally)
		lineups := apiV1.Group("/lineups")
		lineups.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			lineups.GET("", lineupHandler.GetUserLineups)
			lineups.POST("", lineupHandler.CreateLineup)
			lineups.GET("/:id", lineupHandler.GetLineup)
			lineups.PUT("/:id", lineupHandler.UpdateLineup)
			lineups.DELETE("/:id", lineupHandler.DeleteLineup)
			lineups.POST("/:id/export", lineupHandler.ExportLineup)
		}

		// Sports endpoints (proxied to golf service)
		sports := apiV1.Group("/sports")
		{
			sports.Any("/*path", serviceProxy.ProxyGolfRequest)
		}

		// Contest endpoints (proxied to golf service)
		contests := apiV1.Group("/contests")
		{
			contests.Any("/*path", serviceProxy.ProxyGolfRequest)
		}

		// Golf endpoints (proxied to golf service)
		golf := apiV1.Group("/golf")
		{
			golf.Any("/*path", serviceProxy.ProxyGolfRequest)
		}

		// Optimization endpoints (proxied to optimization service)
		optimization := apiV1.Group("/optimize")
		optimization.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			optimization.Any("", serviceProxy.ProxyOptimizationRequest)
			optimization.Any("/*path", serviceProxy.ProxyOptimizationRequest)
		}

		// Simulation endpoints (proxied to optimization service)
		simulation := apiV1.Group("/simulate")
		simulation.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			simulation.Any("", serviceProxy.ProxyOptimizationRequest)
			simulation.Any("/*path", serviceProxy.ProxyOptimizationRequest)
		}

		// AI Recommendations endpoints (proxied to ai-recommendations service)
		aiRecommendations := apiV1.Group("/ai-recommendations")
		aiRecommendations.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			aiRecommendations.Any("", serviceProxy.ProxyAIRecommendationsRequest)
			aiRecommendations.Any("/*path", serviceProxy.ProxyAIRecommendationsRequest)
		}

		// Analysis endpoints (proxied to ai-recommendations service)
		analysis := apiV1.Group("/analyze")
		analysis.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			analysis.Any("", serviceProxy.ProxyAIRecommendationsRequest)
			analysis.Any("/*path", serviceProxy.ProxyAIRecommendationsRequest)
		}

		// Ownership endpoints (proxied to ai-recommendations service)
		ownership := apiV1.Group("/ownership")
		ownership.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			ownership.Any("", serviceProxy.ProxyAIRecommendationsRequest)
			ownership.Any("/*path", serviceProxy.ProxyAIRecommendationsRequest)
		}

		// Real-time data endpoints (proxied to realtime service)
		realtime := apiV1.Group("/realtime")
		realtime.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			realtime.Any("", serviceProxy.ProxyRealtimeRequest)
			realtime.Any("/*path", serviceProxy.ProxyRealtimeRequest)
		}

		// Late swap endpoints (proxied to realtime service)
		lateswap := apiV1.Group("/lateswap")
		lateswap.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			lateswap.Any("", serviceProxy.ProxyRealtimeRequest)
			lateswap.Any("/*path", serviceProxy.ProxyRealtimeRequest)
		}

		// Alert endpoints (proxied to realtime service)  
		alerts := apiV1.Group("/alerts")
		alerts.Use(middleware.AuthRequired(cfg.SupabaseJWTSecret))
		{
			alerts.Any("", serviceProxy.ProxyRealtimeRequest)
			alerts.Any("/*path", serviceProxy.ProxyRealtimeRequest)
		}
	}

	// WebSocket endpoint for optimization progress (proxied to optimization service)
	router.GET("/ws/optimization-progress/:user_id", wsHub.HandleOptimizationProgress)

	// WebSocket endpoint for AI recommendations (proxied to ai-recommendations service)
	router.GET("/ws/ai-recommendations/:user_id", wsHub.HandleAIRecommendations)

	// WebSocket endpoint for real-time events (proxied to realtime service)
	router.GET("/ws/realtime-events/:user_id", wsHub.HandleRealtimeEvents)

	// WebSocket endpoint for late swap notifications (proxied to realtime service)
	router.GET("/ws/lateswap-notifications/:user_id", wsHub.HandleLateSwapNotifications)

	// WebSocket endpoint for alert notifications (proxied to realtime service)
	router.GET("/ws/alert-notifications/:user_id", wsHub.HandleAlertNotifications)

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