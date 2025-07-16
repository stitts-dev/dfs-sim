package simulator

import (
	"hash/crc32"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/optimizer"
)

// SimulationConfig represents configuration for Monte Carlo simulation
type SimulationConfig struct {
	NumSimulations     int
	SimulationWorkers  int
	UseCorrelations    bool
	ContestSize        int
	EntryFee          float64
	PayoutStructure   []PayoutTier
}

// SimulationRun represents a single simulation run
type SimulationRun struct {
	LineupScore  float64
	PlayerScores map[uuid.UUID]float64
	Rank         int
	Percentile   float64
	Payout       float64
}

// SimulationResult represents the aggregate results of multiple simulation runs
type SimulationResult struct {
	LineupID           string
	NumSimulations     int
	Mean               float64
	Median             float64
	StandardDeviation  float64
	Min                float64
	Max                float64
	Percentile25       float64
	Percentile75       float64
	Percentile90       float64
	Percentile95       float64
	Percentile99       float64
	TopPercentFinishes map[string]float64
	WinProbability     float64
	CashProbability    float64
	ROI                float64
}

// SimulationProgress represents progress of a simulation
type SimulationProgress struct {
	LineupID               string
	TotalSimulations       int
	Completed              int
	StartTime              time.Time
	EstimatedTimeRemaining time.Duration
}

// Simulator runs Monte Carlo simulations for lineups
type Simulator struct {
	config       *SimulationConfig
	correlations *optimizer.CorrelationMatrix
	rng          *rand.Rand
	mu           sync.Mutex
}

// NewSimulator creates a new Monte Carlo simulator
func NewSimulator(config *SimulationConfig, players []types.Player) *Simulator {
	// Convert to OptimizationPlayer for correlation matrix
	optimizationPlayers := convertPlayersToOptimizationMC(players)
	
	return &Simulator{
		config:       config,
		correlations: optimizer.NewCorrelationMatrix(optimizationPlayers),
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SimulateContest runs Monte Carlo simulations for a contest
func (s *Simulator) SimulateContest(lineups []types.GeneratedLineup, progressChan chan<- SimulationProgress) (*SimulationResult, error) {
	numWorkers := runtime.NumCPU()
	if s.config.SimulationWorkers > 0 {
		numWorkers = s.config.SimulationWorkers
	}

	// Create channels for work distribution
	simulationsChan := make(chan int, s.config.NumSimulations)
	resultsChan := make(chan SimulationRun, s.config.NumSimulations)

	// Start progress reporter if channel provided
	if progressChan != nil {
		go s.reportProgress(lineups[0].ID, progressChan, resultsChan)
	}

	// Start worker goroutines
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go s.simulationWorker(lineups, simulationsChan, resultsChan, &wg)
	}

	// Queue simulations
	for i := 0; i < s.config.NumSimulations; i++ {
		simulationsChan <- i
	}
	close(simulationsChan)

	// Wait for all workers to complete
	wg.Wait()
	close(resultsChan)

	// Aggregate results
	return s.aggregateResults(lineups[0], resultsChan), nil
}

func (s *Simulator) simulationWorker(lineups []types.GeneratedLineup, simChan <-chan int, resultsChan chan<- SimulationRun, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create local RNG for this worker
	localRng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for simNum := range simChan {
		// Generate player outcomes for this simulation
		_ = simNum // Mark as used
		playerOutcomes := s.generatePlayerOutcomes(lineups[0].Players, localRng)

		// Calculate lineup scores
		lineupScores := make([]float64, len(lineups))
		for i, lineup := range lineups {
			score := 0.0
			for _, player := range lineup.Players {
				score += playerOutcomes[player.ID]
			}
			lineupScores[i] = score
		}

		// Determine ranks and payouts
		ranks := s.calculateRanks(lineupScores)

		// Store result for primary lineup
		result := SimulationRun{
			LineupScore:  lineupScores[0],
			PlayerScores: playerOutcomes,
			Rank:         ranks[0],
			// Percentile:   CalculatePercentile(lineupScores[0], lineupScores),
			// Payout:       GetPayoutForRank(ranks[0], s.config.PayoutStructure),
		}

		resultsChan <- result
	}
}

func (s *Simulator) generatePlayerOutcomes(players []types.LineupPlayer, rng *rand.Rand) map[uuid.UUID]float64 {
	outcomes := make(map[uuid.UUID]float64)

	if s.config.UseCorrelations {
		// Generate correlated outcomes
		outcomes = s.generateCorrelatedOutcomes(players, rng)
	} else {
		// Generate independent outcomes
		for _, player := range players {
			outcomes[player.ID] = s.generatePlayerScore(player, rng)
		}
	}

	return outcomes
}

func (s *Simulator) generateCorrelatedOutcomes(players []types.LineupPlayer, rng *rand.Rand) map[uuid.UUID]float64 {
	n := len(players)
	outcomes := make(map[uuid.UUID]float64)

	// Generate base scores
	baseScores := make([]float64, n)
	for i, player := range players {
		baseScores[i] = s.generatePlayerScore(player, rng)
	}

	// Apply correlations using Cholesky decomposition approximation
	// For simplicity, we'll use a simpler approach here
	for i, player1 := range players {
		adjustedScore := baseScores[i]

		// Apply correlation adjustments
		for j, player2 := range players {
			if i != j {
				// Convert UUID to uint for correlation lookup
				// TODO: This is a temporary solution - ideally correlation matrix should use UUIDs
				player1Hash := hashUUID(player1.ID)
				player2Hash := hashUUID(player2.ID)
				correlation := s.correlations.GetCorrelation(player1Hash, player2Hash)
				if correlation != 0 {
					// Adjust score based on correlation and other player's performance
					deviation := (baseScores[j] - player2.ProjectedPoints) / player2.ProjectedPoints
					adjustment := correlation * deviation * player1.ProjectedPoints * 0.1
					adjustedScore += adjustment
				}
			}
		}

		// Ensure score stays within reasonable bounds
		adjustedScore = math.Max(0, adjustedScore)
		adjustedScore = math.Min(player1.ProjectedPoints*1.8, adjustedScore)

		outcomes[player1.ID] = adjustedScore
	}

	return outcomes
}

func (s *Simulator) generatePlayerScore(player types.LineupPlayer, rng *rand.Rand) float64 {
	// Calculate standard deviation from player's projected points
	// Since LineupPlayer doesn't have ceiling/floor, use approximation
	stdDev := player.ProjectedPoints * 0.25 // 25% variance

	// Generate score using normal distribution
	score := rng.NormFloat64()*stdDev + player.ProjectedPoints

	// Apply approximate floor and ceiling constraints
	minScore := player.ProjectedPoints * 0.3
	maxScore := player.ProjectedPoints * 1.8

	// Small chance of injury/DNP
	if rng.Float64() < 0.02 { // 2% chance
		return 0
	}

	// Ensure score is within bounds
	score = math.Max(minScore, score)
	score = math.Min(maxScore, score)

	return score
}

func (s *Simulator) calculateRanks(scores []float64) []int {
	// Create indexed scores
	type indexedScore struct {
		index int
		score float64
	}

	indexed := make([]indexedScore, len(scores))
	for i, score := range scores {
		indexed[i] = indexedScore{index: i, score: score}
	}

	// Sort by score descending
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].score > indexed[j].score
	})

	// Assign ranks
	ranks := make([]int, len(scores))
	for rank, item := range indexed {
		ranks[item.index] = rank + 1
	}

	return ranks
}

