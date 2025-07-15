package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/services"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/utils"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// GolfHandler handles golf-specific API endpoints
type GolfHandler struct {
	db                *database.DB
	cache             *services.CacheService
	logger            *logrus.Logger
	rapidAPIProvider  *providers.RapidAPIGolfClient
	espnProvider      *providers.ESPNGolfClient
	projectionService *services.GolfProjectionService
	syncService       *services.GolfTournamentSyncService
	golfSportID       uuid.UUID
}

// NewGolfHandler creates a new golf handler for the golf microservice
func NewGolfHandler(
	db *database.DB,
	cache *services.CacheService,
	projectionService *services.GolfProjectionService,
	syncService *services.GolfTournamentSyncService,
	rapidAPIProvider *providers.RapidAPIGolfClient,
	espnProvider *providers.ESPNGolfClient,
	logger *logrus.Logger,
) *GolfHandler {
	// Get golf sport ID once during initialization
	var golfSportID uuid.UUID
	var sport struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	err := db.Table("sports").Select("id").Where("name = ?", "Golf").First(&sport).Error
	if err != nil {
		logger.WithError(err).Fatal("Failed to get golf sport ID - ensure 'Golf' sport exists in database")
	}
	golfSportID = sport.ID

	return &GolfHandler{
		db:                db,
		cache:             cache,
		logger:            logger,
		rapidAPIProvider:  rapidAPIProvider,
		espnProvider:      espnProvider,
		projectionService: projectionService,
		syncService:       syncService,
		golfSportID:       golfSportID,
	}
}

// GolfProvider defines the interface for golf data providers
type GolfProvider interface {
	GetCurrentTournament() (*providers.GolfTournamentData, error)
	GetTournamentSchedule() ([]providers.GolfTournamentData, error)
	GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error)
}

// getGolfProvider returns the best available golf provider with fallback logic
func (h *GolfHandler) getGolfProvider() GolfProvider {
	if h.rapidAPIProvider != nil {
		return h.rapidAPIProvider
	}
	return h.espnProvider
}

