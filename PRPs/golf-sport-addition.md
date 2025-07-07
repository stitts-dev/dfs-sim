# PRP: Golf Sport Addition to DFS Lineup Optimizer

## Purpose
Implement PGA Tour golf as a new sport in the DFS lineup optimizer, leveraging free golf APIs to provide comprehensive tournament optimization with golf-specific features including 4-day tournament handling, cut line projections, and course history analysis.

## Context & Background

### Project Structure Understanding
The DFS optimizer follows these patterns for adding new sports:
1. Sport constants defined in `backend/internal/dfs/types.go`
2. Provider-specific implementations in `backend/internal/providers/`
3. Sport-specific constraints in `backend/internal/optimizer/constraints.go`
4. Correlation matrices in `backend/internal/optimizer/correlation.go`
5. Frontend sport selection in `frontend/src/pages/Dashboard.tsx`

### Golf-Specific Challenges
- **4-Day Tournaments**: Unlike daily sports, golf spans Thursday-Sunday
- **Cut Risk**: ~50% of field eliminated after Friday (Round 2)
- **Weather Impact**: Significant performance factor across 4 days
- **Course History**: Player performance varies significantly by venue
- **Scoring Complexity**: DK/FD use different scoring systems with placement bonuses

### Free API Resources
1. **ESPN Golf API** (Primary): `site.api.espn.com/apis/site/v2/sports/golf/pga/`
2. **Free Golf API**: `freewebapi.com` (requires free signup)
3. **Live Golf API**: `livegolfapi.com` (completely free)
4. **Golf Course API**: `golfcourseapi.com` (free with email)

## Implementation Blueprint

### Phase 1: Core Data Models & API Integration

#### 1.1 Add Golf Sport Constant
**File**: `backend/internal/dfs/types.go`
```go
const (
    SportNBA  Sport = "nba"
    SportNFL  Sport = "nfl"
    SportMLB  Sport = "mlb"
    SportGolf Sport = "golf" // Add this
)
```

#### 1.2 Create Golf-Specific Database Tables
**File**: `backend/migrations/005_add_golf_support.sql`
```sql
-- Golf tournaments table
CREATE TABLE golf_tournaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    purse DECIMAL(10,2),
    course_id VARCHAR(50),
    course_name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'scheduled',
    current_round INTEGER DEFAULT 0,
    cut_line INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Golf player stats table
CREATE TABLE golf_player_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID REFERENCES players(id),
    tournament_id UUID REFERENCES golf_tournaments(id),
    round_number INTEGER NOT NULL,
    score INTEGER,
    strokes INTEGER,
    position INTEGER,
    made_cut BOOLEAN DEFAULT true,
    rounds_data JSONB, -- Store detailed round-by-round data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(player_id, tournament_id, round_number)
);

-- Golf course history table
CREATE TABLE golf_course_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID REFERENCES players(id),
    course_id VARCHAR(50) NOT NULL,
    tournaments_played INTEGER DEFAULT 0,
    avg_score DECIMAL(5,2),
    best_finish INTEGER,
    cuts_made INTEGER,
    top_10s INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(player_id, course_id)
);

-- Indexes for performance
CREATE INDEX idx_golf_tournaments_status ON golf_tournaments(status);
CREATE INDEX idx_golf_player_stats_tournament ON golf_player_stats(tournament_id);
CREATE INDEX idx_golf_course_history_player ON golf_course_history(player_id);
```

#### 1.3 Implement ESPN Golf Provider
**File**: `backend/internal/providers/espn_golf.go`
```go
package providers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "github.com/username/dfs-optimizer/internal/dfs"
)

type ESPNGolfProvider struct {
    client      *http.Client
    baseURL     string
    rateLimiter *time.Ticker
}

func NewESPNGolfProvider() *ESPNGolfProvider {
    return &ESPNGolfProvider{
        client:      &http.Client{Timeout: 10 * time.Second},
        baseURL:     "https://site.api.espn.com/apis/site/v2/sports/golf",
        rateLimiter: time.NewTicker(time.Second), // 1 request per second
    }
}

func (p *ESPNGolfProvider) GetTournamentLeaderboard(tournamentID string) (*GolfLeaderboard, error) {
    <-p.rateLimiter.C // Rate limiting
    
    url := fmt.Sprintf("%s/pga/scoreboard", p.baseURL)
    // Implementation continues...
}

func (p *ESPNGolfProvider) GetPlayerStats(playerID string) (*GolfPlayerStats, error) {
    // Implementation
}

func (p *ESPNGolfProvider) GetHistoricalData(playerID string, season int) (*PlayerHistory, error) {
    // Implementation
}
```

