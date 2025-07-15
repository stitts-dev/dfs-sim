package optimizer

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// GolfCorrelationBuilder builds correlation matrices specific to golf
type GolfCorrelationBuilder struct {
	players           []types.Player
	entries           map[uint]*types.GolfPlayerEntry
	courseHistory     map[uint]*types.GolfCourseHistory
	weatherService    WeatherServiceInterface
	tournament        *types.GolfTournament
	weatherConditions *types.WeatherConditions
}

// WeatherServiceInterface defines interface for weather correlation calculations
type WeatherServiceInterface interface {
	GetWeatherConditions(ctx context.Context, courseID string) (*types.WeatherConditions, error)
	CalculateGolfImpact(conditions *types.WeatherConditions) *types.WeatherImpact
}

// NewGolfCorrelationBuilder creates a new golf correlation builder
func NewGolfCorrelationBuilder(players []types.Player) *GolfCorrelationBuilder {
	return &GolfCorrelationBuilder{
		players:       players,
		entries:       make(map[uint]*types.GolfPlayerEntry),
		courseHistory: make(map[uint]*types.GolfCourseHistory),
	}
}

// SetWeatherService sets the weather service for weather-based correlations
func (gb *GolfCorrelationBuilder) SetWeatherService(ws WeatherServiceInterface) {
	gb.weatherService = ws
}

// SetTournament sets the tournament data for context
func (gb *GolfCorrelationBuilder) SetTournament(tournament *types.GolfTournament) {
	gb.tournament = tournament
}

// SetPlayerEntries sets the golf-specific player entry data
func (gb *GolfCorrelationBuilder) SetPlayerEntries(entries []types.GolfPlayerEntry) {
	for _, entry := range entries {
		gb.entries[entry.PlayerID] = &entry
	}
}

// SetCourseHistory sets the course history data for players
func (gb *GolfCorrelationBuilder) SetCourseHistory(history []types.GolfCourseHistory) {
	for _, h := range history {
		gb.courseHistory[h.PlayerID] = &h
	}
}

// BuildCorrelationMatrix builds the correlation matrix for golf
func (gb *GolfCorrelationBuilder) BuildCorrelationMatrix() map[string]map[string]float64 {
	correlations := make(map[string]map[string]float64)

	for i, p1 := range gb.players {
		p1ID := p1.GetID().String()
		correlations[p1ID] = make(map[string]float64)

		for j, p2 := range gb.players {
			p2ID := p2.GetID().String()
			if i == j {
				correlations[p1ID][p2ID] = 1.0
				continue
			}

			correlation := gb.calculateGolfCorrelation(p1, p2)
			correlations[p1ID][p2ID] = correlation
		}
	}

	return correlations
}

// calculateGolfCorrelation calculates correlation between two golf players
func (gb *GolfCorrelationBuilder) calculateGolfCorrelation(p1, p2 types.Player) float64 {
	correlation := 0.0

	// Get player entries if available
	// Note: This requires a mapping from Player UUID to entry PlayerID (uint)
	// For now, we'll skip entry-based correlations until proper mapping is implemented
	var entry1, entry2 *types.GolfPlayerEntry

	if entry1 != nil && entry2 != nil {
		// Enhanced tee time correlation with weather context
		if gb.haveSameTeeTime(entry1, entry2) {
			teeTimeCorrelation := 0.15
			
			// Weather enhances tee time correlation
			if weatherCorr := gb.calculateWeatherTeeTimeCorrelation(entry1, entry2); weatherCorr != 0 {
				teeTimeCorrelation += weatherCorr
			}
			
			correlation += teeTimeCorrelation
		}

		// Enhanced wave correlation with weather considerations
		if gb.inSameWave(entry1, entry2) {
			waveCorrelation := 0.05
			
			// Weather conditions can make wave timing more important
			if weatherCorr := gb.calculateWeatherWaveCorrelation(entry1, entry2); weatherCorr != 0 {
				waveCorrelation += weatherCorr
			}
			
			correlation += waveCorrelation
		}

		// Similar current position correlation
		if gb.areSimilarPosition(entry1, entry2) {
			correlation += 0.08
		}
	}

	// Country/region correlation (Ryder Cup effect)
	if gb.sameCountry(p1, p2) {
		correlation += 0.10
	}

	// Similar skill level correlation (based on salary as proxy for ranking)
	if gb.similarSkillLevel(p1, p2) {
		correlation += 0.08
	}

	// Course history correlation
	// Note: This requires a mapping from Player UUID to course history PlayerID (uint)
	// For now, we'll skip course history correlations until proper mapping is implemented

	// Weather-based skill correlation
	weatherSkillCorr := gb.calculateWeatherSkillCorrelation(p1, p2)
	correlation += weatherSkillCorr

	// Dynamic weather correlation based on current conditions
	dynamicWeatherCorr := gb.calculateDynamicWeatherCorrelation(p1, p2)
	correlation += dynamicWeatherCorr

	// Cap correlation between -0.3 and 0.6 for golf (expanded range for weather)
	return math.Max(-0.3, math.Min(0.6, correlation))
}