// ListTournaments returns available golf tournaments
func (h *GolfHandler) ListTournaments(c *gin.Context) {
	// Query parameters
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Validate limit
	if limit > 100 {
		limit = 100
	}

	// Build query
	query := h.db.Model(&models.GolfTournament{})

	// Apply filters
	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		// Default to active and upcoming tournaments
		query = query.Where("status IN ?", []string{"scheduled", "in_progress"})
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply pagination and sorting
	query = query.Offset(offset).Limit(limit).Order("start_date ASC")

	var tournaments []models.GolfTournament
	if err := query.Find(&tournaments).Error; err != nil {
		h.logger.Error("Failed to fetch tournaments", "error", err)
		utils.SendInternalError(c, "Failed to fetch tournaments")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tournaments": tournaments,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetTournament returns a specific tournament with details
func (h *GolfHandler) GetTournament(c *gin.Context) {
	tournamentID := c.Param("id")

	var tournament models.GolfTournament

	// Try to parse as UUID first, then as external ID
	if _, err := uuid.Parse(tournamentID); err == nil {
		// It's a UUID
		if err := h.db.Preload("PlayerEntries.Player").First(&tournament, "id = ?", tournamentID).Error; err != nil {
			utils.SendNotFound(c, "Tournament not found")
			return
		}
	} else {
		// Try as external ID
		if err := h.db.Preload("PlayerEntries.Player").First(&tournament, "external_id = ?", tournamentID).Error; err != nil {
			utils.SendNotFound(c, "Tournament not found")
			return
		}
	}

	c.JSON(http.StatusOK, tournament)
}

// GetTournamentLeaderboard returns the current leaderboard
func (h *GolfHandler) GetTournamentLeaderboard(c *gin.Context) {
	tournamentID := c.Param("id")

	// Check cache first
	cacheKey := "golf:leaderboard:" + tournamentID
	var cachedData interface{}
	if err := h.cache.GetSimple(cacheKey, &cachedData); err == nil {
		c.JSON(http.StatusOK, cachedData)
		return
	}

	// Get tournament
	var tournament models.GolfTournament
	if err := h.db.First(&tournament, "id = ? OR external_id = ?", tournamentID, tournamentID).Error; err != nil {
		utils.SendNotFound(c, "Tournament not found")
		return
	}

	// Get player entries sorted by position
	var entries []models.GolfPlayerEntry
	query := h.db.Where("tournament_id = ?", tournament.ID).
		Preload("Player").
		Order("current_position ASC, total_score ASC")

	if err := query.Find(&entries).Error; err != nil {
		h.logger.Error("Failed to fetch leaderboard", "error", err)
		utils.SendInternalError(c, "Failed to fetch leaderboard")
		return
	}

	// Build leaderboard response
	leaderboard := gin.H{
		"tournament": tournament,
		"entries":    entries,
		"cut_line":   tournament.CutLine,
		"updated_at": time.Now(),
	}

	// Cache for 1 minute if tournament is live, 1 hour otherwise
	cacheDuration := 1 * time.Hour
	if tournament.Status == models.TournamentInProgress {
		cacheDuration = 1 * time.Minute
	}
	h.cache.SetSimple(cacheKey, leaderboard, cacheDuration)

	c.JSON(http.StatusOK, leaderboard)
}

// GetTournamentPlayers returns players for a tournament with projections
func (h *GolfHandler) GetTournamentPlayers(c *gin.Context) {
	
	tournamentID := c.Param("id")
	platform := c.DefaultQuery("platform", "draftkings")

	// Get tournament
	var tournament models.GolfTournament
	if err := h.db.First(&tournament, "id = ? OR external_id = ?", tournamentID, tournamentID).Error; err != nil {
		utils.SendNotFound(c, "Tournament not found")
		return
	}

	// Get player entries
	var entries []models.GolfPlayerEntry
	if err := h.db.Where("tournament_id = ?", tournament.ID).
		Preload("Player").
		Find(&entries).Error; err != nil {
		h.logger.Error("Failed to fetch player entries", "error", err)
		utils.SendInternalError(c, "Failed to fetch players")
		return
	}

	// Generate projections for all players using PlayerInterface
	playerInterfaces := make([]types.PlayerInterface, 0, len(entries))
	for _, entry := range entries {
		if entry.Player != nil {
			playerInterfaces = append(playerInterfaces, entry.Player)
		}
	}

	projections := make(map[uuid.UUID]*models.GolfProjection)
	if len(playerInterfaces) > 0 {
		projectionMap, _, err := h.projectionService.GenerateProjections(
			c.Request.Context(),
			playerInterfaces,
			tournament.ID.String(),
		)
		if err != nil {
			h.logger.Warn("Failed to generate projections", "error", err)
		} else {
			projections = projectionMap
		}
	}

	// Convert to player format with golf-specific data and projections
	players := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		if entry.Player == nil {
			continue
		}

		player := gin.H{
			"id":               entry.Player.GetID(),
			"name":             entry.Player.GetName(),
			"external_id":      entry.Player.GetExternalID(),
			"position":         "G",                        // Golfer
			"team":             entry.Player.GetTeam(),     // Country in golf
			"status":           entry.Status,
			"current_position": entry.CurrentPosition,
			"total_score":      entry.TotalScore,
			"thru_holes":       entry.ThruHoles,
			"rounds_scores":    entry.RoundsScores,
		}

		// Add platform-specific salary and ownership
		if platform == "draftkings" {
			player["salary"] = entry.DKSalary
			player["ownership"] = entry.DKOwnership
		} else {
			player["salary"] = entry.FDSalary
			player["ownership"] = entry.FDOwnership
		}

		// Add projection data if available
		if projection, exists := projections[entry.Player.GetID()]; exists {
			player["cut_probability"] = projection.CutProbability
			player["top10_probability"] = projection.Top10Probability
			player["top25_probability"] = projection.Top25Probability
			player["win_probability"] = projection.WinProbability
			player["expected_score"] = projection.ExpectedScore
			player["confidence"] = projection.Confidence

			// Add platform-specific projected points
			var projectedPoints float64
			if platform == "draftkings" {
				projectedPoints = projection.DKPoints
			} else {
				projectedPoints = projection.FDPoints
			}
			
			player["projected_points"] = projectedPoints
			
			// Calculate floor and ceiling based on confidence and probabilities
			confidenceFactor := projection.Confidence
			cutRisk := 1.0 - projection.CutProbability
			
			// Floor: Conservative estimate considering cut risk
			floorMultiplier := 0.6 + (confidenceFactor * 0.2) - (cutRisk * 0.4)
			if floorMultiplier < 0.1 {
				floorMultiplier = 0.1
			}
			player["floor_points"] = projectedPoints * floorMultiplier
			
			// Ceiling: Optimistic estimate based on upside potential
			ceilingMultiplier := 1.2 + (projection.Top10Probability * 0.8) + (projection.WinProbability * 1.0)
			player["ceiling_points"] = projectedPoints * ceilingMultiplier
		} else {
			// Provide reasonable defaults if no projections available
			var salary float64
			if platform == "draftkings" {
				salary = float64(entry.DKSalary)
			} else {
				salary = float64(entry.FDSalary)
			}
			
			// Default projections based on salary tier
			basePoints := salary / 200 // Simple salary-to-points conversion
			player["projected_points"] = basePoints
			player["floor_points"] = basePoints * 0.5
			player["ceiling_points"] = basePoints * 1.8
			player["cut_probability"] = 0.5
		}

		players = append(players, player)
	}

	// Add tournament context information
	tournamentInfo := gin.H{
		"id":           tournament.ID,
		"name":         tournament.Name,
		"course_name":  tournament.CourseName,
		"course_par":   tournament.CoursePar,
		"purse":        tournament.Purse,
		"status":       tournament.Status,
		"cut_line":     tournament.CutLine,
		"current_round": tournament.CurrentRound,
		"field_size":   len(players),
	}

	c.JSON(http.StatusOK, gin.H{
		"tournament": tournamentInfo,
		"players":    players,
		"platform":   platform,
		"stats": gin.H{
			"total_players": len(players),
			"avg_salary":    calculateAverageSalary(entries, platform),
			"projection_coverage": fmt.Sprintf("%.1f%%", float64(len(projections))/float64(len(players))*100),
		},
	})
}

// SyncTournamentData syncs tournament data from external provider
func (h *GolfHandler) SyncTournamentData(c *gin.Context) {
	// Sync all active tournaments using the existing sync service
	if err := h.syncService.SyncAllActiveTournaments(); err != nil {
		h.logger.Error("Failed to sync golf tournaments", "error", err)
		utils.SendInternalError(c, "Failed to sync tournament data")
		return
	}

	// Also sync tournament details for display using best available provider
	provider := h.getGolfProvider()
	tournamentData, err := provider.GetCurrentTournament()
	if err != nil {
		h.logger.Error("Failed to fetch tournament from provider", "error", err)
		utils.SendInternalError(c, "Failed to get current tournament")
		return
	}

	// Update tournament in database
	tournament := &models.GolfTournament{
		ExternalID:   tournamentData.ID,
		Name:         tournamentData.Name,
		StartDate:    tournamentData.StartDate,
		EndDate:      tournamentData.EndDate,
		Status:       models.TournamentStatus(tournamentData.Status),
		CurrentRound: tournamentData.CurrentRound,
		CourseID:     tournamentData.CourseID,
		CourseName:   tournamentData.CourseName,
		CoursePar:    tournamentData.CoursePar,
		CourseYards:  tournamentData.CourseYards,
		Purse:        tournamentData.Purse,
		CutLine:      tournamentData.CutLine,
	}

	// Upsert tournament
	if err := h.db.Where("external_id = ?", tournament.ExternalID).
		Assign(tournament).
		FirstOrCreate(&tournament).Error; err != nil {
		h.logger.Error("Failed to update tournament", "error", err)
		utils.SendInternalError(c, "Failed to update tournament")
		return
	}

	// Sync player data using best available provider
	players, err := provider.GetPlayers(types.SportGolf, time.Now().Format("2006-01-02"))
	if err != nil {
		h.logger.Error("Failed to fetch players", "error", err)
		utils.SendInternalError(c, "Failed to sync player data")
		return
	}

	// Update player entries
	for _, playerData := range players {
		// Find or create player
		var player models.Player
		if err := h.db.Where("external_id = ?", playerData.ExternalID).
			FirstOrCreate(&player, models.Player{
				ExternalID:      playerData.ExternalID,
				Name:            playerData.Name,
				Team:            playerData.Team,
				Position:        "G",
				SportID:         h.golfSportID,
				ProjectedPoints: playerData.ProjectedPoints,
				FloorPoints:     playerData.FloorPoints,
				CeilingPoints:   playerData.CeilingPoints,
				GameTime:        playerData.GameTime,
			}).Error; err != nil {
			h.logger.Warn("Failed to create player", "error", err, "player", playerData.Name)
			continue
		}

		// Create or update player entry
		entry := models.GolfPlayerEntry{
			PlayerID:     player.ID,
			TournamentID: tournament.ID,
		}

		// Update entry data from stats with safe type assertions
		if playerData.Stats != nil {
			if statsMap, ok := playerData.Stats.(map[string]interface{}); ok {
				// Safely extract score
				if score, exists := statsMap["score"]; exists && score != nil {
					switch v := score.(type) {
					case float64:
						entry.TotalScore = int(v)
					case int:
						entry.TotalScore = v
					case string:
						// Try to parse string as number if needed
						h.logger.Warn("Score provided as string, expected numeric", "score", v, "player", playerData.Name)
					default:
						h.logger.Warn("Unexpected score type", "type", fmt.Sprintf("%T", score), "player", playerData.Name)
					}
				}
				
				// Safely extract position
				if pos, exists := statsMap["position"]; exists && pos != nil {
					switch v := pos.(type) {
					case float64:
						entry.CurrentPosition = int(v)
					case int:
						entry.CurrentPosition = v
					case string:
						// Try to parse string as number if needed
						h.logger.Warn("Position provided as string, expected numeric", "position", v, "player", playerData.Name)
					default:
						h.logger.Warn("Unexpected position type", "type", fmt.Sprintf("%T", pos), "player", playerData.Name)
					}
				}
			} else {
				h.logger.Warn("Player stats is not a map[string]interface{}", "type", fmt.Sprintf("%T", playerData.Stats), "player", playerData.Name)
			}
		}

		if err := h.db.Where("player_id = ? AND tournament_id = ?", player.ID, tournament.ID).
			Assign(entry).
			FirstOrCreate(&entry).Error; err != nil {
			h.logger.Warn("Failed to update player entry", "error", err)
		}
	}

	// Get updated contest count
	var contestCount int64
	h.db.Model(&types.Contest{}).Where("sport_id = ? AND is_active = ?", h.golfSportID, true).Count(&contestCount)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Tournament data synced successfully",
		"tournament":       tournament,
		"player_count":     len(players),
		"contests_created": contestCount,
	})
}

