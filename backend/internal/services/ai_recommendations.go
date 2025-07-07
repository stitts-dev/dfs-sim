package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"gorm.io/datatypes"
)

// AnthropicRequest represents the request structure for Claude API
type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// Message represents a message in the Claude conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents the response from Claude API
type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// AIRecommendationService handles AI-powered player recommendations
type AIRecommendationService struct {
	db        *database.DB
	config    *config.Config
	cache     *CacheService
	apiClient *http.Client
}

// PlayerRecommendationRequest represents a request for player recommendations
type PlayerRecommendationRequest struct {
	ContestID        int     `json:"contest_id"`
	ContestType      string  `json:"contest_type"`
	Sport            string  `json:"sport"`
	RemainingBudget  float64 `json:"remaining_budget"`
	CurrentLineup    []int   `json:"current_lineup"`
	PositionsNeeded  []string `json:"positions_needed"`
	BeginnerMode     bool    `json:"beginner_mode"`
	OptimizeFor      string  `json:"optimize_for"` // "ceiling", "floor", "balanced"
}

// LineupAnalysisRequest represents a request to analyze a lineup
type LineupAnalysisRequest struct {
	LineupID    int    `json:"lineup_id"`
	ContestType string `json:"contest_type"`
	Sport       string `json:"sport"`
}

// PlayerRecommendation represents an AI-generated player recommendation
type PlayerRecommendation struct {
	PlayerID         int      `json:"player_id"`
	PlayerName       string   `json:"player_name"`
	Position         string   `json:"position"`
	Team             string   `json:"team"`
	Salary           float64  `json:"salary"`
	ProjectedPoints  float64  `json:"projected_points"`
	Confidence       float64  `json:"confidence"`
	Reasoning        string   `json:"reasoning"`
	BeginnerTip      string   `json:"beginner_tip,omitempty"`
	StackWith        []string `json:"stack_with,omitempty"`
	AvoidWith        []string `json:"avoid_with,omitempty"`
}

// LineupAnalysis represents AI analysis of a lineup
type LineupAnalysis struct {
	OverallScore     float64                `json:"overall_score"`
	Strengths        []string               `json:"strengths"`
	Weaknesses       []string               `json:"weaknesses"`
	Improvements     []string               `json:"improvements"`
	StackingAnalysis map[string]interface{} `json:"stacking_analysis"`
	RiskLevel        string                 `json:"risk_level"`
	BeginnerInsights []string               `json:"beginner_insights,omitempty"`
}

