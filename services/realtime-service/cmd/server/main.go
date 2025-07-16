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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/alerts"
	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/events"
	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/lateswap"
	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/ownership"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level based on environment
	if cfg.IsDevelopment() {
		logger.SetLevel(logrus.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	logger.WithFields(logrus.Fields{
		"service": "realtime-service",
		"port":    cfg.Port,
		"env":     cfg.Env,
	}).Info("Starting realtime service")

	// Initialize database connection
	db, err := initDatabase(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize database")
	}

	// Initialize Redis connection
	redisClient, err := initRedis(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Redis")
	}
	defer redisClient.Close()

	// Initialize event processor
	eventProcessor := events.NewEventProcessor(redisClient, db, logger)

	// Initialize ownership tracker
	ownershipTracker := ownership.NewOwnershipTracker(db, redisClient, logger)

	// Initialize alert engine
	alertEngine := alerts.NewAlertEngine(db, redisClient, logger)

	// Initialize late swap engine
	lateSwapEngine := lateswap.NewRecommendationEngine(db, redisClient, logger)

	// Initialize API handlers
	apiHandlers := handlers.NewHandlers(
		db,
		redisClient,
		eventProcessor,
		ownershipTracker,
		alertEngine,
		lateSwapEngine,
		logger,
	)

	// Set up router
	router := setupRouter(apiHandlers, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start event processor
	go func() {
		if err := eventProcessor.Start(ctx); err != nil {
			logger.WithError(err).Error("Event processor failed")
		}
	}()

	// Start ownership tracker
	go func() {
		if err := ownershipTracker.Start(ctx); err != nil {
			logger.WithError(err).Error("Ownership tracker failed")
		}
	}()

	// Start alert engine
	go func() {
		if err := alertEngine.Start(ctx); err != nil {
			logger.WithError(err).Error("Alert engine failed")
		}
	}()

	// Start server
	go func() {
		logger.WithField("addr", server.Addr).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Cancel background services
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server exited")
}

func initDatabase(cfg *config.Config, logger *logrus.Logger) (*gorm.DB, error) {
	logger.Info("Connecting to database...")

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: NewGormLogger(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Info("Database connection established")
	return db, nil
}

func initRedis(cfg *config.Config, logger *logrus.Logger) (*redis.Client, error) {
	logger.Info("Connecting to Redis...")

	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Use dedicated DB for realtime service
	opts.DB = 5 // Use DB 5 for realtime service

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.Info("Redis connection established")
	return client, nil
}

func setupRouter(handlers *handlers.Handlers, logger *logrus.Logger) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(loggingMiddleware(logger))

	// Health check
	router.GET("/health", handlers.HealthCheck)
	router.GET("/ready", handlers.ReadinessCheck)

	// API routes
	api := router.Group("/api/v1")
	{
		// Real-time events
		api.POST("/events", handlers.CreateEvent)
		api.GET("/events", handlers.GetEvents)
		api.GET("/events/:id", handlers.GetEvent)

		// Ownership tracking
		api.GET("/ownership/:contest_id", handlers.GetOwnership)
		api.GET("/ownership/:contest_id/trends", handlers.GetOwnershipTrends)

		// Alert rules
		api.GET("/alerts/rules", handlers.GetAlertRules)
		api.POST("/alerts/rules", handlers.CreateAlertRule)
		api.PUT("/alerts/rules/:id", handlers.UpdateAlertRule)
		api.DELETE("/alerts/rules/:id", handlers.DeleteAlertRule)

		// Late swap recommendations
		api.GET("/lateswap/recommendations", handlers.GetLateSwapRecommendations)
		api.POST("/lateswap/recommendations/:id/accept", handlers.AcceptLateSwap)
		api.POST("/lateswap/recommendations/:id/reject", handlers.RejectLateSwap)

		// WebSocket endpoint
		api.GET("/ws/:user_id", handlers.HandleWebSocket)
	}

	// Admin routes
	admin := router.Group("/admin")
	{
		admin.GET("/status", handlers.GetServiceStatus)
		admin.GET("/metrics", handlers.GetMetrics)
		admin.POST("/events/simulate", handlers.SimulateEvent)
	}

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func loggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithWriter(logger.Writer())
}

// NewGormLogger creates a GORM logger that integrates with logrus
func NewGormLogger(logger *logrus.Logger) *GormLogger {
	return &GormLogger{logger: logger}
}

type GormLogger struct {
	logger *logrus.Logger
}

func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.WithContext(ctx).Infof(msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.WithContext(ctx).Warnf(msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.WithContext(ctx).Errorf(msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	
	if err != nil {
		l.logger.WithContext(ctx).WithFields(logrus.Fields{
			"elapsed": elapsed,
			"rows":    rows,
			"sql":     sql,
		}).WithError(err).Error("Database query failed")
	} else {
		l.logger.WithContext(ctx).WithFields(logrus.Fields{
			"elapsed": elapsed,
			"rows":    rows,
			"sql":     sql,
		}).Debug("Database query executed")
	}
}