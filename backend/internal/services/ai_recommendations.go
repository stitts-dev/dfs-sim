package services

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

// CircuitBreaker represents a circuit breaker for API calls
type CircuitBreaker struct {
	failureCount    int
	lastFailureTime time.Time
	state          string // "closed", "open", "half-open"
	mutex          sync.RWMutex
	threshold      int
	timeout        time.Duration
}

// RecommendationCacheKey represents the cache key structure
type RecommendationCacheKey struct {
	ContestID       int     `json:"contest_id"`
	RemainingBudget float64 `json:"remaining_budget"`
	Positions       string  `json:"positions_hash"`
	OptimizeFor     string  `json:"optimize_for"`
	Sport           string  `json:"sport"`
}

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
	db            *database.DB
	config        *config.Config
	cache         *CacheService
	apiClient     *http.Client
	logger        *logrus.Logger
	circuitBreaker *CircuitBreaker
	requestCounts  map[int]int // userID -> request count for rate limiting
	mutex         sync.RWMutex
}

// PlayerRecommendationRequest represents a request for player recommendations
type PlayerRecommendationRequest struct {
	ContestID       int      `json:"contest_id"`
	ContestType     string   `json:"contest_type"`
	Sport           string   `json:"sport"`
	RemainingBudget float64  `json:"remaining_budget"`
	CurrentLineup   []int    `json:"current_lineup"`
	PositionsNeeded []string `json:"positions_needed"`
	BeginnerMode    bool     `json:"beginner_mode"`
	OptimizeFor     string   `json:"optimize_for"` // "ceiling", "floor", "balanced"
}

// LineupAnalysisRequest represents a request to analyze a lineup
type LineupAnalysisRequest struct {
	LineupID    int    `json:"lineup_id"`
	ContestType string `json:"contest_type"`
	Sport       string `json:"sport"`
}

// PlayerRecommendation represents an AI-generated player recommendation
type PlayerRecommendation struct {
	PlayerID        int      `json:"player_id"`
	PlayerName      string   `json:"player_name"`
	Position        string   `json:"position"`
	Team            string   `json:"team"`
	Salary          float64  `json:"salary"`
	ProjectedPoints float64  `json:"projected_points"`
	Confidence      float64  `json:"confidence"`
	Reasoning       string   `json:"reasoning"`
	BeginnerTip     string   `json:"beginner_tip,omitempty"`
	StackWith       []string `json:"stack_with,omitempty"`
	AvoidWith       []string `json:"avoid_with,omitempty"`
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
		UserID:     userID,
		ContestID:  req.ContestID,
		Request:    datatypes.JSON(requestData),
		Response:   datatypes.JSON(responseData),
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
		UserID:     userID,
		ContestID:  int(lineup.ContestID),
		Request:    datatypes.JSON(requestData),
		Response:   datatypes.JSON(responseData),
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
	if strings.ToLower(req.Sport) == "golf" {
		return s.buildGolfRecommendationPrompt(req, contest, players)
	}
	
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
		prompt.WriteString(fmt.Sprintf("- ID:%d %s (%s, %s): $%d, Projected: %.2f pts, Ownership: %.1f%%\n",
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
		Model:     "claude-sonnet-4-20250514",
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
		Model:     "claude-sonnet-4-20250514",
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Reset counts every hour
	key := fmt.Sprintf("ai_rate_limit:%d", userID)
	var count int
	s.cache.Get(ctx, key, &count)
	
	rateLimit := 10 // Default rate limit
	if s.config.AIRateLimit > 0 {
		rateLimit = s.config.AIRateLimit
	}
	
	if count >= rateLimit {
		s.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"count":   count,
			"limit":   rateLimit,
		}).Warn("AI rate limit exceeded")
		return errors.New("AI rate limit exceeded, please try again later")
	}
	
	// Increment count
	count++
	s.cache.Set(ctx, key, count, time.Hour)
	
	return nil
}