// hashUUID converts a UUID to a uint32 for compatibility with correlation matrix
// TODO: CRITICAL - This is a temporary hash-based solution with significant limitations:
// 1. Hash collisions can cause incorrect correlations between different players
// 2. Correlation matrix should be refactored to use UUID keys instead of uint
// 3. This breaks historical correlation data that used sequential uint IDs
// 4. Performance impact from hash computation on every correlation lookup
// 
// PROPER SOLUTION: Update CorrelationMatrix struct to use map[uuid.UUID]map[uuid.UUID]float64
// and migrate existing correlation data to use UUID keys
// TEMP FIX: Using CRC32 hash for compilation compatibility only
func hashUUID(id uuid.UUID) uint {
	return uint(crc32.ChecksumIEEE(id[:]))
}

func (s *Simulator) aggregateResults(lineup types.GeneratedLineup, resultsChan <-chan SimulationRun) *SimulationResult {
	scores := make([]float64, 0, s.config.NumSimulations)
	ranks := make([]int, 0, s.config.NumSimulations)
	payouts := make([]float64, 0, s.config.NumSimulations)

	// Collect all results
	for result := range resultsChan {
		scores = append(scores, result.LineupScore)
		ranks = append(ranks, result.Rank)
		payouts = append(payouts, result.Payout)
	}

	// Sort scores for percentile calculation
	sort.Float64s(scores)

	// Calculate statistics
	result := &SimulationResult{
		LineupID:           lineup.ID,
		// ContestID:          lineup.ContestID, // ContestID not in GeneratedLineup
		NumSimulations:     len(scores),
		Mean:               calculateMean(scores),
		Median:             calculateMedian(scores),
		StandardDeviation:  calculateStdDev(scores),
		Min:                scores[0],
		Max:                scores[len(scores)-1],
		Percentile25:       calculatePercentile(scores, 25),
		Percentile75:       calculatePercentile(scores, 75),
		Percentile90:       calculatePercentile(scores, 90),
		Percentile95:       calculatePercentile(scores, 95),
		Percentile99:       calculatePercentile(scores, 99),
		TopPercentFinishes: calculateTopPercentFinishes(ranks, s.config.ContestSize),
		WinProbability:     calculateWinProbability(ranks),
		CashProbability:    calculateCashProbability(ranks, s.config.ContestSize),
		ROI:                calculateROI(payouts, s.config.EntryFee),
	}

	return result
}

