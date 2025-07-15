package optimizer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// TournamentStateTracker tracks live tournament state and provides late swap recommendations
type TournamentStateTracker struct {
	tournamentID      string
	currentState      *types.TournamentState
	weatherService    WeatherServiceInterface
	cutProbEngine     *CutProbabilityEngine
	logger            *logrus.Logger
	updateInterval    time.Duration
	lastUpdate        time.Time
	mu                sync.RWMutex
	subscribers       map[string]chan *types.TournamentState
	stopChan          chan struct{}
	isTracking        bool
}

// PlayerStateUpdate represents a live update for a player's state
type PlayerStateUpdate struct {
	PlayerID        string    `json:"player_id"`
	CurrentPosition int       `json:"current_position"`
	TotalScore      int       `json:"total_score"`
	ThruHoles       int       `json:"thru_holes"`
	RoundScore      int       `json:"round_score"`
	Status          string    `json:"status"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WeatherStateUpdate represents a live weather update
type WeatherStateUpdate struct {
	CourseID       string                     `json:"course_id"`
	Conditions     *types.WeatherConditions   `json:"conditions"`
	Impact         *types.WeatherImpact       `json:"impact"`
	ChangesSince   time.Time                  `json:"changes_since"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

// LateSwapAnalysis represents analysis for late swap decisions
type LateSwapAnalysis struct {
	TournamentID     string                           `json:"tournament_id"`
	AnalysisTime     time.Time                        `json:"analysis_time"`
	Recommendations  []types.LateSwapRecommendation   `json:"recommendations"`
	WeatherChanges   *WeatherStateUpdate              `json:"weather_changes,omitempty"`
	CutLineUpdates   *CutLineUpdate                   `json:"cut_line_updates,omitempty"`
	PlayerUpdates    []PlayerStateUpdate              `json:"player_updates"`
	TimeToDeadline   time.Duration                    `json:"time_to_deadline"`
}

// CutLineUpdate represents changes to the projected cut line
type CutLineUpdate struct {
	CurrentCutLine      int       `json:"current_cut_line"`
	ProjectedCutLine    int       `json:"projected_cut_line"`
	PlayersOnBubble     []string  `json:"players_on_bubble"`
	CutLineProbability  float64   `json:"cut_line_probability"`
	LastUpdated         time.Time `json:"last_updated"`
}

// NewTournamentStateTracker creates a new tournament state tracker
func NewTournamentStateTracker(
	tournamentID string,
	weatherService WeatherServiceInterface,
	cutProbEngine *CutProbabilityEngine,
	logger *logrus.Logger,
) *TournamentStateTracker {
	return &TournamentStateTracker{
		tournamentID:   tournamentID,
		weatherService: weatherService,
		cutProbEngine:  cutProbEngine,
		logger:         logger,
		updateInterval: 2 * time.Minute, // Update every 2 minutes during live tournaments
		subscribers:    make(map[string]chan *types.TournamentState),
		stopChan:       make(chan struct{}),
		isTracking:     false,
	}
}

// StartTracking begins live tournament tracking
func (tst *TournamentStateTracker) StartTracking(ctx context.Context) error {
	tst.mu.Lock()
	defer tst.mu.Unlock()

	if tst.isTracking {
		return fmt.Errorf("tournament tracking already active")
	}

	tst.isTracking = true
	tst.logger.WithField("tournament_id", tst.tournamentID).Info("Starting tournament state tracking")

	// Initialize current state
	if err := tst.updateTournamentState(ctx); err != nil {
		tst.isTracking = false
		return fmt.Errorf("failed to initialize tournament state: %w", err)
	}

	// Start update goroutine
	go tst.trackingLoop(ctx)

	return nil
}

// StopTracking stops live tournament tracking
func (tst *TournamentStateTracker) StopTracking() {
	tst.mu.Lock()
	defer tst.mu.Unlock()

	if !tst.isTracking {
		return
	}

	tst.logger.WithField("tournament_id", tst.tournamentID).Info("Stopping tournament state tracking")
	
	close(tst.stopChan)
	tst.isTracking = false

	// Close all subscriber channels
	for subID, ch := range tst.subscribers {
		close(ch)
		delete(tst.subscribers, subID)
	}
}

// GetCurrentState returns the current tournament state
func (tst *TournamentStateTracker) GetCurrentState() *types.TournamentState {
	tst.mu.RLock()
	defer tst.mu.RUnlock()

	if tst.currentState == nil {
		return nil
	}

	// Return a copy to prevent modification
	stateCopy := *tst.currentState
	return &stateCopy
}

// Subscribe subscribes to tournament state updates
func (tst *TournamentStateTracker) Subscribe(subscriberID string) <-chan *types.TournamentState {
	tst.mu.Lock()
	defer tst.mu.Unlock()

	ch := make(chan *types.TournamentState, 10) // Buffered channel
	tst.subscribers[subscriberID] = ch

	tst.logger.WithFields(logrus.Fields{
		"tournament_id": tst.tournamentID,
		"subscriber_id": subscriberID,
	}).Info("New subscriber added to tournament tracking")

	return ch
}

// Unsubscribe unsubscribes from tournament state updates
func (tst *TournamentStateTracker) Unsubscribe(subscriberID string) {
	tst.mu.Lock()
	defer tst.mu.Unlock()

	if ch, exists := tst.subscribers[subscriberID]; exists {
		close(ch)
		delete(tst.subscribers, subscriberID)
		tst.logger.WithField("subscriber_id", subscriberID).Info("Subscriber removed from tournament tracking")
	}
}

// GenerateLateSwapRecommendations generates late swap recommendations based on current state
func (tst *TournamentStateTracker) GenerateLateSwapRecommendations(
	ctx context.Context,
	currentLineup []string,
	strategy types.TournamentPositionStrategy,
	swapDeadline time.Time,
	maxSwaps int,
) (*LateSwapAnalysis, error) {
	tst.mu.RLock()
	state := tst.currentState
	tst.mu.RUnlock()

	if state == nil {
		return nil, fmt.Errorf("tournament state not available")
	}

	analysis := &LateSwapAnalysis{
		TournamentID:   tst.tournamentID,
		AnalysisTime:   time.Now(),
		TimeToDeadline: swapDeadline.Sub(time.Now()),
		Recommendations: make([]types.LateSwapRecommendation, 0),
	}

	// Check if swap deadline has passed
	if analysis.TimeToDeadline <= 0 {
		return analysis, nil // No swaps possible
	}

	tst.logger.WithFields(logrus.Fields{
		"tournament_id":  tst.tournamentID,
		"lineup_size":    len(currentLineup),
		"strategy":       strategy,
		"time_remaining": analysis.TimeToDeadline.Minutes(),
	}).Info("Generating late swap recommendations")

	// Analyze weather changes
	if weatherUpdate := tst.analyzeWeatherChanges(ctx); weatherUpdate != nil {
		analysis.WeatherChanges = weatherUpdate
		recommendations := tst.generateWeatherBasedSwaps(currentLineup, weatherUpdate, strategy, maxSwaps)
		analysis.Recommendations = append(analysis.Recommendations, recommendations...)
	}

	// Analyze cut line changes
	if cutUpdate := tst.analyzeCutLineChanges(ctx, state); cutUpdate != nil {
		analysis.CutLineUpdates = cutUpdate
		recommendations := tst.generateCutLineBasedSwaps(currentLineup, cutUpdate, strategy, maxSwaps)
		analysis.Recommendations = append(analysis.Recommendations, recommendations...)
	}

	// Analyze player performance changes
	playerUpdates := tst.analyzePlayerPerformanceChanges(currentLineup)
	analysis.PlayerUpdates = playerUpdates
	recommendations := tst.generatePerformanceBasedSwaps(currentLineup, playerUpdates, strategy, maxSwaps)
	analysis.Recommendations = append(analysis.Recommendations, recommendations...)

	// Sort recommendations by impact score
	sort.Slice(analysis.Recommendations, func(i, j int) bool {
		return analysis.Recommendations[i].ImpactScore > analysis.Recommendations[j].ImpactScore
	})

	// Limit to maxSwaps
	if len(analysis.Recommendations) > maxSwaps {
		analysis.Recommendations = analysis.Recommendations[:maxSwaps]
	}

	tst.logger.WithFields(logrus.Fields{
		"recommendations_generated": len(analysis.Recommendations),
		"weather_changes":          analysis.WeatherChanges != nil,
		"cut_line_changes":         analysis.CutLineUpdates != nil,
		"player_updates":           len(analysis.PlayerUpdates),
	}).Info("Late swap analysis completed")

	return analysis, nil
}

// trackingLoop runs the continuous tracking loop
func (tst *TournamentStateTracker) trackingLoop(ctx context.Context) {
	ticker := time.NewTicker(tst.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			tst.logger.Info("Tournament tracking stopped due to context cancellation")
			return
		case <-tst.stopChan:
			tst.logger.Info("Tournament tracking stopped")
			return
		case <-ticker.C:
			if err := tst.updateTournamentState(ctx); err != nil {
				tst.logger.WithError(err).Error("Failed to update tournament state")
			}
		}
	}
}

