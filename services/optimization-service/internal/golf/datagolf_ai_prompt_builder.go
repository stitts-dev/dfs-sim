package golf

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

type DataGolfAIPromptBuilder struct {
	contextBuilder        *ContextualDataBuilder
	metricAnalyzer        *SGMetricAnalyzer
	courseFitAnalyzer     *CourseFitAnalyzer
	correlationAnalyzer   *CorrelationInsightAnalyzer
	strategyRecommender   *StrategyRecommendationEngine
	weatherAnalyzer       *WeatherAnalyzer
	cutLineAnalyzer       *CutLineAnalyzer
	ownershipAnalyzer     *OwnershipAnalyzer
}

type ContextualDataBuilder struct {
	tournamentProcessor   *TournamentContextProcessor
	playerProcessor       *PlayerContextProcessor
	courseProcessor       *CourseContextProcessor
	marketProcessor       *MarketContextProcessor
}

type SGMetricAnalyzer struct {
	categoryAnalyzer      *SGCategoryAnalyzer
	trendAnalyzer         *SGTrendAnalyzer
	comparisonEngine      *SGComparisonEngine
	outlierDetector       *SGOutlierDetector
}

type CourseFitAnalyzer struct {
	fitCalculator         *CourseFitCalculator
	historicalAnalyzer    *HistoricalCourseFitAnalyzer
	strengthAnalyzer      *StrengthMatchAnalyzer
	weaknessAnalyzer      *WeaknessAnalyzer
}

type CorrelationInsightAnalyzer struct {
	correlationEngine     *AdvancedCorrelationEngine
	stackingAnalyzer      *StackingInsightAnalyzer
	contrarian            *ContrarianInsightAnalyzer
	lineupSynergyAnalyzer *LineupSynergyAnalyzer
}

type StrategyRecommendationEngine struct {
	strategyAnalyzer      *StrategyAnalyzer
	riskProfileAnalyzer   *RiskProfileAnalyzer
	objectiveOptimizer    *ObjectiveOptimizer
	scenarioPlanner       *ScenarioPlanner
}

type AIPromptContext struct {
	TournamentContext       *TournamentContext       `json:"tournament_context"`
	PlayerAnalysis          *PlayerAnalysis          `json:"player_analysis"`
	CourseFitMatrix         *CourseFitMatrix         `json:"course_fit_matrix"`
	WeatherAnalysis         *WeatherAnalysis         `json:"weather_analysis"`
	CorrelationInsights     *CorrelationInsights     `json:"correlation_insights"`
	StrategyRecommendations *StrategyRecommendations `json:"strategy_recommendations"`
	MetricPrioritization    *MetricPrioritization    `json:"metric_prioritization"`
	MarketInefficiencies    *MarketInefficiencies    `json:"market_inefficiencies"`
	RiskAssessment          *RiskAssessment          `json:"risk_assessment"`
	OptimizationFocus       *OptimizationFocus       `json:"optimization_focus"`
}

type TournamentContext struct {
	TournamentInfo          *TournamentInfo          `json:"tournament_info"`
	CourseCharacteristics   *CourseCharacteristics   `json:"course_characteristics"`
	HistoricalData          *HistoricalTournamentData `json:"historical_data"`
	CutLineProjection       *CutLineProjection       `json:"cut_line_projection"`
	WeatherConditions       *WeatherConditions       `json:"weather_conditions"`
	FieldStrength           *FieldStrength           `json:"field_strength"`
	KeyNarratives           []string                 `json:"key_narratives"`
}

type PlayerAnalysis struct {
	TopPerformers           []*PlayerInsight         `json:"top_performers"`
	ValuePlays              []*PlayerInsight         `json:"value_plays"`
	CourseFitExcellence     []*PlayerInsight         `json:"course_fit_excellence"`
	RecentFormAnalysis      []*PlayerInsight         `json:"recent_form_analysis"`
	SGCategoryLeaders       map[string][]*PlayerInsight `json:"sg_category_leaders"`
	ConsistencyAnalysis     []*PlayerInsight         `json:"consistency_analysis"`
	VolatilityAnalysis      []*PlayerInsight         `json:"volatility_analysis"`
	InjuryRiskAssessment    []*PlayerInsight         `json:"injury_risk_assessment"`
}

type CourseFitMatrix struct {
	TopCourseFits           []*CourseFitInsight      `json:"top_course_fits"`
	CourseFitMismatches     []*CourseFitInsight      `json:"course_fit_mismatches"`
	SkillPremiumAnalysis    *SkillPremiumAnalysis    `json:"skill_premium_analysis"`
	HistoricalAdvantages    []*HistoricalAdvantage   `json:"historical_advantages"`
	WeatherAdjustments      map[string]float64       `json:"weather_adjustments"`
}

type WeatherAnalysis struct {
	CurrentConditions       *WeatherConditions       `json:"current_conditions"`
	ForecastImpact          *WeatherImpact           `json:"forecast_impact"`
	PlayerWeatherProfiles   map[string]*WeatherProfile `json:"player_weather_profiles"`
	ConditionAdvantages     []*WeatherAdvantage      `json:"condition_advantages"`
	TeeTimeImpact          *TeeTimeWeatherImpact    `json:"tee_time_impact"`
}

type CorrelationInsights struct {
	HighCorrelationPairs    []*CorrelatedPair        `json:"high_correlation_pairs"`
	StackingOpportunities   []*StackingOpportunity   `json:"stacking_opportunities"`
	ContrarianPlays         []*ContrarianPlay        `json:"contrarian_plays"`
	LineupSynergies         []*LineupSynergy         `json:"lineup_synergies"`
	AntiCorrelationBenefits []*AntiCorrelationBenefit `json:"anti_correlation_benefits"`
}

