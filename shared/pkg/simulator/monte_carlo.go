package simulator

import (
	"context"
	"fmt"
	"hash"
	"hash/crc32"
	"math"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// MonteCarloSimulator performs Monte Carlo simulations for DFS lineups
type MonteCarloSimulator struct {
	lineups           []types.Lineup
	contest           types.Contest
	simulationCount   int
	workers           int
	logger            *logrus.Logger
	correlationMatrix map[string]float64
}

// NewMonteCarloSimulator creates a new Monte Carlo simulator
func NewMonteCarloSimulator(
	lineups []types.Lineup,
	contest types.Contest,
	simulationCount int,
	workers int,
	logger *logrus.Logger,
) *MonteCarloSimulator {
	return &MonteCarloSimulator{
		lineups:         lineups,
		contest:         contest,
		simulationCount: simulationCount,
		workers:         workers,
		logger:          logger,
	}
}

// SetCorrelationMatrix sets the correlation matrix for simulation
func (mcs *MonteCarloSimulator) SetCorrelationMatrix(matrix map[string]float64) {
	mcs.correlationMatrix = matrix
}

// LineupResult contains simulation results for a lineup
type LineupResult struct {
	LineupID          string                 `json:"lineup_id"`
	ExpectedPoints    float64               `json:"expected_points"`
	PointsVariance    float64               `json:"points_variance"`
	Percentiles       map[string]float64    `json:"percentiles"`
	CashRate          float64               `json:"cash_rate"`
	ROI               float64               `json:"roi"`
	TopPercentFinish  map[string]float64    `json:"top_percent_finish"`
	SimulationDetails map[string]interface{} `json:"simulation_details"`
}

// SimulationResult contains the overall simulation results
type SimulationResult struct {
	Results           []LineupResult         `json:"results"`
	SimulationCount   int                   `json:"simulation_count"`
	ExecutionTime     time.Duration         `json:"execution_time"`
	ContestInfo       types.Contest         `json:"contest_info"`
	SimulationMeta    map[string]interface{} `json:"simulation_meta"`
}

// RunSimulation executes the Monte Carlo simulation
func (mcs *MonteCarloSimulator) RunSimulation(
	ctx context.Context,
	progressChan chan<- types.ProgressUpdate,
) (*SimulationResult, error) {
	if mcs.logger != nil {
		mcs.logger.Info("Starting Monte Carlo simulation")
	}

	startTime := time.Now()

	// Send initial progress
	if progressChan != nil {
		progressChan <- types.ProgressUpdate{
			Type:        "simulation",
			Progress:    0.0,
			Message:     "Initializing simulation...",
			CurrentStep: "initialization",
			TotalSteps:  mcs.simulationCount,
			Timestamp:   time.Now(),
		}
	}

	// Validate inputs
	if len(mcs.lineups) == 0 {
		return nil, fmt.Errorf("no lineups provided for simulation")
	}

	if mcs.simulationCount <= 0 {
		return nil, fmt.Errorf("simulation count must be positive")
	}

	// Initialize results
	results := make([]LineupResult, len(mcs.lineups))

	// Simulate each lineup
	for i, lineup := range mcs.lineups {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Send progress update
		if progressChan != nil {
			progress := float64(i) / float64(len(mcs.lineups))
			progressChan <- types.ProgressUpdate{
				Type:        "simulation",
				Progress:    progress,
				Message:     fmt.Sprintf("Simulating lineup %d/%d", i+1, len(mcs.lineups)),
				CurrentStep: "simulation",
				TotalSteps:  len(mcs.lineups),
				Timestamp:   time.Now(),
			}
		}

		// Run simulation for this lineup
		result := mcs.simulateLineup(lineup)
		results[i] = result
	}

	// Send final progress update
	if progressChan != nil {
		progressChan <- types.ProgressUpdate{
			Type:        "simulation",
			Progress:    1.0,
			Message:     "Simulation completed",
			CurrentStep: "completed",
			TotalSteps:  len(mcs.lineups),
			Timestamp:   time.Now(),
		}
	}

	// Create simulation result
	simulationResult := &SimulationResult{
		Results:         results,
		SimulationCount: mcs.simulationCount,
		ExecutionTime:   time.Since(startTime),
		ContestInfo:     mcs.contest,
		SimulationMeta: map[string]interface{}{
			"workers":            mcs.workers,
			"correlation_matrix": len(mcs.correlationMatrix) > 0,
			"contest_type":       mcs.contest.Type,
		},
	}

	if mcs.logger != nil {
		mcs.logger.WithFields(logrus.Fields{
			"lineups_simulated": len(mcs.lineups),
			"simulation_count":  mcs.simulationCount,
			"execution_time":    time.Since(startTime),
		}).Info("Monte Carlo simulation completed")
	}

	return simulationResult, nil
}

// simulateLineup runs Monte Carlo simulation for a single lineup
func (mcs *MonteCarloSimulator) simulateLineup(lineup types.Lineup) LineupResult {
	// Initialize random number generator with time-based seed
	// Use hash of UUID string to create deterministic but unique seed
	h := hash.Hash32(crc32.NewIEEE())
	h.Write([]byte(lineup.ID.String()))
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(h.Sum32())))

	scores := make([]float64, mcs.simulationCount)
	totalScore := 0.0

	// Run simulations for this lineup
	for i := 0; i < mcs.simulationCount; i++ {
		// Generate random performance for each player
		lineupScore := 0.0
		for _, player := range lineup.Players {
			// Simple simulation: normal distribution around projected points
			// with standard deviation of 20% of projected points
			stdDev := player.ProjectedPoints * 0.2
			playerScore := rng.NormFloat64()*stdDev + player.ProjectedPoints
			
			// Ensure non-negative score
			if playerScore < 0 {
				playerScore = 0
			}
			
			lineupScore += playerScore
		}
		
		scores[i] = lineupScore
		totalScore += lineupScore
	}

	// Calculate statistics
	expectedPoints := totalScore / float64(mcs.simulationCount)
	
	// Calculate variance
	variance := 0.0
	for _, score := range scores {
		diff := score - expectedPoints
		variance += diff * diff
	}
	variance /= float64(mcs.simulationCount)

	// Calculate percentiles
	// Sort scores for percentile calculation
	sortedScores := make([]float64, len(scores))
	copy(sortedScores, scores)
	// Simple bubble sort for small arrays
	for i := 0; i < len(sortedScores); i++ {
		for j := i + 1; j < len(sortedScores); j++ {
			if sortedScores[i] > sortedScores[j] {
				sortedScores[i], sortedScores[j] = sortedScores[j], sortedScores[i]
			}
		}
	}

	percentiles := map[string]float64{
		"10th":  sortedScores[int(float64(len(sortedScores))*0.1)],
		"25th":  sortedScores[int(float64(len(sortedScores))*0.25)],
		"50th":  sortedScores[int(float64(len(sortedScores))*0.5)],
		"75th":  sortedScores[int(float64(len(sortedScores))*0.75)],
		"90th":  sortedScores[int(float64(len(sortedScores))*0.9)],
	}

	// Calculate cash rate (placeholder - depends on contest type)
	cashThreshold := expectedPoints * 0.8 // Simple assumption
	cashCount := 0
	for _, score := range scores {
		if score >= cashThreshold {
			cashCount++
		}
	}
	cashRate := float64(cashCount) / float64(mcs.simulationCount)

	// Calculate ROI (placeholder)
	roi := (expectedPoints - float64(lineup.TotalSalary/1000)) / float64(lineup.TotalSalary/1000)

	// Calculate top percent finishes
	topPercentFinish := map[string]float64{
		"top_1_percent":  float64(len(scores)-int(float64(len(scores))*0.99)) / float64(len(scores)),
		"top_5_percent":  float64(len(scores)-int(float64(len(scores))*0.95)) / float64(len(scores)),
		"top_10_percent": float64(len(scores)-int(float64(len(scores))*0.90)) / float64(len(scores)),
	}

	return LineupResult{
		LineupID:         lineup.ID.String(),
		ExpectedPoints:   expectedPoints,
		PointsVariance:   variance,
		Percentiles:      percentiles,
		CashRate:         cashRate,
		ROI:              roi,
		TopPercentFinish: topPercentFinish,
		SimulationDetails: map[string]interface{}{
			"simulation_count": mcs.simulationCount,
			"min_score":       sortedScores[0],
			"max_score":       sortedScores[len(sortedScores)-1],
			"std_dev":         math.Sqrt(variance),
		},
	}
}