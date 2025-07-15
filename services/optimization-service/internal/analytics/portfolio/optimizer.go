package portfolio

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
	"gonum.org/v1/gonum/stat"
)

// PortfolioConfig defines configuration for portfolio optimization
type PortfolioConfig struct {
	RiskAversion        float64                    `json:"risk_aversion"`
	MinDiversification  float64                    `json:"min_diversification"`
	MaxPositionSize     float64                    `json:"max_position_size"`
	MinPositionSize     float64                    `json:"min_position_size"`
	UseRiskParity       bool                       `json:"use_risk_parity"`
	Constraints         []PortfolioConstraint      `json:"constraints"`
	RegularizationParam float64                    `json:"regularization_param"`
}

// PortfolioConstraint defines portfolio-level constraints
type PortfolioConstraint struct {
	Type  string  `json:"type"` // "sport", "team", "position"
	Value string  `json:"value"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

// PortfolioResult contains optimization results
type PortfolioResult struct {
	Weights             map[string]float64 `json:"weights"`
	ExpectedReturn      float64            `json:"expected_return"`
	Risk                float64            `json:"risk"`
	SharpeRatio         float64            `json:"sharpe_ratio"`
	DiversificationScore float64            `json:"diversification_score"`
	RiskContribution    map[string]float64 `json:"risk_contribution"`
	OptimizationTime    int64              `json:"optimization_time_ms"`
}

// LineupData represents historical lineup performance
type LineupData struct {
	LineupID   string    `json:"lineup_id"`
	Sport      string    `json:"sport"`
	Team       string    `json:"team"`
	Players    []string  `json:"players"`
	Return     float64   `json:"return"`
	Date       time.Time `json:"date"`
	ContestID  string    `json:"contest_id"`
}

// OptimizePortfolio performs Modern Portfolio Theory optimization
func OptimizePortfolio(ctx context.Context, lineups []LineupData, config PortfolioConfig) (*PortfolioResult, error) {
	startTime := time.Now()
	log := logger.GetLogger()

	log.WithFields(logrus.Fields{
		"lineup_count":  len(lineups),
		"risk_aversion": config.RiskAversion,
	}).Info("Starting portfolio optimization")

	if len(lineups) < 2 {
		return nil, fmt.Errorf("need at least 2 lineups for portfolio optimization")
	}

	// Step 1: Calculate returns matrix
	returns := calculateReturnsMatrix(lineups)
	if returns == nil {
		return nil, fmt.Errorf("failed to calculate returns matrix")
	}

	// Step 2: Compute covariance matrix with regularization
	covMatrix := calculateCovarianceMatrix(returns, config.RegularizationParam)
	if covMatrix == nil {
		return nil, fmt.Errorf("failed to calculate covariance matrix")
	}

	// Step 3: Calculate expected returns
	expectedReturns := calculateExpectedReturns(returns)

	// Step 4: Solve optimization problem
	var weights []float64
	var err error
	
	if config.UseRiskParity {
		weights, err = solveRiskParity(covMatrix, config)
	} else {
		weights, err = solveQuadraticProgramming(covMatrix, expectedReturns, config)
	}
	
	if err != nil {
		return nil, fmt.Errorf("optimization failed: %w", err)
	}

	// Step 5: Apply portfolio constraints
	constrainedWeights := applyPortfolioConstraints(weights, lineups, config)

	// Step 6: Calculate portfolio metrics
	result := calculatePortfolioMetrics(constrainedWeights, expectedReturns, covMatrix)
	result.OptimizationTime = time.Since(startTime).Milliseconds()

	log.WithFields(logrus.Fields{
		"expected_return": result.ExpectedReturn,
		"risk":           result.Risk,
		"sharpe_ratio":   result.SharpeRatio,
		"time_ms":        result.OptimizationTime,
	}).Info("Portfolio optimization completed")

	return result, nil
}

// calculateReturnsMatrix converts lineup data to returns matrix
func calculateReturnsMatrix(lineups []LineupData) *mat.Dense {
	if len(lineups) == 0 {
		return nil
	}

	// Group lineups by date to create time series
	dateGroups := make(map[string][]float64)
	for _, lineup := range lineups {
		dateKey := lineup.Date.Format("2006-01-02")
		dateGroups[dateKey] = append(dateGroups[dateKey], lineup.Return)
	}

	// Create matrix with lineups as columns and dates as rows
	dates := make([]string, 0, len(dateGroups))
	for date := range dateGroups {
		dates = append(dates, date)
	}

	rows := len(dates)
	cols := len(lineups)
	data := make([]float64, rows*cols)

	// Fill matrix with returns
	for i, date := range dates {
		for j, lineup := range lineups {
			if lineup.Date.Format("2006-01-02") == date {
				data[i*cols+j] = lineup.Return
			}
		}
	}

	return mat.NewDense(rows, cols, data)
}

// calculateCovarianceMatrix computes covariance with regularization
func calculateCovarianceMatrix(returns *mat.Dense, regularization float64) *mat.Dense {
	r, c := returns.Dims()
	if r < 2 || c < 2 {
		return nil
	}

	// Calculate covariance
	cov := mat.NewSymDense(c, nil)
	stat.CovarianceMatrix(cov, returns, nil)

	// Add regularization to diagonal to avoid singular matrix
	for i := 0; i < c; i++ {
		cov.SetSym(i, i, cov.At(i, i)+regularization)
	}

	// Convert SymDense to Dense for compatibility
	dense := mat.NewDense(c, c, nil)
	for i := 0; i < c; i++ {
		for j := 0; j < c; j++ {
			dense.Set(i, j, cov.At(i, j))
		}
	}
	return dense
}

// calculateExpectedReturns computes expected returns from historical data
func calculateExpectedReturns(returns *mat.Dense) []float64 {
	_, c := returns.Dims()
	expectedReturns := make([]float64, c)

	for j := 0; j < c; j++ {
		col := mat.Col(nil, j, returns)
		expectedReturns[j] = stat.Mean(col, nil)
	}

	return expectedReturns
}

// solveQuadraticProgramming solves the mean-variance optimization problem
func solveQuadraticProgramming(cov *mat.Dense, expectedReturns []float64, config PortfolioConfig) ([]float64, error) {
	n := len(expectedReturns)
	
	// Define optimization problem
	problem := optimize.Problem{
		Func: func(x []float64) float64 {
			// Objective: minimize 0.5 * w^T * Σ * w - λ * w^T * μ
			// where λ is risk aversion parameter
			var variance float64
			for i := 0; i < n; i++ {
				for j := 0; j < n; j++ {
					variance += x[i] * x[j] * cov.At(i, j)
				}
			}
			
			var expectedReturn float64
			for i := 0; i < n; i++ {
				expectedReturn += x[i] * expectedReturns[i]
			}
			
			return 0.5*variance - config.RiskAversion*expectedReturn
		},
		Grad: func(grad, x []float64) {
			// Gradient of objective function
			for i := 0; i < n; i++ {
				grad[i] = 0
				for j := 0; j < n; j++ {
					grad[i] += x[j] * cov.At(i, j)
				}
				grad[i] -= config.RiskAversion * expectedReturns[i]
			}
		},
	}

	// Initial guess: equal weights
	x0 := make([]float64, n)
	for i := range x0 {
		x0[i] = 1.0 / float64(n)
	}

	// Set optimization method and constraints
	method := &optimize.LBFGS{}
	settings := optimize.Settings{
		FuncEvaluations: 1000,
		GradientThreshold: 1e-6,
	}

	result, err := optimize.Minimize(problem, x0, &settings, method)
	if err != nil {
		return nil, err
	}

	// Normalize weights to sum to 1
	weights := result.X
	sum := 0.0
	for _, w := range weights {
		sum += math.Abs(w)
	}
	
	if sum > 0 {
		for i := range weights {
			weights[i] = math.Max(0, weights[i]) / sum
		}
	}

	return weights, nil
}

// solveRiskParity implements risk parity optimization
func solveRiskParity(cov *mat.Dense, config PortfolioConfig) ([]float64, error) {
	n, _ := cov.Dims()
	
	// Risk parity seeks equal risk contribution from each asset
	// Solve: w_i * (Σw)_i = 1/n * w^T * Σ * w for all i
	
	problem := optimize.Problem{
		Func: func(x []float64) float64 {
			// Calculate risk contributions
			sigmaw := mat.NewVecDense(n, nil)
			sigmaw.MulVec(cov, mat.NewVecDense(n, x))
			
			totalRisk := mat.Dot(mat.NewVecDense(n, x), sigmaw)
			targetContribution := totalRisk / float64(n)
			
			// Minimize squared deviations from equal risk contribution
			var obj float64
			for i := 0; i < n; i++ {
				contribution := x[i] * sigmaw.AtVec(i)
				obj += math.Pow(contribution-targetContribution, 2)
			}
			
			return obj
		},
	}

	// Initial guess: equal weights
	x0 := make([]float64, n)
	for i := range x0 {
		x0[i] = 1.0 / float64(n)
	}

	method := &optimize.LBFGS{}
	settings := optimize.Settings{
		FuncEvaluations: 1000,
		GradientThreshold: 1e-8,
	}

	result, err := optimize.Minimize(problem, x0, &settings, method)
	if err != nil {
		return nil, err
	}

	// Normalize weights
	weights := result.X
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	
	for i := range weights {
		weights[i] /= sum
	}

	return weights, nil
}

// applyPortfolioConstraints applies position limits and other constraints
func applyPortfolioConstraints(weights []float64, lineups []LineupData, config PortfolioConfig) map[string]float64 {
	weightMap := make(map[string]float64)
	
	// Map weights to lineup IDs
	for i, lineup := range lineups {
		if i < len(weights) {
			weight := weights[i]
			
			// Apply position size limits
			if weight > config.MaxPositionSize {
				weight = config.MaxPositionSize
			} else if weight < config.MinPositionSize {
				weight = 0 // Exclude positions below minimum
			}
			
			weightMap[lineup.LineupID] = weight
		}
	}

	// Apply constraint-based adjustments
	for _, constraint := range config.Constraints {
		adjustConstraintWeights(weightMap, lineups, constraint)
	}

	// Re-normalize weights
	normalizeWeights(weightMap)

	return weightMap
}

// adjustConstraintWeights adjusts weights based on specific constraints
func adjustConstraintWeights(weights map[string]float64, lineups []LineupData, constraint PortfolioConstraint) {
	// Calculate current allocation to constraint category
	totalWeight := 0.0
	constraintWeight := 0.0
	
	for id, weight := range weights {
		totalWeight += weight
		
		// Find corresponding lineup
		for _, lineup := range lineups {
			if lineup.LineupID == id {
				switch constraint.Type {
				case "sport":
					if lineup.Sport == constraint.Value {
						constraintWeight += weight
					}
				case "team":
					if lineup.Team == constraint.Value {
						constraintWeight += weight
					}
				}
				break
			}
		}
	}

	// Adjust weights if constraint is violated
	if totalWeight > 0 {
		currentRatio := constraintWeight / totalWeight
		
		if currentRatio < constraint.Min || currentRatio > constraint.Max {
			// Scale weights to meet constraint
			targetRatio := math.Max(constraint.Min, math.Min(constraint.Max, currentRatio))
			scaleFactor := targetRatio / currentRatio
			
			for id, weight := range weights {
				for _, lineup := range lineups {
					if lineup.LineupID == id {
						if (constraint.Type == "sport" && lineup.Sport == constraint.Value) ||
						   (constraint.Type == "team" && lineup.Team == constraint.Value) {
							weights[id] = weight * scaleFactor
						}
						break
					}
				}
			}
		}
	}
}

// normalizeWeights ensures weights sum to 1
func normalizeWeights(weights map[string]float64) {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	
	if sum > 0 {
		for id := range weights {
			weights[id] /= sum
		}
	}
}

// calculatePortfolioMetrics computes portfolio performance metrics
func calculatePortfolioMetrics(weights map[string]float64, expectedReturns []float64, cov *mat.Dense) *PortfolioResult {
	// Convert weight map to vector
	n := len(expectedReturns)
	weightVec := mat.NewVecDense(n, nil)
	i := 0
	for _, w := range weights {
		if i < n {
			weightVec.SetVec(i, w)
			i++
		}
	}

	// Calculate expected portfolio return
	portfolioReturn := mat.Dot(weightVec, mat.NewVecDense(n, expectedReturns))

	// Calculate portfolio variance
	sigmaw := mat.NewVecDense(n, nil)
	sigmaw.MulVec(cov, weightVec)
	portfolioVariance := mat.Dot(weightVec, sigmaw)
	portfolioRisk := math.Sqrt(portfolioVariance)

	// Calculate Sharpe ratio (assuming risk-free rate = 0)
	sharpeRatio := 0.0
	if portfolioRisk > 0 {
		sharpeRatio = portfolioReturn / portfolioRisk
	}

	// Calculate diversification score (1 - HHI)
	hhi := 0.0
	for _, w := range weights {
		hhi += w * w
	}
	diversificationScore := 1.0 - hhi

	// Calculate risk contributions
	riskContributions := make(map[string]float64)
	j := 0
	for id, w := range weights {
		if j < n {
			contribution := w * sigmaw.AtVec(j) / portfolioRisk
			riskContributions[id] = contribution
			j++
		}
	}

	return &PortfolioResult{
		Weights:             weights,
		ExpectedReturn:      portfolioReturn,
		Risk:                portfolioRisk,
		SharpeRatio:         sharpeRatio,
		DiversificationScore: diversificationScore,
		RiskContribution:    riskContributions,
	}
}