// updateTournamentState updates the current tournament state
func (tst *TournamentStateTracker) updateTournamentState(ctx context.Context) error {
	// This would fetch live tournament data from external APIs
	// For now, create a mock update
	newState := &types.TournamentState{
		TournamentID:     tst.tournamentID,
		CurrentRound:     2, // Example: Second round
		CutLine:          2,
		ProjectedCutLine: 1,
		PlayersActive:    144,
		PlayersWithdrawn: 6,
		LastUpdate:       time.Now().Format(time.RFC3339),
	}

	// Get weather update
	if tst.weatherService != nil {
		if weather, err := tst.weatherService.GetWeatherConditions(ctx, "course_id"); err == nil {
			newState.WeatherUpdate = weather
		}
	}

	tst.mu.Lock()
	oldState := tst.currentState
	tst.currentState = newState
	tst.lastUpdate = time.Now()
	tst.mu.Unlock()

	// Notify subscribers if state changed significantly
	if tst.hasSignificantStateChange(oldState, newState) {
		tst.notifySubscribers(newState)
	}

	return nil
}

// hasSignificantStateChange checks if the state change is significant enough to notify
func (tst *TournamentStateTracker) hasSignificantStateChange(oldState, newState *types.TournamentState) bool {
	if oldState == nil {
		return true // First update is always significant
	}

	// Check for significant changes
	return oldState.CutLine != newState.CutLine ||
		oldState.ProjectedCutLine != newState.ProjectedCutLine ||
		oldState.CurrentRound != newState.CurrentRound ||
		math.Abs(float64(oldState.PlayersActive-newState.PlayersActive)) > 5
}