type StrategyRecommendations struct {
	OptimalStrategies       []*StrategyRecommendation `json:"optimal_strategies"`
	TournamentStrategy      *TournamentStrategy      `json:"tournament_strategy"`
	RiskManagement          *RiskManagement          `json:"risk_management"`
	OwnershipStrategy       *OwnershipStrategy       `json:"ownership_strategy"`
	StackingGuidance        *StackingGuidance        `json:"stacking_guidance"`
	CashGameVsTournament    *CashVsTournamentGuidance `json:"cash_vs_tournament"`
}

type MetricPrioritization struct {
	PrimaryMetrics          []string                 `json:"primary_metrics"`
	SecondaryMetrics        []string                 `json:"secondary_metrics"`
	ContextualImportance    map[string]float64       `json:"contextual_importance"`
	WeatherAdjustedMetrics  []string                 `json:"weather_adjusted_metrics"`
	CourseSpecificMetrics   []string                 `json:"course_specific_metrics"`
}

type PlayerInsight struct {
	PlayerID                string                   `json:"player_id"`
	PlayerName              string                   `json:"player_name"`
	Salary                  int                      `json:"salary"`
	Projection              float64                  `json:"projection"`
	Value                   float64                  `json:"value"`
	RecentForm              *RecentForm              `json:"recent_form"`
	SGMetrics               *StrokesGainedMetrics    `json:"sg_metrics"`
	CourseFit               *CourseFitResult         `json:"course_fit"`
	WeatherImpact           float64                  `json:"weather_impact"`
	OwnershipProjection     float64                  `json:"ownership_projection"`
	Volatility              float64                  `json:"volatility"`
	KeyInsights             []string                 `json:"key_insights"`
	RiskFactors             []string                 `json:"risk_factors"`
	Advantages              []string                 `json:"advantages"`
}

type DataGolfInsights struct {
	StrokesGainedData       map[string]*StrokesGainedMetrics `json:"strokes_gained_data"`
	CourseAnalytics         *CourseAnalytics         `json:"course_analytics"`
	PlayerCourseHistory     map[string]*PlayerCourseHistory `json:"player_course_history"`
	WeatherImpactData       *WeatherImpactAnalysis   `json:"weather_impact_data"`
	TournamentPredictions   *TournamentPredictions   `json:"tournament_predictions"`
	LiveTournamentData      *LiveTournamentData      `json:"live_tournament_data"`
	MarketData              *MarketData              `json:"market_data"`
}

func NewDataGolfAIPromptBuilder() *DataGolfAIPromptBuilder {
	return &DataGolfAIPromptBuilder{
		contextBuilder:        NewContextualDataBuilder(),
		metricAnalyzer:        NewSGMetricAnalyzer(),
		courseFitAnalyzer:     NewCourseFitAnalyzer(),
		correlationAnalyzer:   NewCorrelationInsightAnalyzer(),
		strategyRecommender:   NewStrategyRecommendationEngine(),
		weatherAnalyzer:       NewWeatherAnalyzer(),
		cutLineAnalyzer:       NewCutLineAnalyzer(),
		ownershipAnalyzer:     NewOwnershipAnalyzer(),
	}
}

func (dgpb *DataGolfAIPromptBuilder) BuildAdvancedGolfPrompt(
	tournament *types.GolfTournament,
	players []*types.GolfPlayer,
	constraints *types.OptimizationConstraints,
	dataGolfInsights *DataGolfInsights,
) (*AIPromptContext, error) {
	tournamentContext := dgpb.buildTournamentContext(tournament, dataGolfInsights)
	playerAnalysis := dgpb.buildPlayerAnalysis(players, dataGolfInsights)
	courseFitMatrix := dgpb.buildCourseFitMatrix(players, tournament, dataGolfInsights)
	weatherAnalysis := dgpb.buildWeatherAnalysis(tournament, dataGolfInsights)
	correlationInsights := dgpb.buildCorrelationInsights(players, dataGolfInsights)
	strategyRecommendations := dgpb.buildStrategyRecommendations(constraints, dataGolfInsights)
	metricPrioritization := dgpb.buildMetricPrioritization(tournament, dataGolfInsights)
	marketInefficiencies := dgpb.buildMarketInefficiencies(players, dataGolfInsights)
	riskAssessment := dgpb.buildRiskAssessment(tournament, players, dataGolfInsights)
	optimizationFocus := dgpb.buildOptimizationFocus(constraints, tournament, dataGolfInsights)

	return &AIPromptContext{
		TournamentContext:       tournamentContext,
		PlayerAnalysis:          playerAnalysis,
		CourseFitMatrix:         courseFitMatrix,
		WeatherAnalysis:         weatherAnalysis,
		CorrelationInsights:     correlationInsights,
		StrategyRecommendations: strategyRecommendations,
		MetricPrioritization:    metricPrioritization,
		MarketInefficiencies:    marketInefficiencies,
		RiskAssessment:          riskAssessment,
		OptimizationFocus:       optimizationFocus,
	}, nil
}

