## FEATURE:

Implement golf-specific correlation matrix with weather, tee times, course history, and recent form factors

## EXAMPLES:

Current generic implementation:
```go
// No golf-specific correlations
correlationMatrix := make(map[string]map[string]float64)
```

Golf-specific correlation matrix should include:
```go
type GolfCorrelationMatrix struct {
    PlayerCorrelations map[string]map[string]float64
    WeatherImpact      WeatherCorrelation
    TeeTimeGroups      map[string][]string
    CourseHistory      map[string]CourseStats
}

type WeatherCorrelation struct {
    WindPlayers    []PlayerWindProfile    // Players who excel/struggle in wind
    RainPlayers    []PlayerConditionProfile
    Temperature    map[string]float64     // Correlation by temp range
}

type TeeTimeCorrelation struct {
    AMWave  []string  // Players in morning wave
    PMWave  []string  // Players in afternoon wave
    Correlation float64 // How correlated their scores will be
}

func BuildGolfCorrelationMatrix(tournament Tournament, players []Player) GolfCorrelationMatrix {
    matrix := GolfCorrelationMatrix{}
    
    // 1. Tee time correlations (same wave = higher correlation)
    matrix.TeeTimeGroups = groupByTeeTime(players)
    
    // 2. Weather impact correlations
    matrix.WeatherImpact = analyzeWeatherProfiles(players, tournament.Weather)
    
    // 3. Course history correlations
    for _, player := range players {
        matrix.CourseHistory[player.ID] = getCourseStats(player, tournament.Course)
    }
    
    // 4. Playing style correlations
    // Bombers correlate on long courses
    // Accurate players correlate on tight courses
    
    // 5. Recent form momentum
    // Hot players tend to stay hot
    
    return matrix
}

// Example correlation factors:
// - Same tee time wave: +0.15 correlation
// - Similar wind performance: +0.20 correlation  
// - Both struggle at this course: +0.25 correlation
// - Opposite playing styles: -0.10 correlation
```

## DOCUMENTATION:

- Golf weather impact studies
- PGA Tour tee time analysis
- Course architecture and player performance
- Statistical correlation in golf tournaments

## OTHER CONSIDERATIONS:

- Current system has no golf-specific correlations
- Tee time waves create natural correlation (weather changes)
- Course history is a strong predictor
- Weather conditions affect players differently
- Playing styles (bomber vs accurate) matter by course
- Recent form creates momentum correlation
- Major championships have different correlation patterns
- Cut line creates binary correlation risk
- Need different correlation for 2-day vs 4-day contests
- Should account for paired groupings
- Consider implementing "narrative" correlations (rivalry, etc.)