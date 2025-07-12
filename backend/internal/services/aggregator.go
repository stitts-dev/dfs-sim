package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
)

// DataAggregator combines data from multiple providers
type DataAggregator struct {
	db                *database.DB
	cache             *CacheService
	logger            *logrus.Logger
	espnClient        *providers.ESPNClient
	sportsDBClient    *providers.TheSportsDBClient
	ballDontLieClient *providers.BallDontLieClient
	golfProvider      dfs.Provider
	// Add DraftKingsProvider
	draftKingsProvider *providers.DraftKingsProvider
	config             *config.Config
}

// NewDataAggregator creates a new data aggregator
func NewDataAggregator(
	db *database.DB,
	cache *CacheService,
	logger *logrus.Logger,
	ballDontLieAPIKey string,
	cfg *config.Config,
) *DataAggregator {
	return &DataAggregator{
		db:                 db,
		cache:              cache,
		logger:             logger,
		espnClient:         providers.NewESPNClient(cache, logger),
		sportsDBClient:     providers.NewTheSportsDBClient(cache, logger),
		ballDontLieClient:  providers.NewBallDontLieClient(ballDontLieAPIKey, cache, logger),
		golfProvider:       providers.NewESPNGolfClient(cache, logger), // Default to ESPN
		draftKingsProvider: providers.NewDraftKingsProvider(),
		config:             cfg,
	}
}

// SetGolfProvider sets the golf provider (used to inject RapidAPI when available)
func (a *DataAggregator) SetGolfProvider(provider dfs.Provider) {
	a.golfProvider = provider
}

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	Provider string
	Players  []dfs.PlayerData
	Error    error
}

// AggregatePlayersForContest fetches and aggregates player data for a contest
func (a *DataAggregator) AggregatePlayersForContest(contestID uint, sportStr string) ([]dfs.AggregatedPlayer, error) {
	// Check if golf-only mode is enabled and this is not a golf sport
	if a.config.GolfOnlyMode && strings.ToLower(sportStr) != "golf" {
		a.logger.Infof("Golf-only mode enabled, skipping player aggregation for sport: %s", sportStr)
		return []dfs.AggregatedPlayer{}, nil
	}

	// Convert sport string to dfs.Sport
	var sport dfs.Sport
	switch sportStr {
	case "NBA", "nba":
		sport = dfs.SportNBA
	case "NFL", "nfl":
		sport = dfs.SportNFL
	case "MLB", "mlb":
		sport = dfs.SportMLB
	case "GOLF", "golf":
		sport = dfs.SportGolf
	default:
		return nil, fmt.Errorf("unsupported sport: %s", sportStr)
	}
	cacheKey := fmt.Sprintf("aggregated:contest:%d", contestID)

	// Check cache first
	var cachedPlayers []dfs.AggregatedPlayer
	err := a.cache.GetSimple(cacheKey, &cachedPlayers)
	if err == nil {
		return cachedPlayers, nil
	}

	// Fetch from all providers in parallel
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results := a.fetchFromAllProviders(ctx, sport, time.Now().Format("2006-01-02"))

	// Aggregate results
	aggregatedPlayers := a.mergePlayerData(results)

	// Calculate projections
	for i := range aggregatedPlayers {
		a.calculateProjections(&aggregatedPlayers[i])
	}

	// Update database
	err = a.updateDatabasePlayers(aggregatedPlayers, contestID)
	if err != nil {
		a.logger.Errorf("Failed to update database: %v", err)
	}

	// Cache for 1 hour
	if len(aggregatedPlayers) > 0 {
		a.cache.SetSimple(cacheKey, aggregatedPlayers, 1*time.Hour)
	}

	return aggregatedPlayers, nil
}

