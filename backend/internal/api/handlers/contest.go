package handlers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

type ContestHandler struct {
	db    *database.DB
	cache *services.CacheService
}

func NewContestHandler(db *database.DB, cache *services.CacheService) *ContestHandler {
	return &ContestHandler{
		db:    db,
		cache: cache,
	}
}

// ListContests returns available contests
func (h *ContestHandler) ListContests(c *gin.Context) {
	// Query parameters
	sport := c.Query("sport")
	platform := c.Query("platform")
	contestType := c.Query("contest_type")
	active := c.Query("active")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "20"))

	// Build query
	query := h.db.Model(&models.Contest{})

	// Apply filters
	if sport != "" {
		query = query.Where("sport = ?", sport)
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if contestType != "" {
		query = query.Where("contest_type = ?", contestType)
	}
	if active != "" {
		isActive, _ := strconv.ParseBool(active)
		query = query.Where("is_active = ?", isActive)
	}

	// Default to active contests starting in the future
	if active == "" {
		query = query.Where("is_active = ? AND start_time > ?", true, time.Now())
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply pagination and sorting
	offset := (page - 1) * perPage
	query = query.Offset(offset).Limit(perPage).Order("start_time ASC")

	var contests []models.Contest
	if err := query.Find(&contests).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch contests")
		return
	}

	// Calculate metadata
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	meta := &utils.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	utils.SendSuccessWithMeta(c, contests, meta)
}

// GetContest returns a single contest
func (h *ContestHandler) GetContest(c *gin.Context) {
	contestIDStr := c.Param("id")
	contestID, err := strconv.ParseUint(contestIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid contest ID", err.Error())
		return
	}

	var contest models.Contest
	if err := h.db.First(&contest, contestID).Error; err != nil {
		utils.SendNotFound(c, "Contest not found")
		return
	}

	// Add additional contest info
	response := gin.H{
		"contest":               contest,
		"stats":                 h.getContestStats(contest.ID),
		"position_requirements": contest.PositionRequirements,
	}

	utils.SendSuccess(c, response)
}

// CreateContest creates a new contest (admin only)
func (h *ContestHandler) CreateContest(c *gin.Context) {
	var req struct {
		Platform          string  `json:"platform" binding:"required,oneof=draftkings fanduel"`
		Sport             string  `json:"sport" binding:"required,oneof=nba nfl mlb nhl"`
		ContestType       string  `json:"contest_type" binding:"required,oneof=gpp cash"`
		Name              string  `json:"name" binding:"required"`
		EntryFee          float64 `json:"entry_fee" binding:"required,min=0"`
		PrizePool         float64 `json:"prize_pool" binding:"required,min=0"`
		MaxEntries        int     `json:"max_entries" binding:"required,min=1"`
		SalaryCap         int     `json:"salary_cap" binding:"required"`
		StartTime         string  `json:"start_time" binding:"required"`
		IsMultiEntry      bool    `json:"is_multi_entry"`
		MaxLineupsPerUser int     `json:"max_lineups_per_user"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Parse start time
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		utils.SendValidationError(c, "Invalid start time format", "Use RFC3339 format")
		return
	}

	// Create contest
	contest := models.Contest{
		Platform:             req.Platform,
		Sport:                req.Sport,
		ContestType:          req.ContestType,
		Name:                 req.Name,
		EntryFee:             req.EntryFee,
		PrizePool:            req.PrizePool,
		MaxEntries:           req.MaxEntries,
		SalaryCap:            req.SalaryCap,
		StartTime:            startTime,
		IsActive:             true,
		IsMultiEntry:         req.IsMultiEntry,
		MaxLineupsPerUser:    req.MaxLineupsPerUser,
		PositionRequirements: models.GetPositionRequirements(req.Sport, req.Platform),
	}

	// Set defaults
	if contest.MaxLineupsPerUser == 0 {
		if contest.IsMultiEntry {
			contest.MaxLineupsPerUser = 150
		} else {
			contest.MaxLineupsPerUser = 1
		}
	}

	// Save contest
	if err := h.db.Create(&contest).Error; err != nil {
		utils.SendInternalError(c, "Failed to create contest")
		return
	}

	utils.SendSuccess(c, contest)
}

// UpdateContest updates contest details (admin only)
func (h *ContestHandler) UpdateContest(c *gin.Context) {
	contestIDStr := c.Param("id")
	contestID, err := strconv.ParseUint(contestIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid contest ID", err.Error())
		return
	}

	var req struct {
		Name              string  `json:"name"`
		EntryFee          float64 `json:"entry_fee"`
		PrizePool         float64 `json:"prize_pool"`
		MaxEntries        int     `json:"max_entries"`
		StartTime         string  `json:"start_time"`
		IsActive          *bool   `json:"is_active"`
		MaxLineupsPerUser int     `json:"max_lineups_per_user"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Get contest
	var contest models.Contest
	if err := h.db.First(&contest, contestID).Error; err != nil {
		utils.SendNotFound(c, "Contest not found")
		return
	}

	// Check if contest has started
	if time.Now().After(contest.StartTime) {
		utils.SendValidationError(c, "Cannot update contest after it has started", "")
		return
	}

	// Update fields
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.EntryFee > 0 {
		updates["entry_fee"] = req.EntryFee
	}
	if req.PrizePool > 0 {
		updates["prize_pool"] = req.PrizePool
	}
	if req.MaxEntries > 0 {
		updates["max_entries"] = req.MaxEntries
	}
	if req.MaxLineupsPerUser > 0 {
		updates["max_lineups_per_user"] = req.MaxLineupsPerUser
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err == nil {
			updates["start_time"] = startTime
		}
	}

	// Update contest
	if err := h.db.Model(&contest).Updates(updates).Error; err != nil {
		utils.SendInternalError(c, "Failed to update contest")
		return
	}

	utils.SendSuccess(c, contest)
}

// GetContestLeaderboard returns contest standings
func (h *ContestHandler) GetContestLeaderboard(c *gin.Context) {
	contestIDStr := c.Param("id")
	contestID, err := strconv.ParseUint(contestIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid contest ID", err.Error())
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "50"))

	// Get contest
	var contest models.Contest
	if err := h.db.First(&contest, contestID).Error; err != nil {
		utils.SendNotFound(c, "Contest not found")
		return
	}

	// Check if contest has started
	if time.Now().Before(contest.StartTime) {
		utils.SendValidationError(c, "Contest has not started yet", "")
		return
	}

	// Get lineups with actual points
	var lineups []models.Lineup
	query := h.db.Model(&models.Lineup{}).
		Where("contest_id = ? AND is_submitted = ? AND actual_points IS NOT NULL", contestID, true).
		Order("actual_points DESC")

	// Count total
	var total int64
	query.Count(&total)

	// Apply pagination
	offset := (page - 1) * perPage
	query = query.Offset(offset).Limit(perPage).Preload("Players")

	if err := query.Find(&lineups).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch leaderboard")
		return
	}

	// Format leaderboard
	leaderboard := make([]gin.H, len(lineups))
	for i, lineup := range lineups {
		rank := offset + i + 1
		leaderboard[i] = gin.H{
			"rank":          rank,
			"lineup_id":     lineup.ID,
			"user_id":       lineup.UserID,
			"lineup_name":   lineup.Name,
			"actual_points": lineup.ActualPoints,
			"payout":        calculatePayout(rank, contest),
			"players":       lineup.Players,
		}
	}

	// Calculate metadata
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	meta := &utils.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response := gin.H{
		"contest":     contest,
		"leaderboard": leaderboard,
		"meta":        meta,
	}

	utils.SendSuccess(c, response)
}

