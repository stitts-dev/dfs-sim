package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/api/handlers"
	"github.com/jstittsworth/dfs-optimizer/internal/api/middleware"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
)

// SetupRoutes configures all API routes on the given router group
func SetupRoutes(group *gin.RouterGroup, db *database.DB, cache *services.CacheService, wsHub *services.WebSocketHub, cfg *config.Config, aggregator *services.DataAggregator, dataFetcher *services.DataFetcherService) {
	// Initialize services
	aiService := services.NewAIRecommendationService(db, cfg, cache)

	// Initialize handlers
	playerHandler := handlers.NewPlayerHandler(db, cache, aggregator, dataFetcher)
	lineupHandler := handlers.NewLineupHandler(db, cache)
	optimizerHandler := handlers.NewOptimizerHandler(db, cache, cfg)
	simulationHandler := handlers.NewSimulationHandler(db, cache, wsHub)
	contestHandler := handlers.NewContestHandler(db, cache, dataFetcher)
	exportHandler := handlers.NewExportHandler(db)
	glossaryHandler := handlers.NewGlossaryHandler(db, cache)
	preferencesHandler := handlers.NewPreferencesHandler(db, cache)
	aiRecommendationHandler := handlers.NewAIRecommendationHandler(aiService)
	// Create a logger for the golf handler
	// TODO: Add logger to config or use a global logger
	logger := logrus.New()
	golfHandler := handlers.NewGolfHandler(db, cache, logger, cfg)

	// Log that we're setting up routes
	fmt.Println("DEBUG: Setting up API routes...")

	// Public routes - no leading slash
	// Contest endpoints
	group.GET("/contests", contestHandler.ListContests)
	group.GET("/contests/:id", contestHandler.GetContest)
	group.GET("/contests/:id/players", playerHandler.GetPlayers)
	group.POST("/contests/:id/fetch-data", contestHandler.FetchContestData)
	group.GET("/contests/:id/data-status", contestHandler.GetContestDataStatus)
	group.POST("/contests/:id/sync", contestHandler.SyncContest)
	group.POST("/contests/discover", contestHandler.DiscoverContests)
	group.GET("/contests/discovery/status", contestHandler.GetContestDiscoveryStatus)

	// Player endpoints
	group.GET("/players/:id", playerHandler.GetPlayer)

	// Glossary endpoints
	group.GET("/glossary", glossaryHandler.GetGlossaryTerms)
	group.GET("/glossary/search", glossaryHandler.SearchGlossaryTerms)
	group.GET("/glossary/:term", glossaryHandler.GetGlossaryTerm)

	// Lineup endpoints (temporarily public for development)
	// TODO: Re-enable authentication in production
	group.GET("/lineups", lineupHandler.GetLineups)
	group.GET("/lineups/:id", lineupHandler.GetLineup)
	group.POST("/lineups", lineupHandler.CreateLineup)
	group.PUT("/lineups/:id", lineupHandler.UpdateLineup)
	group.DELETE("/lineups/:id", lineupHandler.DeleteLineup)
	group.POST("/lineups/:id/submit", lineupHandler.SubmitLineup)

	// Optimization endpoints (temporarily public for development)
	// TODO: Re-enable authentication in production
	group.POST("/optimize", optimizerHandler.OptimizeLineups)
	group.POST("/optimize/validate", optimizerHandler.ValidateLineup)

	// User preferences endpoints (with optional authentication)
	prefGroup := group.Group("/user/preferences")
	prefGroup.Use(middleware.OptionalAuth(cfg.JWTSecret))
	{
		prefGroup.GET("", preferencesHandler.GetPreferences)
		prefGroup.PUT("", preferencesHandler.UpdatePreferences)
		prefGroup.POST("/reset", preferencesHandler.ResetPreferences)
	}

	// Authenticated preference migration endpoint
	authPrefGroup := group.Group("/user/preferences")
	authPrefGroup.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		authPrefGroup.POST("/migrate", preferencesHandler.MigratePreferences)
	}

	// AI recommendation endpoints (temporarily public for development)
	// TODO: Re-enable authentication in production
	fmt.Println("DEBUG: About to register AI routes...")
	aiRecommendationHandler.RegisterRoutes(group)
	fmt.Println("DEBUG: AI routes registration complete")

	// Golf endpoints
	group.GET("/golf/tournaments", golfHandler.ListTournaments)
	group.GET("/golf/tournaments/schedule", golfHandler.GetTournamentSchedule)
	group.GET("/golf/tournaments/:id", golfHandler.GetTournament)
	group.GET("/golf/tournaments/:id/leaderboard", golfHandler.GetTournamentLeaderboard)
	group.GET("/golf/tournaments/:id/players", golfHandler.GetTournamentPlayers)
	group.GET("/golf/tournaments/:id/projections", golfHandler.GetGolfProjections)
	group.GET("/golf/players/:id/history", golfHandler.GetPlayerHistory)
	// Admin endpoint for syncing data (should be protected in production)
	group.POST("/golf/tournaments/:id/sync", golfHandler.SyncTournamentData)

	// Authenticated routes
	auth := group.Group("")
	auth.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		auth.GET("/optimize/constraints/:contestId", optimizerHandler.GetConstraints)

		// Simulation endpoints
		auth.POST("/simulate", simulationHandler.RunSimulation)
		auth.GET("/simulations/:lineupId", simulationHandler.GetSimulationResult)
		auth.POST("/simulate/batch", simulationHandler.BatchSimulate)

		// Export endpoints
		auth.POST("/export", exportHandler.ExportLineups)
		auth.GET("/export/formats", exportHandler.GetExportFormats)
	}

	// WebSocket endpoint (optional auth) - note: WebSocket at root level, not under /api/v1
	// We'll handle this separately in main.go
}