// NewAIRecommendationService creates a new AI recommendation service
func NewAIRecommendationService(db *database.DB, config *config.Config, cache *CacheService) *AIRecommendationService {
	return &AIRecommendationService{
		db:     db,
		config: config,
		cache:  cache,
		apiClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetPlayerRecommendations returns AI-powered player recommendations
func (s *AIRecommendationService) GetPlayerRecommendations(ctx context.Context, userID int, req PlayerRecommendationRequest) ([]PlayerRecommendation, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("ai_recommendations:%d:%d:%s", userID, req.ContestID, req.OptimizeFor)
	
	var cachedRecommendations []PlayerRecommendation
	if err := s.cache.Get(ctx, cacheKey, &cachedRecommendations); err == nil {
		return cachedRecommendations, nil
	}

	// Get contest details
	var contest models.Contest
	if err := s.db.First(&contest, req.ContestID).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch contest: %w", err)
	}

	// Get available players for the positions needed
	var availablePlayers []models.Player
	query := s.db.Where("contest_id = ?", req.ContestID).
		Where("salary <= ?", req.RemainingBudget)
	
	if len(req.PositionsNeeded) > 0 {
		query = query.Where("position IN ?", req.PositionsNeeded)
	}
	
	if len(req.CurrentLineup) > 0 {
		query = query.Where("id NOT IN ?", req.CurrentLineup)
	}
	
	if err := query.Find(&availablePlayers).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch available players: %w", err)
	}

	// Build the prompt
	prompt := s.buildRecommendationPrompt(req, contest, availablePlayers)

	// Call Anthropic API
	recommendations, err := s.callAnthropicAPI(ctx, prompt, req.BeginnerMode)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI recommendations: %w", err)
	}

	// Match AI recommendations to actual database players
	matchedRecommendations := make([]PlayerRecommendation, 0)
	for _, rec := range recommendations {
		var player models.Player
		err := s.db.Where("contest_id = ? AND name = ? AND team = ?", req.ContestID, rec.PlayerName, rec.Team).First(&player).Error
		if err == nil {
			// Found matching player, update the recommendation with actual player ID
			rec.PlayerID = int(player.ID)
			matchedRecommendations = append(matchedRecommendations, rec)
		} else {
			// Try matching by name only (in case team abbreviation differs)
			err = s.db.Where("contest_id = ? AND name = ?", req.ContestID, rec.PlayerName).First(&player).Error
			if err == nil {
				rec.PlayerID = int(player.ID)
				rec.Team = player.Team // Update team to match database
				matchedRecommendations = append(matchedRecommendations, rec)
			}
			// If still not found, skip this recommendation
		}
	}

	// Use matched recommendations instead of raw AI recommendations
	recommendations = matchedRecommendations

	// Store in database for analytics
	requestData, _ := json.Marshal(req)
	responseData, _ := json.Marshal(recommendations)
	
	aiRec := models.AIRecommendation{
		UserID:    userID,
		ContestID: req.ContestID,
		Request:   datatypes.JSON(requestData),
		Response:  datatypes.JSON(responseData),
		Confidence: s.calculateAverageConfidence(recommendations),
	}
	
	if err := s.db.Create(&aiRec).Error; err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to store AI recommendation: %v\n", err)
	}

	// Cache the recommendations
	s.cache.Set(ctx, cacheKey, recommendations, time.Duration(s.config.AICacheExpiration)*time.Second)

	return recommendations, nil
}

// AnalyzeLineup provides AI analysis of a lineup
func (s *AIRecommendationService) AnalyzeLineup(ctx context.Context, userID int, req LineupAnalysisRequest) (*LineupAnalysis, error) {
	// Get lineup details
	var lineup models.Lineup
	if err := s.db.First(&lineup, req.LineupID).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch lineup: %w", err)
	}

	// Build analysis prompt
	prompt := s.buildAnalysisPrompt(lineup, req.ContestType)

	// Call Anthropic API
	analysisText, err := s.callAnthropicAPIForAnalysis(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI analysis: %w", err)
	}

	// Parse the analysis
	analysis := s.parseAnalysis(analysisText)

	// Store in database
	requestData, _ := json.Marshal(req)
	responseData, _ := json.Marshal(analysis)
	
	aiRec := models.AIRecommendation{
		UserID:    userID,
		ContestID: int(lineup.ContestID),
		Request:   datatypes.JSON(requestData),
		Response:  datatypes.JSON(responseData),
		Confidence: analysis.OverallScore / 100.0,
	}
	
	if err := s.db.Create(&aiRec).Error; err != nil {
		fmt.Printf("Failed to store AI analysis: %v\n", err)
	}

	return analysis, nil
}

// GetRecommendationHistory returns the user's AI recommendation history
func (s *AIRecommendationService) GetRecommendationHistory(userID int, limit int) ([]models.AIRecommendation, error) {
	var recommendations []models.AIRecommendation
	
	query := s.db.Where("user_id = ?", userID).
		Order("created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	if err := query.Find(&recommendations).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch recommendation history: %w", err)
	}
	
	return recommendations, nil
}