#### 1.4 Create Golf Models
**File**: `backend/internal/models/golf.go`
```go
package models

import (
    "time"
    "gorm.io/datatypes"
    "github.com/google/uuid"
)

type GolfTournament struct {
    ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    ExternalID string    `gorm:"uniqueIndex;not null"`
    Name       string    `gorm:"not null"`
    StartDate  time.Time `gorm:"not null"`
    EndDate    time.Time `gorm:"not null"`
    Purse      float64
    CourseID   string
    CourseName string
    Status     string    `gorm:"default:'scheduled'"` // scheduled, in_progress, completed
    CurrentRound int     `gorm:"default:0"`
    CutLine    int       // Score to make cut (e.g., +2)
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type GolfPlayerStats struct {
    ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    PlayerID     uuid.UUID      `gorm:"not null"`
    Player       Player         `gorm:"foreignKey:PlayerID"`
    TournamentID uuid.UUID      `gorm:"not null"`
    Tournament   GolfTournament `gorm:"foreignKey:TournamentID"`
    RoundNumber  int            `gorm:"not null"`
    Score        int            // Relative to par
    Strokes      int            // Total strokes
    Position     int
    MadeCut      bool           `gorm:"default:true"`
    RoundsData   datatypes.JSON // Detailed round data
    CreatedAt    time.Time
}

type GolfCourseHistory struct {
    ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    PlayerID          uuid.UUID `gorm:"not null"`
    Player            Player    `gorm:"foreignKey:PlayerID"`
    CourseID          string    `gorm:"not null"`
    TournamentsPlayed int       `gorm:"default:0"`
    AvgScore          float64
    BestFinish        int
    CutsMade          int
    Top10s            int
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

### Phase 2: Golf-Specific Optimization Logic

#### 2.1 Implement Golf Constraints
**File**: Update `backend/internal/optimizer/constraints.go`
```go
func (lc *LineupConstraints) setupGolfConstraints(platform string) {
    lc.SalaryCap = 50000 // DraftKings
    if platform == "fanduel" {
        lc.SalaryCap = 60000
    }
    
    lc.MaxPlayers = 6
    lc.MinPlayers = 6
    
    // Golf has no position constraints - just 6 golfers
    lc.PositionConstraints = map[string]PositionConstraint{
        "G": {Position: "G", MinRequired: 6, MaxAllowed: 6}, // Golfer
    }
    
    // No team constraints in golf
    lc.MaxPlayersPerTeam = 6
    
    // Tournament-specific constraints
    lc.CustomConstraints = map[string]interface{}{
        "requireCutProjection": true,
        "minProjectedCutProbability": 0.5, // At least 50% chance to make cut
    }
}
```

#### 2.2 Create Golf Correlation Matrix
**File**: `backend/internal/optimizer/golf_correlation.go`
```go
package optimizer

import "github.com/username/dfs-optimizer/internal/models"

type GolfCorrelationBuilder struct {
    players []models.Player
    tournaments map[string]*models.GolfTournament
}

func (gb *GolfCorrelationBuilder) BuildCorrelationMatrix() map[string]map[string]float64 {
    correlations := make(map[string]map[string]float64)
    
    for i, p1 := range gb.players {
        correlations[p1.ID.String()] = make(map[string]float64)
        
        for j, p2 := range gb.players {
            if i == j {
                correlations[p1.ID.String()][p2.ID.String()] = 1.0
                continue
            }
            
            correlation := gb.calculateGolfCorrelation(p1, p2)
            correlations[p1.ID.String()][p2.ID.String()] = correlation
        }
    }
    
    return correlations
}