func (dgpb *DataGolfAIPromptBuilder) GeneratePromptString(context *AIPromptContext) string {
	var promptBuilder strings.Builder

	promptBuilder.WriteString("# Advanced Golf DFS Optimization Analysis\n\n")
	
	dgpb.writeTournamentSection(&promptBuilder, context.TournamentContext)
	dgpb.writePlayerAnalysisSection(&promptBuilder, context.PlayerAnalysis)
	dgpb.writeCourseFitSection(&promptBuilder, context.CourseFitMatrix)
	dgpb.writeWeatherSection(&promptBuilder, context.WeatherAnalysis)
	dgpb.writeCorrelationSection(&promptBuilder, context.CorrelationInsights)
	dgpb.writeStrategySection(&promptBuilder, context.StrategyRecommendations)
	dgpb.writeMetricPrioritizationSection(&promptBuilder, context.MetricPrioritization)
	dgpb.writeMarketInefficienciesSection(&promptBuilder, context.MarketInefficiencies)
	dgpb.writeRiskAssessmentSection(&promptBuilder, context.RiskAssessment)
	dgpb.writeOptimizationFocusSection(&promptBuilder, context.OptimizationFocus)

	promptBuilder.WriteString("\n## AI Optimization Guidance Request\n\n")
	promptBuilder.WriteString("Based on the comprehensive DataGolf analysis above, provide specific lineup optimization recommendations including:\n\n")
	promptBuilder.WriteString("1. **Core Player Selections**: High-confidence plays with strong course fit and strokes gained profiles\n")
	promptBuilder.WriteString("2. **Value Identification**: Underpriced players with favorable analytics\n")
	promptBuilder.WriteString("3. **Correlation Strategy**: Optimal stacking and contrarian approaches\n")
	promptBuilder.WriteString("4. **Weather Adjustments**: Condition-specific player advantages\n")
	promptBuilder.WriteString("5. **Risk Management**: Balanced volatility and ceiling considerations\n")
	promptBuilder.WriteString("6. **Tournament-Specific Strategy**: Cut line, leaderboard, and variance considerations\n\n")
	promptBuilder.WriteString("Prioritize actionable insights that can be directly implemented in lineup construction.")

	return promptBuilder.String()
}

func (dgpb *DataGolfAIPromptBuilder) buildTournamentContext(
	tournament *types.GolfTournament,
	insights *DataGolfInsights,
) *TournamentContext {
	return &TournamentContext{
		TournamentInfo: &TournamentInfo{
			Name:     tournament.Name,
			Date:     tournament.Date,
			Purse:    tournament.Purse,
			Field:    tournament.FieldSize,
			Format:   tournament.Format,
		},
		CourseCharacteristics: dgpb.analyzeCourseCharacteristics(insights.CourseAnalytics),
		HistoricalData:        dgpb.processHistoricalData(tournament, insights),
		CutLineProjection:     dgpb.cutLineAnalyzer.ProjectCutLine(tournament, insights),
		WeatherConditions:     insights.WeatherImpactData.CurrentConditions,
		FieldStrength:         dgpb.analyzeFieldStrength(tournament, insights),
		KeyNarratives:         dgpb.generateKeyNarratives(tournament, insights),
	}
}

func (dgpb *DataGolfAIPromptBuilder) buildPlayerAnalysis(
	players []*types.GolfPlayer,
	insights *DataGolfInsights,
) *PlayerAnalysis {
	playerInsights := make([]*PlayerInsight, 0, len(players))
	
	for _, player := range players {
		insight := dgpb.generatePlayerInsight(player, insights)
		playerInsights = append(playerInsights, insight)
	}

	return &PlayerAnalysis{
		TopPerformers:       dgpb.identifyTopPerformers(playerInsights),
		ValuePlays:          dgpb.identifyValuePlays(playerInsights),
		CourseFitExcellence: dgpb.identifyCourseFitExcellence(playerInsights),
		RecentFormAnalysis:  dgpb.analyzeRecentForm(playerInsights),
		SGCategoryLeaders:   dgpb.analyzeSGCategoryLeaders(playerInsights),
		ConsistencyAnalysis: dgpb.analyzeConsistency(playerInsights),
		VolatilityAnalysis:  dgpb.analyzeVolatility(playerInsights),
		InjuryRiskAssessment: dgpb.assessInjuryRisk(playerInsights),
	}
}

func (dgpb *DataGolfAIPromptBuilder) buildCourseFitMatrix(
	players []*types.GolfPlayer,
	tournament *types.GolfTournament,
	insights *DataGolfInsights,
) *CourseFitMatrix {
	courseFits := make([]*CourseFitInsight, 0, len(players))
	
	for _, player := range players {
		fit := dgpb.courseFitAnalyzer.AnalyzeCourseFit(player, tournament, insights)
		courseFits = append(courseFits, fit)
	}

	return &CourseFitMatrix{
		TopCourseFits:        dgpb.identifyTopCourseFits(courseFits),
		CourseFitMismatches:  dgpb.identifyCourseFitMismatches(courseFits),
		SkillPremiumAnalysis: dgpb.analyzeSkillPremiums(insights.CourseAnalytics),
		HistoricalAdvantages: dgpb.identifyHistoricalAdvantages(players, insights),
		WeatherAdjustments:   dgpb.calculateWeatherAdjustments(insights),
	}
}

func (dgpb *DataGolfAIPromptBuilder) buildWeatherAnalysis(
	tournament *types.GolfTournament,
	insights *DataGolfInsights,
) *WeatherAnalysis {
	return dgpb.weatherAnalyzer.AnalyzeWeatherImpact(tournament, insights)
}

func (dgpb *DataGolfAIPromptBuilder) buildCorrelationInsights(
	players []*types.GolfPlayer,
	insights *DataGolfInsights,
) *CorrelationInsights {
	return dgpb.correlationAnalyzer.AnalyzeCorrelations(players, insights)
}

func (dgpb *DataGolfAIPromptBuilder) buildStrategyRecommendations(
	constraints *types.OptimizationConstraints,
	insights *DataGolfInsights,
) *StrategyRecommendations {
	return dgpb.strategyRecommender.GenerateRecommendations(constraints, insights)
}