// notifySubscribers notifies all subscribers of state changes
func (tst *TournamentStateTracker) notifySubscribers(state *types.TournamentState) {
	tst.mu.RLock()
	defer tst.mu.RUnlock()

	for subID, ch := range tst.subscribers {
		select {
		case ch <- state:
			// Successfully sent
		default:
			// Channel full, skip this update
			tst.logger.WithField("subscriber_id", subID).Warn("Subscriber channel full, skipping update")
		}
	}
}

// analyzeWeatherChanges analyzes recent weather changes
func (tst *TournamentStateTracker) analyzeWeatherChanges(ctx context.Context) *WeatherStateUpdate {
	if tst.weatherService == nil {
		return nil
	}

	// Get current weather and compare with previous conditions
	// This is a simplified implementation
	weather, err := tst.weatherService.GetWeatherConditions(ctx, "course_id")
	if err != nil {
		return nil
	}

	impact := tst.weatherService.CalculateGolfImpact(weather)

	return &WeatherStateUpdate{
		CourseID:     "course_id",
		Conditions:   weather,
		Impact:       impact,
		ChangesSince: time.Now().Add(-tst.updateInterval),
		UpdatedAt:    time.Now(),
	}
}

// analyzeCutLineChanges analyzes changes to the cut line projection
func (tst *TournamentStateTracker) analyzeCutLineChanges(ctx context.Context, state *types.TournamentState) *CutLineUpdate {
	// This would analyze actual tournament data to project cut line changes
	// For now, return a mock update
	return &CutLineUpdate{
		CurrentCutLine:     state.CutLine,
		ProjectedCutLine:   state.ProjectedCutLine,
		PlayersOnBubble:    []string{"player1", "player2", "player3"},
		CutLineProbability: 0.75,
		LastUpdated:        time.Now(),
	}
}