func (gb *GolfCorrelationBuilder) calculateGolfCorrelation(p1, p2 models.Player) float64 {
    correlation := 0.0
    
    // Same tee time correlation (playing partners)
    if gb.haveSameTeeTime(p1, p2) {
        correlation += 0.15
    }
    
    // Same wave (AM/PM) correlation
    if gb.inSameWave(p1, p2) {
        correlation += 0.05
    }
    
    // Country/region correlation (Ryder Cup effect)
    if p1.Metadata["country"] == p2.Metadata["country"] {
        correlation += 0.10
    }
    
    // Similar skill level correlation (world ranking)
    rankDiff := abs(p1.Metadata["worldRank"].(int) - p2.Metadata["worldRank"].(int))
    if rankDiff < 20 {
        correlation += 0.08
    }
    
    return correlation
}
```

#### 2.3 Golf-Specific Projections
**File**: `backend/internal/services/golf_projections.go`
```go
package services

import (
    "math"
    "github.com/username/dfs-optimizer/internal/models"
)

type GolfProjectionService struct {
    playerStats     map[string]*PlayerGolfStats
    courseHistory   map[string]map[string]*models.GolfCourseHistory
    weatherService  *WeatherService
}

func (gps *GolfProjectionService) GenerateProjections(
    players []models.Player,
    tournament *models.GolfTournament,
) map[string]*GolfProjection {
    projections := make(map[string]*GolfProjection)
    
    for _, player := range players {
        projection := &GolfProjection{
            PlayerID: player.ID.String(),
            ExpectedScore: gps.calculateExpectedScore(player, tournament),
            CutProbability: gps.calculateCutProbability(player, tournament),
            Top10Probability: gps.calculateTop10Probability(player, tournament),
            WinProbability: gps.calculateWinProbability(player, tournament),
        }
        
        // Apply course history adjustment
        if history := gps.getCourseHistory(player.ID.String(), tournament.CourseID); history != nil {
            projection.ExpectedScore = gps.adjustForCourseHistory(projection.ExpectedScore, history)
        }
        
        // Apply weather adjustment
        weatherImpact := gps.weatherService.GetImpactFactor(tournament.StartDate)
        projection.ExpectedScore *= weatherImpact
        
        // Calculate DFS points based on platform scoring
        projection.DKPoints = gps.calculateDKPoints(projection)
        projection.FDPoints = gps.calculateFDPoints(projection)
        
        projections[player.ID.String()] = projection
    }
    
    return projections
}

func (gps *GolfProjectionService) calculateCutProbability(
    player models.Player,
    tournament *models.GolfTournament,
) float64 {
    // Based on recent form, course history, and field strength
    baseProbability := 0.5
    
    // Adjust based on recent cuts made
    recentCutsMade := gps.getRecentCutsMade(player.ID.String(), 10)
    baseProbability += (float64(recentCutsMade) / 10.0 - 0.5) * 0.3
    
    // Adjust based on world ranking
    if rank, ok := player.Metadata["worldRank"].(int); ok {
        if rank <= 50 {
            baseProbability += 0.2
        } else if rank <= 100 {
            baseProbability += 0.1
        }
    }
    
    return math.Min(math.Max(baseProbability, 0.1), 0.95)
}
```

### Phase 3: Frontend Integration

#### 3.1 Add Golf to Sport Selector
**File**: `frontend/src/pages/Dashboard.tsx`
```typescript
const sports = [
    { value: 'all', label: 'All Sports', icon: '🏆' },
    { value: 'nba', label: 'NBA', icon: '🏀' },
    { value: 'nfl', label: 'NFL', icon: '🏈' },
    { value: 'mlb', label: 'MLB', icon: '⚾' },
    { value: 'nhl', label: 'NHL', icon: '🏒' },
    { value: 'golf', label: 'Golf', icon: '⛳' }, // Add this
]
```

#### 3.2 Create Golf-Specific Components
**File**: `frontend/src/components/golf/GolfPlayerCard.tsx`
```typescript
import React from 'react';
import { GolfPlayer } from '../../types/golf';

interface GolfPlayerCardProps {
    player: GolfPlayer;
    onAdd: (player: GolfPlayer) => void;
    isSelected: boolean;
}