func (dgpb *DataGolfAIPromptBuilder) buildMetricPrioritization(
	tournament *types.GolfTournament,
	insights *DataGolfInsights,
) *MetricPrioritization {
	courseAnalytics := insights.CourseAnalytics
	
	primaryMetrics := dgpb.identifyPrimaryMetrics(courseAnalytics)
	secondaryMetrics := dgpb.identifySecondaryMetrics(courseAnalytics)
	contextualImportance := dgpb.calculateContextualImportance(tournament, insights)

	return &MetricPrioritization{
		PrimaryMetrics:         primaryMetrics,
		SecondaryMetrics:       secondaryMetrics,
		ContextualImportance:   contextualImportance,
		WeatherAdjustedMetrics: dgpb.identifyWeatherAdjustedMetrics(insights),
		CourseSpecificMetrics:  dgpb.identifyCourseSpecificMetrics(courseAnalytics),
	}
}

func (dgpb *DataGolfAIPromptBuilder) writeTournamentSection(builder *strings.Builder, context *TournamentContext) {
	builder.WriteString("## Tournament Context\n\n")
	builder.WriteString(fmt.Sprintf("**Tournament:** %s (%s)\n", context.TournamentInfo.Name, context.TournamentInfo.Date.Format("Jan 2, 2006")))
	builder.WriteString(fmt.Sprintf("**Purse:** $%s | **Field Size:** %d\n\n", formatMoney(context.TournamentInfo.Purse), context.TournamentInfo.Field))
	
	if context.CourseCharacteristics != nil {
		builder.WriteString("### Course Characteristics\n")
		builder.WriteString(fmt.Sprintf("- **Difficulty Rating:** %.1f\n", context.CourseCharacteristics.DifficultyRating))
		builder.WriteString(fmt.Sprintf("- **Length:** %d yards, Par %d\n", context.CourseCharacteristics.Length, context.CourseCharacteristics.Par))
		builder.WriteString("- **Key Skill Premiums:**\n")
		for skill, premium := range context.CourseCharacteristics.SkillPremiums {
			builder.WriteString(fmt.Sprintf("  - %s: %.2fx importance\n", skill, premium))
		}
		builder.WriteString("\n")
	}
	
	if context.CutLineProjection != nil {
		builder.WriteString("### Cut Line Projection\n")
		builder.WriteString(fmt.Sprintf("- **Projected Cut:** %+.1f\n", context.CutLineProjection.ProjectedCut))
		builder.WriteString(fmt.Sprintf("- **Confidence Interval:** %+.1f to %+.1f\n", context.CutLineProjection.LowerBound, context.CutLineProjection.UpperBound))
		builder.WriteString("\n")
	}

	if len(context.KeyNarratives) > 0 {
		builder.WriteString("### Key Tournament Narratives\n")
		for _, narrative := range context.KeyNarratives {
			builder.WriteString(fmt.Sprintf("- %s\n", narrative))
		}
		builder.WriteString("\n")
	}
}

func (dgpb *DataGolfAIPromptBuilder) writePlayerAnalysisSection(builder *strings.Builder, analysis *PlayerAnalysis) {
	builder.WriteString("## Player Analysis\n\n")
	
	if len(analysis.TopPerformers) > 0 {
		builder.WriteString("### Top Performers\n")
		for i, player := range analysis.TopPerformers[:min(5, len(analysis.TopPerformers))] {
			builder.WriteString(fmt.Sprintf("%d. **%s** ($%d) - Proj: %.1f | Value: %.2f\n",
				i+1, player.PlayerName, player.Salary, player.Projection, player.Value))
			dgpb.writePlayerInsights(builder, player)
		}
		builder.WriteString("\n")
	}

	if len(analysis.ValuePlays) > 0 {
		builder.WriteString("### Value Plays\n")
		for i, player := range analysis.ValuePlays[:min(5, len(analysis.ValuePlays))] {
			builder.WriteString(fmt.Sprintf("%d. **%s** ($%d) - Value: %.2f\n",
				i+1, player.PlayerName, player.Salary, player.Value))
			dgpb.writePlayerInsights(builder, player)
		}
		builder.WriteString("\n")
	}

	if len(analysis.SGCategoryLeaders) > 0 {
		builder.WriteString("### Strokes Gained Category Leaders\n")
		for category, leaders := range analysis.SGCategoryLeaders {
			builder.WriteString(fmt.Sprintf("**%s:**\n", formatSGCategory(category)))
			for i, leader := range leaders[:min(3, len(leaders))] {
				builder.WriteString(fmt.Sprintf("  %d. %s (%.2f SG)\n", i+1, leader.PlayerName, getSGValue(leader, category)))
			}
		}
		builder.WriteString("\n")
	}
}

func (dgpb *DataGolfAIPromptBuilder) writeCourseFitSection(builder *strings.Builder, matrix *CourseFitMatrix) {
	builder.WriteString("## Course Fit Analysis\n\n")
	
	if len(matrix.TopCourseFits) > 0 {
		builder.WriteString("### Excellent Course Fits\n")
		for i, fit := range matrix.TopCourseFits[:min(8, len(matrix.TopCourseFits))] {
			builder.WriteString(fmt.Sprintf("%d. **%s** - Fit Score: %.2f (%.0f%% confidence)\n",
				i+1, fit.PlayerName, fit.FitScore, fit.ConfidenceLevel*100))
			if len(fit.KeyAdvantages) > 0 {
				builder.WriteString("   - Advantages: " + strings.Join(fit.KeyAdvantages, ", ") + "\n")
			}
		}
		builder.WriteString("\n")
	}

	if matrix.SkillPremiumAnalysis != nil {
		builder.WriteString("### Course Skill Premiums\n")
		spa := matrix.SkillPremiumAnalysis
		builder.WriteString(fmt.Sprintf("- **Driving Distance:** %.1fx importance\n", spa.DrivingDistanceWeight))
		builder.WriteString(fmt.Sprintf("- **Driving Accuracy:** %.1fx importance\n", spa.DrivingAccuracyWeight))
		builder.WriteString(fmt.Sprintf("- **Approach Play:** %.1fx importance\n", spa.ApproachWeight))
		builder.WriteString(fmt.Sprintf("- **Short Game:** %.1fx importance\n", spa.ShortGameWeight))
		builder.WriteString(fmt.Sprintf("- **Putting:** %.1fx importance\n", spa.PuttingWeight))
		builder.WriteString("\n")
	}
}

