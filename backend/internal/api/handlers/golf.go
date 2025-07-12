package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
	"github.com/sirupsen/logrus"
)

// GolfHandler handles golf-specific API endpoints
type GolfHandler struct {
	db           *database.DB
	cache        *services.CacheService
	logger       *logrus.Logger
	golfProvider interface {
		GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error)
		GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error)
		GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error)
		GetCurrentTournament() (*providers.GolfTournamentData, error)
		GetTournamentSchedule() ([]providers.GolfTournamentData, error)
	}
	projectionService *services.GolfProjectionService
}

// NewGolfHandler creates a new golf handler
func NewGolfHandler(db *database.DB, cache *services.CacheService, logger *logrus.Logger, cfg *config.Config) *GolfHandler {
	projectionService := services.NewGolfProjectionService(db, cache, logger)

	// Use RapidAPI provider if API key is available, otherwise fall back to ESPN
	var golfProvider interface {
		GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error)
		GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error)
		GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error)
		GetCurrentTournament() (*providers.GolfTournamentData, error)
		GetTournamentSchedule() ([]providers.GolfTournamentData, error)
	}

	if cfg.RapidAPIKey != "" {
		logger.Info("Using RapidAPI Golf provider")
		golfProvider = providers.NewRapidAPIGolfClient(cfg.RapidAPIKey, cache, logger)
	} else {
		logger.Info("Using ESPN Golf provider (no RapidAPI key configured)")
		golfProvider = providers.NewESPNGolfClient(cache, logger)
	}

	return &GolfHandler{
		db:                db,
		cache:             cache,
		logger:            logger,
		golfProvider:      golfProvider,
		projectionService: projectionService,
	}
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

	// Convert to player format with golf-specific data
	players := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		if entry.Player == nil {
			continue
		}

		player := gin.H{
			"id":               entry.Player.ID,
			"name":             entry.Player.Name,
			"external_id":      entry.Player.ExternalID,
			"position":         "G",               // Golfer
			"team":             entry.Player.Team, // Country in golf
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

		players = append(players, player)
	}

	c.JSON(http.StatusOK, gin.H{
		"tournament": tournament,
		"players":    players,
		"platform":   platform,
	})
}

// SyncTournamentData syncs tournament data from external provider
func (h *GolfHandler) SyncTournamentData(c *gin.Context) {
	// Create sync service
	syncService := services.NewGolfTournamentSyncService(h.db, h.golfProvider, h.logger)

	// Sync all active tournaments
	if err := syncService.SyncAllActiveTournaments(); err != nil {
		h.logger.Error("Failed to sync golf tournaments", "error", err)
		utils.SendInternalError(c, "Failed to sync tournament data")
		return
	}

	// Also sync tournament details for display
	tournamentData, err := h.golfProvider.GetCurrentTournament()
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

	// Sync player data
	players, err := h.golfProvider.GetPlayers("golf", time.Now().Format("2006-01-02"))
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
				ExternalID: playerData.ExternalID,
				Name:       playerData.Name,
				Team:       playerData.Team,
				Position:   "G",
				Sport:      "golf",
			}).Error; err != nil {
			h.logger.Warn("Failed to create player", "error", err, "player", playerData.Name)
			continue
		}

		// Create or update player entry
		entry := models.GolfPlayerEntry{
			PlayerID:     player.ID,
			TournamentID: tournament.ID,
		}

		// Update entry data from stats
		if score, ok := playerData.Stats["score"]; ok {
			entry.TotalScore = int(score)
		}
		if pos, ok := playerData.Stats["position"]; ok {
			entry.CurrentPosition = int(pos)
		}

		if err := h.db.Where("player_id = ? AND tournament_id = ?", player.ID, tournament.ID).
			Assign(entry).
			FirstOrCreate(&entry).Error; err != nil {
			h.logger.Warn("Failed to update player entry", "error", err)
		}
	}

	// Get updated contest count
	var contestCount int64
	h.db.Model(&models.Contest{}).Where("sport = ? AND is_active = ?", "golf", true).Count(&contestCount)

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

	// Convert to player models
	players := make([]models.Player, 0, len(entries))
	for _, entry := range entries {
		if entry.Player != nil {
			players = append(players, *entry.Player)
		}
	}

	// Generate projections
	projections, correlations, err := h.projectionService.GenerateProjections(
		c.Request.Context(),
		players,
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
		utils.SendValidationError(c, "Invalid year parameter", err.Error())
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
	schedule, err := h.golfProvider.GetTournamentSchedule()
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
