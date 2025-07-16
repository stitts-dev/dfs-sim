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

	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

func main() {
	// Load configuration with AI recommendations service defaults
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Set service-specific configuration
	cfg.ServiceName = "ai-recommendations"
	if cfg.Port == "8080" { // Only override if using default
		cfg.Port = "8084"
	}

	// Initialize structured logger with service context
	structuredLogger := logger.InitLogger("info", cfg.IsDevelopment())
	logger.WithService("ai-recommendations-service").WithFields(logrus.Fields{
		"version":     "1.0.0",
		"environment": cfg.Env,
		"port":        cfg.Port,
	}).Info("Starting AI Recommendations Service")

	// Setup Gin mode
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database with AI recommendations service connection pool
	db, err := database.NewAIRecommendationsServiceConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.WithService("ai-recommendations-service").Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis with AI recommendations-specific DB
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.WithService("ai-recommendations-service").Fatalf("Failed to parse Redis URL: %v", err)
	}
	opt.DB = 4 // Use DB 4 for AI recommendations service
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithService("ai-recommendations-service").Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize core services
	cacheService := services.NewCacheService(redisClient, structuredLogger)
	claudeClient := services.NewClaudeClient(cfg, structuredLogger)
	promptBuilder := services.NewPromptBuilder(cacheService, structuredLogger)
	realtimeAggregator := services.NewRealtimeAggregator(cacheService, structuredLogger)
	ownershipAnalyzer := services.NewOwnershipAnalyzer(db.DB, cacheService, structuredLogger)
	aiEngine := services.NewAIEngine(claudeClient, promptBuilder, realtimeAggregator, ownershipAnalyzer, structuredLogger)

	// Initialize WebSocket hub for real-time recommendation updates
	wsHub := websocket.NewRecommendationHub(structuredLogger)
	go wsHub.Run()

	// Initialize router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Initialize handlers
	recommendationHandler := handlers.NewRecommendationHandler(
		db.DB,
		aiEngine,
		wsHub,
		cfg,
		structuredLogger,
	)
	analysisHandler := handlers.NewAnalysisHandler(
		db.DB,
		aiEngine,
		ownershipAnalyzer,
		cfg,
		structuredLogger,
	)
	ownershipHandler := handlers.NewOwnershipHandler(
		ownershipAnalyzer,
		cfg,
		structuredLogger,
	)
	healthHandler := handlers.NewHealthHandler(db.DB, redisClient, claudeClient, structuredLogger)

	// Setup API routes for AI recommendations service
	apiV1 := router.Group("/api/v1")
	{
		// AI Recommendations endpoints
		apiV1.POST("/recommendations/players", recommendationHandler.GetPlayerRecommendations)
		apiV1.POST("/recommendations/lineup", recommendationHandler.GetLineupRecommendations)
		apiV1.POST("/recommendations/swap", recommendationHandler.GetSwapRecommendations)
		
		// Analysis endpoints
		apiV1.POST("/analyze/lineup", analysisHandler.AnalyzeLineup)
		apiV1.POST("/analyze/contest", analysisHandler.AnalyzeContest)
		apiV1.GET("/analyze/trends/:sport", analysisHandler.GetTrends)
		
		// Ownership intelligence endpoints
		apiV1.GET("/ownership/:contestId", ownershipHandler.GetOwnershipData)
		apiV1.GET("/ownership/:contestId/leverage", ownershipHandler.GetLeverageOpportunities)
		apiV1.GET("/ownership/:contestId/trends", ownershipHandler.GetOwnershipTrends)
	}

	// WebSocket endpoint for real-time recommendation updates
	router.GET("/ws/ai-recommendations/:user_id", wsHub.HandleWebSocket)

	// Health check endpoints (support both GET and HEAD)
	router.GET("/health", healthHandler.GetHealth)
	router.HEAD("/health", healthHandler.GetHealth)
	router.GET("/ready", healthHandler.GetReady)
	router.HEAD("/ready", healthHandler.GetReady)
	router.GET("/metrics", healthHandler.GetMetrics)
	router.HEAD("/metrics", healthHandler.GetMetrics)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.WithService("ai-recommendations-service").WithField("port", cfg.Port).Info("AI recommendations service started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithService("ai-recommendations-service").Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.WithService("ai-recommendations-service").Info("Shutting down AI recommendations service...")

	// The server has 5 seconds to finish the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithService("ai-recommendations-service").Fatalf("AI recommendations service forced to shutdown: %v", err)
	}

	logger.WithService("ai-recommendations-service").Info("AI recommendations service exited")
}