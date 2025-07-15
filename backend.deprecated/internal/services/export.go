package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
)

// ExportService handles lineup exports for different platforms
type ExportService struct{}

func NewExportService() *ExportService {
	return &ExportService{}
}

// ExportFormat represents a supported export format
type ExportFormat struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Platform    string   `json:"platform"`
	Sport       string   `json:"sport"`
	Description string   `json:"description"`
	Headers     []string `json:"headers"`
}

// GetAvailableFormats returns all supported export formats
func (s *ExportService) GetAvailableFormats() []ExportFormat {
	return []ExportFormat{
		{
			ID:          "dk_nba",
			Name:        "DraftKings NBA",
			Platform:    "draftkings",
			Sport:       "nba",
			Description: "CSV format for DraftKings NBA contests",
			Headers:     []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UTIL"},
		},
		{
			ID:          "fd_nba",
			Name:        "FanDuel NBA",
			Platform:    "fanduel",
			Sport:       "nba",
			Description: "CSV format for FanDuel NBA contests",
			Headers:     []string{"PG", "PG", "SG", "SG", "SF", "SF", "PF", "PF", "C"},
		},
		{
			ID:          "dk_nfl",
			Name:        "DraftKings NFL",
			Platform:    "draftkings",
			Sport:       "nfl",
			Description: "CSV format for DraftKings NFL contests",
			Headers:     []string{"QB", "RB", "RB", "WR", "WR", "WR", "TE", "FLEX", "DST"},
		},
		{
			ID:          "fd_nfl",
			Name:        "FanDuel NFL",
			Platform:    "fanduel",
			Sport:       "nfl",
			Description: "CSV format for FanDuel NFL contests",
			Headers:     []string{"QB", "RB", "RB", "WR", "WR", "WR", "TE", "FLEX", "D/ST"},
		},
		{
			ID:          "dk_mlb",
			Name:        "DraftKings MLB",
			Platform:    "draftkings",
			Sport:       "mlb",
			Description: "CSV format for DraftKings MLB contests",
			Headers:     []string{"P", "P", "C", "1B", "2B", "3B", "SS", "OF", "OF", "OF"},
		},
		{
			ID:          "fd_mlb",
			Name:        "FanDuel MLB",
			Platform:    "fanduel",
			Sport:       "mlb",
			Description: "CSV format for FanDuel MLB contests",
			Headers:     []string{"P", "C/1B", "2B", "3B", "SS", "OF", "OF", "OF", "UTIL"},
		},
		{
			ID:          "dk_nhl",
			Name:        "DraftKings NHL",
			Platform:    "draftkings",
			Sport:       "nhl",
			Description: "CSV format for DraftKings NHL contests",
			Headers:     []string{"C", "C", "W", "W", "W", "D", "D", "G", "UTIL"},
		},
		{
			ID:          "fd_nhl",
			Name:        "FanDuel NHL",
			Platform:    "fanduel",
			Sport:       "nhl",
			Description: "CSV format for FanDuel NHL contests",
			Headers:     []string{"C", "C", "W", "W", "W", "W", "D", "D", "G"},
		},
	}
}