func (dgpb *DataGolfAIPromptBuilder) writeWeatherSection(builder *strings.Builder, analysis *WeatherAnalysis) {
	builder.WriteString("## Weather Impact Analysis\n\n")
	
	if analysis.CurrentConditions != nil {
		conditions := analysis.CurrentConditions
		builder.WriteString("### Current Conditions\n")
		builder.WriteString(fmt.Sprintf("- **Wind:** %s at %.1f mph (gusts to %.1f mph)\n",
			conditions.Direction, conditions.Speed, conditions.Gusts))
		builder.WriteString(fmt.Sprintf("- **Temperature:** Variable conditions expected\n"))
		builder.WriteString("\n")
	}

	if len(analysis.ConditionAdvantages) > 0 {
		builder.WriteString("### Weather Advantages\n")
		for i, advantage := range analysis.ConditionAdvantages[:min(5, len(analysis.ConditionAdvantages))] {
			builder.WriteString(fmt.Sprintf("%d. **%s** - %s (+%.1f expected boost)\n",
				i+1, advantage.PlayerName, advantage.AdvantageType, advantage.ImpactMagnitude))
		}
		builder.WriteString("\n")
	}
}

func (dgpb *DataGolfAIPromptBuilder) writeCorrelationSection(builder *strings.Builder, insights *CorrelationInsights) {
	builder.WriteString("## Correlation & Stacking Insights\n\n")
	
	if len(insights.StackingOpportunities) > 0 {
		builder.WriteString("### Top Stacking Opportunities\n")
		for i, opportunity := range insights.StackingOpportunities[:min(3, len(insights.StackingOpportunities))] {
			builder.WriteString(fmt.Sprintf("%d. **%s Strategy** (%.2f correlation)\n",
				i+1, opportunity.StackType, opportunity.ExpectedCorrelation))
			builder.WriteString(fmt.Sprintf("   - Players: %s\n", strings.Join(opportunity.PlayerNames, ", ")))
			builder.WriteString(fmt.Sprintf("   - Rationale: %s\n", opportunity.Rationale))
		}
		builder.WriteString("\n")
	}

	if len(insights.ContrarianPlays) > 0 {
		builder.WriteString("### Contrarian Opportunities\n")
		for i, play := range insights.ContrarianPlays[:min(5, len(insights.ContrarianPlays))] {
			builder.WriteString(fmt.Sprintf("%d. **%s** - %.1f%% ownership vs %.1f%% optimal\n",
				i+1, play.PlayerName, play.ProjectedOwnership*100, play.OptimalOwnership*100))
		}
		builder.WriteString("\n")
	}
}

func (dgpb *DataGolfAIPromptBuilder) writeStrategySection(builder *strings.Builder, recommendations *StrategyRecommendations) {
	builder.WriteString("## Strategy Recommendations\n\n")
	
	if recommendations.TournamentStrategy != nil {
		strategy := recommendations.TournamentStrategy
		builder.WriteString("### Tournament-Specific Strategy\n")
		builder.WriteString(fmt.Sprintf("- **Primary Approach:** %s\n", strategy.PrimaryApproach))
		builder.WriteString(fmt.Sprintf("- **Volatility Target:** %s\n", strategy.VolatilityTarget))
		builder.WriteString(fmt.Sprintf("- **Cut Probability Focus:** %.0f%% minimum\n", strategy.CutProbabilityFocus*100))
		builder.WriteString("\n")
	}

	if recommendations.OwnershipStrategy != nil {
		ownership := recommendations.OwnershipStrategy
		builder.WriteString("### Ownership Strategy\n")
		builder.WriteString(fmt.Sprintf("- **Chalk Exposure:** %s\n", ownership.ChalkExposure))
		builder.WriteString(fmt.Sprintf("- **Contrarian Focus:** %s\n", ownership.ContrarianFocus))
		builder.WriteString(fmt.Sprintf("- **Leverage Opportunities:** %d identified\n", len(ownership.LeverageOpportunities)))
		builder.WriteString("\n")
	}
}

func (dgpb *DataGolfAIPromptBuilder) writeMetricPrioritizationSection(builder *strings.Builder, prioritization *MetricPrioritization) {
	builder.WriteString("## Metric Prioritization\n\n")
	
	builder.WriteString("### Primary Metrics (Highest Weight)\n")
	for i, metric := range prioritization.PrimaryMetrics {
		importance := prioritization.ContextualImportance[metric]
		builder.WriteString(fmt.Sprintf("%d. %s (%.1fx weight)\n", i+1, formatMetricName(metric), importance))
	}
	builder.WriteString("\n")

	if len(prioritization.WeatherAdjustedMetrics) > 0 {
		builder.WriteString("### Weather-Adjusted Focus\n")
		for _, metric := range prioritization.WeatherAdjustedMetrics {
			builder.WriteString(fmt.Sprintf("- %s (conditions-specific importance)\n", formatMetricName(metric)))
		}
		builder.WriteString("\n")
	}
}

func (dgpb *DataGolfAIPromptBuilder) writePlayerInsights(builder *strings.Builder, player *PlayerInsight) {
	if len(player.KeyInsights) > 0 {
		builder.WriteString("   - " + strings.Join(player.KeyInsights, "; ") + "\n")
	}
	if len(player.RiskFactors) > 0 {
		builder.WriteString("   - **Risks:** " + strings.Join(player.RiskFactors, ", ") + "\n")
	}
}

