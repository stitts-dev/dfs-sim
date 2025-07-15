package services

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
)

// AIEngine orchestrates the AI recommendation generation process
type AIEngine struct {
	claudeClient       *ClaudeClient
	promptBuilder      *PromptBuilder
	realtimeAggregator *RealtimeAggregator
	ownershipAnalyzer  *OwnershipAnalyzer
	logger             *logrus.Logger
}

// RecommendationRequest represents a request for AI recommendations
type RecommendationRequest struct {
	ContestID            uint                     `json:"contest_id"`
	UserID               uint                     `json:"user_id"`
	Players              []models.PlayerRecommendation `json:"players"`
	Context              models.PromptContext     `json:"context"`
	RequestType          string                   `json:"request_type"` // "player_recommendations", "lineup_analysis", "late_swap"
	IncludeRealTimeData  bool                     `json:"include_realtime_data"`
	IncludeLeverageAnalysis bool                  `json:"include_leverage_analysis"`
	MaxRecommendations   int                      `json:"max_recommendations"`
	CacheResults         bool                     `json:"cache_results"`
}

// RecommendationResponse represents the complete AI recommendation response
type RecommendationResponse struct {
	Recommendations       []models.PlayerRecommendation  `json:"recommendations"`
	ContextInsights       []models.ContextInsight        `json:"context_insights"`
	OwnershipAnalysis     *models.OwnershipAnalysis      `json:"ownership_analysis"`
	StackSuggestions      []models.StackSuggestion       `json:"stack_suggestions"`
	LeverageOpportunities []LeveragePlay                 `json:"leverage_opportunities"`
	RealTimeAlerts        []models.RealTimeAlert         `json:"realtime_alerts"`
	Confidence            float64                        `json:"confidence"`
	ReasoningPath         []string                       `json:"reasoning_path"`
	ModelUsed             string                         `json:"model_used"`
	TimestampGenerated    time.Time                      `json:"timestamp_generated"`
	ProcessingTimeMs      int64                          `json:"processing_time_ms"`
	TokensUsed            int                            `json:"tokens_used"`
	CacheHit              bool                           `json:"cache_hit"`
	RequestID             string                         `json:"request_id"`
}

// LineupAnalysisRequest represents a request to analyze an existing lineup
type LineupAnalysisRequest struct {
	ContestID     uint                     `json:"contest_id"`
	UserID        uint                     `json:"user_id"`
	Lineup        []models.PlayerRecommendation `json:"lineup"`
	Context       models.PromptContext     `json:"context"`
	AnalysisType  string                   `json:"analysis_type"` // "full", "quick", "ownership_focused"
}

// LineupAnalysisResponse represents the AI analysis of a lineup
type LineupAnalysisResponse struct {
	OverallRating      float64                    `json:"overall_rating"`
	StrengthAreas      []string                   `json:"strength_areas"`
	WeaknessAreas      []string                   `json:"weakness_areas"`
	RiskAssessment     string                     `json:"risk_assessment"`
	OwnershipProfile   string                     `json:"ownership_profile"`
	ExpectedROI        float64                    `json:"expected_roi"`
	ImprovementSuggestions []string               `json:"improvement_suggestions"`
	AlternativeLineups [][]models.PlayerRecommendation `json:"alternative_lineups"`
	Confidence         float64                    `json:"confidence"`
	TimestampGenerated time.Time                  `json:"timestamp_generated"`
}

// NewAIEngine creates a new AI engine instance
func NewAIEngine(
	claudeClient *ClaudeClient,
	promptBuilder *PromptBuilder,
	realtimeAggregator *RealtimeAggregator,
	ownershipAnalyzer *OwnershipAnalyzer,
	logger *logrus.Logger,
) *AIEngine {
	return &AIEngine{
		claudeClient:       claudeClient,
		promptBuilder:      promptBuilder,
		realtimeAggregator: realtimeAggregator,
		ownershipAnalyzer:  ownershipAnalyzer,
		logger:             logger,
	}
}