export const GolfPlayerCard: React.FC<GolfPlayerCardProps> = ({ 
    player, 
    onAdd, 
    isSelected 
}) => {
    return (
        <div className={`border rounded-lg p-4 ${isSelected ? 'bg-green-100' : 'bg-white'}`}>
            <div className="flex justify-between items-start">
                <div>
                    <h3 className="font-semibold">{player.name}</h3>
                    <p className="text-sm text-gray-600">World Rank: #{player.worldRank}</p>
                    <p className="text-sm text-gray-600">Recent Form: {player.recentForm}</p>
                </div>
                <div className="text-right">
                    <p className="font-bold">${player.salary.toLocaleString()}</p>
                    <p className="text-sm">{player.projectedPoints} pts</p>
                    <p className="text-xs text-gray-600">Cut: {(player.cutProbability * 100).toFixed(0)}%</p>
                </div>
            </div>
            
            <div className="mt-2 grid grid-cols-3 gap-2 text-xs">
                <div>
                    <span className="text-gray-600">Top 10:</span>
                    <span className="ml-1 font-semibold">{(player.top10Probability * 100).toFixed(0)}%</span>
                </div>
                <div>
                    <span className="text-gray-600">Win:</span>
                    <span className="ml-1 font-semibold">{(player.winProbability * 100).toFixed(0)}%</span>
                </div>
                <div>
                    <span className="text-gray-600">Own:</span>
                    <span className="ml-1 font-semibold">{player.ownership}%</span>
                </div>
            </div>
            
            <button
                onClick={() => onAdd(player)}
                disabled={isSelected}
                className="mt-3 w-full py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:bg-gray-400"
            >
                {isSelected ? 'Selected' : 'Add to Lineup'}
            </button>
        </div>
    );
};
```

### Phase 4: Testing & Validation

#### 4.1 API Test Documentation
**File**: `backend/test-golf-api.md`
```markdown
# Golf API Testing Guide

## Prerequisites
- Backend server running on port 8080
- PostgreSQL database with golf tables
- Redis running for caching

## Test Scenarios

### 1. Fetch Golf Tournaments
```bash
curl -X GET http://localhost:8080/api/golf/tournaments \
  -H "Authorization: Bearer $TOKEN"
```

### 2. Get Tournament Leaderboard
```bash
curl -X GET http://localhost:8080/api/golf/tournaments/{tournamentId}/leaderboard \
  -H "Authorization: Bearer $TOKEN"
```

### 3. Create Golf Contest
```bash
curl -X POST http://localhost:8080/api/contests \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "PGA Championship GPP",
    "sport": "golf",
    "platform": "draftkings",
    "entryFee": 20,
    "maxEntries": 150,
    "startTime": "2024-05-16T07:00:00Z",
    "tournamentId": "pga-championship-2024"
  }'
```

### 4. Optimize Golf Lineup
```bash
curl -X POST http://localhost:8080/api/optimize \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contestId": "contest-uuid",
    "optimizerConfig": {
      "numLineups": 20,
      "minCutProbability": 0.6,
      "useCorrelations": true,
      "diversityMultiplier": 0.3,
      "lockedPlayerIds": [],
      "excludedPlayerIds": []
    }
  }'
```
```

#### 4.2 Integration Test Script
**File**: `backend/scripts/test-golf-integration.sh`
```bash
#!/bin/bash

echo "Testing Golf Integration..."

# 1. Test ESPN API integration
echo "Testing ESPN Golf API..."
curl -s https://site.api.espn.com/apis/site/v2/sports/golf/pga/scoreboard | jq '.events[0].name'

# 2. Test database operations
echo "Testing Golf Database..."
psql -U postgres -d dfs_optimizer -c "SELECT COUNT(*) FROM golf_tournaments;"

# 3. Test optimization with golf constraints
echo "Testing Golf Optimization..."
# Create test contest and run optimization

# 4. Validate results
echo "Validating Golf Lineups..."
# Check that all lineups have 6 golfers and meet salary cap

echo "Golf Integration Tests Complete!"
```

## Validation Gates

### Backend Validation
```bash
# Navigate to backend
cd backend

# Run Go tests (when implemented)
go test ./internal/providers/espn_golf_test.go -v
go test ./internal/optimizer/golf_correlation_test.go -v
go test ./internal/services/golf_projections_test.go -v

