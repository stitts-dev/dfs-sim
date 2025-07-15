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

	"github.com/stitts-dev/dfs-sim/services/user-service/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/user-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/user-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

func main() {
	// Load configuration with user service defaults
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Set service-specific configuration
	cfg.ServiceName = "user"
	if cfg.Port == "8080" { // Only override if using default
		cfg.Port = "8083"
	}

	// Initialize structured logger with service context
	structuredLogger := logger.InitLogger("info", cfg.IsDevelopment())
	logger.WithService("user-service").WithFields(logrus.Fields{
		"version":     "1.0.0",
		"environment": cfg.Env,
		"port":        cfg.Port,
	}).Info("Starting User Service")

	// Setup Gin mode
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database with user service connection pool
	db, err := database.NewUserServiceConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logger.WithService("user-service").Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Auto-migrate database schema to match Go models
	// This ensures Supabase tables are compatible with our Go models
	// Temporarily disabled to test API functionality
	// if err := autoMigrateUserTables(db); err != nil {
	//	logger.WithService("user-service").Fatalf("Failed to migrate database schema: %v", err)
	// }

	// Connect to Redis (use DB 3 for user service)
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.WithService("user-service").Fatalf("Failed to parse Redis URL: %v", err)
	}
	opt.DB = 3 // User service uses Redis DB 3
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithService("user-service").Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize user services
	userService := services.NewUserService(db, cfg, structuredLogger)
	
	// Initialize Stripe service
	// fmt.Printf("💳 Creating Stripe service...\n")
	// stripeService := services.NewStripeService(db, cfg, structuredLogger)

	// Initialize router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Initialize handlers
	fmt.Printf("👥 Creating user handler...\n")
	userHandler := handlers.NewSimpleUserHandler(userService, structuredLogger, db)
	fmt.Printf("🔑 Creating auth handler...\n")
	authHandler := handlers.NewAuthHandler(db, cfg, structuredLogger)
	// fmt.Printf("💳 Creating Stripe handler...\n")
	// stripeHandler := handlers.NewStripeHandler(stripeService, userService)
	fmt.Printf("✅ Handlers created successfully\n")

	// Setup API routes for user service only
	apiV1 := router.Group("/api/v1")
	{
		// User authentication endpoints
		fmt.Printf("🚀 Registering auth routes...\n")
		auth := apiV1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/verify", authHandler.VerifyOTP)
			auth.POST("/resend", authHandler.ResendCode)
			auth.GET("/me", authHandler.GetCurrentUser)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}
		fmt.Printf("✅ Auth routes registered successfully\n")

		// User profile endpoints
		users := apiV1.Group("/users")
		{
			users.GET("/me", userHandler.GetProfile)
			users.PUT("/me", userHandler.UpdateProfile)
			users.GET("/preferences", userHandler.GetPreferences)
			users.PUT("/preferences", userHandler.UpdatePreferences)
			users.GET("/subscription", userHandler.GetSubscription)
			users.PUT("/subscription", userHandler.UpdateSubscription)
		}

		// User management endpoints (admin)
		admin := apiV1.Group("/admin/users")
		{
			admin.GET("/", userHandler.ListUsers)
			admin.GET("/:id", userHandler.GetUser)
			admin.PUT("/:id", userHandler.UpdateUser)
			admin.DELETE("/:id", userHandler.DeleteUser)
		}

		// Stripe payment endpoints - TEMPORARILY DISABLED
		// fmt.Printf("💳 Registering Stripe routes...\n")
		// stripe := apiV1.Group("/stripe")
		// {
		//	stripe.POST("/customers", stripeHandler.CreateCustomer)
		//	stripe.GET("/customers/ensure", stripeHandler.EnsureCustomer)
		// }

		// Subscription management endpoints
		// subscriptions := apiV1.Group("/subscriptions")
		// {
		//	subscriptions.POST("/", stripeHandler.CreateSubscription)
		//	subscriptions.GET("/status", stripeHandler.GetSubscriptionStatus)
		//	subscriptions.DELETE("/:subscription_id", stripeHandler.CancelSubscription)
		// }

		// Subscription tiers (public)
		// apiV1.GET("/subscription-tiers", stripeHandler.GetSubscriptionTiers)

		// Webhook endpoints (no auth required)
		// webhooks := apiV1.Group("/webhooks")
		// {
		//	webhooks.POST("/stripe", stripeHandler.WebhookHandler)
		// }
		// fmt.Printf("✅ Stripe routes registered successfully\n")
	}

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "user-service",
			"timestamp": time.Now().Unix(),
		})
	})
	router.GET("/ready", func(c *gin.Context) {
		// Check database connectivity
		if err := db.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": "database unavailable",
			})
			return
		}
		
		// Check Redis connectivity
		if err := redisClient.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": "redis unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": "user-service",
		})
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.WithService("user-service").WithField("port", cfg.Port).Info("User service started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithService("user-service").Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.WithService("user-service").Info("Shutting down user service...")

	// The server has 5 seconds to finish the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithService("user-service").Fatalf("User service forced to shutdown: %v", err)
	}

	logger.WithService("user-service").Info("User service exited")
}

// autoMigrateUserTables ensures Supabase database schema matches Go models
func autoMigrateUserTables(db *database.DB) error {
	logger.WithService("user-service").Info("Auto-migrating user database schema to match Go models")
	
	// AutoMigrate will create tables if they don't exist and add missing columns
	// It won't delete existing columns or change column types
	err := db.AutoMigrate(
		&models.User{},
		&models.UserPreferences{},
		&models.SubscriptionTier{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to auto-migrate user tables: %w", err)
	}
	
	// Create default subscription tiers if they don't exist
	if err := createDefaultSubscriptionTiers(db); err != nil {
		return fmt.Errorf("failed to create default subscription tiers: %w", err)
	}
	
	logger.WithService("user-service").Info("Database schema migration completed successfully")
	return nil
}

// createDefaultSubscriptionTiers ensures default subscription tiers exist
func createDefaultSubscriptionTiers(db *database.DB) error {
	defaultTiers := []models.SubscriptionTier{
		{
			Name:                 "free",
			PriceCents:           0,
			Currency:             "USD",
			MonthlyOptimizations: 10,
			MonthlySimulations:   5,
			AIRecommendations:    false,
			BankVerification:     false,
			PrioritySupport:      false,
		},
		{
			Name:                 "basic",
			PriceCents:           999,
			Currency:             "USD",
			MonthlyOptimizations: 50,
			MonthlySimulations:   25,
			AIRecommendations:    true,
			BankVerification:     false,
			PrioritySupport:      false,
		},
		{
			Name:                 "premium",
			PriceCents:           2999,
			Currency:             "USD",
			MonthlyOptimizations: -1, // unlimited
			MonthlySimulations:   -1, // unlimited
			AIRecommendations:    true,
			BankVerification:     true,
			PrioritySupport:      true,
		},
	}
	
	for _, tier := range defaultTiers {
		// Check if tier already exists
		var existingTier models.SubscriptionTier
		err := db.Where("name = ?", tier.Name).First(&existingTier).Error
		if err != nil {
			// Tier doesn't exist, create it
			if err := db.Create(&tier).Error; err != nil {
				return fmt.Errorf("failed to create subscription tier %s: %w", tier.Name, err)
			}
			logger.WithService("user-service").WithField("tier", tier.Name).Info("Created default subscription tier")
		}
	}
	
	return nil
}