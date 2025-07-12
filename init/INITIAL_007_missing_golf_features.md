## FEATURE:

Implement critical missing golf DFS features: cut line predictions, ownership projections, late swap functionality, and tournament format support

## EXAMPLES:

Missing features that should be implemented:

### 1. Cut Line Predictions
```go
type CutLinePredictor struct {
    TournamentID    string
    PredictedLine   float64  // e.g., -2, +1, E
    Confidence      float64
    LastUpdated     time.Time
}

func PredictCutLine(tournament Tournament, field []Player) CutLinePredictor {
    // Analyze:
    // - Course difficulty history
    // - Weather conditions
    // - Field strength
    // - Scoring conditions
    
    historicalCuts := getHistoricalCuts(tournament.CourseID)
    weatherAdjustment := calculateWeatherImpact(tournament.Weather)
    fieldStrength := calculateFieldStrength(field)
    
    return CutLinePredictor{
        PredictedLine: historicalCuts.Average + weatherAdjustment,
        Confidence: calculateCutConfidence(historicalCuts.StdDev),
    }
}
```

### 2. Ownership Projections
```go
type OwnershipProjection struct {
    PlayerID       string
    ProjectedOwn   float64  // Percentage
    GPPOwnership   float64
    CashOwnership  float64
    Factors        []string // Why high/low ownership
}

func ProjectOwnership(player Player, contest Contest) OwnershipProjection {
    factors := []string{}
    
    // Factors affecting ownership:
    // - Recent performance/wins
    // - Media coverage
    // - Price changes
    // - Course history
    // - Popular tout recommendations
    
    if player.RecentWin {
        factors = append(factors, "Recent winner")
        ownership += 5.0
    }
    
    if player.PriceDropped {
        factors = append(factors, "Price dropped")
        ownership += 3.0
    }
}
```

### 3. Late Swap Engine
```go
type LateSwapEngine struct {
    EnableAutoSwap bool
    SwapRules      []SwapRule
}

type SwapRule struct {
    Trigger    string  // "missed_cut", "injury", "weather_delay"
    Action     string  // "replace_with_next_best", "optimize_remaining"
    Conditions map[string]interface{}
}

func (e *LateSwapEngine) MonitorAndSwap(lineup Lineup) {
    for _, player := range lineup.Players {
        if player.Status == "CUT" || player.Status == "WD" {
            replacement := findBestReplacement(player, lineup)
            executeSwap(player, replacement, lineup)
        }
    }
}
```

### 4. Tournament Format Support
```go
type TournamentFormat struct {
    Type           string  // "stroke_play", "match_play", "team"
    CutAfterRound  int     // Usually 2, but varies
    PlayoffFormat  string  // "sudden_death", "aggregate"
    SpecialRules   []Rule
}

// Different optimization for different formats
func OptimizeForFormat(format TournamentFormat, players []Player) []Lineup {
    switch format.Type {
    case "match_play":
        // Favor high-variance players
        return optimizeMatchPlay(players)
    case "team":
        // Consider team dynamics
        return optimizeTeamEvent(players)
    case "no_cut":
        // Different strategy without cut risk
        return optimizeNoCut(players)
    }
}
```

### 5. Live Scoring Integration
```go
type LiveScoring struct {
    TournamentID string
    Players      map[string]LiveScore
    LastUpdate   time.Time
}

type LiveScore struct {
    Position      int
    Score         string  // "-5", "E", "+2"
    Thru          int     // Holes completed
    ProjectedFinal float64
    Trending      string  // "up", "down", "stable"
}
```

## DOCUMENTATION:

- PGA Tour tournament formats
- DraftKings late swap rules
- FanDuel ownership statistics
- Live scoring APIs and webhooks

## OTHER CONSIDERATIONS:

- Cut line prediction crucial for lineup construction
- Ownership projections help find leverage
- Late swap can save lineups after cuts
- Different tournament formats need different strategies
- Live scoring enables real-time adjustments
- Weather delays create swap opportunities
- Injury news integration needed
- Should support 6-man and 5-man contests
- Playoff/overtime scoring rules
- Season-long contest support
- Head-to-head optimization different from GPPs