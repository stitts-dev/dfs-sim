package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
)

// PromptBuilder creates dynamic AI prompts based on context
type PromptBuilder struct {
	templates      map[string]*PromptTemplate
	sportModifiers map[string]SportModifier
	cache          *CacheService
	logger         *logrus.Logger
}

// PromptTemplate represents a template for AI prompts
type PromptTemplate struct {
	ID              string
	Name            string
	Sport           string
	ContestType     string
	BasePrompt      string
	SystemPrompt    string
	Variables       []TemplateVariable
	OptimalLength   int    // Optimal prompt length for best results
	ComplexityLevel string // "simple", "moderate", "complex"
}

// TemplateVariable represents a dynamic variable in the template
type TemplateVariable struct {
	Name        string
	Type        string // "string", "number", "array", "object"
	Required    bool
	Description string
	DefaultValue interface{}
}

// SportModifier adjusts prompts based on sport-specific characteristics
type SportModifier struct {
	Sport              string
	PositionContext    map[string]string
	StrategyWeighting  map[string]float64
	RiskFactors        []string
	CorrelationRules   []string
	UniqueConsiderations []string
}

// NewPromptBuilder creates a new prompt builder instance
func NewPromptBuilder(cache *CacheService, logger *logrus.Logger) *PromptBuilder {
	pb := &PromptBuilder{
		templates:      make(map[string]*PromptTemplate),
		sportModifiers: make(map[string]SportModifier),
		cache:          cache,
		logger:         logger,
	}

	// Initialize default templates and modifiers
	pb.initializeDefaultTemplates()
	pb.initializeSportModifiers()

	return pb
}

// BuildRecommendationPrompt creates a dynamic recommendation prompt
func (pb *PromptBuilder) BuildRecommendationPrompt(ctx models.PromptContext, players []models.PlayerRecommendation) (string, string, error) {
	// Select appropriate template
	template, err := pb.selectTemplate(ctx.Sport, ctx.ContestType, "player_recommendations")
	if err != nil {
		return "", "", fmt.Errorf("failed to select template: %w", err)
	}

	// Build context-specific prompt
	prompt, err := pb.buildPromptFromTemplate(template, ctx, players)
	if err != nil {
		return "", "", fmt.Errorf("failed to build prompt from template: %w", err)
	}

	// Get system prompt
	systemPrompt := pb.buildSystemPrompt(template, ctx)

	pb.logger.WithFields(logrus.Fields{
		"sport":        ctx.Sport,
		"contest_type": ctx.ContestType,
		"template":     template.Name,
		"prompt_length": len(prompt),
	}).Debug("Built recommendation prompt")

	return prompt, systemPrompt, nil
}

// selectTemplate chooses the best template for the context
func (pb *PromptBuilder) selectTemplate(sport, contestType, promptType string) (*PromptTemplate, error) {
	// Priority order: sport+contest+type -> sport+type -> sport -> default
	templateKeys := []string{
		fmt.Sprintf("%s_%s_%s", sport, contestType, promptType),
		fmt.Sprintf("%s_%s", sport, promptType),
		fmt.Sprintf("%s_default", sport),
		"default_recommendations",
	}

	for _, key := range templateKeys {
		if template, exists := pb.templates[key]; exists {
			return template, nil
		}
	}

	return nil, fmt.Errorf("no suitable template found for sport=%s, contest_type=%s, prompt_type=%s", sport, contestType, promptType)
}

// buildPromptFromTemplate constructs the final prompt from template and context
func (pb *PromptBuilder) buildPromptFromTemplate(template *PromptTemplate, ctx models.PromptContext, players []models.PlayerRecommendation) (string, error) {
	prompt := template.BasePrompt

	// Replace template variables
	replacements := pb.buildReplacements(ctx, players)
	
	for variable, value := range replacements {
		placeholder := fmt.Sprintf("{{%s}}", variable)
		prompt = strings.ReplaceAll(prompt, placeholder, fmt.Sprintf("%v", value))
	}

	// Add sport-specific modifiers
	if modifier, exists := pb.sportModifiers[ctx.Sport]; exists {
		prompt = pb.applySportModifier(prompt, modifier, ctx)
	}

	// Add real-time data integration
	if len(ctx.RealTimeData) > 0 {
		prompt = pb.addRealTimeContext(prompt, ctx.RealTimeData)
	}

	// Add user personalization
	if ctx.UserProfile != nil {
		prompt = pb.addUserPersonalization(prompt, ctx.UserProfile)
	}

	return prompt, nil
}

