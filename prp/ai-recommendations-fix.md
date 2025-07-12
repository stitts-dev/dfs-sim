# PRP: AI Recommendations Service Fix

## FEATURE:

Fix and enhance the AI recommendations service to use latest Claude models, improve player name matching, add golf-specific DFS strategies, and implement robust error handling with caching optimizations.

## CONTEXT & MOTIVATION:

The current AI recommendations service in `backend/internal/services/ai_recommendations.go` has several critical issues that prevent reliable operation:

**Critical Issues:**
1. **Outdated Claude Model**: Using deprecated `claude-3-haiku-20240307` instead of latest `claude-sonnet-4-20250514`
2. **Poor Player Matching**: Brittle exact string matching fails when AI returns slightly different player names or team abbreviations
3. **Missing Golf Context**: Generic prompts lack golf-specific DFS strategies (strokes gained, course fit, cut probability)
4. **Inadequate Error Handling**: No fallback mechanisms when API calls fail or player matching fails
5. **Performance Issues**: No request batching, inefficient caching, missing rate limiting safeguards

**Business Impact:**
- Users receive empty or inaccurate recommendations due to player matching failures
- High API costs from using expensive models for simple tasks
- Poor user experience in golf DFS due to lack of sport-specific insights
- Service reliability issues during high-traffic periods

## EXAMPLES:

**Current Problematic Code:**

```go
// Line 343: Outdated model
Model: "claude-3-haiku-20240307",

// Lines 156-170: Brittle player matching
err := s.db.Where("contest_id = ? AND name = ? AND team = ?", req.ContestID, rec.PlayerName, rec.Team).First(&player).Error
if err == nil {
    // Found matching player
} else {
    // Try matching by name only - still fails for "J. Smith" vs "John Smith"
    err = s.db.Where("contest_id = ? AND name = ?", req.ContestID, rec.PlayerName).First(&player).Error
}
```

**Should be:**

```go
// Use latest model with better reasoning
Model: "claude-sonnet-4-20250514",

// Fuzzy matching with confidence scoring
player, confidence := s.fuzzyMatchPlayer(rec.PlayerName, rec.Team, availablePlayers)
if confidence > 0.8 {
    rec.PlayerID = int(player.ID)
    matchedRecommendations = append(matchedRecommendations, rec)
}
```

## CURRENT STATE ANALYSIS:

**Existing Infrastructure:**
- Anthropic API integration in place with proper headers/auth (`lines 358-366`)
- Redis caching via CacheService (`backend/internal/services/cache.go`)
- Structured logging with logrus available (`github.com/sirupsen/logrus`)
- Database models for AI recommendations storage (`models.AIRecommendation`)
- WebSocket notification framework ready (`data_fetcher.go:196-199`)

**Golf Data Context Available:**
- RapidAPI Golf provider with tournament/leaderboard data (`rapidapi_golf.go`)
- ESPN Golf fallback with player statistics (`espn_golf.go`)
- Contest discovery for golf tournaments (`contest_discovery.go`)
- Strokes gained and performance metrics in player models

**Error Patterns Observed:**
- Player name mismatches: "Jon Rahm" vs "J. Rahm", "Scottie Scheffler" vs "S. Scheffler"
- Team abbreviation differences: "USA" vs "United States"
- Missing players when AI suggests names not in current contest
- JSON parsing failures when Claude response format varies

## IMPLEMENTATION BLUEPRINT:

### Phase 1: Core Infrastructure Improvements

1. **Update Claude API Integration**
   - Upgrade to `claude-sonnet-4-20250514` for better reasoning and JSON consistency
   - Implement structured prompt engineering with clear JSON schema requirements
   - Add response validation and retry logic for malformed JSON

2. **Implement Fuzzy Player Matching**
   ```go
   func (s *AIRecommendationService) fuzzyMatchPlayer(aiName, aiTeam string, players []models.Player) (*models.Player, float64) {
       bestMatch := (*models.Player)(nil)
       bestScore := 0.0
       
       for _, player := range players {
           nameScore := s.calculateSimilarity(aiName, player.Name)
           teamScore := s.calculateSimilarity(aiTeam, player.Team)
           combinedScore := (nameScore * 0.7) + (teamScore * 0.3)
           
           if combinedScore > bestScore && combinedScore > 0.6 {
               bestScore = combinedScore
               bestMatch = &player
           }
       }
       return bestMatch, bestScore
   }
   
   func (s *AIRecommendationService) calculateSimilarity(s1, s2 string) float64 {
       // Levenshtein distance implementation
       distance := s.levenshteinDistance(strings.ToLower(s1), strings.ToLower(s2))
       maxLen := len(s1)
       if len(s2) > maxLen {
           maxLen = len(s2)
       }
       return 1.0 - (float64(distance) / float64(maxLen))
   }
   ```

3. **Enhanced Error Handling and Resilience**
   - Circuit breaker pattern for Anthropic API failures
   - Graceful degradation to cached recommendations
   - Comprehensive logging with request tracking
   - Rate limiting with user-specific quotas

### Phase 2: Golf-Specific DFS Strategy Integration