// GenerateRecommendations orchestrates the AI recommendation generation process
func (ae *AIEngine) GenerateRecommendations(ctx context.Context, request *RecommendationRequest) (*RecommendationResponse, error) {
	startTime := time.Now()
	requestID := ae.generateRequestID(request)

	ae.logger.WithFields(logrus.Fields{
		"request_id":   requestID,
		"contest_id":   request.ContestID,
		"user_id":      request.UserID,
		"request_type": request.RequestType,
	}).Info("Starting AI recommendation generation")

	// Check cache first if enabled
	if request.CacheResults {
		if cachedResponse, err := ae.getCachedResponse(requestID); err == nil {
			ae.logger.WithField("request_id", requestID).Debug("Returning cached response")
			cachedResponse.CacheHit = true
			return cachedResponse, nil
		}
	}

	// Initialize response
	response := &RecommendationResponse{
		RequestID:          requestID,
		TimestampGenerated: time.Now(),
		CacheHit:          false,
		ReasoningPath:     []string{},
	}

	// Step 1: Enhance context with real-time data
	if request.IncludeRealTimeData {
		ae.addToReasoningPath(response, "Collecting real-time data updates")
		if err := ae.enhanceContextWithRealTimeData(ctx, request); err != nil {
			ae.logger.WithError(err).Warn("Failed to enhance context with real-time data")
		}
	}

	// Step 2: Get ownership analysis
	if request.IncludeLeverageAnalysis {
		ae.addToReasoningPath(response, "Analyzing ownership and leverage opportunities")
		ownershipAnalysis, err := ae.ownershipAnalyzer.GetOwnershipInsights(request.ContestID)
		if err != nil {
			ae.logger.WithError(err).Warn("Failed to get ownership analysis")
		} else {
			response.OwnershipAnalysis = ownershipAnalysis
		}

		// Calculate leverage opportunities
		leveragePlays, err := ae.ownershipAnalyzer.CalculateLeverageOpportunities(
			request.ContestID,
			request.Players,
			request.Context.ContestType,
			request.Context.ExistingLineups,
		)
		if err != nil {
			ae.logger.WithError(err).Warn("Failed to calculate leverage opportunities")
		} else {
			response.LeverageOpportunities = leveragePlays
		}
	}

	// Step 3: Build AI prompt
	ae.addToReasoningPath(response, "Building dynamic AI prompt")
	prompt, systemPrompt, err := ae.promptBuilder.BuildRecommendationPrompt(request.Context, request.Players)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Step 4: Generate AI response
	ae.addToReasoningPath(response, "Generating AI recommendations with Claude")
	claudeConfig := ae.claudeClient.BuildDefaultConfig(request.RequestType)
	claudeResponse, err := ae.claudeClient.SendMessage(ctx, prompt, systemPrompt, claudeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI response: %w", err)
	}

	response.ModelUsed = claudeResponse.Model
	response.TokensUsed = claudeResponse.Usage.InputTokens + claudeResponse.Usage.OutputTokens

	// Step 5: Parse AI response
	ae.addToReasoningPath(response, "Parsing and structuring AI response")
	if err := ae.parseAIResponse(claudeResponse, response); err != nil {
		ae.logger.WithError(err).Error("Failed to parse AI response")
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Step 6: Enhance with computed metrics
	ae.addToReasoningPath(response, "Computing additional metrics and insights")
	ae.enhanceResponseWithMetrics(response, request)

	// Step 7: Generate real-time alerts
	if request.IncludeRealTimeData {
		ae.addToReasoningPath(response, "Checking for real-time alerts")
		alerts := ae.generateRealTimeAlerts(ctx, request.ContestID, response.Recommendations)
		response.RealTimeAlerts = alerts
	}

	// Step 8: Calculate final confidence and quality scores
	ae.addToReasoningPath(response, "Calculating confidence and quality metrics")
	response.Confidence = ae.calculateOverallConfidence(response, claudeResponse.Usage)

	// Calculate processing time
	response.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Cache response if enabled
	if request.CacheResults && response.Confidence > 0.7 {
		ae.cacheResponse(requestID, response)
	}

	ae.logger.WithFields(logrus.Fields{
		"request_id":         requestID,
		"processing_time_ms": response.ProcessingTimeMs,
		"tokens_used":        response.TokensUsed,
		"confidence":         response.Confidence,
		"recommendations":    len(response.Recommendations),
	}).Info("AI recommendation generation completed")

	return response, nil
}

// AnalyzeLineup provides detailed analysis of an existing lineup
func (ae *AIEngine) AnalyzeLineup(ctx context.Context, request *LineupAnalysisRequest) (*LineupAnalysisResponse, error) {
	startTime := time.Now()

	ae.logger.WithFields(logrus.Fields{
		"contest_id":    request.ContestID,
		"user_id":       request.UserID,
		"analysis_type": request.AnalysisType,
		"lineup_size":   len(request.Lineup),
	}).Info("Starting lineup analysis")

	// Build analysis prompt
	prompt, systemPrompt := ae.buildLineupAnalysisPrompt(request)

	// Get AI analysis
	claudeConfig := ae.claudeClient.BuildDefaultConfig("lineup_analysis")
	claudeResponse, err := ae.claudeClient.SendMessage(ctx, prompt, systemPrompt, claudeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI analysis: %w", err)
	}

	// Parse response
	response := &LineupAnalysisResponse{
		TimestampGenerated: time.Now(),
	}

	if err := ae.parseLineupAnalysisResponse(claudeResponse, response); err != nil {
		return nil, fmt.Errorf("failed to parse analysis response: %w", err)
	}

	// Enhance with computed metrics
	ae.enhanceLineupAnalysisWithMetrics(response, request)

	ae.logger.WithFields(logrus.Fields{
		"contest_id":       request.ContestID,
		"processing_time":  time.Since(startTime).Milliseconds(),
		"overall_rating":   response.OverallRating,
		"expected_roi":     response.ExpectedROI,
	}).Info("Lineup analysis completed")

	return response, nil
}

// Helper methods for recommendation generation

func (ae *AIEngine) enhanceContextWithRealTimeData(ctx context.Context, request *RecommendationRequest) error {
	// Get real-time updates for the contest
	updateChan := ae.realtimeAggregator.StreamUpdates(ctx, request.ContestID)
	
	// Collect updates for a short period (non-blocking)
	timeout := time.After(2 * time.Second)
	var updates []models.RealtimeDataPoint

	for {
		select {
		case update := <-updateChan:
			updates = append(updates, update)
			if len(updates) >= 10 { // Limit to prevent overflow
				break
			}
		case <-timeout:
			goto done
		default:
			goto done
		}
	}

done:
	// Add collected updates to context
	request.Context.RealTimeData = append(request.Context.RealTimeData, updates...)

	ae.logger.WithFields(logrus.Fields{
		"contest_id":    request.ContestID,
		"updates_count": len(updates),
	}).Debug("Enhanced context with real-time data")

	return nil
}

func (ae *AIEngine) parseAIResponse(claudeResponse *ClaudeResponse, response *RecommendationResponse) error {
	// Extract text content from Claude response
	var aiText string
	for _, content := range claudeResponse.Content {
		if content.Type == "text" {
			aiText += content.Text
		}
	}

	// Parse structured response from AI text
	// This would involve more sophisticated parsing depending on the AI's output format
	recommendations, insights, stacks := ae.parseRecommendationText(aiText)
	
	response.Recommendations = recommendations
	response.ContextInsights = insights
	response.StackSuggestions = stacks

	return nil
}

func (ae *AIEngine) parseRecommendationText(aiText string) (
	[]models.PlayerRecommendation,
	[]models.ContextInsight,
	[]models.StackSuggestion,
) {
	// This is a simplified parser - in production, you'd want more robust parsing
	var recommendations []models.PlayerRecommendation
	var insights []models.ContextInsight
	var stacks []models.StackSuggestion

	// Parse sections of the AI response
	lines := strings.Split(aiText, "\n")
	currentSection := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Identify sections
		if strings.Contains(strings.ToUpper(line), "RECOMMENDATIONS") {
			currentSection = "recommendations"
			continue
		} else if strings.Contains(strings.ToUpper(line), "INSIGHTS") {
			currentSection = "insights"
			continue
		} else if strings.Contains(strings.ToUpper(line), "STACKS") {
			currentSection = "stacks"
			continue
		}

		// Parse content based on section
		switch currentSection {
		case "recommendations":
			if rec := ae.parseRecommendationLine(line); rec != nil {
				recommendations = append(recommendations, *rec)
			}
		case "insights":
			if insight := ae.parseInsightLine(line); insight != nil {
				insights = append(insights, *insight)
			}
		case "stacks":
			if stack := ae.parseStackLine(line); stack != nil {
				stacks = append(stacks, *stack)
			}
		}
	}

	return recommendations, insights, stacks
}