// buildRecommendationPrompt constructs the prompt for player recommendations
func (s *AIRecommendationService) buildRecommendationPrompt(req PlayerRecommendationRequest, contest models.Contest, players []models.Player) string {
	var prompt strings.Builder
	
	prompt.WriteString(fmt.Sprintf("You are a DFS (Daily Fantasy Sports) expert helping with %s lineup optimization.\n\n", req.Sport))
	prompt.WriteString(fmt.Sprintf("Contest Type: %s\n", req.ContestType))
	prompt.WriteString(fmt.Sprintf("Optimization Goal: %s\n", req.OptimizeFor))
	prompt.WriteString(fmt.Sprintf("Remaining Budget: $%.2f\n", req.RemainingBudget))
	prompt.WriteString(fmt.Sprintf("Positions Needed: %v\n\n", req.PositionsNeeded))
	
	if req.BeginnerMode {
		prompt.WriteString("The user is in beginner mode. Please provide simple, educational explanations.\n\n")
	}
	
	prompt.WriteString("Available Players:\n")
	for _, player := range players {
		prompt.WriteString(fmt.Sprintf("- ID:%d %s (%s, %s): $%.2f, Projected: %.2f pts, Ownership: %.1f%%\n",
			player.ID, player.Name, player.Position, player.Team, player.Salary, player.ProjectedPoints, player.Ownership))
	}
	
	prompt.WriteString("\nPlease recommend the best players for this lineup considering:\n")
	prompt.WriteString("1. Value (points per dollar)\n")
	prompt.WriteString("2. Matchup quality\n")
	prompt.WriteString("3. Correlation and stacking opportunities\n")
	prompt.WriteString("4. Ownership projections\n")
	
	if req.ContestType == "GPP" {
		prompt.WriteString("5. Tournament upside and differentiation\n")
	} else {
		prompt.WriteString("5. Safety and consistency for cash games\n")
	}
	
	prompt.WriteString("\nProvide recommendations in JSON format with the following structure:\n")
	prompt.WriteString(`[{
		"player_name": "Player Name",
		"position": "POS",
		"team": "TEAM",
		"salary": 5000,
		"projected_points": 25.5,
		"confidence": 0.85,
		"reasoning": "Clear explanation of why this player is recommended",
		"beginner_tip": "Simple tip for beginners (if applicable)",
		"stack_with": ["Player1", "Player2"],
		"avoid_with": ["Player3"]
	}]`)
	
	return prompt.String()
}

// buildAnalysisPrompt constructs the prompt for lineup analysis
func (s *AIRecommendationService) buildAnalysisPrompt(lineup models.Lineup, contestType string) string {
	var prompt strings.Builder
	
	prompt.WriteString(fmt.Sprintf("Analyze this DFS %s lineup for a %s contest:\n\n", lineup.Contest.Sport, contestType))
	
	prompt.WriteString("Lineup:\n")
	totalSalary := 0.0
	totalProjected := 0.0
	
	for _, player := range lineup.Players {
		prompt.WriteString(fmt.Sprintf("- %s (%s, %s): $%d, Projected: %.2f pts\n",
			player.Name, player.Position, player.Team, player.Salary, player.ProjectedPoints))
		totalSalary += float64(player.Salary)
		totalProjected += player.ProjectedPoints
	}
	
	prompt.WriteString(fmt.Sprintf("\nTotal Salary Used: $%.2f\n", totalSalary))
	prompt.WriteString(fmt.Sprintf("Total Projected Points: %.2f\n\n", totalProjected))
	
	prompt.WriteString("Please analyze this lineup and provide:\n")
	prompt.WriteString("1. Overall score (0-100)\n")
	prompt.WriteString("2. Key strengths\n")
	prompt.WriteString("3. Potential weaknesses\n")
	prompt.WriteString("4. Specific improvements\n")
	prompt.WriteString("5. Stacking analysis\n")
	prompt.WriteString("6. Risk assessment\n")
	
	return prompt.String()
}