// buildReplacements creates variable replacements for the template
func (pb *PromptBuilder) buildReplacements(ctx models.PromptContext, players []models.PlayerRecommendation) map[string]interface{} {
	replacements := make(map[string]interface{})

	// Basic context
	replacements["sport"] = ctx.Sport
	replacements["contest_type"] = ctx.ContestType
	replacements["optimization_goal"] = ctx.OptimizationGoal
	replacements["ownership_strategy"] = ctx.OwnershipStrategy
	replacements["risk_tolerance"] = ctx.RiskTolerance
	replacements["time_to_lock"] = pb.formatTimeToLock(ctx.TimeToLock)

	// Contest metadata
	if ctx.ContestMeta != nil {
		replacements["contest_name"] = ctx.ContestMeta.ContestName
		replacements["entry_fee"] = ctx.ContestMeta.EntryFee
		replacements["total_prize"] = ctx.ContestMeta.TotalPrize
		replacements["entries"] = fmt.Sprintf("%d/%d", ctx.ContestMeta.CurrentEntries, ctx.ContestMeta.MaxEntries)
		replacements["salary_cap"] = ctx.ContestMeta.SalaryCap
	}

	// Player context
	if len(players) > 0 {
		replacements["player_count"] = len(players)
		replacements["top_players"] = pb.formatTopPlayers(players[:min(5, len(players))])
		replacements["salary_range"] = pb.formatSalaryRange(players)
		replacements["position_breakdown"] = pb.formatPositionBreakdown(players)
	}

	// Existing lineups context
	if len(ctx.ExistingLineups) > 0 {
		replacements["existing_lineup_count"] = len(ctx.ExistingLineups)
		replacements["lineup_diversity_needed"] = pb.calculateDiversityNeeded(ctx.ExistingLineups)
	}

	// User analytics
	if ctx.UserProfile != nil {
		replacements["user_roi"] = fmt.Sprintf("%.2f", ctx.UserProfile.HistoricalROI)
		replacements["user_risk_profile"] = ctx.UserProfile.RiskProfile
		replacements["user_expertise"] = pb.formatUserExpertise(ctx.UserProfile, ctx.Sport)
	}

	return replacements
}

// buildSystemPrompt creates the system prompt for Claude
func (pb *PromptBuilder) buildSystemPrompt(template *PromptTemplate, ctx models.PromptContext) string {
	systemPrompt := template.SystemPrompt

	// Add role definition
	role := "You are an expert Daily Fantasy Sports (DFS) analyst and strategist."
	
	// Add sport-specific expertise
	switch ctx.Sport {
	case "golf":
		role += " You specialize in golf tournament analysis, understanding course conditions, player form, weather impacts, and cut line dynamics."
	case "nfl":
		role += " You specialize in NFL player analysis, understanding game scripts, weather conditions, and positional correlations."
	case "nba":
		role += " You specialize in NBA player analysis, understanding pace, usage rates, rest situations, and slate dynamics."
	}

	// Add contest type specialization
	switch ctx.ContestType {
	case "gpp":
		role += " You focus on tournament strategy, seeking high-upside plays and unique lineup construction."
	case "cash":
		role += " You prioritize consistent, high-floor plays that provide steady returns."
	case "satellite":
		role += " You balance risk and reward to secure tournament entries efficiently."
	}

	// Combine role with template system prompt
	fullSystemPrompt := fmt.Sprintf("%s\n\n%s", role, systemPrompt)

	// Add behavioral guidelines
	guidelines := pb.buildBehavioralGuidelines(ctx)
	fullSystemPrompt = fmt.Sprintf("%s\n\n%s", fullSystemPrompt, guidelines)

	return fullSystemPrompt
}