1. **Golf-Aware Prompt Engineering**
   ```go
   func (s *AIRecommendationService) buildGolfRecommendationPrompt(req PlayerRecommendationRequest, contest models.Contest, players []models.Player) string {
       prompt := `You are a golf DFS expert. Analyze players using these key factors:
       
       GOLF DFS STRATEGY PRIORITIES:
       1. Strokes Gained Analysis:
          - Off the Tee: Driving distance and accuracy
          - Approach: GIR percentage and proximity
          - Around Green: Scrambling and up/down percentage  
          - Putting: Putts per GIR and overall putting average
       
       2. Course Fit Assessment:
          - Course length vs player driving distance
          - Course difficulty vs player's scrambling ability
          - Green speed vs putting statistics
          - Weather conditions impact
       
       3. Recent Form & Cut Probability:
          - Last 5 tournament finishes
          - Missed cuts in similar course conditions
          - Current world ranking trends
       
       4. Tournament Strategy:
          - GPP: High ceiling players with low ownership
          - Cash: Consistent players with high cut probability
          - Stacking: Same country/sponsor correlations`
   }
   ```

2. **Dynamic Context Loading**
   - Course-specific statistics from golf providers
   - Weather condition integration for outdoor tournaments
   - Historical performance data for venue analysis
   - Cut line probability calculations

### Phase 3: Performance and Caching Optimizations

1. **Intelligent Caching Strategy**
   ```go
   type RecommendationCacheKey struct {
       ContestID       int     `json:"contest_id"`
       RemainingBudget float64 `json:"remaining_budget"`
       Positions       string  `json:"positions_hash"`
       OptimizeFor     string  `json:"optimize_for"`
       Sport           string  `json:"sport"`
   }
   
   func (s *AIRecommendationService) getCacheKey(req PlayerRecommendationRequest) string {
       key := RecommendationCacheKey{
           ContestID:       req.ContestID,
           RemainingBudget: math.Floor(req.RemainingBudget/1000) * 1000, // Round to nearest $1k
           Positions:       s.hashPositions(req.PositionsNeeded),
           OptimizeFor:     req.OptimizeFor,
           Sport:           req.Sport,
       }
       return fmt.Sprintf("ai_rec_v2:%s", s.hashStruct(key))
   }
   ```

2. **Request Batching and Deduplication**
   - Aggregate similar requests within time windows
   - Deduplicate concurrent requests for same parameters
   - Background cache warming for popular contests

## VALIDATION GATES:

```bash
# Backend validation
cd backend

# Syntax and imports
go mod tidy && go build ./internal/services/

# Unit tests for fuzzy matching
go test ./internal/services/ -run TestFuzzyPlayerMatching -v

# Integration tests with mock Claude API
go test ./internal/services/ -run TestAIRecommendations -v

# Load testing for caching efficiency
go test ./internal/services/ -run TestRecommendationCaching -bench=. -v

# Lint and security check
golangci-lint run ./internal/services/ai_recommendations.go
```

## TASK IMPLEMENTATION ORDER:

1. **Upgrade Claude API model and improve JSON parsing** (`ai_recommendations.go:342-402`)
2. **Implement fuzzy string matching with Levenshtein distance** (new methods)
3. **Add golf-specific prompt engineering** (`buildRecommendationPrompt` method)
4. **Enhance error handling with circuit breaker pattern** (new error handling)
5. **Optimize caching strategy with intelligent cache keys** (cache key generation)
6. **Add comprehensive logging and monitoring** (throughout service)
7. **Implement rate limiting safeguards** (new rate limiting logic)
8. **Add unit tests for fuzzy matching and caching** (new test files)
9. **Update API handlers to use enhanced service** (`handlers/ai_recommendations.go`)
10. **Validate integration with golf data providers** (integration testing)

## EXTERNAL REFERENCES:

**Claude API Documentation:**
- Model specifications: https://docs.anthropic.com/claude/docs/models-overview
- Structured outputs: https://docs.anthropic.com/claude/docs/tool-use
- Rate limits and best practices: https://docs.anthropic.com/claude/reference/rate-limits

**Golf DFS Strategy Resources:**
- Data Golf projections: https://datagolf.com/fantasy-projections
- Strokes gained methodology: https://www.pgatour.com/stats
- Course fit analysis examples: https://www.stokastic.com/pga/

**Existing Codebase Patterns:**
- Cache service: `backend/internal/services/cache.go`
- Error handling: `backend/internal/services/data_fetcher.go:163-180`
- Golf data providers: `backend/internal/providers/rapidapi_golf.go`
- Structured logging: `backend/internal/services/aggregator.go`

## GOTCHAS & IMPLEMENTATION NOTES:

1. **Claude API Version**: Must use `anthropic-version: 2023-06-01` header (line 365)
2. **Rate Limiting**: Basic RapidAPI plan has 20 requests/day - implement aggressive caching
3. **JSON Parsing**: Claude sometimes wraps JSON in markdown blocks - extract content between `[` and `]`
4. **Player Name Variations**: Handle common abbreviations (J. vs John, Jr. vs Junior)
5. **Team Matching**: Normalize team names (USA vs United States, LIV vs LIV Golf)
6. **Context Size**: Golf tournaments have 150+ players - may need prompt compression
7. **Concurrent Requests**: Implement request deduplication to avoid duplicate API calls
8. **Cache Invalidation**: Clear caches when player data updates (injuries, withdrawals)

## SUCCESS METRICS:

- Player matching accuracy > 95% (currently ~60% due to exact string matching)
- API response time < 3 seconds (with caching)
- Cache hit rate > 80% for similar requests
- Zero failed recommendations due to JSON parsing errors
- Support for golf-specific strategy terminology in responses
- Graceful degradation when Anthropic API is unavailable

**Confidence Score: 9/10** - Well-scoped implementation with existing infrastructure, clear patterns to follow, and comprehensive validation strategy.