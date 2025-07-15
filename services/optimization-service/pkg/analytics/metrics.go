package analytics

import (
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

// MetricsCalculator provides comprehensive analytics metrics calculations
type MetricsCalculator struct {
	logger *logrus.Logger
}

// PerformanceMetrics represents comprehensive performance metrics
type PerformanceMetrics struct {
	ROI                  float64   `json:"roi"`
	SharpeRatio          float64   `json:"sharpe_ratio"`
	SortinoRatio         float64   `json:"sortino_ratio"`
	MaxDrawdown          float64   `json:"max_drawdown"`
	MaxDrawdownDuration  int       `json:"max_drawdown_duration"`
	Volatility           float64   `json:"volatility"`
	DownsideDeviation    float64   `json:"downside_deviation"`
	CalmarRatio          float64   `json:"calmar_ratio"`
	WinRate              float64   `json:"win_rate"`
	ProfitFactor         float64   `json:"profit_factor"`
	ExpectedValue        float64   `json:"expected_value"`
	KellyFraction        float64   `json:"kelly_fraction"`
	ConsistencyScore     float64   `json:"consistency_score"`
	RiskAdjustedReturn   float64   `json:"risk_adjusted_return"`
	InformationRatio     float64   `json:"information_ratio"`
	UpsideCapture        float64   `json:"upside_capture"`
	DownsideCapture      float64   `json:"downside_capture"`
	Beta                 float64   `json:"beta"`
	Alpha                float64   `json:"alpha"`
	TreynorRatio         float64   `json:"treynor_ratio"`
}

// CorrelationMatrix represents correlation data between assets
type CorrelationMatrix struct {
	Assets      []string    `json:"assets"`
	Matrix      [][]float64 `json:"matrix"`
	Size        int         `json:"size"`
	AverageCorr float64     `json:"average_correlation"`
}

// ReturnData represents return data for performance calculations
type ReturnData struct {
	Date   time.Time `json:"date"`
	Return float64   `json:"return"`
	Value  float64   `json:"value"`
}

// RiskMetrics represents risk-related metrics
type RiskMetrics struct {
	VaR95         float64 `json:"var_95"`
	VaR99         float64 `json:"var_99"`
	CVaR95        float64 `json:"cvar_95"`
	CVaR99        float64 `json:"cvar_99"`
	StandardDev   float64 `json:"standard_deviation"`
	Variance      float64 `json:"variance"`
	Skewness      float64 `json:"skewness"`
	Kurtosis      float64 `json:"kurtosis"`
	MaxLoss       float64 `json:"max_loss"`
	MinReturn     float64 `json:"min_return"`
	MaxReturn     float64 `json:"max_return"`
}

// NewMetricsCalculator creates a new metrics calculator instance
func NewMetricsCalculator() *MetricsCalculator {
	return &MetricsCalculator{
		logger: logger.GetLogger(),
	}
}

// CalculatePerformanceMetrics calculates comprehensive performance metrics
func (mc *MetricsCalculator) CalculatePerformanceMetrics(returns []float64, benchmarkReturns []float64, riskFreeRate float64) *PerformanceMetrics {
	if len(returns) == 0 {
		return &PerformanceMetrics{}
	}

	metrics := &PerformanceMetrics{}

	// Basic return metrics
	totalReturn := mc.CalculateTotalReturn(returns)
	metrics.ROI = totalReturn
	metrics.ExpectedValue = mc.CalculateMean(returns)

	// Risk metrics
	metrics.Volatility = mc.CalculateStandardDeviation(returns)
	metrics.MaxDrawdown = mc.CalculateMaxDrawdown(returns)
	
	// Risk-adjusted metrics
	metrics.SharpeRatio = mc.CalculateSharpeRatio(returns, riskFreeRate)
	metrics.SortinoRatio = mc.CalculateSortinoRatio(returns, riskFreeRate)
	metrics.CalmarRatio = mc.CalculateCalmarRatio(returns)
	
	// Win/loss metrics
	metrics.WinRate = mc.CalculateWinRate(returns)
	metrics.ProfitFactor = mc.CalculateProfitFactor(returns)
	
	// Advanced metrics
	metrics.KellyFraction = mc.CalculateKellyFraction(returns)
	metrics.ConsistencyScore = mc.CalculateConsistencyScore(returns)
	metrics.DownsideDeviation = mc.CalculateDownsideDeviation(returns, 0)
	
	// Benchmark-relative metrics (if benchmark provided)
	if len(benchmarkReturns) == len(returns) && len(benchmarkReturns) > 0 {
		metrics.Beta = mc.CalculateBeta(returns, benchmarkReturns)
		metrics.Alpha = mc.CalculateAlpha(returns, benchmarkReturns, riskFreeRate, metrics.Beta)
		metrics.TreynorRatio = mc.CalculateTreynorRatio(returns, riskFreeRate, metrics.Beta)
		metrics.InformationRatio = mc.CalculateInformationRatio(returns, benchmarkReturns)
		metrics.UpsideCapture = mc.CalculateUpsideCapture(returns, benchmarkReturns)
		metrics.DownsideCapture = mc.CalculateDownsideCapture(returns, benchmarkReturns)
	}
	
	// Risk-adjusted return
	if metrics.Volatility > 0 {
		metrics.RiskAdjustedReturn = metrics.ExpectedValue / metrics.Volatility
	}

	return metrics
}

// CalculateCorrelationMatrix calculates correlation matrix between multiple return series
func (mc *MetricsCalculator) CalculateCorrelationMatrix(returnSeries [][]float64, assetNames []string) *CorrelationMatrix {
	n := len(returnSeries)
	if n == 0 || len(assetNames) != n {
		return &CorrelationMatrix{}
	}

	// Initialize correlation matrix
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	// Calculate correlations
	totalCorr := 0.0
	corrCount := 0

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0
			} else {
				corr := mc.CalculateCorrelation(returnSeries[i], returnSeries[j])
				matrix[i][j] = corr
				
				if i < j { // Count unique pairs only
					totalCorr += corr
					corrCount++
				}
			}
		}
	}

	avgCorr := 0.0
	if corrCount > 0 {
		avgCorr = totalCorr / float64(corrCount)
	}

	return &CorrelationMatrix{
		Assets:      assetNames,
		Matrix:      matrix,
		Size:        n,
		AverageCorr: avgCorr,
	}
}