func (s *Simulator) reportProgress(lineupID string, progressChan chan<- SimulationProgress, resultsChan <-chan SimulationRun) {
	startTime := time.Now()
	completed := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-resultsChan:
			completed++
		case <-ticker.C:
			if completed > 0 {
				elapsed := time.Since(startTime)
				rate := float64(completed) / elapsed.Seconds()
				remaining := s.config.NumSimulations - completed
				eta := time.Duration(float64(remaining)/rate) * time.Second

				progress := SimulationProgress{
					LineupID:               lineupID,
					TotalSimulations:       s.config.NumSimulations,
					Completed:              completed,
					StartTime:              startTime,
					EstimatedTimeRemaining: eta,
				}

				select {
				case progressChan <- progress:
				default:
					// Don't block if channel is full
				}
			}
		}

		if completed >= s.config.NumSimulations {
			break
		}
	}
}

// Statistical helper functions

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateMedian(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func calculateStdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func calculatePercentile(sorted []float64, percentile int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(percentile) / 100.0 * float64(len(sorted)-1))
	return sorted[index]
}

func calculateTopPercentFinishes(ranks []int, contestSize int) map[string]float64 {
	finishes := map[string]float64{
		"top_1":  0,
		"top_10": 0,
		"top_20": 0,
		"top_50": 0,
	}

	for _, rank := range ranks {
		percentile := float64(rank) / float64(contestSize) * 100
		if percentile <= 1 {
			finishes["top_1"]++
		}
		if percentile <= 10 {
			finishes["top_10"]++
		}
		if percentile <= 20 {
			finishes["top_20"]++
		}
		if percentile <= 50 {
			finishes["top_50"]++
		}
	}

	// Convert to percentages
	n := float64(len(ranks))
	for k := range finishes {
		finishes[k] = finishes[k] / n * 100
	}

	return finishes
}

func calculateWinProbability(ranks []int) float64 {
	wins := 0
	for _, rank := range ranks {
		if rank == 1 {
			wins++
		}
	}
	return float64(wins) / float64(len(ranks)) * 100
}

func calculateCashProbability(ranks []int, contestSize int) float64 {
	// Assuming top 20% cash in GPP
	cashLine := int(float64(contestSize) * 0.2)
	cashes := 0
	for _, rank := range ranks {
		if rank <= cashLine {
			cashes++
		}
	}
	return float64(cashes) / float64(len(ranks)) * 100
}

func calculateROI(payouts []float64, entryFee float64) float64 {
	if entryFee == 0 {
		return 0
	}
	avgPayout := calculateMean(payouts)
	return (avgPayout - entryFee) / entryFee * 100
}

// convertPlayersToOptimizationMC converts types.Player slice to optimizer.OptimizationPlayer slice
func convertPlayersToOptimizationMC(players []types.Player) []optimizer.OptimizationPlayer {
	result := make([]optimizer.OptimizationPlayer, len(players))
	for i, p := range players {
		result[i] = optimizer.OptimizationPlayer{
			ID:              p.ID,
			ExternalID:      p.ExternalID,
			Name:            p.Name,
			Team:            getStringValueMC(p.Team),
			Opponent:        getStringValueMC(p.Opponent),
			Position:        getStringValueMC(p.Position),
			SalaryDK:        getIntValueMC(p.SalaryDK),
			SalaryFD:        getIntValueMC(p.SalaryFD),
			ProjectedPoints: getFloatValueMC(p.ProjectedPoints),
			FloorPoints:     getFloatValueMC(p.FloorPoints),
			CeilingPoints:   getFloatValueMC(p.CeilingPoints),
			OwnershipDK:     getFloatValueMC(p.OwnershipDK),
			OwnershipFD:     getFloatValueMC(p.OwnershipFD),
			GameTime:        getTimeValueMC(p.GameTime),
			IsInjured:       getBoolValueMC(p.IsInjured),
			InjuryStatus:    getStringValueMC(p.InjuryStatus),
			ImageURL:        getStringValueMC(p.ImageURL),
			TeeTime:         "",
			CutProbability:  0.0,
			CreatedAt:       p.CreatedAt,
			UpdatedAt:       p.UpdatedAt,
		}
	}
	return result
}

// Helper functions to safely extract values from pointers
func getStringValueMC(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getIntValueMC(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getFloatValueMC(ptr *float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return 0.0
}

func getBoolValueMC(ptr *bool) bool {
	if ptr != nil {
		return *ptr
	}
	return false
}

func getTimeValueMC(ptr *time.Time) time.Time {
	if ptr != nil {
		return *ptr
	}
	return time.Time{}
}
