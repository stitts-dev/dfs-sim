package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
)

// SportsHandler handles sports configuration endpoints
type SportsHandler struct {
	config *config.Config
}

// NewSportsHandler creates a new sports handler
func NewSportsHandler(cfg *config.Config) *SportsHandler {
	return &SportsHandler{
		config: cfg,
	}
}

// SportInfo represents information about a sport
type SportInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Enabled  bool   `json:"enabled"`
	Position []string `json:"positions,omitempty"`
}

// SportsResponse represents the response structure for available sports
type SportsResponse struct {
	Sports     []SportInfo `json:"sports"`
	GolfOnly   bool        `json:"golf_only_mode"`
	AllSports  []SportInfo `json:"all_sports"`
}

// GetAvailableSports returns the list of available/supported sports based on configuration
func (h *SportsHandler) GetAvailableSports(c *gin.Context) {
	// Define all possible sports with metadata
	allSports := []SportInfo{
		{
			ID:        "golf",
			Name:      "Golf",
			Icon:      "⛳",
			Enabled:   false,
			Position: []string{"G"},
		},
		{
			ID:        "nba",
			Name:      "NBA",
			Icon:      "🏀",
			Enabled:   false,
			Position: []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UTIL"},
		},
		{
			ID:        "nfl",
			Name:      "NFL",
			Icon:      "🏈",
			Enabled:   false,
			Position: []string{"QB", "RB", "WR", "TE", "K", "DST", "FLEX"},
		},
		{
			ID:        "mlb",
			Name:      "MLB",
			Icon:      "⚾",
			Enabled:   false,
			Position: []string{"C", "1B", "2B", "3B", "SS", "OF", "DH", "P"},
		},
		{
			ID:        "nhl",
			Name:      "NHL",
			Icon:      "🏒",
			Enabled:   false,
			Position: []string{"C", "LW", "RW", "D", "G", "F"},
		},
	}

	// Determine enabled sports based on configuration
	var enabledSports []SportInfo
	
	if h.config.GolfOnlyMode {
		// In golf-only mode, only enable golf
		for i := range allSports {
			if allSports[i].ID == "golf" {
				allSports[i].Enabled = true
				enabledSports = append(enabledSports, allSports[i])
				break
			}
		}
	} else {
		// Enable sports based on SupportedSports configuration
		supportedMap := make(map[string]bool)
		for _, sport := range h.config.SupportedSports {
			supportedMap[sport] = true
		}
		
		for i := range allSports {
			if supportedMap[allSports[i].ID] {
				allSports[i].Enabled = true
				enabledSports = append(enabledSports, allSports[i])
			}
		}
		
		// If no supported sports configured, default to all except golf-only scenarios
		if len(enabledSports) == 0 {
			for i := range allSports {
				allSports[i].Enabled = true
				enabledSports = append(enabledSports, allSports[i])
			}
		}
	}

	response := SportsResponse{
		Sports:    enabledSports,
		GolfOnly:  h.config.GolfOnlyMode,
		AllSports: allSports,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    response,
		"message": "Available sports retrieved successfully",
	})
}

// GetSportConfiguration returns detailed configuration for a specific sport
func (h *SportsHandler) GetSportConfiguration(c *gin.Context) {
	sportID := c.Param("sport")
	
	// Find the sport in configuration
	var sportInfo *SportInfo
	allSports := []SportInfo{
		{ID: "golf", Name: "Golf", Icon: "⛳", Position: []string{"G"}},
		{ID: "nba", Name: "NBA", Icon: "🏀", Position: []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UTIL"}},
		{ID: "nfl", Name: "NFL", Icon: "🏈", Position: []string{"QB", "RB", "WR", "TE", "K", "DST", "FLEX"}},
		{ID: "mlb", Name: "MLB", Icon: "⚾", Position: []string{"C", "1B", "2B", "3B", "SS", "OF", "DH", "P"}},
		{ID: "nhl", Name: "NHL", Icon: "🏒", Position: []string{"C", "LW", "RW", "D", "G", "F"}},
	}
	
	for i := range allSports {
		if allSports[i].ID == sportID {
			sportInfo = &allSports[i]
			break
		}
	}
	
	if sportInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Sport not found",
		})
		return
	}
	
	// Check if sport is enabled
	isEnabled := false
	if h.config.GolfOnlyMode {
		isEnabled = (sportID == "golf")
	} else {
		for _, supported := range h.config.SupportedSports {
			if supported == sportID {
				isEnabled = true
				break
			}
		}
	}
	
	sportInfo.Enabled = isEnabled
	
	c.JSON(http.StatusOK, gin.H{
		"data":    sportInfo,
		"message": "Sport configuration retrieved successfully",
	})
}