func (ae *AIEngine) parseRecommendationLine(line string) *models.PlayerRecommendation {
	// Simplified parsing - would be more sophisticated in production
	if strings.Contains(line, "$") && strings.Contains(line, "%") {
		return &models.PlayerRecommendation{
			PlayerName:      "Parsed Player", // Would extract from line
			RecommendReason: line,
			Confidence:      0.8,
		}
	}
	return nil
}

func (ae *AIEngine) parseInsightLine(line string) *models.ContextInsight {
	return &models.ContextInsight{
		InsightType: "general",
		Message:     line,
		Confidence:  0.7,
	}
}

func (ae *AIEngine) parseStackLine(line string) *models.StackSuggestion {
	return &models.StackSuggestion{
		StackType:  "team",
		Reasoning:  line,
		Confidence: 0.7,
	}
}

func (ae *AIEngine) enhanceResponseWithMetrics(response *RecommendationResponse, request *RecommendationRequest) {
	// Add computed metrics to recommendations
	for i := range response.Recommendations {
		rec := &response.Recommendations[i]
		
		// Calculate value metrics
		if rec.Salary > 0 {
			rec.Value = rec.Projection / (rec.Salary / 1000)
		}
		
		// Add risk level assessment
		if rec.Value > 3.5 {
			rec.RiskLevel = "low"
		} else if rec.Value > 2.5 {
			rec.RiskLevel = "medium"
		} else {
			rec.RiskLevel = "high"
		}
	}

	// Sort recommendations by confidence
	// sort.Slice(response.Recommendations, func(i, j int) bool {
	// 	return response.Recommendations[i].Confidence > response.Recommendations[j].Confidence
	// })

	// Limit to requested max
	if request.MaxRecommendations > 0 && len(response.Recommendations) > request.MaxRecommendations {
		response.Recommendations = response.Recommendations[:request.MaxRecommendations]
	}
}