// analyzePlayerPerformanceChanges analyzes performance changes for lineup players
func (tst *TournamentStateTracker) analyzePlayerPerformanceChanges(lineup []string) []PlayerStateUpdate {
	// This would fetch live scoring data for lineup players
	// For now, return mock updates
	updates := make([]PlayerStateUpdate, 0, len(lineup))
	
	for i, playerID := range lineup {
		update := PlayerStateUpdate{
			PlayerID:        playerID,
			CurrentPosition: 15 + i*5, // Mock positions
			TotalScore:      -2 - i,   // Mock scores
			ThruHoles:       9,
			RoundScore:      -1,
			Status:          "active",
			UpdatedAt:       time.Now(),
		}
		updates = append(updates, update)
	}
	
	return updates
}

// generateWeatherBasedSwaps generates swap recommendations based on weather changes
func (tst *TournamentStateTracker) generateWeatherBasedSwaps(
	lineup []string,
	weatherUpdate *WeatherStateUpdate,
	strategy types.TournamentPositionStrategy,
	maxSwaps int,
) []types.LateSwapRecommendation {
	recommendations := make([]types.LateSwapRecommendation, 0)

	// Example weather-based swap logic
	if weatherUpdate.Impact.WindAdvantage < -0.1 {
		// High wind conditions - recommend swapping to better wind players
		recommendation := types.LateSwapRecommendation{
			PlayerOut:        lineup[0], // Example
			PlayerIn:         "wind-specialist-player",
			ReasonCode:       "WEATHER_WIND",
			Reasoning:        "High wind conditions favor players with superior ball-striking skills",
			ImpactScore:      0.85,
			Confidence:       0.75,
			SwapDeadline:     time.Now().Add(15 * time.Minute).Format(time.RFC3339),
			WeatherRelated:   true,
			TeeTimeAdvantage: false,
		}
		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

// generateCutLineBasedSwaps generates swaps based on cut line changes
func (tst *TournamentStateTracker) generateCutLineBasedSwaps(
	lineup []string,
	cutUpdate *CutLineUpdate,
	strategy types.TournamentPositionStrategy,
	maxSwaps int,
) []types.LateSwapRecommendation {
	recommendations := make([]types.LateSwapRecommendation, 0)

	// If cut line moved, recommend swapping players unlikely to make cut
	if cutUpdate.ProjectedCutLine > cutUpdate.CurrentCutLine {
		recommendation := types.LateSwapRecommendation{
			PlayerOut:        lineup[len(lineup)-1], // Example: last player
			PlayerIn:         "safer-cut-player",
			ReasonCode:       "CUT_LINE_MOVE",
			Reasoning:        "Cut line projection moved higher, favoring safer players",
			ImpactScore:      0.70,
			Confidence:       0.80,
			SwapDeadline:     time.Now().Add(20 * time.Minute).Format(time.RFC3339),
			WeatherRelated:   false,
			TeeTimeAdvantage: false,
		}
		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

// generatePerformanceBasedSwaps generates swaps based on live performance
func (tst *TournamentStateTracker) generatePerformanceBasedSwaps(
	lineup []string,
	playerUpdates []PlayerStateUpdate,
	strategy types.TournamentPositionStrategy,
	maxSwaps int,
) []types.LateSwapRecommendation {
	recommendations := make([]types.LateSwapRecommendation, 0)

	// Analyze each player's performance
	for _, update := range playerUpdates {
		if update.CurrentPosition > 80 && strategy == types.CutStrategy {
			// Player struggling and strategy prioritizes making cut
			recommendation := types.LateSwapRecommendation{
				PlayerOut:        update.PlayerID,
				PlayerIn:         "consistent-performer",
				ReasonCode:       "POOR_PERFORMANCE",
				Reasoning:        "Player struggling and unlikely to make cut given current position",
				ImpactScore:      0.65,
				Confidence:       0.70,
				SwapDeadline:     time.Now().Add(10 * time.Minute).Format(time.RFC3339),
				WeatherRelated:   false,
				TeeTimeAdvantage: false,
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}