## FEATURE:

Enhance golf-specific optimization algorithm with proper lineup generation, cut considerations, and tournament dynamics

## EXAMPLES:

Current issues in `backend/internal/optimizer/algorithm.go`:
```go
// Line 267: Simplistic golf lineup generation
if sport == "GOLF" {
    newLineups := a.generateGolfLineups(validPlayers, contests[0], req.NumLineups-len(lineups))
    // No cut line consideration
    // No tournament position weighting
    // Random selection without strategy
}

// Line 289-340: Basic random selection
for len(lineup) < totalSlots && attempts < maxAttempts {
    randomIndex := rand.Intn(len(validPlayers))
    player := validPlayers[randomIndex]
    // No position-specific logic for golf
    // No consideration of player correlations
}
```

Improved implementation should include:
```go
type GolfOptimizer struct {
    CutLineThreshold float64  // Probability threshold
    PositionWeights  map[int]float64  // T1-T10 weights
    VolatilityBonus  float64  // Reward for high variance
}

func (a *Algorithm) generateGolfLineups(players []Player, contest Contest, count int) []Lineup {
    // Group players by cut probability
    likelyToCut := filterByCutProbability(players, 0.7)
    bubble := filterByCutProbability(players, 0.4, 0.7)
    unlikely := filterByCutProbability(players, 0, 0.4)
    
    // Build lineups with cut line strategy
    for i := 0; i < count; i++ {
        lineup := Lineup{}
        
        // Core (4-5 players likely to make cut)
        core := selectCore(likelyToCut, 4, 5)
        
        // Leverage (1-2 high upside players)
        leverage := selectLeverage(bubble, unlikely, 1, 2)
        
        // Optimize for tournament position upside
        lineup = optimizeForPosition(core, leverage, contest)
        
        lineups = append(lineups, lineup)
    }
}

func optimizeForPosition(core, leverage []Player, contest Contest) Lineup {
    // Weight players by:
    // - Top 10 probability
    // - Course history
    // - Recent form
    // - Strokes gained categories matching course
}
```

## DOCUMENTATION:

- PGA Tour cut rules: https://www.pgatour.com/news/2019/09/26/pga-tour-cut-rules-making-the-cut
- Golf DFS strategy guides
- Tournament position probability models
- Strokes gained statistics interpretation

## OTHER CONSIDERATIONS:

- Current algorithm treats golf like other sports (wrong approach)
- No cut line modeling (crucial for golf DFS)
- Missing tournament position optimization (T5, T10, etc.)
- No consideration of course fit
- Ignoring player volatility (important for GPPs)
- Not using strokes gained data effectively
- Missing late swap considerations
- No weather impact on lineup construction
- Should implement "stars and scrubs" vs "balanced" lineup types
- Need different strategies for GPP vs cash games
- Consider implementing ownership leverage strategies