// haveSameTeeTime checks if two players have the same tee time
func (gb *GolfCorrelationBuilder) haveSameTeeTime(e1, e2 *types.GolfPlayerEntry) bool {
	if len(e1.TeeTimes) == 0 || len(e2.TeeTimes) == 0 {
		return false
	}

	// Check latest round tee time
	latestRound := len(e1.TeeTimes) - 1
	if latestRound >= 0 && latestRound < len(e2.TeeTimes) {
		t1, _ := time.Parse(time.RFC3339, e1.TeeTimes[latestRound])
		t2, _ := time.Parse(time.RFC3339, e2.TeeTimes[latestRound])

		// Same tee time (within 10 minutes)
		diff := math.Abs(t1.Sub(t2).Minutes())
		return diff < 10
	}

	return false
}

// inSameWave checks if players are in the same wave (AM/PM)
func (gb *GolfCorrelationBuilder) inSameWave(e1, e2 *types.GolfPlayerEntry) bool {
	if len(e1.TeeTimes) == 0 || len(e2.TeeTimes) == 0 {
		return false
	}

	latestRound := len(e1.TeeTimes) - 1
	if latestRound >= 0 && latestRound < len(e2.TeeTimes) {
		t1, _ := time.Parse(time.RFC3339, e1.TeeTimes[latestRound])
		t2, _ := time.Parse(time.RFC3339, e2.TeeTimes[latestRound])

		// Both AM (before 1 PM) or both PM
		return (t1.Hour() < 13) == (t2.Hour() < 13)
	}

	return false
}

// areSimilarPosition checks if players are in similar leaderboard positions
func (gb *GolfCorrelationBuilder) areSimilarPosition(e1, e2 *types.GolfPlayerEntry) bool {
	if e1.CurrentPosition == 0 || e2.CurrentPosition == 0 {
		return false
	}

	diff := math.Abs(float64(e1.CurrentPosition - e2.CurrentPosition))
	return diff <= 10 // Within 10 positions
}

// sameCountry checks if players are from the same country
func (gb *GolfCorrelationBuilder) sameCountry(p1, p2 types.Player) bool {
	// In golf, Team often represents country
	return p1.GetTeam() == p2.GetTeam() && p1.GetTeam() != ""
}

// similarSkillLevel checks if players have similar skill levels based on salary
func (gb *GolfCorrelationBuilder) similarSkillLevel(p1, p2 types.Player) bool {
	salary1 := p1.GetSalaryDK()
	if salary1 == 0 {
		salary1 = p1.GetSalaryFD()
	}
	salary2 := p2.GetSalaryDK()
	if salary2 == 0 {
		salary2 = p2.GetSalaryFD()
	}
	
	if salary1 == 0 || salary2 == 0 {
		return false
	}

	salaryDiff := math.Abs(float64(salary1 - salary2))
	avgSalary := float64(salary1+salary2) / 2

	// Within 15% of each other's salary
	return salaryDiff/avgSalary < 0.15
}

// similarCourseHistory checks if players have similar course history
func (gb *GolfCorrelationBuilder) similarCourseHistory(p1ID, p2ID uint) bool {
	h1, ok1 := gb.courseHistory[p1ID]
	h2, ok2 := gb.courseHistory[p2ID]

	if !ok1 || !ok2 || h1.ScoringAvg == 0 || h2.ScoringAvg == 0 {
		return false
	}

	// Similar scoring average at this course
	scoringDiff := math.Abs(h1.ScoringAvg - h2.ScoringAvg)
	return scoringDiff < 1.5 // Within 1.5 strokes
}