// GetPlayerHistory returns a player's course history
func (h *GolfHandler) GetPlayerHistory(c *gin.Context) {
	playerID := c.Param("id")
	courseID := c.Query("course_id")

	var histories []models.GolfCourseHistory
	query := h.db.Where("player_id = ?", playerID)

	if courseID != "" {
		query = query.Where("course_id = ?", courseID)
	}

	if err := query.Find(&histories).Error; err != nil {
		h.logger.Error("Failed to fetch player history", "error", err)
		utils.SendInternalError(c, "Failed to fetch player history")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player_id": playerID,
		"histories": histories,
	})
}

// GetGolfProjections returns projections for a tournament
func (h *GolfHandler) GetGolfProjections(c *gin.Context) {
	
	tournamentID := c.Param("id")

	// Get tournament
	var tournament models.GolfTournament
	if err := h.db.First(&tournament, "id = ? OR external_id = ?", tournamentID, tournamentID).Error; err != nil {
		utils.SendNotFound(c, "Tournament not found")
		return
	}

	// Get all players in tournament
	var entries []models.GolfPlayerEntry
	if err := h.db.Where("tournament_id = ?", tournament.ID).
		Preload("Player").
		Find(&entries).Error; err != nil {
		h.logger.Error("Failed to fetch player entries", "error", err)
		utils.SendInternalError(c, "Failed to fetch players")
		return
	}

	// Convert to player interfaces
	playerInterfaces := make([]types.PlayerInterface, 0, len(entries))
	for _, entry := range entries {
		if entry.Player != nil {
			playerInterfaces = append(playerInterfaces, entry.Player)
		}
	}

	// Generate projections
	projections, correlations, err := h.projectionService.GenerateProjections(
		c.Request.Context(),
		playerInterfaces,
		tournament.ID.String(),
	)
	if err != nil {
		h.logger.Error("Failed to generate projections", "error", err)
		utils.SendInternalError(c, "Failed to generate projections")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tournament":   tournament,
		"projections":  projections,
		"correlations": correlations,
	})
}