// buildBehavioralGuidelines creates guidelines for AI behavior
func (pb *PromptBuilder) buildBehavioralGuidelines(ctx models.PromptContext) string {
	guidelines := []string{
		"CRITICAL GUIDELINES:",
		"1. Always provide specific, actionable recommendations",
		"2. Include confidence levels (0-100) for each recommendation",
		"3. Explain the reasoning behind each suggestion",
		"4. Consider real-time factors and late-breaking news",
		"5. Account for ownership projections and leverage opportunities",
		"6. Maintain consistency with user's risk tolerance and strategy",
		"7. Format responses in clear, digestible sections",
		"8. Never recommend players without proper justification",
	}

	// Add time-sensitive guidelines
	if ctx.TimeToLock < 2*time.Hour {
		guidelines = append(guidelines, "9. URGENT: Focus on late-swap opportunities and breaking news")
		guidelines = append(guidelines, "10. Prioritize immediately actionable insights")
	}

	// Add contest-specific guidelines
	switch ctx.ContestType {
	case "gpp":
		guidelines = append(guidelines, "11. Emphasize contrarian plays and correlation stacks")
		guidelines = append(guidelines, "12. Identify players with 'tournament-winning' upside")
	case "cash":
		guidelines = append(guidelines, "11. Focus on safety, consistency, and high floors")
		guidelines = append(guidelines, "12. Avoid highly volatile or risky plays")
	}

	return strings.Join(guidelines, "\n")
}

// Sport-specific modifier methods
func (pb *PromptBuilder) applySportModifier(prompt string, modifier SportModifier, ctx models.PromptContext) string {
	// Add sport-specific context
	if len(modifier.RiskFactors) > 0 {
		riskSection := fmt.Sprintf("\n\nSPORT-SPECIFIC RISK FACTORS:\n%s",
			strings.Join(modifier.RiskFactors, "\n"))
		prompt += riskSection
	}

	if len(modifier.UniqueConsiderations) > 0 {
		considerationsSection := fmt.Sprintf("\n\nUNIQUE CONSIDERATIONS FOR %s:\n%s",
			strings.ToUpper(ctx.Sport), strings.Join(modifier.UniqueConsiderations, "\n"))
		prompt += considerationsSection
	}

	return prompt
}

// Real-time data integration
func (pb *PromptBuilder) addRealTimeContext(prompt string, realTimeData []models.RealtimeDataPoint) string {
	if len(realTimeData) == 0 {
		return prompt
	}

	realTimeSection := "\n\nREAL-TIME UPDATES:\n"
	
	// Group data by type
	dataByType := make(map[string][]models.RealtimeDataPoint)
	for _, data := range realTimeData {
		dataByType[data.DataType] = append(dataByType[data.DataType], data)
	}

	// Format by importance
	priorityOrder := []string{"injury", "weather", "lineup", "ownership", "news"}
	
	for _, dataType := range priorityOrder {
		if data, exists := dataByType[dataType]; exists {
			realTimeSection += fmt.Sprintf("\n%s UPDATES:\n", strings.ToUpper(dataType))
			for _, item := range data {
				realTimeSection += fmt.Sprintf("- %s (Confidence: %.0f%%, Impact: %.1f)\n",
					pb.formatRealTimeData(item), item.Confidence*100, item.ImpactRating)
			}
		}
	}

	return prompt + realTimeSection
}

// User personalization
func (pb *PromptBuilder) addUserPersonalization(prompt string, userProfile *models.UserAnalytics) string {
	personalizationSection := fmt.Sprintf(`

USER PROFILE CONTEXT:
- Historical ROI: %.2f%%
- Risk Profile: %s
- Successful Patterns: %s
- Preferred Strategies: %s

PERSONALIZATION INSTRUCTIONS:
- Tailor recommendations to user's historical success patterns
- Match suggested risk level to user's risk profile
- Reference user's preferred strategies when applicable
- Consider user's sport expertise level in explanations`,
		userProfile.HistoricalROI*100,
		userProfile.RiskProfile,
		pb.formatSuccessfulPatterns(userProfile.SuccessfulPatterns),
		strings.Join(userProfile.PreferredStrategies, ", "))

	return prompt + personalizationSection
}

