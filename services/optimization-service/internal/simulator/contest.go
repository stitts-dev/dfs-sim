package simulator

import (
	"math"
	"math/rand"
	"sort"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
)

// ContestSimulator simulates entire contest fields
type ContestSimulator struct {
	contest         *models.Contest
	fieldSize       int
	payoutStructure []models.PayoutTier
	ownershipModel  *OwnershipModel
}

// NewContestSimulator creates a new contest simulator
func NewContestSimulator(contest *models.Contest) *ContestSimulator {
	return &ContestSimulator{
		contest:         contest,
		fieldSize:       contest.TotalEntries,
		payoutStructure: GetPayoutStructure(contest),
		ownershipModel:  NewOwnershipModel(contest.ContestType),
	}
}

// SimulateFullContest simulates an entire contest with all entries
func (cs *ContestSimulator) SimulateFullContest(userLineups []models.Lineup, players []models.Player, rng *rand.Rand) *ContestResult {
	// Generate field lineups based on ownership
	fieldLineups := cs.generateFieldLineups(players, cs.fieldSize-len(userLineups), rng)

	// Combine user and field lineups
	allLineups := append(userLineups, fieldLineups...)

	// Generate player outcomes
	playerDists := make(map[uint]*PlayerDistribution)
	for _, player := range players {
		playerDists[player.ID] = NewPlayerDistribution(player)
	}

	playerOutcomes := make(map[uint]float64)
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

func (cs *ContestSimulator) generateFieldLineups(players []models.Player, count int, rng *rand.Rand) []models.Lineup {
	fieldLineups := make([]models.Lineup, 0, count)

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

func (cs *ContestSimulator) createWeightedPool(players []models.Player, ownership map[uint]float64) []weightedPlayer {
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

func (cs *ContestSimulator) generateSingleLineup(pool []weightedPlayer, allPlayers []models.Player, rng *rand.Rand) *models.Lineup {
	requirements := cs.contest.PositionRequirements
	lineup := &models.Lineup{
		ContestID: cs.contest.ID,
		Players:   make([]models.Player, 0, requirements.GetTotalPlayers()),
	}

	// Try to fill each position
	positionsFilled := make(map[string]int)
	usedPlayers := make(map[uint]bool)

	// Multiple attempts to create valid lineup
	for attempt := 0; attempt < 100; attempt++ {
		lineup.Players = lineup.Players[:0]
		lineup.TotalSalary = 0
		lineup.ProjectedPoints = 0
		positionsFilled = make(map[string]int)
		usedPlayers = make(map[uint]bool)

		// Fill positions in order
		for position, required := range requirements {
			for i := 0; i < required; i++ {
				player := cs.selectPlayer(pool, position, cs.contest.SalaryCap-lineup.TotalSalary, usedPlayers, rng)
				if player == nil {
					break
				}

				lineup.Players = append(lineup.Players, *player)
				lineup.TotalSalary += player.Salary
				lineup.ProjectedPoints += player.ProjectedPoints
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

func (cs *ContestSimulator) selectPlayer(pool []weightedPlayer, position string, remainingSalary int, used map[uint]bool, rng *rand.Rand) *models.Player {
	// Filter eligible players
	eligible := make([]weightedPlayer, 0)
	totalWeight := 0.0

	for _, wp := range pool {
		if wp.player.Position == position &&
			wp.player.Salary <= remainingSalary &&
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
		scores[i].Payout = models.GetPayoutForRank(rank, cs.payoutStructure)
		scores[i].Percentile = float64(rank) / float64(len(scores)) * 100
	}
}

func (cs *ContestSimulator) extractUserResults(allScores []LineupScore, userLineups []models.Lineup) []UserResult {
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
func GetPayoutStructure(contest *models.Contest) []models.PayoutTier {
	if contest.ContestType == "cash" {
		// Double-up structure
		return []models.PayoutTier{
			{MinRank: 1, MaxRank: contest.TotalEntries / 2, Payout: contest.EntryFee * 1.8},
		}
	}

	// GPP structure (simplified)
	prizePool := contest.PrizePool
	entries := contest.TotalEntries

	tiers := []models.PayoutTier{
		{MinRank: 1, MaxRank: 1, Payout: prizePool * 0.20},                   // 1st: 20%
		{MinRank: 2, MaxRank: 2, Payout: prizePool * 0.12},                   // 2nd: 12%
		{MinRank: 3, MaxRank: 3, Payout: prizePool * 0.08},                   // 3rd: 8%
		{MinRank: 4, MaxRank: 5, Payout: prizePool * 0.05},                   // 4-5th: 5% each
		{MinRank: 6, MaxRank: 10, Payout: prizePool * 0.03},                  // 6-10th: 3% each
		{MinRank: 11, MaxRank: 20, Payout: prizePool * 0.015},                // 11-20th: 1.5% each
		{MinRank: 21, MaxRank: 50, Payout: prizePool * 0.005},                // 21-50th: 0.5% each
		{MinRank: 51, MaxRank: 100, Payout: prizePool * 0.002},               // 51-100th: 0.2% each
		{MinRank: 101, MaxRank: entries / 5, Payout: contest.EntryFee * 1.5}, // Min cash
	}

	return tiers
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

func (om *OwnershipModel) GenerateOwnership(players []models.Player, rng *rand.Rand) map[uint]float64 {
	ownership := make(map[uint]float64)

	// Group by position
	byPosition := make(map[string][]models.Player)
	for _, p := range players {
		byPosition[p.Position] = append(byPosition[p.Position], p)
	}

	// Generate ownership for each position
	for _, posPlayers := range byPosition {
		// Sort by value (projected points per dollar)
		sort.Slice(posPlayers, func(i, j int) bool {
			valueI := posPlayers[i].ProjectedPoints / float64(posPlayers[i].Salary)
			valueJ := posPlayers[j].ProjectedPoints / float64(posPlayers[j].Salary)
			return valueI > valueJ
		})

		// Assign ownership based on rank
		for rank, player := range posPlayers {
			baseOwnership := om.calculateBaseOwnership(rank, len(posPlayers))

			// Add noise
			noise := (rng.Float64() - 0.5) * 0.1
			ownership[player.ID] = math.Max(0.01, math.Min(0.50, baseOwnership+noise))

			// Boost if already has ownership data
			if player.Ownership > 0 {
				ownership[player.ID] = player.Ownership / 100.0
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
	player models.Player
	weight float64
}

type LineupScore struct {
	LineupID    uint
	Score       float64
	Rank        int
	Percentile  float64
	Payout      float64
	IsUserEntry bool
}

type UserResult struct {
	LineupID   uint
	Score      float64
	Rank       int
	Percentile float64
	Payout     float64
	ROI        float64
}

type ContestResult struct {
	Contest        *models.Contest
	LineupScores   []LineupScore
	PlayerOutcomes map[uint]float64
	UserResults    []UserResult
}