// GetTournamentSchedule returns the upcoming tournament schedule
func (h *GolfHandler) GetTournamentSchedule(c *gin.Context) {
	// Get year parameter, default to current year
	yearStr := c.DefaultQuery("year", strconv.Itoa(time.Now().Year()))
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		utils.SendValidationError(c, "Invalid year parameter: "+err.Error())
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("golf:schedule:%d", year)
	var cachedSchedule interface{}
	if err := h.cache.GetSimple(cacheKey, &cachedSchedule); err == nil {
		c.JSON(http.StatusOK, cachedSchedule)
		return
	}

	// Get schedule from provider
	provider := h.getGolfProvider()
	schedule, err := provider.GetTournamentSchedule()
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch tournament schedule")
		// Try to get from database as fallback
		var tournaments []models.GolfTournament
		query := h.db.Where("EXTRACT(YEAR FROM start_date) = ?", year).
			Order("start_date ASC")

		if err := query.Find(&tournaments).Error; err != nil {
			utils.SendInternalError(c, "Failed to fetch tournament schedule")
			return
		}

		// Return database results
		c.JSON(http.StatusOK, gin.H{
			"tournaments": tournaments,
			"source":      "database",
			"cached_at":   time.Now(),
		})
		return
	}

	// Filter to requested year and get next 4 upcoming tournaments
	now := time.Now()
	upcoming := make([]providers.GolfTournamentData, 0)
	allYearTournaments := make([]providers.GolfTournamentData, 0)

	for _, tournament := range schedule {
		if tournament.StartDate.Year() == year {
			allYearTournaments = append(allYearTournaments, tournament)
			if tournament.StartDate.After(now) && len(upcoming) < 4 {
				upcoming = append(upcoming, tournament)
			}
		}
	}

	// If we don't have 4 upcoming, include recent past tournaments
	if len(upcoming) < 4 {
		for _, tournament := range allYearTournaments {
			if tournament.StartDate.Before(now) || tournament.StartDate.Equal(now) {
				upcoming = append(upcoming, tournament)
				if len(upcoming) >= 4 {
					break
				}
			}
		}
	}

	// Build response with cache metadata
	response := gin.H{
		"tournaments": upcoming,
		"total_year":  len(allYearTournaments),
		"year":        year,
		"source":      "api",
		"cached_at":   time.Now(),
		"next_update": time.Now().Add(24 * time.Hour),
	}

	// Cache for 24 hours
	h.cache.SetSimple(cacheKey, response, 24*time.Hour)

	c.JSON(http.StatusOK, response)
}