// Helper formatting methods
func (pb *PromptBuilder) formatTimeToLock(duration time.Duration) string {
	if duration < time.Hour {
		return fmt.Sprintf("%d minutes", int(duration.Minutes()))
	}
	return fmt.Sprintf("%.1f hours", duration.Hours())
}

func (pb *PromptBuilder) formatTopPlayers(players []models.PlayerRecommendation) string {
	var formatted []string
	for _, player := range players {
		formatted = append(formatted, fmt.Sprintf("%s (%s, $%g, %.1f proj)",
			player.PlayerName, player.Position, player.Salary, player.Projection))
	}
	return strings.Join(formatted, "\n")
}

func (pb *PromptBuilder) formatSalaryRange(players []models.PlayerRecommendation) string {
	if len(players) == 0 {
		return "No players available"
	}
	
	min, max := players[0].Salary, players[0].Salary
	for _, player := range players {
		if player.Salary < min {
			min = player.Salary
		}
		if player.Salary > max {
			max = player.Salary
		}
	}
	
	return fmt.Sprintf("$%g - $%g", min, max)
}

func (pb *PromptBuilder) formatPositionBreakdown(players []models.PlayerRecommendation) string {
	positions := make(map[string]int)
	for _, player := range players {
		positions[player.Position]++
	}
	
	var breakdown []string
	for pos, count := range positions {
		breakdown = append(breakdown, fmt.Sprintf("%s: %d", pos, count))
	}
	
	return strings.Join(breakdown, ", ")
}

func (pb *PromptBuilder) calculateDiversityNeeded(lineups []models.LineupReference) string {
	// Simple diversity calculation based on lineup count
	if len(lineups) < 3 {
		return "High - need significantly different player combinations"
	} else if len(lineups) < 10 {
		return "Medium - avoid heavy overlap with existing lineups"
	}
	return "Low - focus on optimal plays regardless of overlap"
}

func (pb *PromptBuilder) formatUserExpertise(profile *models.UserAnalytics, sport string) string {
	if expertise, exists := profile.SportExpertise[sport]; exists {
		if expertise >= 0.8 {
			return "Expert"
		} else if expertise >= 0.6 {
			return "Advanced"
		} else if expertise >= 0.4 {
			return "Intermediate"
		}
		return "Beginner"
	}
	return "Unknown"
}

func (pb *PromptBuilder) formatRealTimeData(data models.RealtimeDataPoint) string {
	// This would parse the JSON value and format it appropriately
	return fmt.Sprintf("Player %d: %s update from %s", data.PlayerID, data.DataType, data.Source)
}

func (pb *PromptBuilder) formatSuccessfulPatterns(patterns map[string]float64) string {
	var formatted []string
	for pattern, value := range patterns {
		formatted = append(formatted, fmt.Sprintf("%s (%.0f%% success)", pattern, value*100))
	}
	if len(formatted) == 0 {
		return "No historical patterns available"
	}
	return strings.Join(formatted, ", ")
}