// fetchFromAllProviders fetches data from all providers in parallel
func (a *DataAggregator) fetchFromAllProviders(ctx context.Context, sport dfs.Sport, date string) []FetchResult {
	var wg sync.WaitGroup
	results := make(chan FetchResult, 6) // Increased for DK on all sports

	// Golf uses dedicated provider
	if sport == dfs.SportGolf {
		wg.Add(1)
		go func() {
			defer wg.Done()
			players, err := a.golfProvider.GetPlayers(sport, date)
			results <- FetchResult{Provider: "golf", Players: players, Error: err}
		}()

		// DraftKings for Golf
		wg.Add(1)
		go func() {
			defer wg.Done()
			players, err := a.draftKingsProvider.GetPlayers(sport, date)
			results <- FetchResult{Provider: "draftkings", Players: players, Error: err}
		}()
	} else {
		// Skip non-golf providers in golf-only mode
		if a.config.GolfOnlyMode {
			a.logger.Info("Golf-only mode enabled, skipping non-golf providers for other sports")
		} else {
			// ESPN for other sports
			wg.Add(1)
			go func() {
				defer wg.Done()
				players, err := a.espnClient.GetPlayers(sport, date)
				results <- FetchResult{Provider: "espn", Players: players, Error: err}
			}()

			// DraftKings for NBA, NFL and LOL
			if sport == dfs.SportNBA || sport == dfs.SportNFL || string(sport) == "lol" {
				wg.Add(1)
				go func() {
					defer wg.Done()
					players, err := a.draftKingsProvider.GetPlayers(sport, date)
					results <- FetchResult{Provider: "draftkings", Players: players, Error: err}
				}()
			}
		}
	}

	// Skip non-golf providers in golf-only mode
	if !a.config.GolfOnlyMode {
		// TheSportsDB (only for NBA/NFL/MLB)
		if sport == dfs.SportNBA || sport == dfs.SportNFL || sport == dfs.SportMLB {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// TheSportsDB doesn't support date-based queries, so we'll skip for now
				results <- FetchResult{Provider: "thesportsdb", Players: []dfs.PlayerData{}, Error: nil}
			}()
		}

		// BALLDONTLIE (only for NBA)
		if sport == dfs.SportNBA {
			wg.Add(1)
			go func() {
				defer wg.Done()
				players, err := a.ballDontLieClient.GetPlayers(sport, date)
				results <- FetchResult{Provider: "balldontlie", Players: players, Error: err}
			}()
		}
	}

	// Close results channel when all fetches complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []FetchResult
	for result := range results {
		if result.Error != nil {
			a.logger.Warnf("Provider %s failed: %v", result.Provider, result.Error)
		}
		allResults = append(allResults, result)
	}

	return allResults
}

// mergePlayerData merges player data from multiple providers
func (a *DataAggregator) mergePlayerData(results []FetchResult) []dfs.AggregatedPlayer {
	// Group players by name and team
	playerMap := make(map[string]*dfs.AggregatedPlayer)

	for _, result := range results {
		if result.Error != nil {
			continue
		}

		for _, player := range result.Players {
			key := a.generatePlayerKey(player.Name, player.Team)

			if existing, exists := playerMap[key]; exists {
				// Merge data
				a.mergeIntoExisting(existing, player, result.Provider)
			} else {
				// Create new aggregated player
				playerMap[key] = a.createAggregatedPlayer(player, result.Provider)
			}
		}
	}

	// Convert map to slice
	aggregated := make([]dfs.AggregatedPlayer, 0, len(playerMap))
	for _, player := range playerMap {
		aggregated = append(aggregated, *player)
	}

	return aggregated
}

// generatePlayerKey creates a unique key for player matching
func (a *DataAggregator) generatePlayerKey(name, team string) string {
	// Normalize name and team for matching
	name = strings.ToLower(strings.TrimSpace(name))
	team = strings.ToLower(strings.TrimSpace(team))
	return fmt.Sprintf("%s:%s", name, team)
}

// createAggregatedPlayer creates a new aggregated player from provider data
func (a *DataAggregator) createAggregatedPlayer(player dfs.PlayerData, provider string) *dfs.AggregatedPlayer {
	aggPlayer := &dfs.AggregatedPlayer{
		PlayerID:    fmt.Sprintf("%s_%s", provider, player.ExternalID),
		Name:        player.Name,
		Team:        player.Team,
		Position:    player.Position,
		Stats:       make(map[string]float64),
		LastUpdated: time.Now(),
	}

	switch provider {
	case "espn":
		aggPlayer.ESPNData = &player
	case "thesportsdb":
		aggPlayer.TheSportsDBData = &player
	case "balldontlie":
		aggPlayer.BallDontLieData = &player
	case "draftkings":
		aggPlayer.DraftKingsData = &player
	}

	// Copy stats
	for k, v := range player.Stats {
		aggPlayer.Stats[k] = v
	}

	return aggPlayer
}