// GetGolfPlayer returns a specific golf player with details
func (h *GolfHandler) GetGolfPlayer(c *gin.Context) {
	
	playerID := c.Param("id")

	var player models.Player
	if err := h.db.First(&player, "id = ? AND sport_id = ?", playerID, h.golfSportID).Error; err != nil {
		utils.SendNotFound(c, "Golf player not found")
		return
	}

	// Get recent tournament entries for this player
	var entries []models.GolfPlayerEntry
	h.db.Where("player_id = ?", player.GetID()).
		Preload("Tournament").
		Order("tournament.start_date DESC").
		Limit(5).
		Find(&entries)

	// Build response with tournament history
	tournamentHistory := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		if entry.Tournament != nil {
			tournamentHistory = append(tournamentHistory, gin.H{
				"tournament_name": entry.Tournament.Name,
				"finish_position": entry.CurrentPosition,
				"total_score":     entry.TotalScore,
				"status":          entry.Status,
				"date":           entry.Tournament.StartDate,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"player":             player,
		"tournament_history": tournamentHistory,
	})
}

// GetPlayerProjections returns projections for a specific player
func (h *GolfHandler) GetPlayerProjections(c *gin.Context) {
	
	playerID := c.Param("id")
	tournamentID := c.Query("tournament_id")

	var player models.Player
	if err := h.db.First(&player, "id = ? AND sport_id = ?", playerID, h.golfSportID).Error; err != nil {
		utils.SendNotFound(c, "Golf player not found")
		return
	}

	if tournamentID == "" {
		utils.SendValidationError(c, "tournament_id query parameter is required")
		return
	}

	// Generate projections for this player
	projections, _, err := h.projectionService.GenerateProjections(
		c.Request.Context(),
		[]types.PlayerInterface{&player},
		tournamentID,
	)
	if err != nil {
		h.logger.Error("Failed to generate player projections", "error", err)
		utils.SendInternalError(c, "Failed to generate projections")
		return
	}

	projection, exists := projections[player.GetID()]
	if !exists {
		utils.SendNotFound(c, "No projections available for this player")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":     player,
		"projection": projection,
	})
}