func (ae *AIEngine) generateRealTimeAlerts(ctx context.Context, contestID uint, recommendations []models.PlayerRecommendation) []models.RealTimeAlert {
	var alerts []models.RealTimeAlert

	// Check for late-breaking news or changes affecting recommended players
	for _, rec := range recommendations {
		// Get latest real-time data for this player
		if latestData, err := ae.realtimeAggregator.GetLatestData(rec.PlayerID, "injury"); err == nil {
			if latestData.ImpactRating < -2 {
				alerts = append(alerts, models.RealTimeAlert{
					AlertType:    "injury",
					Severity:     "high",
					Message:      fmt.Sprintf("Injury concern for %s", rec.PlayerName),
					PlayerID:     &rec.PlayerID,
					Impact:       "negative",
					ActionNeeded: "Consider alternative",
					Timestamp:    time.Now(),
				})
			}
		}
	}

	return alerts
}

func (ae *AIEngine) calculateOverallConfidence(response *RecommendationResponse, usage ClaudeUsage) float64 {
	confidence := 0.8 // Base confidence

	// Adjust based on data quality
	if len(response.RealTimeAlerts) > 0 {
		confidence += 0.1 // Real-time data increases confidence
	}

	if response.OwnershipAnalysis != nil {
		confidence += 0.05 // Ownership data increases confidence
	}

	// Adjust based on token usage (more tokens = more thorough analysis)
	if usage.OutputTokens > 2000 {
		confidence += 0.05
	}

	// Adjust based on number of insights
	if len(response.ContextInsights) > 3 {
		confidence += 0.05
	}

	return min(1.0, confidence)
}

// Lineup analysis methods