// Placeholder implementations for missing functions and types
type TournamentInfo struct {
	Name   string    `json:"name"`
	Date   time.Time `json:"date"`
	Purse  int64     `json:"purse"`
	Field  int       `json:"field"`
	Format string    `json:"format"`
}

type CourseCharacteristics struct {
	DifficultyRating float64            `json:"difficulty_rating"`
	Length           int                `json:"length"`
	Par              int                `json:"par"`
	SkillPremiums    map[string]float64 `json:"skill_premiums"`
}

type HistoricalTournamentData struct{}
type CutLineProjection struct {
	ProjectedCut float64 `json:"projected_cut"`
	LowerBound   float64 `json:"lower_bound"`
	UpperBound   float64 `json:"upper_bound"`
}
type FieldStrength struct{}
type RecentForm struct{}
type CourseFitInsight struct {
	PlayerName       string   `json:"player_name"`
	FitScore         float64  `json:"fit_score"`
	ConfidenceLevel  float64  `json:"confidence_level"`
	KeyAdvantages    []string `json:"key_advantages"`
}
type SkillPremiumAnalysis struct {
	DrivingDistanceWeight float64 `json:"driving_distance_weight"`
	DrivingAccuracyWeight float64 `json:"driving_accuracy_weight"`
	ApproachWeight        float64 `json:"approach_weight"`
	ShortGameWeight       float64 `json:"short_game_weight"`
	PuttingWeight         float64 `json:"putting_weight"`
}
type HistoricalAdvantage struct{}
type WeatherProfile struct{}
type WeatherAdvantage struct {
	PlayerName      string  `json:"player_name"`
	AdvantageType   string  `json:"advantage_type"`
	ImpactMagnitude float64 `json:"impact_magnitude"`
}
type TeeTimeWeatherImpact struct{}
type StackingOpportunity struct {
	StackType           string   `json:"stack_type"`
	PlayerNames         []string `json:"player_names"`
	ExpectedCorrelation float64  `json:"expected_correlation"`
	Rationale           string   `json:"rationale"`
}
type ContrarianPlay struct {
	PlayerName        string  `json:"player_name"`
	ProjectedOwnership float64 `json:"projected_ownership"`
	OptimalOwnership  float64 `json:"optimal_ownership"`
}
type LineupSynergy struct{}
type AntiCorrelationBenefit struct{}
type StrategyRecommendation struct{}
type TournamentStrategy struct {
	PrimaryApproach      string  `json:"primary_approach"`
	VolatilityTarget     string  `json:"volatility_target"`
	CutProbabilityFocus  float64 `json:"cut_probability_focus"`
}
type RiskManagement struct{}
type OwnershipStrategy struct {
	ChalkExposure        string   `json:"chalk_exposure"`
	ContrarianFocus      string   `json:"contrarian_focus"`
	LeverageOpportunities []string `json:"leverage_opportunities"`
}
type StackingGuidance struct{}
type CashVsTournamentGuidance struct{}
type MarketInefficiencies struct{}
type RiskAssessment struct{}
type OptimizationFocus struct{}
type MarketData struct{}

// Constructor functions
func NewContextualDataBuilder() *ContextualDataBuilder { return &ContextualDataBuilder{} }
func NewSGMetricAnalyzer() *SGMetricAnalyzer { return &SGMetricAnalyzer{} }
func NewCorrelationInsightAnalyzer() *CorrelationInsightAnalyzer { return &CorrelationInsightAnalyzer{} }
func NewStrategyRecommendationEngine() *StrategyRecommendationEngine { return &StrategyRecommendationEngine{} }
func NewWeatherAnalyzer() *WeatherAnalyzer { return &WeatherAnalyzer{} }
func NewCutLineAnalyzer() *CutLineAnalyzer { return &CutLineAnalyzer{} }
func NewOwnershipAnalyzer() *OwnershipAnalyzer { return &OwnershipAnalyzer{} }

// Placeholder analyzer types
type WeatherAnalyzer struct{}
type CutLineAnalyzer struct{}
type OwnershipAnalyzer struct{}

// Placeholder methods
func (dgpb *DataGolfAIPromptBuilder) analyzeCourseCharacteristics(analytics *CourseAnalytics) *CourseCharacteristics {
	if analytics == nil {
		return &CourseCharacteristics{
			DifficultyRating: 3.5,
			Length:          7200,
			Par:             72,
			SkillPremiums:   map[string]float64{"driving": 1.2, "approach": 1.0, "short_game": 0.8, "putting": 0.9},
		}
	}
	return &CourseCharacteristics{
		DifficultyRating: analytics.DifficultyRating,
		Length:          analytics.Length,
		Par:             analytics.Par,
		SkillPremiums:   analytics.SkillPremiums.ToMap(),
	}
}

func (dgpb *DataGolfAIPromptBuilder) processHistoricalData(tournament *models.GolfTournament, insights *DataGolfInsights) *HistoricalTournamentData {
	return &HistoricalTournamentData{}
}

func (dgpb *DataGolfAIPromptBuilder) analyzeFieldStrength(tournament *models.GolfTournament, insights *DataGolfInsights) *FieldStrength {
	return &FieldStrength{}
}

func (dgpb *DataGolfAIPromptBuilder) generateKeyNarratives(tournament *models.GolfTournament, insights *DataGolfInsights) []string {
	return []string{
		"Course favors long, accurate drivers with strong approach play",
		"Weather conditions expected to be challenging with variable winds",
		"Projected cut line indicates competitive field depth",
	}
}