// generateCacheKey creates an intelligent cache key
func (s *AIRecommendationService) generateCacheKey(req PlayerRecommendationRequest, userID int) string {
	key := RecommendationCacheKey{
		ContestID:       req.ContestID,
		RemainingBudget: math.Floor(req.RemainingBudget/1000) * 1000, // Round to nearest $1k
		Positions:       s.hashPositions(req.PositionsNeeded),
		OptimizeFor:     req.OptimizeFor,
		Sport:           req.Sport,
	}
	
	keyBytes, _ := json.Marshal(key)
	hash := md5.Sum(keyBytes)
	return fmt.Sprintf("ai_rec_v2:%d:%x", userID, hash)
}

// hashPositions creates a consistent hash for position arrays
func (s *AIRecommendationService) hashPositions(positions []string) string {
	if len(positions) == 0 {
		return "all"
	}
	
	// Sort positions for consistent hashing
	sorted := make([]string, len(positions))
	copy(sorted, positions)
	sort.Strings(sorted)
	
	return strings.Join(sorted, ",")
}

// fuzzyMatchPlayer finds the best matching player using Levenshtein distance
func (s *AIRecommendationService) fuzzyMatchPlayer(aiName, aiTeam string, players []models.Player) (*models.Player, float64) {
	bestMatch := (*models.Player)(nil)
	bestScore := 0.0
	
	// Normalize inputs
	aiNameNorm := strings.ToLower(strings.TrimSpace(aiName))
	aiTeamNorm := s.normalizeTeam(strings.ToLower(strings.TrimSpace(aiTeam)))
	
	for i := range players {
		player := &players[i]
		playerNameNorm := strings.ToLower(strings.TrimSpace(player.Name))
		playerTeamNorm := s.normalizeTeam(strings.ToLower(strings.TrimSpace(player.Team)))
		
		// Calculate name similarity
		nameScore := s.calculateSimilarity(aiNameNorm, playerNameNorm)
		
		// Handle common name variations
		nameScore = s.adjustForNameVariations(aiNameNorm, playerNameNorm, nameScore)
		
		// Calculate team similarity
		teamScore := s.calculateSimilarity(aiTeamNorm, playerTeamNorm)
		
		// Combined score (name weighted more heavily)
		combinedScore := (nameScore * 0.7) + (teamScore * 0.3)
		
		if combinedScore > bestScore && combinedScore > 0.6 {
			bestScore = combinedScore
			bestMatch = player
		}
	}
	
	return bestMatch, bestScore
}

// calculateSimilarity calculates string similarity using Levenshtein distance
func (s *AIRecommendationService) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	
	distance := s.levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}
	
	if maxLen == 0 {
		return 1.0
	}
	
	return 1.0 - (float64(distance) / float64(maxLen))
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (s *AIRecommendationService) levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	rows := len(r1) + 1
	cols := len(r2) + 1
	
	d := make([][]int, rows)
	for i := range d {
		d[i] = make([]int, cols)
		d[i][0] = i
	}
	
	for j := range d[0] {
		d[0][j] = j
	}
	
	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			
			d[i][j] = min(
				d[i-1][j]+1,    // deletion
				d[i][j-1]+1,    // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}
	
	return d[rows-1][cols-1]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// adjustForNameVariations improves matching for common name patterns
func (s *AIRecommendationService) adjustForNameVariations(aiName, playerName string, baseScore float64) float64 {
	// Handle initials vs full names
	if s.isInitialVariation(aiName, playerName) {
		return math.Max(baseScore, 0.9)
	}
	
	// Handle Jr., Sr., III variations
	if s.isSuffixVariation(aiName, playerName) {
		return math.Max(baseScore, 0.95)
	}
	
	return baseScore
}

// isInitialVariation checks if names are initial variations (J. Smith vs John Smith)
func (s *AIRecommendationService) isInitialVariation(name1, name2 string) bool {
	parts1 := strings.Fields(name1)
	parts2 := strings.Fields(name2)
	
	if len(parts1) != len(parts2) {
		return false
	}
	
	for i := range parts1 {
		p1, p2 := parts1[i], parts2[i]
		
		// Check if one is an initial of the other
		if len(p1) == 2 && strings.HasSuffix(p1, ".") && len(p2) > 2 {
			if strings.ToLower(p1[:1]) != strings.ToLower(p2[:1]) {
				return false
			}
		} else if len(p2) == 2 && strings.HasSuffix(p2, ".") && len(p1) > 2 {
			if strings.ToLower(p2[:1]) != strings.ToLower(p1[:1]) {
				return false
			}
		} else if strings.ToLower(p1) != strings.ToLower(p2) {
			return false
		}
	}
	
	return true
}

// isSuffixVariation checks for Jr., Sr., III variations
func (s *AIRecommendationService) isSuffixVariation(name1, name2 string) bool {
	suffixes := []string{"jr.", "sr.", "jr", "sr", "ii", "iii", "iv"}
	
	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToLower(name1), suffix) && !strings.HasSuffix(strings.ToLower(name2), suffix) {
			base1 := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(name1), suffix))
			if base1 == strings.ToLower(name2) {
				return true
			}
		}
		if strings.HasSuffix(strings.ToLower(name2), suffix) && !strings.HasSuffix(strings.ToLower(name1), suffix) {
			base2 := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(name2), suffix))
			if base2 == strings.ToLower(name1) {
				return true
			}
		}
	}
	
	return false
}

