package simulator

import (
	"math"
	"math/rand"
	"sort"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// PayoutTier represents a payout tier in a contest
type PayoutTier struct {
	MinRank int
	MaxRank int
	Payout  float64
}

// ContestSimulator simulates entire contest fields
type ContestSimulator struct {
	contest         *types.Contest
	fieldSize       int
	payoutStructure []PayoutTier
	ownershipModel  *OwnershipModel
}

// NewContestSimulator creates a new contest simulator
func NewContestSimulator(contest *types.Contest) *ContestSimulator {
	return &ContestSimulator{
		contest:         contest,
		fieldSize:       contest.TotalEntries,
		payoutStructure: GetPayoutStructure(contest),
		ownershipModel:  NewOwnershipModel(contest.ContestType),
	}
}

// SimulateFullContest simulates an entire contest with all entries
func (cs *ContestSimulator) SimulateFullContest(userLineups []types.GeneratedLineup, players []types.Player, rng *rand.Rand) *ContestResult {
	// Generate field lineups based on ownership
	fieldLineups := cs.generateFieldLineups(players, cs.fieldSize-len(userLineups), rng)

	// Combine user and field lineups
	allLineups := append(userLineups, fieldLineups...)

	// Generate player outcomes
	playerDists := make(map[uuid.UUID]*PlayerDistribution)
	for _, player := range players {
		playerDists[player.ID] = NewPlayerDistribution(player)
	}

	playerOutcomes := make(map[uuid.UUID]float64)
	for _, player := range players {
		playerOutcomes[player.ID] = playerDists[player.ID].Sample(rng)
	}

	// Calculate all lineup scores
	lineupScores := make([]LineupScore, len(allLineups))
	for i, lineup := range allLineups {
		score := 0.0
		for _, player := range lineup.Players {
			score += playerOutcomes[player.ID]
		}

		lineupScores[i] = LineupScore{
			LineupID:    lineup.ID,
			Score:       score,
			IsUserEntry: i < len(userLineups),
		}
	}

	// Sort by score descending
	sort.Slice(lineupScores, func(i, j int) bool {
		return lineupScores[i].Score > lineupScores[j].Score
	})

	// Calculate payouts
	cs.calculatePayouts(lineupScores)

	// Create result
	return &ContestResult{
		Contest:        cs.contest,
		LineupScores:   lineupScores,
		PlayerOutcomes: playerOutcomes,
		UserResults:    cs.extractUserResults(lineupScores, userLineups),
	}
}

func (cs *ContestSimulator) generateFieldLineups(players []types.Player, count int, rng *rand.Rand) []types.GeneratedLineup {
	fieldLineups := make([]types.GeneratedLineup, 0, count)

	// Get ownership percentages
	ownership := cs.ownershipModel.GenerateOwnership(players, rng)

	// Create player pool weighted by ownership
	weightedPool := cs.createWeightedPool(players, ownership)

	for i := 0; i < count; i++ {
		// Generate a lineup using ownership-weighted selection
		lineup := cs.generateSingleLineup(weightedPool, players, rng)
		if lineup != nil {
			fieldLineups = append(fieldLineups, *lineup)
		}
	}

	return fieldLineups
}

func (cs *ContestSimulator) createWeightedPool(players []types.Player, ownership map[uuid.UUID]float64) []weightedPlayer {
	pool := make([]weightedPlayer, 0, len(players))

	for _, player := range players {
		weight := ownership[player.ID]
		if weight > 0 {
			pool = append(pool, weightedPlayer{
				player: player,
				weight: weight,
			})
		}
	}

	return pool
}

func (cs *ContestSimulator) generateSingleLineup(pool []weightedPlayer, allPlayers []types.Player, rng *rand.Rand) *types.GeneratedLineup {
	requirements := cs.contest.PositionRequirements
	lineup := &types.GeneratedLineup{
		Players:   make([]types.LineupPlayer, 0, requirements.GetTotalPlayers()),
	}

	// Try to fill each position
	positionsFilled := make(map[string]int)
	usedPlayers := make(map[uuid.UUID]bool)

	// Multiple attempts to create valid lineup
	for attempt := 0; attempt < 100; attempt++ {
		lineup.Players = lineup.Players[:0]
		lineup.TotalSalary = 0
		lineup.ProjectedPoints = 0
		positionsFilled = make(map[string]int)
		usedPlayers = make(map[uuid.UUID]bool)

		// Fill positions in order
		for position, required := range requirements {
			for i := 0; i < required; i++ {
				player := cs.selectPlayer(pool, position, cs.contest.SalaryCap-lineup.TotalSalary, usedPlayers, rng)
				if player == nil {
					break
				}

				lineupPlayer := types.LineupPlayer{
					ID:              player.ID,
					Name:            player.Name,
					Team:            getStringValueSim(player.Team),
					Position:        getStringValueSim(player.Position),
					Salary:          getIntValueSim(player.SalaryDK),
					ProjectedPoints: getFloatValueSim(player.ProjectedPoints),
				}
				lineup.Players = append(lineup.Players, lineupPlayer)
				lineup.TotalSalary += getIntValueSim(player.SalaryDK)
				lineup.ProjectedPoints += getFloatValueSim(player.ProjectedPoints)
				usedPlayers[player.ID] = true
				positionsFilled[position]++
			}
		}

		// Check if lineup is complete
		if len(lineup.Players) == requirements.GetTotalPlayers() {
			return lineup
		}
	}

	return nil // Failed to generate valid lineup
}

func (cs *ContestSimulator) selectPlayer(pool []weightedPlayer, position string, remainingSalary int, used map[uuid.UUID]bool, rng *rand.Rand) *types.Player {
	// Filter eligible players
	eligible := make([]weightedPlayer, 0)
	totalWeight := 0.0

	for _, wp := range pool {
		if getStringValueSim(wp.player.Position) == position &&
			getIntValueSim(wp.player.SalaryDK) <= remainingSalary &&
			!used[wp.player.ID] {
			eligible = append(eligible, wp)
			totalWeight += wp.weight
		}
	}

	if len(eligible) == 0 {
		return nil
	}

	// Weighted random selection
	r := rng.Float64() * totalWeight
	cumWeight := 0.0

	for _, wp := range eligible {
		cumWeight += wp.weight
		if r <= cumWeight {
			return &wp.player
		}
	}

	// Fallback to last player
	return &eligible[len(eligible)-1].player
}

func (cs *ContestSimulator) calculatePayouts(scores []LineupScore) {
	for i := range scores {
		rank := i + 1
		scores[i].Rank = rank
		// scores[i].Payout = types.GetPayoutForRank(rank, cs.payoutStructure) // TODO: Implement GetPayoutForRank
		scores[i].Percentile = float64(rank) / float64(len(scores)) * 100
	}
}

func (cs *ContestSimulator) extractUserResults(allScores []LineupScore, userLineups []types.GeneratedLineup) []UserResult {
	results := make([]UserResult, 0, len(userLineups))

	for _, score := range allScores {
		if score.IsUserEntry {
			result := UserResult{
				LineupID:   score.LineupID,
				Score:      score.Score,
				Rank:       score.Rank,
				Percentile: score.Percentile,
				Payout:     score.Payout,
				ROI:        (score.Payout - cs.contest.EntryFee) / cs.contest.EntryFee * 100,
			}
			results = append(results, result)
		}
	}

	return results
}

// GetPayoutStructure returns the payout structure for a contest
func GetPayoutStructure(contest *types.Contest) []PayoutTier {
	if contest.ContestType == "cash" {
		// Double-up structure
		// TODO: Define PayoutTier in types or create a simple structure
		return []PayoutTier{}
		// {MinRank: 1, MaxRank: contest.TotalEntries / 2, Payout: contest.EntryFee * 1.8},
	}

	// GPP structure (simplified)
	// TODO: Implement full payout structure
	return []PayoutTier{
		{MinRank: 1, MaxRank: 1, Payout: contest.PrizePool * 0.20},
	}
}

// OwnershipModel generates realistic ownership percentages
type OwnershipModel struct {
	contestType string
	chalk       float64 // How much ownership concentrates on top plays
}

func NewOwnershipModel(contestType string) *OwnershipModel {
	chalk := 0.5
	if contestType == "cash" {
		chalk = 0.7 // More chalk in cash games
	}

	return &OwnershipModel{
		contestType: contestType,
		chalk:       chalk,
	}
}

func (om *OwnershipModel) GenerateOwnership(players []types.Player, rng *rand.Rand) map[uuid.UUID]float64 {
	ownership := make(map[uuid.UUID]float64)

	// Group by position
	byPosition := make(map[string][]types.Player)
	for _, p := range players {
		position := getStringValueSim(p.Position)
		byPosition[position] = append(byPosition[position], p)
	}

	// Generate ownership for each position
	for _, posPlayers := range byPosition {
		// Sort by value (projected points per dollar)
		sort.Slice(posPlayers, func(i, j int) bool {
			valueI := getFloatValueSim(posPlayers[i].ProjectedPoints) / float64(getIntValueSim(posPlayers[i].SalaryDK))
			valueJ := getFloatValueSim(posPlayers[j].ProjectedPoints) / float64(getIntValueSim(posPlayers[j].SalaryDK))
			return valueI > valueJ
		})

		// Assign ownership based on rank
		for rank, player := range posPlayers {
			baseOwnership := om.calculateBaseOwnership(rank, len(posPlayers))

			// Add noise
			noise := (rng.Float64() - 0.5) * 0.1
			ownership[player.ID] = math.Max(0.01, math.Min(0.50, baseOwnership+noise))

			// Boost if already has ownership data
			if getFloatValueSim(player.OwnershipDK) > 0 {
				ownership[player.ID] = getFloatValueSim(player.OwnershipDK) / 100.0
			}
		}
	}

	return ownership
}

func (om *OwnershipModel) calculateBaseOwnership(rank, total int) float64 {
	percentile := float64(rank) / float64(total)

	if om.contestType == "cash" {
		// Cash game - heavy on top value plays
		if percentile < 0.2 {
			return 0.40 - percentile*0.5
		} else if percentile < 0.5 {
			return 0.20 - percentile*0.2
		}
		return 0.05
	} else {
		// GPP - more distributed but still top-heavy
		if percentile < 0.1 {
			return 0.30 - percentile*0.8
		} else if percentile < 0.3 {
			return 0.20 - percentile*0.4
		} else if percentile < 0.6 {
			return 0.10 - percentile*0.1
		}
		return 0.02
	}
}

// Types for contest simulation

type weightedPlayer struct {
	player types.Player
	weight float64
}

type LineupScore struct {
	LineupID    string
	Score       float64
	Rank        int
	Percentile  float64
	Payout      float64
	IsUserEntry bool
}

type UserResult struct {
	LineupID   string
	Score      float64
	Rank       int
	Percentile float64
	Payout     float64
	ROI        float64
}

type ContestResult struct {
	Contest        *types.Contest
	LineupScores   []LineupScore
	PlayerOutcomes map[uuid.UUID]float64
	UserResults    []UserResult
}

// Helper functions to safely extract values from pointers
func getStringValueSim(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getIntValueSim(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getFloatValueSim(ptr *float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return 0.0
}