// GetGolfTeammateCorrelation returns correlation for golf (country-based)
func getGolfTeammateCorrelation(pos1, pos2 string) float64 {
	// In golf, all players are position "G"
	if pos1 == "G" && pos2 == "G" {
		return 0.10 // Small positive correlation for same country
	}
	return 0.0
}

// GetGolfOpponentCorrelation returns correlation for golf opponents
func getGolfOpponentCorrelation(pos1, pos2 string) float64 {
	// Golf doesn't have direct opponents like team sports
	return 0.0
}

// getGolfStackingBonus returns bonus correlation for golf stacking strategies
func getGolfStackingBonus(p1, p2 types.Player) float64 {
	bonus := 0.0

	// Ownership-based negative correlation stacking
	// If one player is highly owned and another is low owned
	ownership1 := p1.GetOwnershipDK()
	ownership2 := p2.GetOwnershipDK()

	if (ownership1 > 20 && ownership2 < 10) || (ownership1 < 10 && ownership2 > 20) {
		bonus += 0.05 // Encourage contrarian plays
	}

	// Price-based stacking (stars and scrubs)
	salary1 := p1.GetSalaryDK()
	if salary1 == 0 {
		salary1 = p1.GetSalaryFD()
	}
	salary2 := p2.GetSalaryDK()
	if salary2 == 0 {
		salary2 = p2.GetSalaryFD()
	}

	if (salary1 > 10000 && salary2 < 7000) || (salary1 < 7000 && salary2 > 10000) {
		bonus += 0.03
	}

	return bonus
}

// Helper function to parse player metadata for golf-specific data
func getGolfMetadata(player types.Player, key string) string {
	// For now, return empty string as we don't have metadata storage
	// TODO: Implement proper metadata storage for golf-specific data
	return ""
}

// Helper function to check if string contains golf position
func isGolfPosition(position string) bool {
	golfPos := strings.ToUpper(position)
	return golfPos == "G" || golfPos == "GOLF" || golfPos == "GOLFER"
}

// Weather Correlation Methods

// calculateWeatherTeeTimeCorrelation calculates enhanced tee time correlation based on weather
func (gb *GolfCorrelationBuilder) calculateWeatherTeeTimeCorrelation(e1, e2 *types.GolfPlayerEntry) float64 {
	if gb.weatherService == nil || gb.tournament == nil {
		return 0.0
	}

	// Get current weather conditions
	weather, err := gb.weatherService.GetWeatherConditions(context.Background(), gb.tournament.CourseID)
	if err != nil {
		return 0.0
	}

	weatherImpact := gb.weatherService.CalculateGolfImpact(weather)
	correlation := 0.0

	// Wind correlation - players in same group face same wind conditions
	if weather.WindSpeed > 15 {
		// High wind makes tee time correlation more important
		correlation += 0.08
	} else if weather.WindSpeed > 10 {
		correlation += 0.04
	}

	// Weather variance correlation - changing conditions favor same tee times
	if weatherImpact.VarianceMultiplier > 1.2 {
		correlation += 0.06
	}

	// Rain/precipitation correlation
	if weather.Conditions == "Rain" || weather.Conditions == "Drizzle" {
		correlation += 0.05
	}

	return correlation
}

// calculateWeatherWaveCorrelation calculates wave correlation enhanced by weather
func (gb *GolfCorrelationBuilder) calculateWeatherWaveCorrelation(e1, e2 *types.GolfPlayerEntry) float64 {
	if gb.weatherService == nil || gb.tournament == nil {
		return 0.0
	}

	weather, err := gb.weatherService.GetWeatherConditions(context.Background(), gb.tournament.CourseID)
	if err != nil {
		return 0.0
	}

	correlation := 0.0

	// Morning vs afternoon wind patterns
	if weather.WindSpeed > 10 {
		// Wind typically picks up in afternoon - morning wave advantage
		correlation += 0.03
	}

	// Temperature effects on morning vs afternoon waves
	if weather.Temperature > 85 || weather.Temperature < 50 {
		// Extreme temps affect performance - wave timing matters more
		correlation += 0.02
	}

	return correlation
}