// Helper functions

func (h *ContestHandler) getContestStats(contestID uint) gin.H {
	var stats struct {
		TotalLineups     int64
		UniqueUsers      int64
		AvgProjected     float64
		HighestProjected float64
	}

	// Total lineups
	h.db.Model(&models.Lineup{}).Where("contest_id = ? AND is_submitted = ?", contestID, true).Count(&stats.TotalLineups)

	// Unique users
	h.db.Model(&models.Lineup{}).Where("contest_id = ? AND is_submitted = ?", contestID, true).
		Distinct("user_id").Count(&stats.UniqueUsers)

	// Average and highest projected
	h.db.Model(&models.Lineup{}).Where("contest_id = ? AND is_submitted = ?", contestID, true).
		Select("AVG(projected_points) as avg_projected, MAX(projected_points) as highest_projected").
		Scan(&stats)

	return gin.H{
		"total_lineups":     stats.TotalLineups,
		"unique_users":      stats.UniqueUsers,
		"avg_projected":     stats.AvgProjected,
		"highest_projected": stats.HighestProjected,
		"fill_percentage":   float64(stats.TotalLineups) / float64(h.getContestMaxEntries(contestID)) * 100,
	}
}

func (h *ContestHandler) getContestMaxEntries(contestID uint) int {
	var contest models.Contest
	h.db.Select("max_entries").First(&contest, contestID)
	return contest.MaxEntries
}

func calculatePayout(rank int, contest models.Contest) float64 {
	// This would use the actual payout structure
	// For now, simple calculation
	if contest.ContestType == "cash" {
		if rank <= contest.MaxEntries/2 {
			return contest.EntryFee * 1.8
		}
		return 0
	}

	// GPP payouts (simplified)
	prizePool := contest.PrizePool
	switch {
	case rank == 1:
		return prizePool * 0.20
	case rank == 2:
		return prizePool * 0.12
	case rank == 3:
		return prizePool * 0.08
	case rank <= 5:
		return prizePool * 0.05
	case rank <= 10:
		return prizePool * 0.03
	case rank <= 20:
		return prizePool * 0.015
	case rank <= 50:
		return prizePool * 0.005
	case rank <= 100:
		return prizePool * 0.002
	case rank <= contest.MaxEntries/5:
		return contest.EntryFee * 1.5
	default:
		return 0
	}
}