func (ae *AIEngine) buildLineupAnalysisPrompt(request *LineupAnalysisRequest) (string, string) {
	prompt := fmt.Sprintf(`Analyze this DFS lineup for a %s contest:

LINEUP:
`, request.Context.ContestType)

	totalSalary := 0.0
	totalProjection := 0.0
	for _, player := range request.Lineup {
		prompt += fmt.Sprintf("- %s (%s): $%.0f, %.1f projected points\n",
			player.PlayerName, player.Position, player.Salary, player.Projection)
		totalSalary += player.Salary
		totalProjection += player.Projection
	}

	prompt += fmt.Sprintf(`
LINEUP TOTALS:
- Total Salary: $%.0f
- Total Projection: %.1f points
- Salary Cap: $%.0f

ANALYSIS REQUIREMENTS:
1. Overall lineup rating (0-10)
2. Strength areas
3. Weakness areas  
4. Risk assessment
5. Ownership profile assessment
6. Expected ROI estimate
7. Specific improvement suggestions
8. Alternative lineup suggestions

Provide detailed analysis with specific reasoning.`, 
		totalSalary, totalProjection, request.Context.ContestMeta.SalaryCap)

	systemPrompt := `You are an expert DFS analyst. Provide comprehensive lineup analysis with specific, actionable feedback. Consider player correlations, salary efficiency, upside potential, and contest-specific strategy.`

	return prompt, systemPrompt
}

func (ae *AIEngine) parseLineupAnalysisResponse(claudeResponse *ClaudeResponse, response *LineupAnalysisResponse) error {
	// Extract and parse the AI's lineup analysis
	var aiText string
	for _, content := range claudeResponse.Content {
		if content.Type == "text" {
			aiText += content.Text
		}
	}

	// Set defaults and parse specific sections
	response.OverallRating = 7.0
	response.ExpectedROI = 0.15
	response.Confidence = 0.8
	response.RiskAssessment = "Medium"
	response.OwnershipProfile = "Balanced"

	// Parse key insights from the text
	lines := strings.Split(aiText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "strength") {
			response.StrengthAreas = append(response.StrengthAreas, line)
		} else if strings.Contains(strings.ToLower(line), "weakness") {
			response.WeaknessAreas = append(response.WeaknessAreas, line)
		} else if strings.Contains(strings.ToLower(line), "suggestion") {
			response.ImprovementSuggestions = append(response.ImprovementSuggestions, line)
		}
	}

	return nil
}

func (ae *AIEngine) enhanceLineupAnalysisWithMetrics(response *LineupAnalysisResponse, request *LineupAnalysisRequest) {
	// Add computed metrics to the analysis
	totalValue := 0.0
	for _, player := range request.Lineup {
		if player.Salary > 0 {
			totalValue += player.Projection / (player.Salary / 1000)
		}
	}

	avgValue := totalValue / float64(len(request.Lineup))
	
	// Adjust ratings based on computed metrics
	if avgValue > 3.5 {
		response.OverallRating += 1.0
		response.ExpectedROI += 0.05
	} else if avgValue < 2.5 {
		response.OverallRating -= 1.0
		response.ExpectedROI -= 0.05
	}

	// Clamp ratings
	response.OverallRating = max(0, min(10, response.OverallRating))
	response.ExpectedROI = max(-0.5, min(1.0, response.ExpectedROI))
}

// Utility methods

func (ae *AIEngine) generateRequestID(request *RecommendationRequest) string {
	data := fmt.Sprintf("%d-%d-%s-%d", request.ContestID, request.UserID, request.RequestType, time.Now().Unix())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)[:12]
}

func (ae *AIEngine) addToReasoningPath(response *RecommendationResponse, step string) {
	response.ReasoningPath = append(response.ReasoningPath, step)
}

func (ae *AIEngine) getCachedResponse(requestID string) (*RecommendationResponse, error) {
	// Implementation would check cache for existing response
	return nil, fmt.Errorf("cache miss")
}

func (ae *AIEngine) cacheResponse(requestID string, response *RecommendationResponse) {
	// Implementation would cache the response
	ae.logger.WithField("request_id", requestID).Debug("Cached recommendation response")
}

// Health check
func (ae *AIEngine) IsHealthy() bool {
	return ae.claudeClient.IsHealthy() && 
		   ae.realtimeAggregator.IsHealthy()
}

// Utility functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}