// mergeIntoExisting merges new player data into existing aggregated player
func (a *DataAggregator) mergeIntoExisting(existing *dfs.AggregatedPlayer, player dfs.PlayerData, provider string) {
	switch provider {
	case "espn":
		existing.ESPNData = &player
	case "thesportsdb":
		existing.TheSportsDBData = &player
		// Update image if available
		if player.ImageURL != "" && existing.TheSportsDBData != nil {
			existing.TheSportsDBData.ImageURL = player.ImageURL
		}
	case "balldontlie":
		existing.BallDontLieData = &player
	case "draftkings":
		existing.DraftKingsData = &player
	}

	// Merge stats (prefer newer data)
	for k, v := range player.Stats {
		existing.Stats[k] = v
	}

	existing.LastUpdated = time.Now()
}

// calculateProjections calculates fantasy projections based on available data
func (a *DataAggregator) calculateProjections(player *dfs.AggregatedPlayer) {
	// Calculate confidence based on data availability
	dataPoints := 0
	if player.ESPNData != nil {
		dataPoints++
	}
	if player.TheSportsDBData != nil {
		dataPoints++
	}
	if player.BallDontLieData != nil {
		dataPoints++
	}
	if player.DraftKingsData != nil {
		dataPoints++
	}

	player.Confidence = float64(dataPoints) / 4.0

	// Calculate projected points based on available stats
	// This is a simplified calculation - in production, you'd use more sophisticated algorithms
	projectedPoints := 0.0

	// NBA scoring (DraftKings format)
	if player.Stats["pts"] > 0 {
		projectedPoints += player.Stats["pts"] * 1.0
		projectedPoints += player.Stats["reb"] * 1.25
		projectedPoints += player.Stats["ast"] * 1.5
		projectedPoints += player.Stats["stl"] * 2.0
		projectedPoints += player.Stats["blk"] * 2.0
		projectedPoints += player.Stats["turnover"] * -0.5

		// Bonus for double-double or triple-double
		doubles := 0
		if player.Stats["pts"] >= 10 {
			doubles++
		}
		if player.Stats["reb"] >= 10 {
			doubles++
		}
		if player.Stats["ast"] >= 10 {
			doubles++
		}
		if player.Stats["stl"] >= 10 {
			doubles++
		}
		if player.Stats["blk"] >= 10 {
			doubles++
		}

		if doubles >= 2 {
			projectedPoints += 1.5
		}
		if doubles >= 3 {
			projectedPoints += 3.0
		}
	}

	player.ProjectedPoints = projectedPoints
}