// normalizeTeam normalizes team names for better matching
func (s *AIRecommendationService) normalizeTeam(team string) string {
	// Common team name normalizations
	teamMappings := map[string]string{
		"usa":           "united states",
		"united states": "usa",
		"liv":           "liv golf",
		"liv golf":      "liv",
	}
	
	if normalized, exists := teamMappings[team]; exists {
		return normalized
	}
	
	return team
}

// callAnthropicAPIWithCircuitBreaker wraps API calls with circuit breaker
func (s *AIRecommendationService) callAnthropicAPIWithCircuitBreaker(ctx context.Context, prompt string, beginnerMode bool) ([]PlayerRecommendation, error) {
	// Check circuit breaker state
	if !s.canMakeAPICall() {
		return nil, errors.New("circuit breaker is open - API temporarily unavailable")
	}
	
	recommendations, err := s.callAnthropicAPI(ctx, prompt, beginnerMode)
	
	if err != nil {
		s.recordAPIFailure()
		return nil, err
	}
	
	s.recordAPISuccess()
	return recommendations, nil
}

// canMakeAPICall checks if API calls are allowed based on circuit breaker state
func (s *AIRecommendationService) canMakeAPICall() bool {
	s.circuitBreaker.mutex.RLock()
	defer s.circuitBreaker.mutex.RUnlock()
	
	switch s.circuitBreaker.state {
	case "closed":
		return true
	case "open":
		if time.Since(s.circuitBreaker.lastFailureTime) > s.circuitBreaker.timeout {
			s.circuitBreaker.mutex.RUnlock()
			s.circuitBreaker.mutex.Lock()
			s.circuitBreaker.state = "half-open"
			s.circuitBreaker.mutex.Unlock()
			s.circuitBreaker.mutex.RLock()
			return true
		}
		return false
	case "half-open":
		return true
	default:
		return true
	}
}

// recordAPIFailure records an API failure for circuit breaker
func (s *AIRecommendationService) recordAPIFailure() {
	s.circuitBreaker.mutex.Lock()
	defer s.circuitBreaker.mutex.Unlock()
	
	s.circuitBreaker.failureCount++
	s.circuitBreaker.lastFailureTime = time.Now()
	
	if s.circuitBreaker.failureCount >= s.circuitBreaker.threshold {
		s.circuitBreaker.state = "open"
		s.logger.Warn("Circuit breaker opened due to API failures")
	}
}

// recordAPISuccess records an API success for circuit breaker
func (s *AIRecommendationService) recordAPISuccess() {
	s.circuitBreaker.mutex.Lock()
	defer s.circuitBreaker.mutex.Unlock()
	
	s.circuitBreaker.failureCount = 0
	if s.circuitBreaker.state == "half-open" {
		s.circuitBreaker.state = "closed"
		s.logger.Info("Circuit breaker closed after successful API call")
	}
}

