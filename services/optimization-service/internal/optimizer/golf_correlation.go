package optimizer

import (
	"math"
	"strings"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
)

// GolfCorrelationBuilder builds correlation matrices specific to golf
type GolfCorrelationBuilder struct {
	players       []models.Player
	entries       map[uint]*models.GolfPlayerEntry
	courseHistory map[uint]*models.GolfCourseHistory
}

// NewGolfCorrelationBuilder creates a new golf correlation builder
func NewGolfCorrelationBuilder(players []models.Player) *GolfCorrelationBuilder {
	return &GolfCorrelationBuilder{
		players:       players,
		entries:       make(map[uint]*models.GolfPlayerEntry),
		courseHistory: make(map[uint]*models.GolfCourseHistory),
	}
}

// SetPlayerEntries sets the golf-specific player entry data
func (gb *GolfCorrelationBuilder) SetPlayerEntries(entries []models.GolfPlayerEntry) {
	for _, entry := range entries {
		gb.entries[entry.PlayerID] = &entry
	}
}

// SetCourseHistory sets the course history data for players
func (gb *GolfCorrelationBuilder) SetCourseHistory(history []models.GolfCourseHistory) {
	for _, h := range history {
		gb.courseHistory[h.PlayerID] = &h
	}
}

// BuildCorrelationMatrix builds the correlation matrix for golf
func (gb *GolfCorrelationBuilder) BuildCorrelationMatrix() map[uint]map[uint]float64 {
	correlations := make(map[uint]map[uint]float64)

	for i, p1 := range gb.players {
		correlations[p1.ID] = make(map[uint]float64)

		for j, p2 := range gb.players {
			if i == j {
				correlations[p1.ID][p2.ID] = 1.0
				continue
			}

			correlation := gb.calculateGolfCorrelation(p1, p2)
			correlations[p1.ID][p2.ID] = correlation
		}
	}

	return correlations
}

// calculateGolfCorrelation calculates correlation between two golf players
func (gb *GolfCorrelationBuilder) calculateGolfCorrelation(p1, p2 models.Player) float64 {
	correlation := 0.0

	// Get player entries if available
	entry1 := gb.entries[p1.ID]
	entry2 := gb.entries[p2.ID]

	if entry1 != nil && entry2 != nil {
		// Same tee time correlation (playing partners)
		if gb.haveSameTeeTime(entry1, entry2) {
			correlation += 0.15
		}

		// Same wave (AM/PM) correlation
		if gb.inSameWave(entry1, entry2) {
			correlation += 0.05
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
	if gb.similarCourseHistory(p1.ID, p2.ID) {
		correlation += 0.06
	}

	// Cap correlation between -0.2 and 0.5 for golf
	return math.Max(-0.2, math.Min(0.5, correlation))
}

// haveSameTeeTime checks if two players have the same tee time
func (gb *GolfCorrelationBuilder) haveSameTeeTime(e1, e2 *models.GolfPlayerEntry) bool {
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
func (gb *GolfCorrelationBuilder) inSameWave(e1, e2 *models.GolfPlayerEntry) bool {
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
func (gb *GolfCorrelationBuilder) areSimilarPosition(e1, e2 *models.GolfPlayerEntry) bool {
	if e1.CurrentPosition == 0 || e2.CurrentPosition == 0 {
		return false
	}

	diff := math.Abs(float64(e1.CurrentPosition - e2.CurrentPosition))
	return diff <= 10 // Within 10 positions
}

// sameCountry checks if players are from the same country
func (gb *GolfCorrelationBuilder) sameCountry(p1, p2 models.Player) bool {
	// In golf, Team often represents country
	return p1.Team == p2.Team && p1.Team != ""
}

// similarSkillLevel checks if players have similar skill levels based on salary
func (gb *GolfCorrelationBuilder) similarSkillLevel(p1, p2 models.Player) bool {
	salaryDiff := math.Abs(float64(p1.Salary - p2.Salary))
	avgSalary := float64(p1.Salary+p2.Salary) / 2

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
func (cm *CorrelationMatrix) getGolfTeammateCorrelation(pos1, pos2 string) float64 {
	// In golf, all players are position "G"
	if pos1 == "G" && pos2 == "G" {
		return 0.10 // Small positive correlation for same country
	}
	return 0.0
}

// GetGolfOpponentCorrelation returns correlation for golf opponents
func (cm *CorrelationMatrix) getGolfOpponentCorrelation(pos1, pos2 string) float64 {
	// Golf doesn't have direct opponents like team sports
	return 0.0
}

// getGolfStackingBonus returns bonus correlation for golf stacking strategies
func getGolfStackingBonus(p1, p2 models.Player) float64 {
	bonus := 0.0

	// Ownership-based negative correlation stacking
	// If one player is highly owned and another is low owned
	ownership1 := p1.Ownership
	ownership2 := p2.Ownership

	if (ownership1 > 20 && ownership2 < 10) || (ownership1 < 10 && ownership2 > 20) {
		bonus += 0.05 // Encourage contrarian plays
	}

	// Price-based stacking (stars and scrubs)
	if (p1.Salary > 10000 && p2.Salary < 7000) || (p1.Salary < 7000 && p2.Salary > 10000) {
		bonus += 0.03
	}

	return bonus
}

// Helper function to parse player metadata for golf-specific data
func getGolfMetadata(player models.Player, key string) string {
	// For now, return empty string as we don't have metadata storage
	// TODO: Implement proper metadata storage for golf-specific data
	return ""
}

// Helper function to check if string contains golf position
func isGolfPosition(position string) bool {
	golfPos := strings.ToUpper(position)
	return golfPos == "G" || golfPos == "GOLF" || golfPos == "GOLFER"
}
