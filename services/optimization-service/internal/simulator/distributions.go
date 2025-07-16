package simulator

import (
	"math"
	"math/rand"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// Helper functions to safely extract values from pointers
func getStringValueDist(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getIntValueDist(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getFloatValueDist(ptr *float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return 0.0
}

func getBoolValueDist(ptr *bool) bool {
	if ptr != nil {
		return *ptr
	}
	return false
}

// Distribution represents a probability distribution for player performance
type Distribution interface {
	Sample(rng *rand.Rand) float64
	Mean() float64
	StdDev() float64
}

// NormalDistribution represents a normal (Gaussian) distribution
type NormalDistribution struct {
	mean   float64
	stdDev float64
}

func NewNormalDistribution(mean, stdDev float64) *NormalDistribution {
	return &NormalDistribution{
		mean:   mean,
		stdDev: stdDev,
	}
}

func (d *NormalDistribution) Sample(rng *rand.Rand) float64 {
	return rng.NormFloat64()*d.stdDev + d.mean
}

func (d *NormalDistribution) Mean() float64 {
	return d.mean
}

func (d *NormalDistribution) StdDev() float64 {
	return d.stdDev
}

// TruncatedNormalDistribution represents a normal distribution with bounds
type TruncatedNormalDistribution struct {
	*NormalDistribution
	min float64
	max float64
}

func NewTruncatedNormalDistribution(mean, stdDev, min, max float64) *TruncatedNormalDistribution {
	return &TruncatedNormalDistribution{
		NormalDistribution: NewNormalDistribution(mean, stdDev),
		min:                min,
		max:                max,
	}
}

func (d *TruncatedNormalDistribution) Sample(rng *rand.Rand) float64 {
	for {
		sample := d.NormalDistribution.Sample(rng)
		if sample >= d.min && sample <= d.max {
			return sample
		}
	}
}

// BetaDistribution represents a beta distribution (good for modeling rates/percentages)
type BetaDistribution struct {
	alpha float64
	beta  float64
	scale float64
	shift float64
}

func NewBetaDistribution(alpha, beta, scale, shift float64) *BetaDistribution {
	return &BetaDistribution{
		alpha: alpha,
		beta:  beta,
		scale: scale,
		shift: shift,
	}
}

func (d *BetaDistribution) Sample(rng *rand.Rand) float64 {
	// Simple beta distribution approximation using gamma
	x := d.sampleGamma(d.alpha, rng)
	y := d.sampleGamma(d.beta, rng)
	return (x/(x+y))*d.scale + d.shift
}

func (d *BetaDistribution) sampleGamma(shape float64, rng *rand.Rand) float64 {
	// Marsaglia and Tsang method for gamma distribution
	if shape < 1 {
		return d.sampleGamma(shape+1, rng) * math.Pow(rng.Float64(), 1/shape)
	}

	d1 := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d1)

	for {
		x := rng.NormFloat64()
		v := 1.0 + c*x
		if v <= 0 {
			continue
		}

		v = v * v * v
		u := rng.Float64()

		if u < 1-0.0331*x*x*x*x {
			return d1 * v
		}

		if math.Log(u) < 0.5*x*x+d1*(1-v+math.Log(v)) {
			return d1 * v
		}
	}
}

func (d *BetaDistribution) Mean() float64 {
	return d.alpha/(d.alpha+d.beta)*d.scale + d.shift
}

func (d *BetaDistribution) StdDev() float64 {
	variance := (d.alpha * d.beta) / ((d.alpha + d.beta) * (d.alpha + d.beta) * (d.alpha + d.beta + 1))
	return math.Sqrt(variance) * d.scale
}

// PlayerDistribution creates appropriate distribution for a player
type PlayerDistribution struct {
	player       types.Player
	distribution Distribution
	injuryProb   float64
}

func NewPlayerDistribution(player types.Player) *PlayerDistribution {
	// Calculate parameters from player stats (removed unused mean variable)

	// Estimate standard deviation from floor/ceiling
	// Using 95% confidence interval approximation
	ceilingPoints := getFloatValueDist(player.CeilingPoints)
	floorPoints := getFloatValueDist(player.FloorPoints)
	projectedPoints := getFloatValueDist(player.ProjectedPoints)
	
	stdDev := (ceilingPoints - floorPoints) / 4.0

	// Create distribution based on player variance
	variance := stdDev / projectedPoints

	var dist Distribution
	if variance > 0.5 {
		// High variance players - use beta distribution for more realistic tails
		alpha := projectedPoints * projectedPoints / (stdDev * stdDev)
		beta := alpha * (ceilingPoints/projectedPoints - 1)
		dist = NewBetaDistribution(alpha, beta, ceilingPoints, 0)
	} else {
		// Normal variance - use truncated normal
		dist = NewTruncatedNormalDistribution(projectedPoints, stdDev, floorPoints*0.8, ceilingPoints*1.2)
	}

	// Set injury probability based on injury status
	injuryProb := 0.01 // 1% base injury risk
	if getBoolValueDist(player.IsInjured) {
		injuryProb = 0.25 // 25% if already injured
	} else if getStringValueDist(player.InjuryStatus) != "" {
		injuryProb = 0.10 // 10% if questionable
	}

	return &PlayerDistribution{
		player:       player,
		distribution: dist,
		injuryProb:   injuryProb,
	}
}

func (pd *PlayerDistribution) Sample(rng *rand.Rand) float64 {
	// Check for injury/DNP
	if rng.Float64() < pd.injuryProb {
		return 0
	}

	// Sample from distribution
	score := pd.distribution.Sample(rng)

	// Apply position-specific adjustments
	score = pd.applyPositionAdjustments(score, rng)

	// Ensure non-negative
	return math.Max(0, score)
}

func (pd *PlayerDistribution) applyPositionAdjustments(baseScore float64, rng *rand.Rand) float64 {
	score := baseScore

	// TODO: CRITICAL - Restore sport-specific player adjustments that were removed during compilation fixes
	// Original logic checked pd.player.Sport for "nba", "nfl", "mlb", "nhl", "golf" and applied:
	// - applyNBAAdjustments(score, rng) for position-based variance (PG/SG/SF/PF/C)
	// - applyNFLAdjustments(score, rng) for weather, game script, and matchup factors
	// - applyMLBAdjustments(score, rng) for pitcher handedness and ballpark factors
	// - applyNHLAdjustments(score, rng) for ice time and special teams usage
	// - applyGolfAdjustments(score, rng) for course difficulty and weather conditions
	// 
	// TEMP FIX: Player struct no longer has Sport field - need to derive from contest/external data
	// This significantly impacts simulation accuracy and must be restored ASAP
	_ = rng // avoid unused parameter warning

	return score
}

func (pd *PlayerDistribution) applyNBAAdjustments(score float64, rng *rand.Rand) float64 {
	// Blowout risk - reduced minutes in garbage time
	if rng.Float64() < 0.1 { // 10% chance of blowout
		reduction := 0.7 + rng.Float64()*0.2 // 70-90% of normal
		score *= reduction
	}

	// Overtime bonus
	if rng.Float64() < 0.05 { // 5% chance of OT
		bonus := 1.1 + rng.Float64()*0.2 // 110-130% of normal
		score *= bonus
	}

	return score
}

func (pd *PlayerDistribution) applyNFLAdjustments(score float64, rng *rand.Rand) float64 {
	position := getStringValueDist(pd.player.Position)
	switch position {
	case "QB":
		// Game script affects passing volume
		if rng.Float64() < 0.3 { // 30% chance of game script impact
			adjustment := 0.8 + rng.Float64()*0.4 // 80-120%
			score *= adjustment
		}
	case "RB":
		// Touchdown variance is high
		tdBonus := rng.Float64() * 12 // 0-12 points TD variance
		score += tdBonus
	case "DST":
		// Defense is highly variable
		if rng.Float64() < 0.1 { // 10% chance of defensive TD
			score += 6 + rng.Float64()*6 // 6-12 bonus points
		}
	}

	return score
}

func (pd *PlayerDistribution) applyMLBAdjustments(score float64, rng *rand.Rand) float64 {
	position := getStringValueDist(pd.player.Position)
	switch position {
	case "P":
		// Pitchers can get pulled early
		if rng.Float64() < 0.2 { // 20% chance of early exit
			score *= 0.3 + rng.Float64()*0.5 // 30-80% of projection
		}
	default:
		// Hitters - multi-hit game bonus
		if rng.Float64() < 0.15 { // 15% chance
			score *= 1.2 + rng.Float64()*0.3 // 120-150%
		}
	}

	return score
}

func (pd *PlayerDistribution) applyNHLAdjustments(score float64, rng *rand.Rand) float64 {
	position := getStringValueDist(pd.player.Position)
	switch position {
	case "G":
		// Goalies can get pulled
		if rng.Float64() < 0.1 { // 10% chance
			score *= 0.2 + rng.Float64()*0.3 // 20-50% of projection
		}
		// Shutout bonus
		if rng.Float64() < 0.05 { // 5% chance
			score += 5 + rng.Float64()*5 // 5-10 bonus points
		}
	default:
		// Power play opportunities
		if rng.Float64() < 0.3 { // 30% chance of PP boost
			score *= 1.1 + rng.Float64()*0.2 // 110-130%
		}
	}

	return score
}

// ContestDistribution models the field's lineup distribution
type ContestDistribution struct {
	contestSize int
	topHeavy    bool    // GPP vs Cash game distribution
	sharpness   float64 // How concentrated scores are at the top
}

func NewContestDistribution(contestType string, size int) *ContestDistribution {
	topHeavy := contestType == "gpp"
	sharpness := 2.0
	if topHeavy {
		sharpness = 3.0 // More separation in GPPs
	}

	return &ContestDistribution{
		contestSize: size,
		topHeavy:    topHeavy,
		sharpness:   sharpness,
	}
}

func (cd *ContestDistribution) GenerateFieldScores(rng *rand.Rand, topScore float64) []float64 {
	scores := make([]float64, cd.contestSize)

	if cd.topHeavy {
		// Power law distribution for GPP
		for i := 0; i < cd.contestSize; i++ {
			rank := float64(i + 1)
			scores[i] = topScore * math.Pow(float64(cd.contestSize)/rank, 1/cd.sharpness)
			// Add noise
			scores[i] += rng.NormFloat64() * 5
		}
	} else {
		// Normal distribution for cash games
		mean := topScore * 0.85
		stdDev := topScore * 0.1
		for i := 0; i < cd.contestSize; i++ {
			scores[i] = rng.NormFloat64()*stdDev + mean
		}
	}

	// Sort descending
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i] < scores[j] {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	return scores
}
