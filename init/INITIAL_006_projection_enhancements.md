## FEATURE:

Enhance golf projections with weather integration, strokes gained statistics, confidence scoring, and tournament-specific adjustments

## EXAMPLES:

Current basic projections:
```go
type Player struct {
    ProjectedPoints float64  // Too simple
    Salary         int
}
```

Enhanced projection system:
```go
type GolfProjection struct {
    BaseProjection    float64
    WeatherAdjustment float64
    CourseAdjustment  float64
    FormAdjustment    float64
    Confidence        float64  // 0-100 score
    
    // Strokes Gained Components
    SGOffTheTee      float64
    SGApproach       float64
    SGAroundGreen    float64
    SGPutting        float64
    
    // Outcome Distributions
    CutProbability   float64
    Top10Probability float64
    WinProbability   float64
    
    // DFS Specific
    ProjectedPoints  float64
    PointsFloor      float64  // 20th percentile
    PointsCeiling    float64  // 80th percentile
    Volatility       float64
}

func CalculateGolfProjection(player Player, tournament Tournament) GolfProjection {
    proj := GolfProjection{}
    
    // 1. Base projection from season average
    proj.BaseProjection = player.SeasonAvgPoints
    
    // 2. Weather adjustment
    weatherImpact := calculateWeatherImpact(player, tournament.Weather)
    proj.WeatherAdjustment = weatherImpact
    
    // 3. Course fit adjustment
    coursefit := analyzeCoursefit(player.StrokesGained, tournament.Course)
    proj.CourseAdjustment = coursefit
    
    // 4. Recent form (last 5 tournaments)
    form := calculateFormTrend(player.RecentResults)
    proj.FormAdjustment = form
    
    // 5. Calculate confidence based on data quality
    proj.Confidence = calculateConfidence(player)
    
    // 6. Final projection with uncertainty
    proj.ProjectedPoints = proj.BaseProjection + 
                          proj.WeatherAdjustment + 
                          proj.CourseAdjustment + 
                          proj.FormAdjustment
    
    // 7. Calculate floor/ceiling
    proj.PointsFloor = proj.ProjectedPoints - (proj.Volatility * 1.5)
    proj.PointsCeiling = proj.ProjectedPoints + (proj.Volatility * 2.0)
    
    return proj
}

func calculateConfidence(player Player) float64 {
    confidence := 100.0
    
    // Reduce confidence for:
    // - Missing recent data (-20)
    // - First time at course (-15)
    // - Returning from injury (-25)
    // - Limited sample size (-10)
    // - Weather uncertainty (-10)
    
    return confidence
}
```

## DOCUMENTATION:

- PGA Tour Strokes Gained: https://www.pgatour.com/news/2016/05/31/strokes-gained-defined
- Weather impact on golf scoring studies
- Course architecture analysis
- DFS golf projection methodologies

## OTHER CONSIDERATIONS:

- Current projections don't account for weather (huge factor)
- No strokes gained breakdown (key for course fit)
- Missing confidence scores (helps users trust projections)
- No floor/ceiling projections (important for GPP vs cash)
- Form trending not implemented
- Course history weight not adjusted
- No injury status consideration
- Missing field strength adjustment
- Should integrate Vegas odds for validation
- Need different projections for different contest types
- Consider implementing machine learning model
- Add projection explanations for transparency