// ExportLineups exports lineups to CSV format
func (s *ExportService) ExportLineups(lineups []models.Lineup, format string) ([]byte, error) {
	if len(lineups) == 0 {
		return nil, fmt.Errorf("no lineups to export")
	}

	// Get export format
	formatConfig := s.getFormatConfig(format)
	if formatConfig == nil {
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}

	// Create CSV writer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write headers
	if err := writer.Write(formatConfig.Headers); err != nil {
		return nil, fmt.Errorf("failed to write headers: %w", err)
	}

	// Export each lineup
	for _, lineup := range lineups {
		row, err := s.formatLineup(lineup, formatConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to format lineup %d: %w", lineup.ID, err)
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write lineup: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportSingleLineup exports a single lineup with additional metadata
func (s *ExportService) ExportSingleLineup(lineup models.Lineup, includeStats bool) map[string]interface{} {
	export := map[string]interface{}{
		"lineup_id":        lineup.ID,
		"name":             lineup.Name,
		"total_salary":     lineup.TotalSalary,
		"projected_points": lineup.ProjectedPoints,
		"players":          []map[string]interface{}{},
	}

	// Add player details
	for _, player := range lineup.Players {
		playerData := map[string]interface{}{
			"id":               player.ID,
			"name":             player.Name,
			"position":         player.Position,
			"team":             player.Team,
			"opponent":         player.Opponent,
			"salary":           player.Salary,
			"projected_points": player.ProjectedPoints,
		}

		if includeStats {
			playerData["ownership"] = player.Ownership
			playerData["floor_points"] = player.FloorPoints
			playerData["ceiling_points"] = player.CeilingPoints
		}

		export["players"] = append(export["players"].([]map[string]interface{}), playerData)
	}

	// Add team exposure
	export["team_exposure"] = lineup.GetTeamExposure()

	// Add game exposure
	export["game_exposure"] = lineup.GetGameExposure()

	return export
}

// Helper functions

func (s *ExportService) getFormatConfig(formatID string) *ExportFormat {
	formats := s.GetAvailableFormats()
	for _, f := range formats {
		if f.ID == formatID {
			return &f
		}
	}
	return nil
}

func (s *ExportService) formatLineup(lineup models.Lineup, format *ExportFormat) ([]string, error) {
	// Create position map
	positionMap := make(map[string][]models.Player)
	for _, player := range lineup.Players {
		positionMap[player.Position] = append(positionMap[player.Position], player)
	}

	// Fill positions according to format
	row := make([]string, len(format.Headers))
	usedPlayers := make(map[uint]bool)

	for i, position := range format.Headers {
		player := s.selectPlayerForPosition(position, positionMap, usedPlayers, format.Platform)
		if player == nil {
			return nil, fmt.Errorf("no player available for position %s", position)
		}

		// Format player based on platform
		row[i] = s.formatPlayer(player, format.Platform)
		usedPlayers[player.ID] = true
	}

	return row, nil
}

func (s *ExportService) selectPlayerForPosition(position string, positionMap map[string][]models.Player, used map[uint]bool, platform string) *models.Player {
	// Direct position match
	for _, player := range positionMap[position] {
		if !used[player.ID] {
			return &player
		}
	}

	// Handle flex positions
	switch position {
	case "FLEX":
		// NFL FLEX: RB/WR/TE
		for _, pos := range []string{"RB", "WR", "TE"} {
			for _, player := range positionMap[pos] {
				if !used[player.ID] {
					return &player
				}
			}
		}
	case "UTIL":
		// Any position
		for _, players := range positionMap {
			for _, player := range players {
				if !used[player.ID] {
					return &player
				}
			}
		}
	case "G":
		// NBA Guard: PG or SG
		for _, pos := range []string{"PG", "SG"} {
			for _, player := range positionMap[pos] {
				if !used[player.ID] {
					return &player
				}
			}
		}
	case "F":
		// NBA Forward: SF or PF
		for _, pos := range []string{"SF", "PF"} {
			for _, player := range positionMap[pos] {
				if !used[player.ID] {
					return &player
				}
			}
		}
	case "C/1B":
		// MLB: C or 1B
		for _, pos := range []string{"C", "1B"} {
			for _, player := range positionMap[pos] {
				if !used[player.ID] {
					return &player
				}
			}
		}
	}

	return nil
}

func (s *ExportService) formatPlayer(player *models.Player, platform string) string {
	// DraftKings format: "PlayerID:PlayerName"
	if platform == "draftkings" {
		return fmt.Sprintf("%s:%s", player.ExternalID, player.Name)
	}

	// FanDuel format: "PlayerID - PlayerName"
	if platform == "fanduel" {
		return fmt.Sprintf("%s - %s", player.ExternalID, player.Name)
	}

	// Default format
	return player.Name
}

// ValidateLineupForExport checks if a lineup can be exported
func (s *ExportService) ValidateLineupForExport(lineup models.Lineup, format string) error {
	formatConfig := s.getFormatConfig(format)
	if formatConfig == nil {
		return fmt.Errorf("unsupported export format: %s", format)
	}

	// Check if sport matches
	sport := strings.Split(format, "_")[1]
	if lineup.Contest.Sport != sport {
		return fmt.Errorf("lineup sport %s does not match export format %s", lineup.Contest.Sport, sport)
	}

	// Check if platform matches
	platform := strings.Split(format, "_")[0]
	if platform == "dk" {
		platform = "draftkings"
	} else if platform == "fd" {
		platform = "fanduel"
	}

	if lineup.Contest.Platform != platform {
		return fmt.Errorf("lineup platform %s does not match export format %s", lineup.Contest.Platform, platform)
	}

	// Check if lineup has correct number of players
	expectedPlayers := len(formatConfig.Headers)
	if len(lineup.Players) != expectedPlayers {
		return fmt.Errorf("lineup has %d players, expected %d", len(lineup.Players), expectedPlayers)
	}

	return nil
}

// BatchExportResult represents the result of a batch export
type BatchExportResult struct {
	Success      int      `json:"success"`
	Failed       int      `json:"failed"`
	Errors       []string `json:"errors,omitempty"`
	CSVData      []byte   `json:"-"`
	FileName     string   `json:"file_name"`
	TotalLineups int      `json:"total_lineups"`
	ExportFormat string   `json:"export_format"`
}

// BatchExportLineups exports multiple lineups with validation
func (s *ExportService) BatchExportLineups(lineups []models.Lineup, format string) *BatchExportResult {
	result := &BatchExportResult{
		TotalLineups: len(lineups),
		ExportFormat: format,
		Errors:       make([]string, 0),
	}

	// Validate each lineup
	validLineups := make([]models.Lineup, 0, len(lineups))
	for _, lineup := range lineups {
		if err := s.ValidateLineupForExport(lineup, format); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Lineup %d: %s", lineup.ID, err.Error()))
		} else {
			validLineups = append(validLineups, lineup)
			result.Success++
		}
	}

	// Export valid lineups
	if len(validLineups) > 0 {
		csvData, err := s.ExportLineups(validLineups, format)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Export error: %s", err.Error()))
		} else {
			result.CSVData = csvData
			result.FileName = fmt.Sprintf("lineups_%s_%d.csv", format, len(validLineups))
		}
	}

	return result
}
