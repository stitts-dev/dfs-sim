## FEATURE:

Fix and enhance AI recommendations service for golf-specific insights and player analysis

## EXAMPLES:

Current issues in `backend/internal/services/ai_recommendations.go`:
```go
// Line 35: Outdated model
Model: "claude-3-sonnet-20240229"

// Line 68: Generic prompt without golf context
content := fmt.Sprintf(`Analyze this DFS lineup and provide 3-5 specific recommendations...`)

// Line 111-113: Poor player matching logic
name1 := strings.ToLower(strings.TrimSpace(player.Name))
name2 := strings.ToLower(strings.TrimSpace(p.Name))
if strings.Contains(name1, name2) || strings.Contains(name2, name1)
```

Improved implementation should include:
```go
// Updated model
Model: "claude-3-opus-20240229" // or latest available

// Golf-specific prompt
content := fmt.Sprintf(`As a golf DFS expert, analyze this PGA Tour lineup for the %s tournament.
Consider:
- Course fit and history
- Recent form (last 5 tournaments)
- Strokes gained statistics
- Weather conditions
- Cut probability
- Ownership projections
...`)

// Better player matching with fuzzy logic
func matchPlayer(playerName string, roster []Player) *Player {
    // Implement Levenshtein distance or similar
    // Handle "Scottie Scheffler" vs "S. Scheffler" cases
}
```

## DOCUMENTATION:

- Claude API latest models: https://docs.anthropic.com/claude/docs/models-overview
- Golf statistics explained: https://www.pgatour.com/stats
- DFS golf strategy resources
- Fuzzy string matching algorithms

## OTHER CONSIDERATIONS:

- Current AI model is outdated (using February 2024 version)
- No retry logic or proper error handling
- Golf-specific terminology missing (strokes gained, cut line, etc.)
- Player name matching fails for common variations
- No caching of AI responses leading to unnecessary API calls
- Missing rate limiting protection
- Should add tournament context (course type, weather, field strength)
- Need to handle API timeouts gracefully
- Consider implementing streaming responses for better UX
- Add cost tracking for AI API usage