// GetPlayerCourseHistory returns a player's history at specific courses
func (h *GolfHandler) GetPlayerCourseHistory(c *gin.Context) {
	
	playerID := c.Param("id")
	courseID := c.Query("course_id")

	var player models.Player
	if err := h.db.First(&player, "id = ? AND sport_id = ?", playerID, h.golfSportID).Error; err != nil {
		utils.SendNotFound(c, "Golf player not found")
		return
	}

	// Get course history
	var histories []models.GolfCourseHistory
	query := h.db.Where("player_id = ?", player.GetID())

	if courseID != "" {
		query = query.Where("course_id = ?", courseID)
	}

	if err := query.Order("created_at DESC").Find(&histories).Error; err != nil {
		h.logger.Error("Failed to fetch player course history", "error", err)
		utils.SendInternalError(c, "Failed to fetch course history")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":   player,
		"course_id": courseID,
		"history":  histories,
	})
}

// SyncCurrentTournament syncs the current tournament data
func (h *GolfHandler) SyncCurrentTournament(c *gin.Context) {
	// Get current tournament from provider
	provider := h.getGolfProvider()
	tournamentData, err := provider.GetCurrentTournament()
	if err != nil {
		h.logger.Error("Failed to fetch current tournament", "error", err)
		utils.SendInternalError(c, "Failed to sync current tournament")
		return
	}

	// Use the sync service to update tournament
	if err := h.syncService.SyncCurrentTournament(); err != nil {
		h.logger.Error("Failed to sync current tournament via service", "error", err)
		utils.SendInternalError(c, "Failed to sync tournament data")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Current tournament synced successfully",
		"tournament": tournamentData,
	})
}

// SyncTournamentSchedule syncs the tournament schedule
func (h *GolfHandler) SyncTournamentSchedule(c *gin.Context) {
	// Get tournament schedule from provider
	provider := h.getGolfProvider()
	schedule, err := provider.GetTournamentSchedule()
	if err != nil {
		h.logger.Error("Failed to fetch tournament schedule", "error", err)
		utils.SendInternalError(c, "Failed to sync tournament schedule")
		return
	}

	// Sync via service
	if err := h.syncService.SyncTournamentSchedule(); err != nil {
		h.logger.Error("Failed to sync tournament schedule via service", "error", err)
		utils.SendInternalError(c, "Failed to sync schedule")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Tournament schedule synced successfully",
		"tournaments": len(schedule),
	})
}

// calculateAverageSalary calculates the average salary for a platform
func calculateAverageSalary(entries []models.GolfPlayerEntry, platform string) float64 {
	if len(entries) == 0 {
		return 0
	}

	totalSalary := 0.0
	count := 0

	for _, entry := range entries {
		var salary int
		if platform == "draftkings" {
			salary = entry.DKSalary
		} else {
			salary = entry.FDSalary
		}

		if salary > 0 {
			totalSalary += float64(salary)
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalSalary / float64(count)
}

// SportInfo represents information about a sport
type SportInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Icon     string   `json:"icon"`
	Enabled  bool     `json:"enabled"`
	Position []string `json:"positions,omitempty"`
}

// SportsResponse represents the response structure for available sports
type SportsResponse struct {
	Sports     []SportInfo `json:"sports"`
	GolfOnly   bool        `json:"golf_only_mode"`
	AllSports  []SportInfo `json:"all_sports"`
}

// GetAvailableSports returns golf sport configuration for the golf service
func (h *GolfHandler) GetAvailableSports(c *gin.Context) {
	// Golf service only supports golf
	golfSport := SportInfo{
		ID:       "golf",
		Name:     "Golf",
		Icon:     "⛳",
		Enabled:  true,
		Position: []string{"G"},
	}

	// Define all possible sports but only enable golf
	allSports := []SportInfo{
		golfSport,
		{
			ID:       "nba",
			Name:     "NBA",
			Icon:     "🏀",
			Enabled:  false,
			Position: []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UTIL"},
		},
		{
			ID:       "nfl",
			Name:     "NFL",
			Icon:     "🏈",
			Enabled:  false,
			Position: []string{"QB", "RB", "WR", "TE", "K", "DST", "FLEX"},
		},
		{
			ID:       "mlb",
			Name:     "MLB",
			Icon:     "⚾",
			Enabled:  false,
			Position: []string{"C", "1B", "2B", "3B", "SS", "OF", "DH", "P"},
		},
		{
			ID:       "nhl",
			Name:     "NHL",
			Icon:     "🏒",
			Enabled:  false,
			Position: []string{"C", "LW", "RW", "D", "G", "F"},
		},
	}

	response := SportsResponse{
		Sports:    []SportInfo{golfSport}, // Only golf is enabled
		GolfOnly:  true,                   // Golf service only mode
		AllSports: allSports,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    response,
		"message": "Available sports retrieved successfully",
		"success": true,
	})
}