// calculateWeatherSkillCorrelation calculates correlation based on weather-handling skills
func (gb *GolfCorrelationBuilder) calculateWeatherSkillCorrelation(p1, p2 types.Player) float64 {
	if gb.weatherService == nil || gb.tournament == nil {
		return 0.0
	}

	weather, err := gb.weatherService.GetWeatherConditions(context.Background(), gb.tournament.CourseID)
	if err != nil {
		return 0.0
	}

	correlation := 0.0

	// Players with similar salaries likely have similar weather-handling abilities
	if gb.similarSkillLevel(p1, p2) && weather.WindSpeed > 15 {
		// High wind conditions favor players with similar skill levels
		correlation += 0.04
	}

	// Ball striking correlation in wind
	if weather.WindSpeed > 20 {
		// Extreme wind conditions create higher correlation among elite ball strikers
		salary1 := p1.GetSalaryDK()
		if salary1 == 0 {
			salary1 = p1.GetSalaryFD()
		}
		salary2 := p2.GetSalaryDK()
		if salary2 == 0 {
			salary2 = p2.GetSalaryFD()
		}
		
		if salary1 > 10000 && salary2 > 10000 {
			correlation += 0.06
		}
	}

	return correlation
}

// calculateDynamicWeatherCorrelation calculates real-time weather-based correlation
func (gb *GolfCorrelationBuilder) calculateDynamicWeatherCorrelation(p1, p2 types.Player) float64 {
	if gb.weatherService == nil || gb.tournament == nil {
		return 0.0
	}

	weather, err := gb.weatherService.GetWeatherConditions(context.Background(), gb.tournament.CourseID)
	if err != nil {
		return 0.0
	}

	weatherImpact := gb.weatherService.CalculateGolfImpact(weather)
	correlation := 0.0

	// Course difficulty correlation - harder conditions increase correlation
	if weatherImpact.ScoreImpact > 1.5 {
		// Very difficult conditions create higher correlation across all players
		correlation += 0.05
	}

	// Wind direction correlation - crosswinds vs headwinds/tailwinds
	if weather.WindDir != "" && weather.WindSpeed > 15 {
		// Specific wind directions can favor certain player types similarly
		correlation += 0.03
	}

	// Soft conditions correlation (rain)
	if weatherImpact.SoftConditions {
		// Soft conditions reduce variance and increase correlation
		correlation += 0.04
	}

	// Distance correlation in wind
	if weatherImpact.DistanceReduction > 0.03 {
		// Significant distance loss creates similar challenges for all players
		correlation += 0.03
	}

	return correlation
}

// getWeatherConditionsFromTournament gets weather conditions from tournament data
func (gb *GolfCorrelationBuilder) getWeatherConditionsFromTournament() *types.WeatherConditions {
	if gb.tournament == nil {
		return nil
	}

	// Return stored weather conditions from tournament
	return &gb.tournament.WeatherConditions
}

// updateWeatherConditions fetches and updates current weather conditions
func (gb *GolfCorrelationBuilder) updateWeatherConditions() error {
	if gb.weatherService == nil || gb.tournament == nil {
		return nil
	}

	weather, err := gb.weatherService.GetWeatherConditions(context.Background(), gb.tournament.CourseID)
	if err != nil {
		return err
	}

	gb.weatherConditions = weather
	return nil
}

// getEnhancedTeeTimeCorrelation calculates enhanced tee time correlation
func (gb *GolfCorrelationBuilder) getEnhancedTeeTimeCorrelation(e1, e2 *types.GolfPlayerEntry) float64 {
	baseCorrelation := 0.15 // Base same-group correlation

	// Weather enhancement
	weatherEnhancement := gb.calculateWeatherTeeTimeCorrelation(e1, e2)
	
	// Tournament round enhancement (later rounds more important)
	roundEnhancement := 0.0
	if gb.tournament != nil && gb.tournament.CurrentRound > 2 {
		roundEnhancement = 0.02 * float64(gb.tournament.CurrentRound-2)
	}

	totalCorrelation := baseCorrelation + weatherEnhancement + roundEnhancement
	return math.Min(0.25, totalCorrelation) // Cap at 25%
}
