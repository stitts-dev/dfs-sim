package providers

import (
	"time"
	
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// DataGolfClient is a publicly accessible wrapper around the internal DataGolf client
type DataGolfClient struct {
	*providers.DataGolfClient
}

// NewDataGolfClient creates a new DataGolf client with public access
func NewDataGolfClient(apiKey string, db *gorm.DB, cache types.CacheProvider, logger *logrus.Logger) *DataGolfClient {
	return &DataGolfClient{
		DataGolfClient: providers.NewDataGolfClient(apiKey, db, cache, logger),
	}
}

// Enhanced Golf Data Provider Interface - publicly accessible
type EnhancedGolfDataProvider = providers.EnhancedGolfDataProvider

// Public type aliases for DataGolf integration
type StrokesGainedMetrics = providers.StrokesGainedMetrics
type CourseAnalytics = providers.CourseAnalytics
type TournamentPredictions = providers.TournamentPredictions
type PlayerCourseHistory = providers.PlayerCourseHistory
type WeatherImpactAnalysis = providers.WeatherImpactAnalysis
type EnhancedPlayerPrediction = providers.EnhancedPlayerPrediction
type CourseModelData = providers.CourseModelData
type WeatherModelData = providers.WeatherModelData

// LiveLeaderboardEntry wrapper for public access
type LiveLeaderboardEntry struct {
	PlayerID         int     `json:"player_id"`
	PlayerName       string  `json:"player_name"`
	Position         int     `json:"position"`
	TotalScore       int     `json:"total_score"`
	ThruHoles        int     `json:"thru_holes"`
	RoundScore       int     `json:"round_score"`
	MovementIndicator string `json:"movement_indicator"`
	TeeTime          string  `json:"tee_time"`
	IsOnCourse       bool    `json:"is_on_course"`
}

// LiveTournamentData wrapper for public access
type LiveTournamentData struct {
	TournamentID     string                 `json:"tournament_id"`
	CurrentRound     int                    `json:"current_round"`
	CutLine          int                    `json:"cut_line"`
	CutMade          bool                   `json:"cut_made"`
	LeaderScore      int                    `json:"leader_score"`
	LastUpdated      time.Time              `json:"last_updated"`
	LiveLeaderboard  []*LiveLeaderboardEntry `json:"live_leaderboard"`
	WeatherUpdate    providers.WeatherConditions `json:"weather_update"`
	PlaySuspended    bool                   `json:"play_suspended"`
}

// PlayerPrediction is an alias for EnhancedPlayerPrediction for backward compatibility
type PlayerPrediction = providers.EnhancedPlayerPrediction

// Conversion functions between internal and public types
func convertLiveLeaderboardEntry(internal *providers.LiveLeaderboardEntry) *LiveLeaderboardEntry {
	if internal == nil {
		return nil
	}
	return &LiveLeaderboardEntry{
		PlayerID:         internal.PlayerID,
		PlayerName:       internal.PlayerName,
		Position:         internal.Position,
		TotalScore:       internal.TotalScore,
		ThruHoles:        internal.ThruHoles,
		RoundScore:       internal.RoundScore,
		MovementIndicator: internal.MovementIndicator,
		TeeTime:          internal.TeeTime,
		IsOnCourse:       internal.IsOnCourse,
	}
}

func ConvertLiveTournamentData(internal *providers.LiveTournamentData) *LiveTournamentData {
	if internal == nil {
		return nil
	}
	
	publicLeaderboard := make([]*LiveLeaderboardEntry, len(internal.LiveLeaderboard))
	for i, entry := range internal.LiveLeaderboard {
		publicLeaderboard[i] = convertLiveLeaderboardEntry(&entry)
	}
	
	return &LiveTournamentData{
		TournamentID:     internal.TournamentID,
		CurrentRound:     internal.CurrentRound,
		CutLine:          internal.CutLine,
		CutMade:          internal.CutMade,
		LeaderScore:      internal.LeaderScore,
		LastUpdated:      internal.LastUpdated,
		LiveLeaderboard:  publicLeaderboard,
		WeatherUpdate:    internal.WeatherUpdate,
		PlaySuspended:    internal.PlaySuspended,
	}
}

// GetLiveTournamentData wraps the internal method with public type conversion
func (c *DataGolfClient) GetLiveTournamentData(tournamentID string) (*LiveTournamentData, error) {
	internal, err := c.DataGolfClient.GetLiveTournamentData(tournamentID)
	if err != nil {
		return nil, err
	}
	return ConvertLiveTournamentData(internal), nil
}