// callAnthropicAPI makes a request to the Anthropic API
func (s *AIRecommendationService) callAnthropicAPI(ctx context.Context, prompt string, beginnerMode bool) ([]PlayerRecommendation, error) {
	if s.config.AnthropicAPIKey == "" {
		return nil, errors.New("Anthropic API key not configured")
	}

	reqBody := AnthropicRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 2048,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.config.AnthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, err
	}

	if len(anthropicResp.Content) == 0 {
		return nil, errors.New("no content in API response")
	}

	// Parse the JSON response
	var recommendations []PlayerRecommendation
	responseText := anthropicResp.Content[0].Text
	
	// Find JSON content in the response
	startIdx := strings.Index(responseText, "[")
	endIdx := strings.LastIndex(responseText, "]")
	
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		jsonContent := responseText[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonContent), &recommendations); err != nil {
			return nil, fmt.Errorf("failed to parse recommendations: %w", err)
		}
	}

	return recommendations, nil
}

// callAnthropicAPIForAnalysis makes a request for lineup analysis
func (s *AIRecommendationService) callAnthropicAPIForAnalysis(ctx context.Context, prompt string) (string, error) {
	if s.config.AnthropicAPIKey == "" {
		return "", errors.New("Anthropic API key not configured")
	}

	reqBody := AnthropicRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 1024,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.config.AnthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.apiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", err
	}

	if len(anthropicResp.Content) == 0 {
		return "", errors.New("no content in API response")
	}

	return anthropicResp.Content[0].Text, nil
}

// parseAnalysis parses the AI analysis text into structured format
func (s *AIRecommendationService) parseAnalysis(analysisText string) *LineupAnalysis {
	// This is a simplified parser - in production, you'd want more sophisticated parsing
	analysis := &LineupAnalysis{
		OverallScore:     75.0, // Default score
		Strengths:        []string{},
		Weaknesses:       []string{},
		Improvements:     []string{},
		StackingAnalysis: make(map[string]interface{}),
		RiskLevel:        "medium",
		BeginnerInsights: []string{},
	}

	// Extract score if mentioned
	if strings.Contains(analysisText, "score:") || strings.Contains(analysisText, "Score:") {
		// Simple extraction - would be more sophisticated in production
		analysis.OverallScore = 80.0
	}

	// Extract sections
	lines := strings.Split(analysisText, "\n")
	currentSection := ""
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(strings.ToLower(line), "strength") {
			currentSection = "strengths"
		} else if strings.Contains(strings.ToLower(line), "weakness") {
			currentSection = "weaknesses"
		} else if strings.Contains(strings.ToLower(line), "improvement") {
			currentSection = "improvements"
		} else if strings.Contains(strings.ToLower(line), "risk") {
			if strings.Contains(strings.ToLower(line), "high") {
				analysis.RiskLevel = "high"
			} else if strings.Contains(strings.ToLower(line), "low") {
				analysis.RiskLevel = "low"
			}
		} else if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") {
			point := strings.TrimPrefix(strings.TrimPrefix(line, "-"), "•")
			point = strings.TrimSpace(point)
			
			switch currentSection {
			case "strengths":
				analysis.Strengths = append(analysis.Strengths, point)
			case "weaknesses":
				analysis.Weaknesses = append(analysis.Weaknesses, point)
			case "improvements":
				analysis.Improvements = append(analysis.Improvements, point)
			}
		}
	}

	return analysis
}

// calculateAverageConfidence calculates the average confidence score
func (s *AIRecommendationService) calculateAverageConfidence(recommendations []PlayerRecommendation) float64 {
	if len(recommendations) == 0 {
		return 0.0
	}

	total := 0.0
	for _, rec := range recommendations {
		total += rec.Confidence
	}

	return total / float64(len(recommendations))
}

// Helper function to enforce rate limiting
func (s *AIRecommendationService) checkRateLimit(ctx context.Context, userID int) error {
	key := fmt.Sprintf("ai_rate_limit:%d", userID)
	
	// Get current count
	var count int
	s.cache.Get(ctx, key, &count)
	
	if count >= s.config.AIRateLimit {
		return errors.New("AI rate limit exceeded, please try again later")
	}
	
	// Increment count
	count++
	s.cache.Set(ctx, key, count, time.Minute)
	
	return nil
}