// CalculateRiskMetrics calculates comprehensive risk metrics
func (mc *MetricsCalculator) CalculateRiskMetrics(returns []float64) *RiskMetrics {
	if len(returns) == 0 {
		return &RiskMetrics{}
	}

	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	metrics := &RiskMetrics{
		StandardDev: mc.CalculateStandardDeviation(returns),
		Variance:    mc.CalculateVariance(returns),
		MinReturn:   sortedReturns[0],
		MaxReturn:   sortedReturns[len(sortedReturns)-1],
		MaxLoss:     math.Min(0, sortedReturns[0]),
	}

	// Value at Risk calculations
	metrics.VaR95 = mc.CalculateVaR(sortedReturns, 0.05)
	metrics.VaR99 = mc.CalculateVaR(sortedReturns, 0.01)
	metrics.CVaR95 = mc.CalculateCVaR(sortedReturns, 0.05)
	metrics.CVaR99 = mc.CalculateCVaR(sortedReturns, 0.01)

	// Higher moment calculations
	metrics.Skewness = mc.CalculateSkewness(returns)
	metrics.Kurtosis = mc.CalculateKurtosis(returns)

	return metrics
}

// Core mathematical functions

// CalculateSharpeRatio calculates the Sharpe ratio
func (mc *MetricsCalculator) CalculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	excessReturns := make([]float64, len(returns))
	for i, r := range returns {
		excessReturns[i] = r - riskFreeRate
	}

	mean := mc.CalculateMean(excessReturns)
	stdDev := mc.CalculateStandardDeviation(excessReturns)

	if stdDev == 0 {
		return 0
	}

	return mean / stdDev
}