func (dgpb *DataGolfAIPromptBuilder) generatePlayerInsight(player *models.GolfPlayer, insights *DataGolfInsights) *PlayerInsight {
	sgMetrics := insights.StrokesGainedData[player.PlayerID]
	
	return &PlayerInsight{
		PlayerID:        player.PlayerID,
		PlayerName:      player.Name,
		Salary:          player.Salary,
		Projection:      player.Projection,
		Value:           player.Value,
		SGMetrics:       sgMetrics,
		KeyInsights:     []string{"Strong recent form", "Excellent course fit"},
		RiskFactors:     []string{"Weather sensitivity"},
		Advantages:      []string{"Course history", "Current form"},
	}
}

func (dgpb *DataGolfAIPromptBuilder) identifyTopPerformers(insights []*PlayerInsight) []*PlayerInsight {
	sorted := make([]*PlayerInsight, len(insights))
	copy(sorted, insights)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Projection > sorted[j].Projection
	})
	return sorted
}

func (dgpb *DataGolfAIPromptBuilder) identifyValuePlays(insights []*PlayerInsight) []*PlayerInsight {
	sorted := make([]*PlayerInsight, len(insights))
	copy(sorted, insights)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	return sorted
}

func (dgpb *DataGolfAIPromptBuilder) identifyCourseFitExcellence(insights []*PlayerInsight) []*PlayerInsight {
	return insights
}

func (dgpb *DataGolfAIPromptBuilder) analyzeRecentForm(insights []*PlayerInsight) []*PlayerInsight {
	return insights
}

func (dgpb *DataGolfAIPromptBuilder) analyzeSGCategoryLeaders(insights []*PlayerInsight) map[string][]*PlayerInsight {
	return map[string][]*PlayerInsight{
		"sg_off_the_tee":        insights,
		"sg_approach":           insights,
		"sg_around_the_green":   insights,
		"sg_putting":            insights,
	}
}

func (dgpb *DataGolfAIPromptBuilder) analyzeConsistency(insights []*PlayerInsight) []*PlayerInsight {
	return insights
}

func (dgpb *DataGolfAIPromptBuilder) analyzeVolatility(insights []*PlayerInsight) []*PlayerInsight {
	return insights
}

func (dgpb *DataGolfAIPromptBuilder) assessInjuryRisk(insights []*PlayerInsight) []*PlayerInsight {
	return insights
}

func (dgpb *DataGolfAIPromptBuilder) identifyTopCourseFits(fits []*CourseFitInsight) []*CourseFitInsight {
	sorted := make([]*CourseFitInsight, len(fits))
	copy(sorted, fits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].FitScore > sorted[j].FitScore
	})
	return sorted
}

func (dgpb *DataGolfAIPromptBuilder) identifyCourseFitMismatches(fits []*CourseFitInsight) []*CourseFitInsight {
	return fits
}

func (dgpb *DataGolfAIPromptBuilder) analyzeSkillPremiums(analytics *CourseAnalytics) *SkillPremiumAnalysis {
	if analytics == nil {
		return &SkillPremiumAnalysis{
			DrivingDistanceWeight: 1.2,
			DrivingAccuracyWeight: 1.0,
			ApproachWeight:        1.1,
			ShortGameWeight:       0.9,
			PuttingWeight:         0.8,
		}
	}
	return &SkillPremiumAnalysis{
		DrivingDistanceWeight: analytics.SkillPremiums.DrivingDistance,
		DrivingAccuracyWeight: analytics.SkillPremiums.DrivingAccuracy,
		ApproachWeight:        analytics.SkillPremiums.ApproachPrecision,
		ShortGameWeight:       analytics.SkillPremiums.ShortGameSkill,
		PuttingWeight:         analytics.SkillPremiums.PuttingConsistency,
	}
}

func (dgpb *DataGolfAIPromptBuilder) identifyHistoricalAdvantages(players []*models.GolfPlayer, insights *DataGolfInsights) []*HistoricalAdvantage {
	return make([]*HistoricalAdvantage, 0)
}

func (dgpb *DataGolfAIPromptBuilder) calculateWeatherAdjustments(insights *DataGolfInsights) map[string]float64 {
	return map[string]float64{"wind_adjustment": 0.1, "temperature_adjustment": 0.05}
}

func (dgpb *DataGolfAIPromptBuilder) identifyPrimaryMetrics(analytics *CourseAnalytics) []string {
	return []string{"sg_off_the_tee", "sg_approach", "driving_distance"}
}

func (dgpb *DataGolfAIPromptBuilder) identifySecondaryMetrics(analytics *CourseAnalytics) []string {
	return []string{"sg_around_the_green", "sg_putting", "course_history"}
}

func (dgpb *DataGolfAIPromptBuilder) calculateContextualImportance(tournament *models.GolfTournament, insights *DataGolfInsights) map[string]float64 {
	return map[string]float64{
		"sg_off_the_tee":        1.3,
		"sg_approach":           1.2,
		"driving_distance":      1.1,
		"sg_around_the_green":   0.9,
		"sg_putting":            0.8,
	}
}

func (dgpb *DataGolfAIPromptBuilder) identifyWeatherAdjustedMetrics(insights *DataGolfInsights) []string {
	return []string{"driving_accuracy", "approach_precision"}
}

func (dgpb *DataGolfAIPromptBuilder) identifyCourseSpecificMetrics(analytics *CourseAnalytics) []string {
	return []string{"course_history", "course_fit_rating"}
}

func (dgpb *DataGolfAIPromptBuilder) buildMarketInefficiencies(players []*models.GolfPlayer, insights *DataGolfInsights) *MarketInefficiencies {
	return &MarketInefficiencies{}
}