// updateDatabasePlayers updates or creates players in the database
func (a *DataAggregator) updateDatabasePlayers(players []dfs.AggregatedPlayer, contestID uint) error {
	// Start a transaction for atomic player updates
	tx := a.db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, aggPlayer := range players {
		// Get external ID for lookup - prefer DraftKings, then ESPN, then other sources
		externalID := ""
		if aggPlayer.DraftKingsData != nil && aggPlayer.DraftKingsData.ExternalID != "" {
			externalID = aggPlayer.DraftKingsData.ExternalID
		} else if aggPlayer.ESPNData != nil && aggPlayer.ESPNData.ExternalID != "" {
			externalID = aggPlayer.ESPNData.ExternalID
		} else if aggPlayer.BallDontLieData != nil && aggPlayer.BallDontLieData.ExternalID != "" {
			externalID = aggPlayer.BallDontLieData.ExternalID
		} else if aggPlayer.TheSportsDBData != nil && aggPlayer.TheSportsDBData.ExternalID != "" {
			externalID = aggPlayer.TheSportsDBData.ExternalID
		}

		// If no external ID, use name as fallback (but this should be avoided)
		if externalID == "" {
			externalID = aggPlayer.Name
		}

		// Check if player exists using external_id + contest_id (matches the unique constraint)
		var dbPlayer models.Player
		err := tx.Where("external_id = ? AND contest_id = ?", externalID, contestID).First(&dbPlayer).Error

		if err != nil {
			// Create new player
			salary := a.getSalaryFromProviders(&aggPlayer)
			if salary == 0 {
				salary = a.estimateSalary(aggPlayer.ProjectedPoints)
			}

			dbPlayer = models.Player{
				ExternalID:      externalID,
				Name:            aggPlayer.Name,
				Team:            aggPlayer.Team,
				Position:        aggPlayer.Position,
				Salary:          int(salary),
				ProjectedPoints: aggPlayer.ProjectedPoints,
				Ownership:       0.0, // Will be updated by another service
				ContestID:       contestID,
			}

			err = tx.Create(&dbPlayer).Error
			if err != nil {
				// Check if it's a duplicate key constraint error
				if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
					a.logger.Warnf("Player %s (external_id: %s) already exists in contest %d, skipping creation", aggPlayer.Name, externalID, contestID)
					continue
				}
				a.logger.Errorf("Failed to create player %s (external_id: %s): %v", aggPlayer.Name, externalID, err)
				tx.Rollback()
				return fmt.Errorf("failed to create player %s: %w", aggPlayer.Name, err)
			}
		} else {
			// Update existing player
			dbPlayer.ProjectedPoints = aggPlayer.ProjectedPoints

			// Update salary if we have better data
			salary := a.getSalaryFromProviders(&aggPlayer)
			if salary > 0 {
				dbPlayer.Salary = int(salary)
			}

			// Update external ID if we have better data (but don't overwrite existing)
			if externalID != "" && dbPlayer.ExternalID == "" {
				dbPlayer.ExternalID = externalID
			}

			err = tx.Save(&dbPlayer).Error
			if err != nil {
				a.logger.Errorf("Failed to update player %s (external_id: %s): %v", aggPlayer.Name, externalID, err)
				tx.Rollback()
				return fmt.Errorf("failed to update player %s: %w", aggPlayer.Name, err)
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		a.logger.Errorf("Failed to commit player updates transaction: %v", err)
		return fmt.Errorf("failed to commit player updates: %w", err)
	}

	return nil
}

// getSalaryFromProviders gets salary from provider data, preferring DraftKings
func (a *DataAggregator) getSalaryFromProviders(player *dfs.AggregatedPlayer) float64 {
	// First check DraftKings data for actual salary
	if player.DraftKingsData != nil {
		if salary, exists := player.DraftKingsData.Stats["salary"]; exists && salary > 0 {
			return salary
		}
	}

	// Fall back to other providers if they have salary data
	if player.ESPNData != nil {
		if salary, exists := player.ESPNData.Stats["salary"]; exists && salary > 0 {
			return salary
		}
	}

	if player.TheSportsDBData != nil {
		if salary, exists := player.TheSportsDBData.Stats["salary"]; exists && salary > 0 {
			return salary
		}
	}

	if player.BallDontLieData != nil {
		if salary, exists := player.BallDontLieData.Stats["salary"]; exists && salary > 0 {
			return salary
		}
	}

	return 0 // No salary data found
}

// estimateSalary estimates salary based on projected points
func (a *DataAggregator) estimateSalary(projectedPoints float64) float64 {
	// More realistic DraftKings-style salary estimation
	// Based on analysis of actual DK salaries

	// Minimum and maximum salaries
	minSalary := 3000.0
	maxSalary := 12000.0

	// Different tiers based on projected points
	var salary float64

	switch {
	case projectedPoints >= 50: // Elite tier (50+ points)
		// Elite players: $9,000 - $12,000
		salary = 9000 + (projectedPoints-50)*150

	case projectedPoints >= 40: // Star tier (40-50 points)
		// Star players: $7,000 - $9,000
		salary = 7000 + (projectedPoints-40)*200

	case projectedPoints >= 30: // Above average (30-40 points)
		// Good players: $5,500 - $7,000
		salary = 5500 + (projectedPoints-30)*150

	case projectedPoints >= 20: // Average (20-30 points)
		// Role players: $4,000 - $5,500
		salary = 4000 + (projectedPoints-20)*150

	case projectedPoints >= 10: // Below average (10-20 points)
		// Bench players: $3,000 - $4,000
		salary = 3000 + (projectedPoints-10)*100

	default: // Minimal playing time
		salary = minSalary
	}

	// Apply small variance (±3%) to create more realistic distribution
	// Use hash of projected points for consistent variance per player
	hash := int(projectedPoints * 1000)
	variance := (salary * 0.03) * (2.0*float64(hash%100)/100.0 - 1.0)
	salary += variance

	// Ensure within bounds
	if salary < minSalary {
		salary = minSalary
	} else if salary > maxSalary {
		salary = maxSalary
	}

	// Round to nearest 100
	return float64(int(salary/100) * 100)
}

// EnrichPlayerWithImages enriches player data with images from TheSportsDB
func (a *DataAggregator) EnrichPlayerWithImages(players []models.Player) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent requests

	for i := range players {
		wg.Add(1)
		go func(player *models.Player) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Search for player image
			playerData, err := a.sportsDBClient.GetPlayer(dfs.SportNBA, player.Name)
			if err == nil && playerData != nil && playerData.ImageURL != "" {
				player.ImageURL = playerData.ImageURL
				// Update in database
				a.db.DB.Model(player).Update("image_url", playerData.ImageURL)
			}
		}(&players[i])
	}

	wg.Wait()
}