// Utility function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Template and modifier initialization methods
func (pb *PromptBuilder) initializeDefaultTemplates() {
	// Golf recommendation template
	pb.templates["golf_gpp_player_recommendations"] = &PromptTemplate{
		ID:          "golf_gpp_001",
		Name:        "Golf GPP Player Recommendations",
		Sport:       "golf",
		ContestType: "gpp",
		BasePrompt: `Analyze the current golf tournament and provide DFS recommendations for a GPP contest.

CONTEST CONTEXT:
- Tournament: {{contest_name}}
- Sport: {{sport}}
- Contest Type: {{contest_type}}
- Entry Fee: ${{entry_fee}}
- Total Prize: ${{total_prize}}
- Salary Cap: ${{salary_cap}}
- Time to Lock: {{time_to_lock}}
- Optimization Goal: {{optimization_goal}}
- Ownership Strategy: {{ownership_strategy}}

AVAILABLE PLAYERS:
{{top_players}}

USER CONTEXT:
- Risk Tolerance: {{risk_tolerance}}
- Historical ROI: {{user_roi}}
- Expertise Level: {{user_expertise}}

ANALYSIS REQUIREMENTS:
1. Recommend 6-8 golfers for optimal lineup construction
2. Consider course fit, recent form, and weather conditions
3. Identify potential low-ownership leverage plays
4. Suggest correlation stacks (same country, equipment sponsor, etc.)
5. Account for cut line probability and weekend scoring potential
6. Provide specific reasoning for each recommendation
7. Include confidence levels (0-100) for each player

OUTPUT FORMAT:
For each recommended player, provide:
- Player Name and Salary
- Projected Ownership %
- Course Fit Rating (1-10)
- Recent Form Assessment
- Key Factors (weather, equipment, motivation)
- Confidence Level (0-100)
- Leverage Assessment (High/Medium/Low)

STRATEGIC CONSIDERATIONS:
- This is a GPP contest - prioritize upside over safety
- Look for players who could make a big move on the weekend
- Consider weather forecast impacts on scoring
- Factor in tee times and course conditions
- Identify potential contrarian plays with lower ownership`,

		SystemPrompt: `You are an expert golf analyst with deep knowledge of PGA Tour players, course characteristics, and tournament dynamics. Your recommendations should reflect advanced understanding of:

- Course fit and historical performance
- Weather impact on scoring and player performance  
- Equipment and sponsor correlations
- Cut line dynamics and weekend scoring patterns
- Ownership projections and leverage opportunities
- Recent form, injury status, and motivation factors

Provide detailed, actionable insights that give users a competitive edge in DFS golf contests.`,

		ComplexityLevel: "complex",
		OptimalLength:   3000,
	}

	// Add more templates for different sports and contest types
	pb.addDefaultTemplates()
}

func (pb *PromptBuilder) addDefaultTemplates() {
	// Default recommendation template
	pb.templates["default_recommendations"] = &PromptTemplate{
		ID:          "default_001",
		Name:        "Default Sport Recommendations",
		Sport:       "any",
		ContestType: "any",
		BasePrompt: `Provide DFS recommendations based on the available data.

CONTEXT:
- Sport: {{sport}}
- Contest Type: {{contest_type}}
- Time to Lock: {{time_to_lock}}
- Players Available: {{player_count}}

Please analyze the players and provide recommendations with reasoning.`,

		SystemPrompt: `You are a DFS analyst. Provide clear, actionable recommendations based on the available data.`,
		ComplexityLevel: "simple",
		OptimalLength:   1500,
	}
}

func (pb *PromptBuilder) initializeSportModifiers() {
	// Golf sport modifier
	pb.sportModifiers["golf"] = SportModifier{
		Sport: "golf",
		PositionContext: map[string]string{
			"G": "Golfer - all players compete for the same prize pool",
		},
		StrategyWeighting: map[string]float64{
			"course_fit":     0.25,
			"recent_form":    0.20,
			"weather_impact": 0.15,
			"ownership":      0.15,
			"value":          0.15,
			"upside":         0.10,
		},
		RiskFactors: []string{
			"Cut line risk - players must make the cut to score points",
			"Weather dependency - wind and rain significantly impact scoring",
			"Course fit - some players excel on specific course types",
			"Field strength - stronger fields make top finishes more difficult",
		},
		CorrelationRules: []string{
			"Country stacks - players from same country often perform similarly",
			"Equipment sponsors - players using same equipment may benefit from conditions",
			"Caddie connections - experienced caddies on specific courses",
		},
		UniqueConsiderations: []string{
			"Tee times affect scoring conditions",
			"Cut line typically around +4 to +6",
			"Weekend tee times favor players making the cut",
			"Weather forecasts can change rapidly",
			"Course history more important than overall form",
		},
	}

	// Add modifiers for other sports as needed
}

// GetTemplate returns a specific template by key
func (pb *PromptBuilder) GetTemplate(key string) (*PromptTemplate, bool) {
	template, exists := pb.templates[key]
	return template, exists
}

// ListTemplates returns all available templates
func (pb *PromptBuilder) ListTemplates() map[string]*PromptTemplate {
	return pb.templates
}

// AddCustomTemplate allows adding custom templates
func (pb *PromptBuilder) AddCustomTemplate(key string, template *PromptTemplate) {
	pb.templates[key] = template
	pb.logger.WithFields(logrus.Fields{
		"key":          key,
		"template_name": template.Name,
		"sport":        template.Sport,
	}).Info("Added custom prompt template")
}