func (dgpb *DataGolfAIPromptBuilder) buildRiskAssessment(tournament *models.GolfTournament, players []*models.GolfPlayer, insights *DataGolfInsights) *RiskAssessment {
	return &RiskAssessment{}
}

func (dgpb *DataGolfAIPromptBuilder) buildOptimizationFocus(constraints *models.OptimizationConstraints, tournament *models.GolfTournament, insights *DataGolfInsights) *OptimizationFocus {
	return &OptimizationFocus{}
}

func (dgpb *DataGolfAIPromptBuilder) writeMarketInefficienciesSection(builder *strings.Builder, inefficiencies *MarketInefficiencies) {
	builder.WriteString("## Market Inefficiencies\n\n")
	builder.WriteString("- Analysis of pricing vs. expected performance\n")
	builder.WriteString("- Ownership leverage opportunities identified\n\n")
}

func (dgpb *DataGolfAIPromptBuilder) writeRiskAssessmentSection(builder *strings.Builder, assessment *RiskAssessment) {
	builder.WriteString("## Risk Assessment\n\n")
	builder.WriteString("- Portfolio volatility considerations\n")
	builder.WriteString("- Cut line risk evaluation\n\n")
}

func (dgpb *DataGolfAIPromptBuilder) writeOptimizationFocusSection(builder *strings.Builder, focus *OptimizationFocus) {
	builder.WriteString("## Optimization Focus\n\n")
	builder.WriteString("- Tournament-specific strategy recommendations\n")
	builder.WriteString("- Balanced approach to risk and reward\n\n")
}

// Method implementations for missing analyzer types
func (wa *WeatherAnalyzer) AnalyzeWeatherImpact(tournament *models.GolfTournament, insights *DataGolfInsights) *WeatherAnalysis {
	return &WeatherAnalysis{
		CurrentConditions: insights.WeatherImpactData.CurrentConditions,
		ConditionAdvantages: []*WeatherAdvantage{
			{PlayerName: "Example Player", AdvantageType: "Wind Resistance", ImpactMagnitude: 0.3},
		},
	}
}

func (cla *CutLineAnalyzer) ProjectCutLine(tournament *models.GolfTournament, insights *DataGolfInsights) *CutLineProjection {
	return &CutLineProjection{
		ProjectedCut: -4.5,
		LowerBound:   -6.0,
		UpperBound:   -3.0,
	}
}

func (cfa *CourseFitAnalyzer) AnalyzeCourseFit(player *models.GolfPlayer, tournament *models.GolfTournament, insights *DataGolfInsights) *CourseFitInsight {
	return &CourseFitInsight{
		PlayerName:      player.Name,
		FitScore:        0.75,
		ConfidenceLevel: 0.85,
		KeyAdvantages:   []string{"Length advantage", "Approach precision"},
	}
}

func (cia *CorrelationInsightAnalyzer) AnalyzeCorrelations(players []*models.GolfPlayer, insights *DataGolfInsights) *CorrelationInsights {
	return &CorrelationInsights{
		StackingOpportunities: []*StackingOpportunity{
			{StackType: "Tee Time", PlayerNames: []string{"Player 1", "Player 2"}, ExpectedCorrelation: 0.35, Rationale: "Same wave advantage"},
		},
		ContrarianPlays: []*ContrarianPlay{
			{PlayerName: "Value Player", ProjectedOwnership: 0.08, OptimalOwnership: 0.15},
		},
	}
}

func (sre *StrategyRecommendationEngine) GenerateRecommendations(constraints *models.OptimizationConstraints, insights *DataGolfInsights) *StrategyRecommendations {
	return &StrategyRecommendations{
		TournamentStrategy: &TournamentStrategy{
			PrimaryApproach:     "Balanced Upside",
			VolatilityTarget:    "Medium-High",
			CutProbabilityFocus: 0.75,
		},
		OwnershipStrategy: &OwnershipStrategy{
			ChalkExposure:   "Moderate",
			ContrarianFocus: "High-Value Contrarian",
			LeverageOpportunities: []string{"Underpriced veterans", "Weather-advantaged players"},
		},
	}
}

// Helper functions
func formatMoney(amount int64) string {
	return fmt.Sprintf("%.1fM", float64(amount)/1000000)
}

func formatSGCategory(category string) string {
	switch category {
	case "sg_off_the_tee":
		return "Off the Tee"
	case "sg_approach":
		return "Approach"
	case "sg_around_the_green":
		return "Around the Green"
	case "sg_putting":
		return "Putting"
	default:
		return category
	}
}

func formatMetricName(metric string) string {
	switch metric {
	case "sg_off_the_tee":
		return "Strokes Gained: Off the Tee"
	case "sg_approach":
		return "Strokes Gained: Approach"
	case "driving_distance":
		return "Driving Distance"
	case "driving_accuracy":
		return "Driving Accuracy"
	case "course_history":
		return "Course History"
	default:
		return metric
	}
}

func getSGValue(player *PlayerInsight, category string) float64 {
	if player.SGMetrics == nil {
		return 0.0
	}
	switch category {
	case "sg_off_the_tee":
		return player.SGMetrics.SGOffTheTee
	case "sg_approach":
		return player.SGMetrics.SGApproach
	case "sg_around_the_green":
		return player.SGMetrics.SGAroundTheGreen
	case "sg_putting":
		return player.SGMetrics.SGPutting
	default:
		return 0.0
	}
}

func (sp SkillPremiumWeights) ToMap() map[string]float64 {
	return map[string]float64{
		"driving_distance":    sp.DrivingDistance,
		"driving_accuracy":    sp.DrivingAccuracy,
		"approach_precision":  sp.ApproachPrecision,
		"short_game_skill":    sp.ShortGameSkill,
		"putting_consistency": sp.PuttingConsistency,
	}
}