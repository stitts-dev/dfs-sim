package models

import (
	"time"
)

type SimulationResult struct {
	ID                 uint               `gorm:"primaryKey" json:"id"`
	LineupID           uint               `gorm:"not null;uniqueIndex" json:"lineup_id"`
	ContestID          uint               `gorm:"not null" json:"contest_id"`
	NumSimulations     int                `gorm:"not null" json:"num_simulations"`
	Mean               float64            `gorm:"not null" json:"mean"`
	Median             float64            `gorm:"not null" json:"median"`
	StandardDeviation  float64            `gorm:"not null" json:"standard_deviation"`
	Min                float64            `gorm:"not null" json:"min"`
	Max                float64            `gorm:"not null" json:"max"`
	Percentile25       float64            `gorm:"not null" json:"percentile_25"`
	Percentile75       float64            `gorm:"not null" json:"percentile_75"`
	Percentile90       float64            `gorm:"not null" json:"percentile_90"`
	Percentile95       float64            `gorm:"not null" json:"percentile_95"`
	Percentile99       float64            `gorm:"not null" json:"percentile_99"`
	TopPercentFinishes map[string]float64 `gorm:"type:jsonb" json:"top_percent_finishes"`
	WinProbability     float64            `gorm:"not null" json:"win_probability"`
	CashProbability    float64            `gorm:"not null" json:"cash_probability"`
	ROI                float64            `gorm:"not null" json:"roi"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`

	// Associations
	Lineup  Lineup  `gorm:"foreignKey:LineupID" json:"-"`
	Contest Contest `gorm:"foreignKey:ContestID" json:"-"`
}

func (SimulationResult) TableName() string {
	return "simulation_results"
}

// SimulationRun represents a single simulation iteration
type SimulationRun struct {
	LineupScore  float64          `json:"lineup_score"`
	PlayerScores map[uint]float64 `json:"player_scores"`
	Rank         int              `json:"rank"`
	Percentile   float64          `json:"percentile"`
	Payout       float64          `json:"payout"`
}

// SimulationConfig holds configuration for running simulations
type SimulationConfig struct {
	NumSimulations    int                           `json:"num_simulations"`
	UseCorrelations   bool                          `json:"use_correlations"`
	CorrelationMatrix map[string]map[string]float64 `json:"correlation_matrix"`
	ContestSize       int                           `json:"contest_size"`
	PayoutStructure   []PayoutTier                  `json:"payout_structure"`
	EntryFee          float64                       `json:"entry_fee"`
	SimulationWorkers int                           `json:"simulation_workers"`
}

// PayoutTier represents a payout tier in the contest
type PayoutTier struct {
	MinRank int     `json:"min_rank"`
	MaxRank int     `json:"max_rank"`
	Payout  float64 `json:"payout"`
}

// PlayerProjection holds projection data for simulation
type PlayerProjection struct {
	PlayerID          uint    `json:"player_id"`
	Mean              float64 `json:"mean"`
	StandardDeviation float64 `json:"standard_deviation"`
	Floor             float64 `json:"floor"`
	Ceiling           float64 `json:"ceiling"`
}

// CorrelationPair represents correlation between two players
type CorrelationPair struct {
	Player1ID   uint    `json:"player1_id"`
	Player2ID   uint    `json:"player2_id"`
	Correlation float64 `json:"correlation"`
	Type        string  `json:"type"` // "teammate", "opponent", "stack", etc.
}

// SimulationProgress tracks progress of ongoing simulation
type SimulationProgress struct {
	LineupID               uint          `json:"lineup_id"`
	TotalSimulations       int           `json:"total_simulations"`
	Completed              int           `json:"completed"`
	StartTime              time.Time     `json:"start_time"`
	EstimatedTimeRemaining time.Duration `json:"estimated_time_remaining"`
}

// CalculatePercentile calculates what percentile a score falls into
func CalculatePercentile(score float64, allScores []float64) float64 {
	if len(allScores) == 0 {
		return 0
	}

	count := 0
	for _, s := range allScores {
		if s <= score {
			count++
		}
	}

	return float64(count) / float64(len(allScores)) * 100
}

// GetPayoutForRank returns the payout amount for a given rank
func GetPayoutForRank(rank int, payoutStructure []PayoutTier) float64 {
	for _, tier := range payoutStructure {
		if rank >= tier.MinRank && rank <= tier.MaxRank {
			return tier.Payout
		}
	}
	return 0
}