// buildGolfRecommendationPrompt constructs golf-specific prompts with DFS strategies
func (s *AIRecommendationService) buildGolfRecommendationPrompt(req PlayerRecommendationRequest, contest models.Contest, players []models.Player) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are a Golf DFS expert analyzing players using advanced golf-specific strategies.\n\n")
	
	prompt.WriteString("GOLF DFS STRATEGY PRIORITIES:\n")
	prompt.WriteString("1. Strokes Gained Analysis:\n")
	prompt.WriteString("   - Off the Tee: Driving distance and accuracy\n")
	prompt.WriteString("   - Approach: GIR percentage and proximity to pin\n")
	prompt.WriteString("   - Around Green: Scrambling and up/down percentage\n")
	prompt.WriteString("   - Putting: Putts per GIR and overall putting average\n\n")
	
	prompt.WriteString("2. Course Fit Assessment:\n")
	prompt.WriteString("   - Course length vs player driving distance\n")
	prompt.WriteString("   - Course difficulty vs player's scrambling ability\n")
	prompt.WriteString("   - Green speed vs putting statistics\n")
	prompt.WriteString("   - Weather conditions impact\n\n")
	
	prompt.WriteString("3. Recent Form & Cut Probability:\n")
	prompt.WriteString("   - Last 5 tournament finishes\n")
	prompt.WriteString("   - Missed cuts in similar course conditions\n")
	prompt.WriteString("   - Current world ranking trends\n\n")
	
	prompt.WriteString("4. Tournament Strategy:\n")
	if req.ContestType == "GPP" {
		prompt.WriteString("   - GPP: High ceiling players with low ownership\n")
		prompt.WriteString("   - Target players with major championship upside\n")
		prompt.WriteString("   - Focus on volatile players who can separate from field\n")
	} else {
		prompt.WriteString("   - Cash: Consistent players with high cut probability\n")
		prompt.WriteString("   - Target safe players with steady performance\n")
		prompt.WriteString("   - Prioritize players who rarely miss cuts\n")
	}
	prompt.WriteString("   - Country/sponsor correlations for stacking\n\n")
	
	prompt.WriteString(fmt.Sprintf("Contest: %s\n", contest.Name))
	prompt.WriteString(fmt.Sprintf("Optimization Goal: %s\n", req.OptimizeFor))
	prompt.WriteString(fmt.Sprintf("Remaining Budget: $%.2f\n", req.RemainingBudget))
	prompt.WriteString(fmt.Sprintf("Positions Needed: %v\n\n", req.PositionsNeeded))
	
	if req.BeginnerMode {
		prompt.WriteString("BEGINNER MODE: Provide educational explanations about golf DFS strategy.\n\n")
	}
	
	prompt.WriteString("Available Players:\n")
	for _, player := range players {
		prompt.WriteString(fmt.Sprintf("- %s (%s): $%d, Proj: %.1f pts, Own: %.1f%%, Form: %s\n",
			player.Name, player.Team, player.Salary, player.ProjectedPoints, player.Ownership, s.getPlayerForm(player)))
	}
	
	prompt.WriteString("\nAnalyze each player considering:\n")
	prompt.WriteString("- Current form and recent performance trends\n")
	prompt.WriteString("- Historical performance at similar course types\n")
	prompt.WriteString("- Strokes gained data in key categories\n")
	prompt.WriteString("- Cut probability and floor/ceiling projections\n")
	prompt.WriteString("- Ownership projections for tournament differentiation\n")
	prompt.WriteString("- Weather and course condition adjustments\n\n")
	
	prompt.WriteString("CRITICAL: Return ONLY valid JSON array. Use exact player names as shown above.\n")
	prompt.WriteString("Required JSON format:\n")
	prompt.WriteString(`[{
		"player_name": "Exact Player Name",
		"position": "G",
		"team": "Country/Team",
		"salary": 9500,
		"projected_points": 45.2,
		"confidence": 0.82,
		"reasoning": "Detailed golf-specific analysis including course fit, form, and strategy",
		"beginner_tip": "Golf DFS tip for beginners (if applicable)",
		"stack_with": ["Country teammate", "Sponsor teammate"],
		"avoid_with": ["Contrarian play"]
	}]`)
	
	return prompt.String()
}

// getPlayerForm returns a simple form indicator for golf players
func (s *AIRecommendationService) getPlayerForm(player models.Player) string {
	// Simplified form calculation based on projected points vs salary
	valueRatio := player.ProjectedPoints / float64(player.Salary) * 1000
	
	if valueRatio > 5.0 {
		return "Hot"
	} else if valueRatio > 4.0 {
		return "Good"
	} else if valueRatio > 3.0 {
		return "Average"
	}
	return "Cold"
}