// CalculateMaxDrawdown calculates maximum drawdown
func (mc *MetricsCalculator) CalculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Calculate cumulative returns
	cumulative := make([]float64, len(returns))
	cumulative[0] = 1 + returns[0]
	
	for i := 1; i < len(returns); i++ {
		cumulative[i] = cumulative[i-1] * (1 + returns[i])
	}

	maxDrawdown := 0.0
	peak := cumulative[0]

	for _, value := range cumulative {
		if value > peak {
			peak = value
		}
		
		drawdown := (peak - value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// CalculateCorrelation calculates Pearson correlation coefficient
func (mc *MetricsCalculator) CalculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	meanX := mc.CalculateMean(x)
	meanY := mc.CalculateMean(y)

	numerator := 0.0
	sumXSquared := 0.0
	sumYSquared := 0.0

	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		
		numerator += dx * dy
		sumXSquared += dx * dx
		sumYSquared += dy * dy
	}

	denominator := math.Sqrt(sumXSquared * sumYSquared)
	
	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// CalculateBeta calculates beta coefficient
func (mc *MetricsCalculator) CalculateBeta(returns, benchmarkReturns []float64) float64 {
	if len(returns) != len(benchmarkReturns) || len(returns) < 2 {
		return 0
	}

	meanBench := mc.CalculateMean(benchmarkReturns)
	meanReturns := mc.CalculateMean(returns)

	covariance := 0.0
	benchmarkVariance := 0.0

	for i := 0; i < len(returns); i++ {
		benchDiff := benchmarkReturns[i] - meanBench
		returnDiff := returns[i] - meanReturns
		
		covariance += benchDiff * returnDiff
		benchmarkVariance += benchDiff * benchDiff
	}

	if benchmarkVariance == 0 {
		return 0
	}

	return covariance / benchmarkVariance
}

// CalculateAlpha calculates alpha using CAPM
func (mc *MetricsCalculator) CalculateAlpha(returns, benchmarkReturns []float64, riskFreeRate, beta float64) float64 {
	if len(returns) == 0 || len(benchmarkReturns) == 0 {
		return 0
	}

	portfolioReturn := mc.CalculateMean(returns)
	benchmarkReturn := mc.CalculateMean(benchmarkReturns)

	expectedReturn := riskFreeRate + beta*(benchmarkReturn-riskFreeRate)
	
	return portfolioReturn - expectedReturn
}

// CalculateSortinoRatio calculates Sortino ratio
func (mc *MetricsCalculator) CalculateSortinoRatio(returns []float64, targetReturn float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	excessReturn := mc.CalculateMean(returns) - targetReturn
	downsideDeviation := mc.CalculateDownsideDeviation(returns, targetReturn)

	if downsideDeviation == 0 {
		return 0
	}

	return excessReturn / downsideDeviation
}

// CalculateDownsideDeviation calculates downside deviation
func (mc *MetricsCalculator) CalculateDownsideDeviation(returns []float64, targetReturn float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	sumSquaredDownside := 0.0
	count := 0

	for _, r := range returns {
		if r < targetReturn {
			diff := r - targetReturn
			sumSquaredDownside += diff * diff
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return math.Sqrt(sumSquaredDownside / float64(count))
}

// CalculateCalmarRatio calculates Calmar ratio
func (mc *MetricsCalculator) CalculateCalmarRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	annualizedReturn := mc.CalculateMean(returns) * 252 // Assuming daily returns
	maxDrawdown := mc.CalculateMaxDrawdown(returns)

	if maxDrawdown == 0 {
		return 0
	}

	return annualizedReturn / maxDrawdown
}

// CalculateKellyFraction calculates optimal Kelly fraction
func (mc *MetricsCalculator) CalculateKellyFraction(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	wins := 0
	losses := 0
	totalWinAmount := 0.0
	totalLossAmount := 0.0

	for _, r := range returns {
		if r > 0 {
			wins++
			totalWinAmount += r
		} else if r < 0 {
			losses++
			totalLossAmount += math.Abs(r)
		}
	}

	if wins == 0 || losses == 0 {
		return 0
	}

	winRate := float64(wins) / float64(len(returns))
	avgWin := totalWinAmount / float64(wins)
	avgLoss := totalLossAmount / float64(losses)

	if avgLoss == 0 {
		return 0
	}

	winLossRatio := avgWin / avgLoss
	kellyFraction := winRate - (1-winRate)/winLossRatio

	// Cap Kelly fraction to prevent over-leverage
	return math.Max(0, math.Min(kellyFraction, 0.25))
}

// CalculateVaR calculates Value at Risk at given confidence level
func (mc *MetricsCalculator) CalculateVaR(sortedReturns []float64, alpha float64) float64 {
	if len(sortedReturns) == 0 {
		return 0
	}

	index := int(alpha * float64(len(sortedReturns)))
	if index >= len(sortedReturns) {
		index = len(sortedReturns) - 1
	}

	return sortedReturns[index]
}

// CalculateCVaR calculates Conditional Value at Risk
func (mc *MetricsCalculator) CalculateCVaR(sortedReturns []float64, alpha float64) float64 {
	if len(sortedReturns) == 0 {
		return 0
	}

	varIndex := int(alpha * float64(len(sortedReturns)))
	if varIndex >= len(sortedReturns) {
		varIndex = len(sortedReturns) - 1
	}

	if varIndex == 0 {
		return sortedReturns[0]
	}

	sum := 0.0
	for i := 0; i <= varIndex; i++ {
		sum += sortedReturns[i]
	}

	return sum / float64(varIndex+1)
}

// CalculateSkewness calculates skewness
func (mc *MetricsCalculator) CalculateSkewness(returns []float64) float64 {
	if len(returns) < 3 {
		return 0
	}

	mean := mc.CalculateMean(returns)
	stdDev := mc.CalculateStandardDeviation(returns)

	if stdDev == 0 {
		return 0
	}

	n := float64(len(returns))
	sum := 0.0

	for _, r := range returns {
		sum += math.Pow((r-mean)/stdDev, 3)
	}

	return (n / ((n - 1) * (n - 2))) * sum
}

// CalculateKurtosis calculates excess kurtosis
func (mc *MetricsCalculator) CalculateKurtosis(returns []float64) float64 {
	if len(returns) < 4 {
		return 0
	}

	mean := mc.CalculateMean(returns)
	stdDev := mc.CalculateStandardDeviation(returns)

	if stdDev == 0 {
		return 0
	}

	n := float64(len(returns))
	sum := 0.0

	for _, r := range returns {
		sum += math.Pow((r-mean)/stdDev, 4)
	}

	kurtosis := (n*(n+1))/((n-1)*(n-2)*(n-3))*sum - 3*(n-1)*(n-1)/((n-2)*(n-3))
	
	return kurtosis
}

// Basic statistical functions

// CalculateMean calculates arithmetic mean
func (mc *MetricsCalculator) CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// CalculateVariance calculates variance
func (mc *MetricsCalculator) CalculateVariance(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	mean := mc.CalculateMean(values)
	sumSquares := 0.0

	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	return sumSquares / float64(len(values)-1)
}

// CalculateStandardDeviation calculates standard deviation
func (mc *MetricsCalculator) CalculateStandardDeviation(values []float64) float64 {
	return math.Sqrt(mc.CalculateVariance(values))
}

// CalculateTotalReturn calculates total return from series
func (mc *MetricsCalculator) CalculateTotalReturn(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	totalReturn := 1.0
	for _, r := range returns {
		totalReturn *= (1 + r)
	}

	return totalReturn - 1
}

// CalculateWinRate calculates percentage of positive returns
func (mc *MetricsCalculator) CalculateWinRate(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	wins := 0
	for _, r := range returns {
		if r > 0 {
			wins++
		}
	}

	return float64(wins) / float64(len(returns))
}

// CalculateProfitFactor calculates profit factor
func (mc *MetricsCalculator) CalculateProfitFactor(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	totalWins := 0.0
	totalLosses := 0.0

	for _, r := range returns {
		if r > 0 {
			totalWins += r
		} else if r < 0 {
			totalLosses += math.Abs(r)
		}
	}

	if totalLosses == 0 {
		return math.Inf(1) // Infinite profit factor
	}

	return totalWins / totalLosses
}

// CalculateConsistencyScore calculates consistency score
func (mc *MetricsCalculator) CalculateConsistencyScore(returns []float64) float64 {
	if len(returns) <= 1 {
		return 0
	}

	variance := mc.CalculateVariance(returns)
	mean := mc.CalculateMean(returns)

	if mean == 0 {
		return 0
	}

	// Consistency = 1 / (1 + CV^2) where CV is coefficient of variation
	cv := math.Sqrt(variance) / math.Abs(mean)
	return 1.0 / (1.0 + cv*cv)
}

// CalculateTreynorRatio calculates Treynor ratio
func (mc *MetricsCalculator) CalculateTreynorRatio(returns []float64, riskFreeRate, beta float64) float64 {
	if beta == 0 || len(returns) == 0 {
		return 0
	}

	excessReturn := mc.CalculateMean(returns) - riskFreeRate
	return excessReturn / beta
}

// CalculateInformationRatio calculates information ratio
func (mc *MetricsCalculator) CalculateInformationRatio(returns, benchmarkReturns []float64) float64 {
	if len(returns) != len(benchmarkReturns) || len(returns) == 0 {
		return 0
	}

	// Calculate active returns
	activeReturns := make([]float64, len(returns))
	for i := 0; i < len(returns); i++ {
		activeReturns[i] = returns[i] - benchmarkReturns[i]
	}

	meanActiveReturn := mc.CalculateMean(activeReturns)
	trackingError := mc.CalculateStandardDeviation(activeReturns)

	if trackingError == 0 {
		return 0
	}

	return meanActiveReturn / trackingError
}

// CalculateUpsideCapture calculates upside capture ratio
func (mc *MetricsCalculator) CalculateUpsideCapture(returns, benchmarkReturns []float64) float64 {
	if len(returns) != len(benchmarkReturns) || len(returns) == 0 {
		return 0
	}

	var portfolioUpside, benchmarkUpside []float64

	for i := 0; i < len(returns); i++ {
		if benchmarkReturns[i] > 0 {
			portfolioUpside = append(portfolioUpside, returns[i])
			benchmarkUpside = append(benchmarkUpside, benchmarkReturns[i])
		}
	}

	if len(portfolioUpside) == 0 {
		return 0
	}

	portfolioUpsideReturn := mc.CalculateMean(portfolioUpside)
	benchmarkUpsideReturn := mc.CalculateMean(benchmarkUpside)

	if benchmarkUpsideReturn == 0 {
		return 0
	}

	return portfolioUpsideReturn / benchmarkUpsideReturn
}

// CalculateDownsideCapture calculates downside capture ratio
func (mc *MetricsCalculator) CalculateDownsideCapture(returns, benchmarkReturns []float64) float64 {
	if len(returns) != len(benchmarkReturns) || len(returns) == 0 {
		return 0
	}

	var portfolioDownside, benchmarkDownside []float64

	for i := 0; i < len(returns); i++ {
		if benchmarkReturns[i] < 0 {
			portfolioDownside = append(portfolioDownside, returns[i])
			benchmarkDownside = append(benchmarkDownside, benchmarkReturns[i])
		}
	}

	if len(portfolioDownside) == 0 {
		return 0
	}

	portfolioDownsideReturn := mc.CalculateMean(portfolioDownside)
	benchmarkDownsideReturn := mc.CalculateMean(benchmarkDownside)

	if benchmarkDownsideReturn == 0 {
		return 0
	}

	return portfolioDownsideReturn / benchmarkDownsideReturn
}