// ContestInfo represents a golf contest/tournament
type ContestInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Sport        string    `json:"sport"`
	Platform     string    `json:"platform"`
	ContestType  string    `json:"contest_type"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       string    `json:"status"`
	TournamentID *string   `json:"tournament_id,omitempty"`
}

// ListContests returns available golf contests/tournaments
func (h *GolfHandler) ListContests(c *gin.Context) {
	// Query parameters
	sport := c.DefaultQuery("sport", "golf")
	platform := c.DefaultQuery("platform", "")
	contestType := c.DefaultQuery("contest_type", "")
	active := c.DefaultQuery("active", "true")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "20"))

	// Since this is golf service, enforce sport=golf
	if sport != "golf" && sport != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Golf service only supports golf contests",
			"success": false,
		})
		return
	}

	// Build query for golf tournaments as contests
	query := h.db.Model(&models.GolfTournament{})

	// Apply filters
	if platform != "" {
		// Filter by DFS platform availability in the future
		// For now, we'll accept all platforms
	}
	if contestType != "" {
		// Contest type could map to tournament type in the future
	}
	if active == "true" {
		// Active tournaments are those happening now or in the near future
		now := time.Now()
		query = query.Where("start_date <= ? AND end_date >= ?", now.AddDate(0, 0, 7), now.AddDate(0, 0, -1))
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply pagination and sorting
	offset := (page - 1) * perPage
	query = query.Offset(offset).Limit(perPage).Order("start_date ASC")

	var tournaments []models.GolfTournament
	if err := query.Find(&tournaments).Error; err != nil {
		h.logger.Error("Failed to fetch golf tournaments", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch contests",
			"success": false,
		})
		return
	}

	// Convert tournaments to contest format
	contests := make([]ContestInfo, len(tournaments))
	for i, tournament := range tournaments {
		tournamentID := tournament.ID.String()
		contests[i] = ContestInfo{
			ID:           tournament.ID.String(),
			Name:         tournament.Name,
			Sport:        "golf",
			Platform:     "multi",                     // Golf tournaments are available on multiple platforms
			ContestType:  "gpp",                       // Golf tournaments are typically GPP style
			StartTime:    tournament.StartDate,
			EndTime:      tournament.EndDate,
			Status:       string(tournament.Status),
			TournamentID: &tournamentID,
		}
	}

	// Calculate metadata
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	meta := map[string]interface{}{
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    contests,
		"meta":    meta,
		"success": true,
	})
}

// GetContest returns a single golf contest/tournament
func (h *GolfHandler) GetContest(c *gin.Context) {
	contestIDStr := c.Param("id")
	contestID, err := uuid.Parse(contestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid contest ID",
			"success": false,
		})
		return
	}

	// Find tournament by ID
	var tournament models.GolfTournament
	if err := h.db.First(&tournament, contestID).Error; err != nil {
		h.logger.Error("Failed to fetch golf tournament", "error", err, "id", contestID)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Contest not found",
			"success": false,
		})
		return
	}

	// Convert to contest format
	tournamentID := tournament.ID.String()
	contest := ContestInfo{
		ID:           tournament.ID.String(),
		Name:         tournament.Name,
		Sport:        "golf",
		Platform:     "multi",
		ContestType:  "gpp",
		StartTime:    tournament.StartDate,
		EndTime:      tournament.EndDate,
		Status:       string(tournament.Status),
		TournamentID: &tournamentID,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    contest,
		"success": true,
	})
}
