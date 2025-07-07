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
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
)

// DataAggregator combines data from multiple providers
type DataAggregator struct {
	db              *database.DB
	cache           *CacheService
	logger          *logrus.Logger
	espnClient      *providers.ESPNClient
	sportsDBClient  *providers.TheSportsDBClient
	ballDontLieClient *providers.BallDontLieClient
}

// NewDataAggregator creates a new data aggregator
func NewDataAggregator(
	db *database.DB,
	cache *CacheService,
	logger *logrus.Logger,
	ballDontLieAPIKey string,
) *DataAggregator {
	return &DataAggregator{
		db:              db,
		cache:           cache,
		logger:          logger,
		espnClient:      providers.NewESPNClient(cache, logger),
		sportsDBClient:  providers.NewTheSportsDBClient(cache, logger),
		ballDontLieClient: providers.NewBallDontLieClient(ballDontLieAPIKey, cache, logger),
	}
}

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	Provider string
	Players  []dfs.PlayerData
	Error    error
}

// AggregatePlayersForContest fetches and aggregates player data for a contest
func (a *DataAggregator) AggregatePlayersForContest(contestID uint, sportStr string) ([]dfs.AggregatedPlayer, error) {
	// Convert sport string to dfs.Sport
	var sport dfs.Sport
	switch sportStr {
	case "NBA", "nba":
		sport = dfs.SportNBA
	case "NFL", "nfl":
		sport = dfs.SportNFL
	case "MLB", "mlb":
		sport = dfs.SportMLB
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
	results := make(chan FetchResult, 3)
	
	// ESPN
	wg.Add(1)
	go func() {
		defer wg.Done()
		players, err := a.espnClient.GetPlayers(sport, date)
		results <- FetchResult{Provider: "espn", Players: players, Error: err}
	}()
	
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
	
	player.Confidence = float64(dataPoints) / 3.0
	
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
	for _, aggPlayer := range players {
		// Check if player exists
		var dbPlayer models.Player
		err := a.db.DB.Where("name = ? AND team = ?", aggPlayer.Name, aggPlayer.Team).First(&dbPlayer).Error
		
		if err != nil {
			// Create new player
			dbPlayer = models.Player{
				Name:            aggPlayer.Name,
				Team:            aggPlayer.Team,
				Position:        aggPlayer.Position,
				Salary:          int(a.estimateSalary(aggPlayer.ProjectedPoints)),
				ProjectedPoints: aggPlayer.ProjectedPoints,
				Ownership:       0.0, // Will be updated by another service
				ContestID:       contestID,
			}
			
			// Set external IDs
			if aggPlayer.ESPNData != nil {
				dbPlayer.ExternalID = aggPlayer.ESPNData.ExternalID
			}
			
			err = a.db.DB.Create(&dbPlayer).Error
			if err != nil {
				a.logger.Errorf("Failed to create player %s: %v", aggPlayer.Name, err)
				continue
			}
		} else {
			// Update existing player
			dbPlayer.ProjectedPoints = aggPlayer.ProjectedPoints
			if aggPlayer.ESPNData != nil && dbPlayer.ExternalID == "" {
				dbPlayer.ExternalID = aggPlayer.ESPNData.ExternalID
			}
			
			err = a.db.DB.Save(&dbPlayer).Error
			if err != nil {
				a.logger.Errorf("Failed to update player %s: %v", aggPlayer.Name, err)
			}
		}
	}
	
	return nil
}

// estimateSalary estimates salary based on projected points
func (a *DataAggregator) estimateSalary(projectedPoints float64) float64 {
	// Simple linear estimation
	// In production, you'd use actual DFS site salaries
	baseSalary := 3000.0
	pointValue := 100.0
	
	salary := baseSalary + (projectedPoints * pointValue)
	
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