# Lint code
golangci-lint run

# Check compilation
go build ./...
```

### Frontend Validation
```bash
# Navigate to frontend
cd frontend

# Type checking
npm run type-check

# Linting
npm run lint

# Build verification
npm run build
```

### Integration Validation
```bash
# Run full integration test
./backend/scripts/test-golf-integration.sh

# Verify API endpoints
curl http://localhost:8080/api/golf/tournaments | jq
curl http://localhost:8080/api/contests?sport=golf | jq

# Test optimization with real data
curl -X POST http://localhost:8080/api/optimize \
  -H "Content-Type: application/json" \
  -d @test-golf-optimization-request.json
```

## Implementation Tasks (in order)

1. **Database Setup** ✓
   - Create migration file with golf tables
   - Run migration to create tables
   - Verify tables exist

2. **Backend Sport Integration** ✓
   - Add SportGolf constant
   - Update provider switch statements
   - Add golf case to constraints

3. **ESPN Golf Provider** ✓
   - Implement API client
   - Add tournament fetching
   - Add player stats fetching
   - Implement caching

4. **Golf Models** ✓
   - Create tournament model
   - Create player stats model
   - Create course history model

5. **Golf Projections** ✓
   - Implement cut probability
   - Add course history adjustments
   - Calculate DFS points

6. **Golf Correlations** ✓
   - Build correlation matrix
   - Add tee time correlations
   - Add country correlations

7. **Golf Optimization** ✓
   - Add golf constraints
   - Implement cut line filtering
   - Add golf-specific rules

8. **Frontend Components** ✓
   - Add golf to sport selector
   - Create golf player card
   - Add tournament selector
   - Create cut probability display

9. **API Endpoints** ✓
   - Add golf tournament endpoints
   - Add golf-specific contest creation
   - Update optimization for golf

10. **Testing & Documentation** ✓
    - Create API test documentation
    - Write integration tests
    - Update README

## Success Criteria

1. **Data Integration**
   - ✓ Successfully fetch tournament data from ESPN API
   - ✓ Store and update player statistics
   - ✓ Calculate accurate projections

2. **Optimization**
   - ✓ Generate valid 6-player lineups
   - ✓ Respect salary cap constraints
   - ✓ Apply cut probability filtering
   - ✓ Use golf-specific correlations

3. **User Experience**
   - ✓ Display golf tournaments in UI
   - ✓ Show cut probabilities for players
   - ✓ Allow lineup optimization
   - ✓ Export lineups for DK/FD

4. **Performance**
   - ✓ API responses under 2 seconds
   - ✓ Optimization completes in < 30 seconds
   - ✓ Efficient caching of golf data

## External Resources

1. **ESPN Golf API Documentation**
   - Base: https://site.api.espn.com/apis/site/v2/sports/golf/pga/
   - Examples: https://gist.github.com/nntrn/ee26cb2a0716de0947a0a4e9a157bc1c

2. **Free Golf API**
   - Docs: https://freewebapi.com/golf-api
   - Signup: https://freewebapi.com/register

3. **DraftKings Golf Scoring**
   - Rules: https://www.draftkings.com/help/rules/golf

4. **FanDuel Golf Scoring**
   - Rules: https://www.fanduel.com/rules/golf

## Common Pitfalls to Avoid

1. **Cut Line Handling**: Always filter players who missed cut before final scoring
2. **Tournament Status**: Check if tournament is postponed/cancelled
3. **Time Zones**: PGA tournaments span multiple time zones
4. **Withdrawal Handling**: Players can WD during tournament
5. **API Rate Limits**: Implement proper rate limiting for free APIs

## Notes for AI Implementation

- Start with Phase 1 and complete each phase before moving to the next
- Run validation gates after each major phase
- Use existing patterns from NBA/NFL/MLB implementations
- Test with real PGA tournament data
- Ensure all free APIs are properly integrated without authentication issues
- Focus on MVP features first, enhance later

**Confidence Score: 9/10**

This PRP provides comprehensive context and clear implementation steps that should enable successful one-pass implementation of golf support in the DFS optimizer.