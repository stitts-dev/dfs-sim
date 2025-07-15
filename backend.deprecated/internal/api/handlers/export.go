package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

type ExportHandler struct {
	db            *database.DB
	exportService *services.ExportService
}

func NewExportHandler(db *database.DB) *ExportHandler {
	return &ExportHandler{
		db:            db,
		exportService: services.NewExportService(),
	}
}

// ExportLineups exports lineups in the specified format
func (h *ExportHandler) ExportLineups(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		LineupIDs []uint `json:"lineup_ids" binding:"required,min=1,max=500"`
		Format    string `json:"format" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Verify lineup ownership
	var lineups []models.Lineup
	err := h.db.Preload("Contest").Preload("Players").
		Where("id IN ? AND user_id = ?", req.LineupIDs, userID).
		Find(&lineups).Error
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch lineups")
		return
	}

	if len(lineups) != len(req.LineupIDs) {
		utils.SendUnauthorized(c, "Access denied to some lineups")
		return
	}

	// Verify all lineups are for the same contest
	if len(lineups) > 0 {
		contestID := lineups[0].ContestID
		for _, lineup := range lineups {
			if lineup.ContestID != contestID {
				utils.SendValidationError(c, "All lineups must be for the same contest", "")
				return
			}
		}
	}

	// Export lineups
	result := h.exportService.BatchExportLineups(lineups, req.Format)

	if result.CSVData == nil {
		utils.SendError(c, http.StatusBadRequest,
			utils.NewAppError("EXPORT_FAILED", "Failed to export lineups", fmt.Sprintf("%d errors occurred", len(result.Errors))))
		return
	}

	// Return CSV file
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", result.FileName))
	c.Data(http.StatusOK, "text/csv", result.CSVData)
}

// GetExportFormats returns available export formats
func (h *ExportHandler) GetExportFormats(c *gin.Context) {
	sport := c.Query("sport")
	platform := c.Query("platform")

	formats := h.exportService.GetAvailableFormats()

	// Filter by sport/platform if provided
	if sport != "" || platform != "" {
		filtered := make([]services.ExportFormat, 0)
		for _, format := range formats {
			if (sport == "" || format.Sport == sport) &&
				(platform == "" || format.Platform == platform) {
				filtered = append(filtered, format)
			}
		}
		formats = filtered
	}

	utils.SendSuccess(c, formats)
}

// ExportSingleLineup exports a single lineup with detailed information
func (h *ExportHandler) ExportSingleLineup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	lineupIDStr := c.Param("id")
	lineupID, err := strconv.ParseUint(lineupIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid lineup ID", err.Error())
		return
	}

	includeStats := c.Query("include_stats") == "true"
	format := c.Query("format") // Optional format for CSV

	// Get lineup
	var lineup models.Lineup
	err = h.db.Preload("Contest").Preload("Players").
		Where("id = ? AND user_id = ?", lineupID, userID).
		First(&lineup).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// If format specified, return CSV
	if format != "" {
		csvData, err := h.exportService.ExportLineups([]models.Lineup{lineup}, format)
		if err != nil {
			utils.SendError(c, http.StatusBadRequest,
				utils.NewAppError("EXPORT_FAILED", "Failed to export lineup", err.Error()))
			return
		}

		filename := fmt.Sprintf("lineup_%d_%s.csv", lineup.ID, format)
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Data(http.StatusOK, "text/csv", csvData)
		return
	}

	// Return JSON representation
	export := h.exportService.ExportSingleLineup(lineup, includeStats)
	utils.SendSuccess(c, export)
}

// PreviewExport shows what the export will look like
func (h *ExportHandler) PreviewExport(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		LineupID uint   `json:"lineup_id" binding:"required"`
		Format   string `json:"format" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Get lineup
	var lineup models.Lineup
	err := h.db.Preload("Contest").Preload("Players").
		Where("id = ? AND user_id = ?", req.LineupID, userID).
		First(&lineup).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// Validate export
	if err := h.exportService.ValidateLineupForExport(lineup, req.Format); err != nil {
		utils.SendValidationError(c, "Invalid export", err.Error())
		return
	}

	// Generate preview
	csvData, err := h.exportService.ExportLineups([]models.Lineup{lineup}, req.Format)
	if err != nil {
		utils.SendInternalError(c, "Failed to generate preview")
		return
	}

	// Convert to string for preview
	preview := string(csvData)

	response := gin.H{
		"format":  req.Format,
		"valid":   true,
		"preview": preview,
		"lineup":  lineup,
	}

	utils.SendSuccess(c, response)
}

// ValidateExport checks if lineups can be exported
func (h *ExportHandler) ValidateExport(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		LineupIDs []uint `json:"lineup_ids" binding:"required,min=1"`
		Format    string `json:"format" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Get lineups
	var lineups []models.Lineup
	err := h.db.Preload("Contest").Preload("Players").
		Where("id IN ? AND user_id = ?", req.LineupIDs, userID).
		Find(&lineups).Error
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch lineups")
		return
	}

	// Validate each lineup
	validationResults := make([]gin.H, 0, len(lineups))
	allValid := true

	for _, lineup := range lineups {
		err := h.exportService.ValidateLineupForExport(lineup, req.Format)
		result := gin.H{
			"lineup_id": lineup.ID,
			"valid":     err == nil,
		}

		if err != nil {
			result["error"] = err.Error()
			allValid = false
		}

		validationResults = append(validationResults, result)
	}

	response := gin.H{
		"all_valid":   allValid,
		"validations": validationResults,
		"format":      req.Format,
	}

	utils.SendSuccess(c, response)
}

// GetExportHistory returns user's export history
func (h *ExportHandler) GetExportHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "20"))

	// This would typically query an export_history table
	// For now, return mock data
	history := []gin.H{
		{
			"id":           1,
			"user_id":      userID,
			"format":       "dk_nba",
			"lineup_count": 20,
			"exported_at":  "2024-01-20T15:30:00Z",
			"contest_name": "NBA $100K Tournament",
		},
		{
			"id":           2,
			"user_id":      userID,
			"format":       "fd_nfl",
			"lineup_count": 150,
			"exported_at":  "2024-01-19T12:00:00Z",
			"contest_name": "NFL Sunday Million",
		},
	}

	meta := &utils.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      int64(len(history)),
		TotalPages: 1,
	}

	utils.SendSuccessWithMeta(c, history, meta)
}

// GetExportStats returns export statistics
func (h *ExportHandler) GetExportStats(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Get export statistics
	var stats struct {
		TotalExports   int64
		TotalLineups   int64
		MostUsedFormat string
		LastExportDate string
	}

	// This would query actual export data
	// For now, return mock stats
	stats.TotalExports = 42
	stats.TotalLineups = 1337
	stats.MostUsedFormat = "dk_nba"
	stats.LastExportDate = "2024-01-20T15:30:00Z"

	// Platform breakdown
	platformBreakdown := gin.H{
		"draftkings": 75,
		"fanduel":    25,
	}

	// Sport breakdown
	sportBreakdown := gin.H{
		"nba": 40,
		"nfl": 35,
		"mlb": 15,
		"nhl": 10,
	}

	response := gin.H{
		"user_id":            userID,
		"total_exports":      stats.TotalExports,
		"total_lineups":      stats.TotalLineups,
		"most_used_format":   stats.MostUsedFormat,
		"last_export":        stats.LastExportDate,
		"platform_breakdown": platformBreakdown,
		"sport_breakdown":    sportBreakdown,
	}

	utils.SendSuccess(c, response)
}
