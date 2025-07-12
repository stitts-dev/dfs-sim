# PRP: Golf Sport Addition to DFS Lineup Optimizer (Enhanced)

## Purpose
Implement PGA Tour golf as a new sport in the DFS lineup optimizer with production-ready code following React and Go best practices, ensuring complete frontend-backend integration with real-time updates, comprehensive error handling, and performance optimization.

## Context & Architecture Overview

### Existing Architecture Patterns
```
backend/
├── cmd/server/main.go         # Entry point with dependency injection
├── internal/
│   ├── api/                   # HTTP handlers (thin controllers)
│   ├── services/              # Business logic (testable services)
│   ├── models/                # GORM models with hooks
│   ├── providers/             # External API integrations
│   ├── optimizer/             # Core optimization algorithms
│   └── middleware/            # Auth, logging, error handling

frontend/
├── src/
│   ├── store/                 # Redux/Zustand state management
│   ├── hooks/                 # Custom React hooks
│   ├── services/              # API client with React Query
│   ├── components/            # Reusable UI components
│   └── pages/                 # Route-based page components
```

### Golf-Specific Technical Challenges
1. **Multi-Day Events**: 4-day tournaments vs single-day sports
2. **Dynamic Field Changes**: Players withdraw, cut lines move
3. **Weather Delays**: Rounds suspended/resumed
4. **Scoring Complexity**: Stroke play, match play, team events
5. **Real-Time Updates**: Live scoring during rounds
6. **Historical Data Volume**: Years of course-specific data

## Phase 1: Backend Foundation with Go Best Practices

### 1.1 Database Schema with Advanced Features
```sql
-- Enhanced migration with proper indexes and constraints
-- File: backend/migrations/005_add_golf_support.sql

BEGIN;

-- Golf tournaments with full metadata
CREATE TABLE golf_tournaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    purse DECIMAL(12,2),
    winner_share DECIMAL(10,2),
    fedex_points INTEGER,
    course_id VARCHAR(50),
    course_name VARCHAR(255),
    course_par INTEGER,
    course_yards INTEGER,
    status VARCHAR(50) DEFAULT 'scheduled',
    current_round INTEGER DEFAULT 0,
    cut_line INTEGER,
    cut_rule VARCHAR(100), -- "Top 70 and ties"
    weather_conditions JSONB,
    field_strength DECIMAL(5,2), -- OWGR average
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Partial indexes for performance
CREATE INDEX idx_golf_tournaments_active ON golf_tournaments(status) 
    WHERE status IN ('in_progress', 'scheduled');
CREATE INDEX idx_golf_tournaments_dates ON golf_tournaments(start_date, end_date);

-- Player tournament entries with detailed tracking
CREATE TABLE golf_player_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID REFERENCES players(id) ON DELETE CASCADE,
    tournament_id UUID REFERENCES golf_tournaments(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'entered', -- entered, withdrawn, cut, active
    starting_position INTEGER,
    current_position INTEGER,
    total_score INTEGER, -- relative to par
    thru_holes INTEGER,
    rounds_scores INTEGER[], -- [70, 68, 72, 69]
    tee_times TIMESTAMP[],
    playing_partners UUID[],
    dk_salary INTEGER,
    fd_salary INTEGER,
    dk_ownership DECIMAL(5,2),
    fd_ownership DECIMAL(5,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(player_id, tournament_id)
);

-- Optimized compound indexes
CREATE INDEX idx_player_entries_tournament_status ON golf_player_entries(tournament_id, status);
CREATE INDEX idx_player_entries_position ON golf_player_entries(current_position) 
    WHERE status = 'active';

-- Real-time round scoring
CREATE TABLE golf_round_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id UUID REFERENCES golf_player_entries(id) ON DELETE CASCADE,
    round_number INTEGER NOT NULL CHECK (round_number BETWEEN 1 AND 4),
    holes_completed INTEGER DEFAULT 0,
    score INTEGER,
    strokes INTEGER,
    birdies INTEGER DEFAULT 0,
    eagles INTEGER DEFAULT 0,
    bogeys INTEGER DEFAULT 0,
    double_bogeys INTEGER DEFAULT 0,
    hole_scores JSONB, -- {"1": 4, "2": 3, ...}
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entry_id, round_number)
);

-- Course history with advanced metrics
CREATE TABLE golf_course_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID REFERENCES players(id) ON DELETE CASCADE,
    course_id VARCHAR(50) NOT NULL,
    tournaments_played INTEGER DEFAULT 0,
    rounds_played INTEGER DEFAULT 0,
    total_strokes INTEGER,
    scoring_avg DECIMAL(5,2),
    adj_scoring_avg DECIMAL(5,2), -- Adjusted for field strength
    best_finish INTEGER,
    worst_finish INTEGER,
    cuts_made INTEGER,
    missed_cuts INTEGER,
    top_10s INTEGER,
    top_25s INTEGER,
    wins INTEGER,
    strokes_gained_total DECIMAL(5,2),
    sg_tee_to_green DECIMAL(5,2),
    sg_putting DECIMAL(5,2),
    last_played DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(player_id, course_id)
);

-- Create update trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_golf_tournaments_updated_at BEFORE UPDATE ON golf_tournaments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_golf_player_entries_updated_at BEFORE UPDATE ON golf_player_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMIT;

-- Rollback script
-- File: backend/migrations/005_add_golf_support_down.sql
BEGIN;
DROP TABLE IF EXISTS golf_round_scores CASCADE;
DROP TABLE IF EXISTS golf_player_entries CASCADE;
DROP TABLE IF EXISTS golf_course_history CASCADE;
DROP TABLE IF EXISTS golf_tournaments CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;
COMMIT;
```

### 1.2 Go Models with GORM Best Practices
```go
// File: backend/internal/models/golf.go
package models

import (
    "time"
    "database/sql/driver"
    "encoding/json"
    "gorm.io/gorm"
    "github.com/google/uuid"
    "github.com/lib/pq"
)

// Custom types for better type safety
type TournamentStatus string

const (
    TournamentScheduled   TournamentStatus = "scheduled"
    TournamentInProgress  TournamentStatus = "in_progress"
    TournamentCompleted   TournamentStatus = "completed"
    TournamentPostponed   TournamentStatus = "postponed"
    TournamentCancelled   TournamentStatus = "cancelled"
)

type PlayerEntryStatus string

const (
    EntryStatusEntered    PlayerEntryStatus = "entered"
    EntryStatusWithdrawn  PlayerEntryStatus = "withdrawn"
    EntryStatusCut        PlayerEntryStatus = "cut"
    EntryStatusActive     PlayerEntryStatus = "active"
    EntryStatusCompleted  PlayerEntryStatus = "completed"
)

// WeatherConditions for JSONB storage
type WeatherConditions struct {
    Temperature int     `json:"temperature"`
    WindSpeed   int     `json:"wind_speed"`
    WindDir     string  `json:"wind_direction"`
    Conditions  string  `json:"conditions"`
    Humidity    int     `json:"humidity"`
}

func (w WeatherConditions) Value() (driver.Value, error) {
    return json.Marshal(w)
}

func (w *WeatherConditions) Scan(value interface{}) error {
    bytes, ok := value.([]byte)
    if !ok {
        return errors.New("type assertion to []byte failed")
    }
    return json.Unmarshal(bytes, w)
}

// GolfTournament model with hooks
type GolfTournament struct {
    ID               uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    ExternalID       string             `gorm:"uniqueIndex;not null"`
    Name             string             `gorm:"not null"`
    StartDate        time.Time          `gorm:"not null;index"`
    EndDate          time.Time          `gorm:"not null"`
    Purse            float64
    WinnerShare      float64
    FedexPoints      int
    CourseID         string             `gorm:"index"`
    CourseName       string
    CoursePar        int
    CourseYards      int
    Status           TournamentStatus   `gorm:"type:varchar(50);default:'scheduled';index:idx_active,where:status IN ('in_progress','scheduled')"`
    CurrentRound     int                `gorm:"default:0"`
    CutLine          int
    CutRule          string
    WeatherConditions WeatherConditions `gorm:"type:jsonb"`
    FieldStrength    float64
    CreatedAt        time.Time
    UpdatedAt        time.Time
    
    // Associations
    PlayerEntries    []GolfPlayerEntry  `gorm:"foreignKey:TournamentID"`
}

// BeforeCreate hook for validation
func (t *GolfTournament) BeforeCreate(tx *gorm.DB) error {
    if t.StartDate.After(t.EndDate) {
        return errors.New("start date must be before end date")
    }
    return nil
}

// GolfPlayerEntry with optimized loading
type GolfPlayerEntry struct {
    ID               uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    PlayerID         uuid.UUID          `gorm:"not null;uniqueIndex:idx_player_tournament,priority:1"`
    Player           *Player            `gorm:"foreignKey:PlayerID;preload:false"`
    TournamentID     uuid.UUID          `gorm:"not null;uniqueIndex:idx_player_tournament,priority:2;index:idx_tournament_status,priority:1"`
    Tournament       *GolfTournament    `gorm:"foreignKey:TournamentID;preload:false"`
    Status           PlayerEntryStatus  `gorm:"type:varchar(50);default:'entered';index:idx_tournament_status,priority:2"`
    StartingPosition int
    CurrentPosition  int                `gorm:"index:idx_position,where:status = 'active'"`
    TotalScore       int
    ThruHoles        int
    RoundsScores     pq.Int64Array      `gorm:"type:integer[]"`
    TeeTimes         pq.StringArray     `gorm:"type:timestamp[]"`
    PlayingPartners  pq.StringArray     `gorm:"type:uuid[]"`
    DKSalary         int
    FDSalary         int
    DKOwnership      float64
    FDOwnership      float64
    CreatedAt        time.Time
    UpdatedAt        time.Time
    
    // Associations
    RoundScores      []GolfRoundScore   `gorm:"foreignKey:EntryID"`
}

// GetProjectedScore calculates expected score based on history
func (e *GolfPlayerEntry) GetProjectedScore(courseHistory *GolfCourseHistory) float64 {
    if courseHistory == nil || courseHistory.ScoringAvg == 0 {
        return 280.0 // Default 4-round score
    }
    
    // Adjust for recent form and course history
    baseScore := courseHistory.ScoringAvg * 4
    formAdjustment := e.calculateFormAdjustment()
    
    return baseScore + formAdjustment
}

// Optimized preloading strategies
func (db *gorm.DB) PreloadGolfData() *gorm.DB {
    return db.Preload("Player", func(db *gorm.DB) *gorm.DB {
        return db.Select("id", "name", "external_id", "metadata")
    }).Preload("Tournament", func(db *gorm.DB) *gorm.DB {
        return db.Select("id", "name", "course_name", "status", "current_round")
    })
}
```

### 1.3 ESPN Golf Provider with Error Handling & Caching
```go
// File: backend/internal/providers/espn_golf.go
package providers

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"
    "golang.org/x/time/rate"
    
    "github.com/username/dfs-optimizer/internal/models"
)

// ESPNGolfProvider with dependency injection
type ESPNGolfProvider struct {
    client       *http.Client
    baseURL      string
    cache        *redis.Client
    logger       *zap.Logger
    rateLimiter  *rate.Limiter
    mu           sync.RWMutex
    webhookURL   string // For real-time updates
}

// ProviderConfig for dependency injection
type ProviderConfig struct {
    HTTPClient   *http.Client
    Cache        *redis.Client
    Logger       *zap.Logger
    WebhookURL   string
    RateLimit    rate.Limit
}

// NewESPNGolfProvider with proper initialization
func NewESPNGolfProvider(cfg *ProviderConfig) *ESPNGolfProvider {
    if cfg.HTTPClient == nil {
        cfg.HTTPClient = &http.Client{
            Timeout: 15 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        }
    }
    
    if cfg.RateLimit == 0 {
        cfg.RateLimit = rate.Every(time.Second) // 1 request per second
    }
    
    return &ESPNGolfProvider{
        client:      cfg.HTTPClient,
        baseURL:     "https://site.api.espn.com/apis/site/v2/sports/golf",
        cache:       cfg.Cache,
        logger:      cfg.Logger,
        rateLimiter: rate.NewLimiter(cfg.RateLimit, 1),
        webhookURL:  cfg.WebhookURL,
    }
}

// GetTournamentLeaderboard with caching and error handling
func (p *ESPNGolfProvider) GetTournamentLeaderboard(ctx context.Context, tournamentID string) (*TournamentLeaderboard, error) {
    cacheKey := fmt.Sprintf("golf:leaderboard:%s", tournamentID)
    
    // Try cache first
    if cached, err := p.getFromCache(ctx, cacheKey, &TournamentLeaderboard{}); err == nil {
        p.logger.Debug("cache hit", zap.String("key", cacheKey))
        return cached.(*TournamentLeaderboard), nil
    }
    
    // Rate limiting
    if err := p.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limiter: %w", err)
    }
    
    // Build request with context
    url := fmt.Sprintf("%s/pga/leaderboard?event=%s", p.baseURL, tournamentID)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }
    
    // Add headers
    req.Header.Set("User-Agent", "DFS-Optimizer/1.0")
    req.Header.Set("Accept", "application/json")
    
    // Execute request with retry logic
    resp, err := p.executeWithRetry(ctx, req, 3)
    if err != nil {
        return nil, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()
    
    // Parse response
    var leaderboard TournamentLeaderboard
    if err := json.NewDecoder(resp.Body).Decode(&leaderboard); err != nil {
        return nil, fmt.Errorf("decoding response: %w", err)
    }
    
    // Cache results
    cacheDuration := p.getCacheDuration(leaderboard.Status)
    if err := p.setCache(ctx, cacheKey, &leaderboard, cacheDuration); err != nil {
        p.logger.Warn("cache set failed", zap.Error(err))
    }
    
    // Send webhook if tournament is live
    if leaderboard.Status == "in_progress" && p.webhookURL != "" {
        go p.sendWebhook(ctx, "leaderboard_update", &leaderboard)
    }
    
    return &leaderboard, nil
}

// executeWithRetry implements exponential backoff
func (p *ESPNGolfProvider) executeWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        resp, err := p.client.Do(req)
        if err != nil {
            lastErr = err
            backoff := time.Duration(1<<uint(i)) * time.Second
            
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(backoff):
                continue
            }
        }
        
        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            return resp, nil
        }
        
        // Handle specific status codes
        switch resp.StatusCode {
        case http.StatusTooManyRequests:
            if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
                if seconds, err := strconv.Atoi(retryAfter); err == nil {
                    time.Sleep(time.Duration(seconds) * time.Second)
                    continue
                }
            }
        case http.StatusServiceUnavailable, http.StatusBadGateway:
            // Retry on server errors
            time.Sleep(time.Duration(1<<uint(i)) * time.Second)
            continue
        default:
            // Don't retry on client errors
            body, _ := ioutil.ReadAll(resp.Body)
            resp.Body.Close()
            return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
        }
    }
    
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Batch operations for efficiency
func (p *ESPNGolfProvider) GetMultiplePlayerStats(ctx context.Context, playerIDs []string) (map[string]*PlayerStats, error) {
    results := make(map[string]*PlayerStats)
    mu := sync.Mutex{}
    
    // Use worker pool for concurrent fetching
    workerPool := make(chan struct{}, 10) // Max 10 concurrent requests
    errChan := make(chan error, len(playerIDs))
    wg := sync.WaitGroup{}
    
    for _, playerID := range playerIDs {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            
            workerPool <- struct{}{} // Acquire worker
            defer func() { <-workerPool }() // Release worker
            
            stats, err := p.GetPlayerStats(ctx, id)
            if err != nil {
                errChan <- fmt.Errorf("player %s: %w", id, err)
                return
            }
            
            mu.Lock()
            results[id] = stats
            mu.Unlock()
        }(playerID)
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return results, fmt.Errorf("partial failures: %v", errs)
    }
    
    return results, nil
}

// Cache helpers with proper error handling
func (p *ESPNGolfProvider) getCacheDuration(status string) time.Duration {
    switch status {
    case "in_progress":
        return 30 * time.Second // Live tournaments update frequently
    case "completed":
        return 24 * time.Hour // Completed tournaments are static
    default:
        return 5 * time.Minute // Scheduled tournaments
    }
}

func (p *ESPNGolfProvider) getFromCache(ctx context.Context, key string, dest interface{}) (interface{}, error) {
    if p.cache == nil {
        return nil, errors.New("cache not configured")
    }
    
    data, err := p.cache.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    
    if err := json.Unmarshal([]byte(data), dest); err != nil {
        return nil, err
    }
    
    return dest, nil
}
```

### 1.4 Golf Service Layer with Business Logic
```go
// File: backend/internal/services/golf_service.go
package services

import (
    "context"
    "database/sql"
    "fmt"
    "math"
    "sync"
    
    "github.com/google/uuid"
    "go.uber.org/zap"
    "gorm.io/gorm"
    
    "github.com/username/dfs-optimizer/internal/models"
    "github.com/username/dfs-optimizer/internal/providers"
)

// GolfService encapsulates all golf business logic
type GolfService struct {
    db            *gorm.DB
    provider      providers.GolfProvider
    projections   *GolfProjectionService
    logger        *zap.Logger
    weatherSvc    *WeatherService
}

// NewGolfService with dependency injection
func NewGolfService(
    db *gorm.DB,
    provider providers.GolfProvider,
    projections *GolfProjectionService,
    logger *zap.Logger,
    weatherSvc *WeatherService,
) *GolfService {
    return &GolfService{
        db:          db,
        provider:    provider,
        projections: projections,
        logger:      logger,
        weatherSvc:  weatherSvc,
    }
}

// SyncTournamentData with transaction support
func (s *GolfService) SyncTournamentData(ctx context.Context, tournamentID string) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // Fetch tournament data
        leaderboard, err := s.provider.GetTournamentLeaderboard(ctx, tournamentID)
        if err != nil {
            return fmt.Errorf("fetching leaderboard: %w", err)
        }
        
        // Update tournament
        tournament := &models.GolfTournament{
            ExternalID:   leaderboard.ID,
            Name:         leaderboard.Name,
            Status:       models.TournamentStatus(leaderboard.Status),
            CurrentRound: leaderboard.CurrentRound,
            CutLine:      leaderboard.CutLine,
        }
        
        if err := tx.Where("external_id = ?", tournamentID).
            Assign(tournament).
            FirstOrCreate(&tournament).Error; err != nil {
            return fmt.Errorf("updating tournament: %w", err)
        }
        
        // Batch update player entries
        if err := s.updatePlayerEntries(ctx, tx, tournament.ID, leaderboard.Players); err != nil {
            return fmt.Errorf("updating player entries: %w", err)
        }
        
        // Update projections if tournament is active
        if tournament.Status == models.TournamentInProgress {
            go s.updateLiveProjections(context.Background(), tournament.ID)
        }
        
        return nil
    })
}

// GetOptimizedLineups with advanced filtering
func (s *GolfService) GetOptimizedLineups(ctx context.Context, req *OptimizationRequest) ([]*models.Lineup, error) {
    // Get eligible players with projections
    players, err := s.getEligiblePlayers(ctx, req.TournamentID, req.MinCutProbability)
    if err != nil {
        return nil, fmt.Errorf("getting eligible players: %w", err)
    }
    
    // Generate projections with correlations
    projections, correlations, err := s.projections.GenerateProjections(ctx, players, req.TournamentID)
    if err != nil {
        return nil, fmt.Errorf("generating projections: %w", err)
    }
    
    // Apply advanced filters
    filteredPlayers := s.applyAdvancedFilters(players, projections, req)
    
    // Run optimization
    optimizer := NewGolfOptimizer(s.logger)
    lineups, err := optimizer.Optimize(ctx, &OptimizerInput{
        Players:              filteredPlayers,
        Projections:         projections,
        Correlations:        correlations,
        NumLineups:          req.NumLineups,
        UniqueMultiplier:    req.UniqueMultiplier,
        Platform:            req.Platform,
        LockedPlayers:       req.LockedPlayers,
        ExcludedPlayers:     req.ExcludedPlayers,
    })
    
    if err != nil {
        return nil, fmt.Errorf("optimization failed: %w", err)
    }
    
    // Post-process lineups
    return s.postProcessLineups(ctx, lineups, req), nil
}

// getEligiblePlayers with efficient queries
func (s *GolfService) getEligiblePlayers(ctx context.Context, tournamentID uuid.UUID, minCutProb float64) ([]*models.Player, error) {
    var entries []models.GolfPlayerEntry
    
    // Optimized query with selective loading
    err := s.db.WithContext(ctx).
        Preload("Player", func(db *gorm.DB) *gorm.DB {
            return db.Select("id", "name", "external_id", "metadata")
        }).
        Where("tournament_id = ? AND status IN ?", 
            tournamentID, 
            []models.PlayerEntryStatus{models.EntryStatusEntered, models.EntryStatusActive}).
        Find(&entries).Error
        
    if err != nil {
        return nil, err
    }
    
    // Convert to players with filtering
    players := make([]*models.Player, 0, len(entries))
    for _, entry := range entries {
        if entry.Player != nil {
            // Calculate cut probability if needed
            if minCutProb > 0 {
                cutProb := s.projections.CalculateCutProbability(ctx, entry)
                if cutProb < minCutProb {
                    continue
                }
            }
            
            // Enrich player with golf-specific data
            entry.Player.Metadata["tournamentEntry"] = entry
            players = append(players, entry.Player)
        }
    }
    
    return players, nil
}

// Real-time update handling
func (s *GolfService) HandleLiveUpdate(ctx context.Context, update *LiveUpdate) error {
    s.logger.Info("processing live update", 
        zap.String("tournament", update.TournamentID),
        zap.String("type", update.Type))
    
    switch update.Type {
    case "score_update":
        return s.handleScoreUpdate(ctx, update)
    case "cut_update":
        return s.handleCutUpdate(ctx, update)
    case "weather_update":
        return s.handleWeatherUpdate(ctx, update)
    default:
        return fmt.Errorf("unknown update type: %s", update.Type)
    }
}

// WebSocket broadcast for real-time updates
func (s *GolfService) BroadcastUpdate(ctx context.Context, updateType string, data interface{}) {
    update := map[string]interface{}{
        "type":      updateType,
        "data":      data,
        "timestamp": time.Now().Unix(),
    }
    
    // Broadcast to WebSocket hub (implemented in API layer)
    if s.wsHub != nil {
        s.wsHub.Broadcast(ctx, "golf_update", update)
    }
}
```

## Phase 2: Frontend Implementation with React Best Practices

### 2.1 Redux Store Setup for Golf
```typescript
// File: frontend/src/store/slices/golfSlice.ts
import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { golfAPI } from '../../services/api/golf';
import { GolfTournament, GolfPlayer, GolfLineup } from '../../types/golf';

interface GolfState {
    tournaments: GolfTournament[];
    activeTournament: GolfTournament | null;
    players: GolfPlayer[];
    lineups: GolfLineup[];
    optimizationStatus: 'idle' | 'loading' | 'succeeded' | 'failed';
    optimizationProgress: number;
    filters: {
        minCutProbability: number;
        maxSalary: number;
        minProjectedPoints: number;
        excludeWithdrawn: boolean;
    };
    correlations: Record<string, Record<string, number>>;
    liveUpdates: boolean;
    error: string | null;
}

const initialState: GolfState = {
    tournaments: [],
    activeTournament: null,
    players: [],
    lineups: [],
    optimizationStatus: 'idle',
    optimizationProgress: 0,
    filters: {
        minCutProbability: 0.5,
        maxSalary: 50000,
        minProjectedPoints: 0,
        excludeWithdrawn: true,
    },
    correlations: {},
    liveUpdates: false,
    error: null,
};

// Async thunks with proper error handling
export const fetchTournaments = createAsyncThunk(
    'golf/fetchTournaments',
    async (_, { rejectWithValue }) => {
        try {
            const response = await golfAPI.getTournaments();
            return response.data;
        } catch (error) {
            return rejectWithValue(error.response?.data?.message || 'Failed to fetch tournaments');
        }
    }
);

export const optimizeLineups = createAsyncThunk(
    'golf/optimizeLineups',
    async (params: OptimizationParams, { dispatch, rejectWithValue }) => {
        try {
            // Start WebSocket connection for progress updates
            const ws = new WebSocket(`${process.env.REACT_APP_WS_URL}/golf/optimize`);
            
            ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                if (data.type === 'progress') {
                    dispatch(updateOptimizationProgress(data.progress));
                }
            };
            
            const response = await golfAPI.optimizeLineups(params);
            ws.close();
            
            return response.data;
        } catch (error) {
            return rejectWithValue(error.response?.data?.message || 'Optimization failed');
        }
    }
);

// Slice with reducers
const golfSlice = createSlice({
    name: 'golf',
    initialState,
    reducers: {
        setActiveTournament: (state, action: PayloadAction<GolfTournament>) => {
            state.activeTournament = action.payload;
        },
        updateFilters: (state, action: PayloadAction<Partial<GolfState['filters']>>) => {
            state.filters = { ...state.filters, ...action.payload };
        },
        updateOptimizationProgress: (state, action: PayloadAction<number>) => {
            state.optimizationProgress = action.payload;
        },
        toggleLiveUpdates: (state) => {
            state.liveUpdates = !state.liveUpdates;
        },
        handleLiveUpdate: (state, action: PayloadAction<LiveUpdate>) => {
            const { type, data } = action.payload;
            
            switch (type) {
                case 'score_update':
                    // Update player scores in real-time
                    const playerIndex = state.players.findIndex(p => p.id === data.playerId);
                    if (playerIndex !== -1) {
                        state.players[playerIndex] = {
                            ...state.players[playerIndex],
                            ...data.updates,
                        };
                    }
                    break;
                    
                case 'cut_update':
                    // Update cut line and player statuses
                    if (state.activeTournament) {
                        state.activeTournament.cutLine = data.cutLine;
                    }
                    state.players = state.players.map(player => ({
                        ...player,
                        madecut: player.totalScore <= data.cutLine,
                    }));
                    break;
            }
        },
    },
    extraReducers: (builder) => {
        builder
            // Tournaments
            .addCase(fetchTournaments.fulfilled, (state, action) => {
                state.tournaments = action.payload;
                state.error = null;
            })
            .addCase(fetchTournaments.rejected, (state, action) => {
                state.error = action.payload as string;
            })
            // Optimization
            .addCase(optimizeLineups.pending, (state) => {
                state.optimizationStatus = 'loading';
                state.optimizationProgress = 0;
                state.error = null;
            })
            .addCase(optimizeLineups.fulfilled, (state, action) => {
                state.optimizationStatus = 'succeeded';
                state.lineups = action.payload.lineups;
                state.correlations = action.payload.correlations;
            })
            .addCase(optimizeLineups.rejected, (state, action) => {
                state.optimizationStatus = 'failed';
                state.error = action.payload as string;
            });
    },
});

export const { 
    setActiveTournament, 
    updateFilters, 
    updateOptimizationProgress,
    toggleLiveUpdates,
    handleLiveUpdate 
} = golfSlice.actions;

export default golfSlice.reducer;
```

### 2.2 Advanced Golf Components with Performance Optimization
```typescript
// File: frontend/src/components/golf/GolfDashboard.tsx
import React, { useEffect, useMemo, useCallback, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useQuery, useQueryClient } from 'react-query';
import { motion, AnimatePresence } from 'framer-motion';
import { 
    DndContext, 
    DragEndEvent, 
    useSensor, 
    useSensors, 
    PointerSensor,
    KeyboardSensor,
} from '@dnd-kit/core';
import { 
    SortableContext, 
    verticalListSortingStrategy,
    arrayMove,
} from '@dnd-kit/sortable';
import { restrictToVerticalAxis } from '@dnd-kit/modifiers';

import { RootState } from '../../store';
import { fetchTournaments, setActiveTournament, optimizeLineups } from '../../store/slices/golfSlice';
import { GolfPlayerPool } from './GolfPlayerPool';
import { GolfLineupBuilder } from './GolfLineupBuilder';
import { GolfOptimizationControls } from './GolfOptimizationControls';
import { TournamentSelector } from './TournamentSelector';
import { useWebSocket } from '../../hooks/useWebSocket';
import { useDebounce } from '../../hooks/useDebounce';
import { ErrorBoundary } from '../common/ErrorBoundary';
import { LoadingSpinner } from '../common/LoadingSpinner';

export const GolfDashboard: React.FC = () => {
    const dispatch = useDispatch();
    const queryClient = useQueryClient();
    
    // Redux state
    const { 
        tournaments, 
        activeTournament, 
        players,
        lineups,
        optimizationStatus,
        filters,
        liveUpdates,
    } = useSelector((state: RootState) => state.golf);
    
    // Local state
    const [selectedLineup, setSelectedLineup] = useState<number>(0);
    const [searchTerm, setSearchTerm] = useState('');
    const [sortBy, setSortBy] = useState<'projectedPoints' | 'salary' | 'ownership'>('projectedPoints');
    
    // Debounced search
    const debouncedSearch = useDebounce(searchTerm, 300);
    
    // DnD sensors
    const sensors = useSensors(
        useSensor(PointerSensor, {
            activationConstraint: {
                distance: 8,
            },
        }),
        useSensor(KeyboardSensor)
    );
    
    // WebSocket connection for live updates
    const { sendMessage, lastMessage } = useWebSocket(
        liveUpdates && activeTournament ? `/golf/tournament/${activeTournament.id}` : null,
        {
            onOpen: () => console.log('Golf WebSocket connected'),
            onClose: () => console.log('Golf WebSocket disconnected'),
            shouldReconnect: () => liveUpdates,
            reconnectInterval: 3000,
        }
    );
    
    // React Query for tournament data
    const { data: tournamentData, isLoading } = useQuery(
        ['golf-tournament', activeTournament?.id],
        () => golfAPI.getTournamentDetails(activeTournament!.id),
        {
            enabled: !!activeTournament,
            refetchInterval: liveUpdates ? 30000 : false, // Refetch every 30s if live
            staleTime: 60000,
            cacheTime: 300000,
        }
    );
    
    // Process WebSocket messages
    useEffect(() => {
        if (lastMessage) {
            try {
                const update = JSON.parse(lastMessage.data);
                dispatch(handleLiveUpdate(update));
                
                // Invalidate relevant queries
                if (update.type === 'score_update') {
                    queryClient.invalidateQueries(['golf-player', update.data.playerId]);
                }
            } catch (error) {
                console.error('Failed to process WebSocket message:', error);
            }
        }
    }, [lastMessage, dispatch, queryClient]);
    
    // Memoized filtered and sorted players
    const filteredPlayers = useMemo(() => {
        let filtered = players;
        
        // Apply search filter
        if (debouncedSearch) {
            filtered = filtered.filter(player => 
                player.name.toLowerCase().includes(debouncedSearch.toLowerCase())
            );
        }
        
        // Apply other filters
        if (filters.excludeWithdrawn) {
            filtered = filtered.filter(player => player.status !== 'withdrawn');
        }
        
        if (filters.minCutProbability > 0) {
            filtered = filtered.filter(player => player.cutProbability >= filters.minCutProbability);
        }
        
        if (filters.minProjectedPoints > 0) {
            filtered = filtered.filter(player => player.projectedPoints >= filters.minProjectedPoints);
        }
        
        // Sort
        return [...filtered].sort((a, b) => {
            switch (sortBy) {
                case 'projectedPoints':
                    return b.projectedPoints - a.projectedPoints;
                case 'salary':
                    return b.salary - a.salary;
                case 'ownership':
                    return b.ownership - a.ownership;
                default:
                    return 0;
            }
        });
    }, [players, debouncedSearch, filters, sortBy]);
    
    // Optimization handler
    const handleOptimize = useCallback(async () => {
        if (!activeTournament) return;
        
        const params = {
            tournamentId: activeTournament.id,
            ...filters,
            numLineups: 20,
            platform: 'draftkings',
        };
        
        dispatch(optimizeLineups(params));
    }, [activeTournament, filters, dispatch]);
    
    // Drag end handler
    const handleDragEnd = useCallback((event: DragEndEvent) => {
        const { active, over } = event;
        
        if (active.id !== over?.id) {
            // Handle player reordering in lineup
            // Implementation depends on lineup structure
        }
    }, []);
    
    if (isLoading) {
        return <LoadingSpinner size="large" />;
    }
    
    return (
        <ErrorBoundary>
            <DndContext 
                sensors={sensors} 
                onDragEnd={handleDragEnd}
                modifiers={[restrictToVerticalAxis]}
            >
                <div className="min-h-screen bg-gray-50">
                    {/* Header */}
                    <div className="bg-white shadow-sm border-b">
                        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                            <div className="flex justify-between items-center py-4">
                                <h1 className="text-2xl font-bold text-gray-900">
                                    Golf DFS Optimizer
                                </h1>
                                <TournamentSelector
                                    tournaments={tournaments}
                                    selected={activeTournament}
                                    onChange={(tournament) => dispatch(setActiveTournament(tournament))}
                                />
                            </div>
                        </div>
                    </div>
                    
                    {/* Main Content */}
                    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                            {/* Player Pool */}
                            <div className="lg:col-span-1">
                                <GolfPlayerPool
                                    players={filteredPlayers}
                                    searchTerm={searchTerm}
                                    onSearchChange={setSearchTerm}
                                    sortBy={sortBy}
                                    onSortChange={setSortBy}
                                    onPlayerSelect={(player) => {
                                        // Add to current lineup
                                    }}
                                />
                            </div>
                            
                            {/* Lineup Builder & Controls */}
                            <div className="lg:col-span-2 space-y-6">
                                <GolfOptimizationControls
                                    filters={filters}
                                    onFiltersChange={(newFilters) => dispatch(updateFilters(newFilters))}
                                    onOptimize={handleOptimize}
                                    isOptimizing={optimizationStatus === 'loading'}
                                    progress={optimizationProgress}
                                />
                                
                                <AnimatePresence mode="wait">
                                    {lineups.length > 0 ? (
                                        <motion.div
                                            initial={{ opacity: 0, y: 20 }}
                                            animate={{ opacity: 1, y: 0 }}
                                            exit={{ opacity: 0, y: -20 }}
                                            transition={{ duration: 0.3 }}
                                        >
                                            <GolfLineupBuilder
                                                lineup={lineups[selectedLineup]}
                                                onLineupChange={(index) => setSelectedLineup(index)}
                                                totalLineups={lineups.length}
                                                currentIndex={selectedLineup}
                                            />
                                        </motion.div>
                                    ) : (
                                        <motion.div
                                            initial={{ opacity: 0 }}
                                            animate={{ opacity: 1 }}
                                            className="bg-white rounded-lg shadow p-8 text-center"
                                        >
                                            <p className="text-gray-500">
                                                No lineups generated yet. Configure your settings and click Optimize.
                                            </p>
                                        </motion.div>
                                    )}
                                </AnimatePresence>
                            </div>
                        </div>
                    </div>
                </div>
            </DndContext>
        </ErrorBoundary>
    );
};
```

### 2.3 Custom Hooks for Golf Features
```typescript
// File: frontend/src/hooks/useGolfData.ts
import { useQuery, useMutation, useQueryClient } from 'react-query';
import { useCallback, useEffect } from 'react';
import { golfAPI } from '../services/api/golf';
import { useWebSocket } from './useWebSocket';
import { GolfPlayer, GolfTournament } from '../types/golf';

interface UseGolfDataOptions {
    tournamentId?: string;
    enableLiveUpdates?: boolean;
    refetchInterval?: number;
}

export const useGolfData = ({
    tournamentId,
    enableLiveUpdates = false,
    refetchInterval = 60000,
}: UseGolfDataOptions) => {
    const queryClient = useQueryClient();
    
    // Tournament data query
    const tournamentQuery = useQuery(
        ['golf-tournament', tournamentId],
        () => golfAPI.getTournament(tournamentId!),
        {
            enabled: !!tournamentId,
            staleTime: 30000,
            cacheTime: 300000,
            refetchInterval: enableLiveUpdates ? refetchInterval : false,
        }
    );
    
    // Players query with optimistic updates
    const playersQuery = useQuery(
        ['golf-players', tournamentId],
        () => golfAPI.getTournamentPlayers(tournamentId!),
        {
            enabled: !!tournamentId,
            staleTime: 30000,
            select: (data) => {
                // Transform and enrich player data
                return data.map(player => ({
                    ...player,
                    value: calculatePlayerValue(player),
                    trending: calculateTrending(player),
                }));
            },
        }
    );
    
    // WebSocket for live updates
    const { lastMessage } = useWebSocket(
        enableLiveUpdates && tournamentId ? `/golf/live/${tournamentId}` : null,
        {
            shouldReconnect: () => enableLiveUpdates,
            reconnectInterval: 3000,
        }
    );
    
    // Process live updates
    useEffect(() => {
        if (!lastMessage) return;
        
        try {
            const update = JSON.parse(lastMessage.data);
            
            switch (update.type) {
                case 'score_update':
                    // Optimistically update player data
                    queryClient.setQueryData<GolfPlayer[]>(
                        ['golf-players', tournamentId],
                        (old) => {
                            if (!old) return old;
                            
                            return old.map(player => 
                                player.id === update.playerId
                                    ? { ...player, ...update.data }
                                    : player
                            );
                        }
                    );
                    break;
                    
                case 'tournament_update':
                    // Update tournament data
                    queryClient.setQueryData<GolfTournament>(
                        ['golf-tournament', tournamentId],
                        (old) => old ? { ...old, ...update.data } : old
                    );
                    break;
            }
        } catch (error) {
            console.error('Failed to process live update:', error);
        }
    }, [lastMessage, queryClient, tournamentId]);
    
    // Prefetch related data
    const prefetchPlayerHistory = useCallback(async (playerId: string) => {
        await queryClient.prefetchQuery(
            ['golf-player-history', playerId],
            () => golfAPI.getPlayerHistory(playerId),
            { staleTime: 3600000 } // 1 hour
        );
    }, [queryClient]);
    
    // Mutations with optimistic updates
    const updatePlayerStatus = useMutation(
        (data: { playerId: string; status: string }) => 
            golfAPI.updatePlayerStatus(data.playerId, data.status),
        {
            onMutate: async ({ playerId, status }) => {
                // Cancel outgoing refetches
                await queryClient.cancelQueries(['golf-players', tournamentId]);
                
                // Snapshot previous value
                const previousPlayers = queryClient.getQueryData<GolfPlayer[]>(
                    ['golf-players', tournamentId]
                );
                
                // Optimistically update
                queryClient.setQueryData<GolfPlayer[]>(
                    ['golf-players', tournamentId],
                    (old) => {
                        if (!old) return old;
                        return old.map(player =>
                            player.id === playerId
                                ? { ...player, status }
                                : player
                        );
                    }
                );
                
                return { previousPlayers };
            },
            onError: (err, variables, context) => {
                // Rollback on error
                if (context?.previousPlayers) {
                    queryClient.setQueryData(
                        ['golf-players', tournamentId],
                        context.previousPlayers
                    );
                }
            },
            onSettled: () => {
                // Refetch after mutation
                queryClient.invalidateQueries(['golf-players', tournamentId]);
            },
        }
    );
    
    return {
        tournament: tournamentQuery.data,
        players: playersQuery.data || [],
        isLoading: tournamentQuery.isLoading || playersQuery.isLoading,
        error: tournamentQuery.error || playersQuery.error,
        refetch: () => {
            tournamentQuery.refetch();
            playersQuery.refetch();
        },
        updatePlayerStatus,
        prefetchPlayerHistory,
    };
};

// Helper functions
const calculatePlayerValue = (player: GolfPlayer): number => {
    // Value calculation based on projected points per dollar
    return player.projectedPoints / (player.salary / 1000);
};

const calculateTrending = (player: GolfPlayer): 'up' | 'down' | 'stable' => {
    // Trend calculation based on recent performance
    if (!player.recentScores || player.recentScores.length < 2) return 'stable';
    
    const recent = player.recentScores.slice(-5);
    const avg = recent.reduce((a, b) => a + b, 0) / recent.length;
    const lastScore = recent[recent.length - 1];
    
    if (lastScore < avg - 1) return 'up';
    if (lastScore > avg + 1) return 'down';
    return 'stable';
};
```

## Phase 3: API Integration & Testing

### 3.1 Golf API Routes with Middleware
```go
// File: backend/internal/api/golf_routes.go
package api

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"
    
    "github.com/username/dfs-optimizer/internal/middleware"
    "github.com/username/dfs-optimizer/internal/services"
)

// GolfHandler with dependency injection
type GolfHandler struct {
    golfService *services.GolfService
    logger      *zap.Logger
}

// RegisterGolfRoutes sets up all golf endpoints
func RegisterGolfRoutes(r *gin.RouterGroup, golfService *services.GolfService, logger *zap.Logger) {
    handler := &GolfHandler{
        golfService: golfService,
        logger:      logger,
    }
    
    golf := r.Group("/golf")
    golf.Use(middleware.RateLimit(100)) // 100 requests per minute
    
    // Tournament endpoints
    golf.GET("/tournaments", handler.GetTournaments)
    golf.GET("/tournaments/:id", handler.GetTournament)
    golf.GET("/tournaments/:id/leaderboard", handler.GetLeaderboard)
    golf.POST("/tournaments/:id/sync", middleware.RequireRole("admin"), handler.SyncTournament)
    
    // Player endpoints
    golf.GET("/players", handler.GetPlayers)
    golf.GET("/players/:id", handler.GetPlayer)
    golf.GET("/players/:id/history", handler.GetPlayerHistory)
    
    // Optimization endpoints
    golf.POST("/optimize", middleware.RequireAuth(), handler.OptimizeLineups)
    golf.GET("/optimize/:jobId", handler.GetOptimizationStatus)
    
    // WebSocket endpoint
    golf.GET("/ws/:tournamentId", handler.HandleWebSocket)
}

// GetTournaments with pagination and filtering
func (h *GolfHandler) GetTournaments(c *gin.Context) {
    // Parse query parameters
    status := c.DefaultQuery("status", "")
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
    
    // Validate parameters
    if limit > 100 {
        limit = 100
    }
    
    tournaments, total, err := h.golfService.GetTournaments(c.Request.Context(), &services.TournamentFilter{
        Status: status,
        Limit:  limit,
        Offset: offset,
    })
    
    if err != nil {
        h.logger.Error("failed to get tournaments", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to fetch tournaments",
        })
        return
    }
    
    // Add pagination headers
    c.Header("X-Total-Count", strconv.Itoa(total))
    c.Header("X-Limit", strconv.Itoa(limit))
    c.Header("X-Offset", strconv.Itoa(offset))
    
    c.JSON(http.StatusOK, gin.H{
        "tournaments": tournaments,
        "total":       total,
        "limit":       limit,
        "offset":      offset,
    })
}

// OptimizeLineups with job queue
func (h *GolfHandler) OptimizeLineups(c *gin.Context) {
    var req services.OptimizationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request body",
            "details": err.Error(),
        })
        return
    }
    
    // Validate request
    if err := req.Validate(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Validation failed",
            "details": err.Error(),
        })
        return
    }
    
    // Get user from context
    userID, _ := c.Get("userID")
    req.UserID = userID.(uuid.UUID)
    
    // Submit optimization job
    jobID, err := h.golfService.SubmitOptimizationJob(c.Request.Context(), &req)
    if err != nil {
        h.logger.Error("failed to submit optimization", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to start optimization",
        })
        return
    }
    
    c.JSON(http.StatusAccepted, gin.H{
        "jobId": jobID,
        "status": "processing",
        "message": "Optimization started. Use the jobId to check status.",
    })
}

// HandleWebSocket for real-time updates
func (h *GolfHandler) HandleWebSocket(c *gin.Context) {
    tournamentID := c.Param("tournamentId")
    
    // Upgrade connection
    conn, err := websocket.Upgrade(c.Writer, c.Request, nil, 1024, 1024)
    if err != nil {
        h.logger.Error("websocket upgrade failed", zap.Error(err))
        return
    }
    defer conn.Close()
    
    // Create client
    client := &WSClient{
        conn:         conn,
        send:         make(chan []byte, 256),
        tournamentID: tournamentID,
        hub:          h.wsHub,
    }
    
    // Register client
    h.wsHub.Register(client)
    defer h.wsHub.Unregister(client)
    
    // Start goroutines
    go client.WritePump()
    go client.ReadPump()
    
    // Send initial data
    if err := h.sendInitialData(c.Request.Context(), client); err != nil {
        h.logger.Error("failed to send initial data", zap.Error(err))
    }
}

// Error response helper
func (h *GolfHandler) errorResponse(c *gin.Context, code int, message string, err error) {
    h.logger.Error(message, zap.Error(err))
    
    response := gin.H{
        "error": message,
        "code":  code,
    }
    
    // Add request ID for debugging
    if requestID, exists := c.Get("requestID"); exists {
        response["requestId"] = requestID
    }
    
    c.JSON(code, response)
}
```

### 3.2 Comprehensive Testing Suite
```go
// File: backend/internal/api/golf_handler_test.go
package api

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"
    "go.uber.org/zap"
    
    "github.com/username/dfs-optimizer/internal/models"
    "github.com/username/dfs-optimizer/internal/services"
)

// Test suite setup
type GolfHandlerTestSuite struct {
    suite.Suite
    handler     *GolfHandler
    mockService *MockGolfService
    router      *gin.Engine
    logger      *zap.Logger
}

func (suite *GolfHandlerTestSuite) SetupTest() {
    gin.SetMode(gin.TestMode)
    
    suite.logger = zap.NewNop()
    suite.mockService = new(MockGolfService)
    suite.handler = &GolfHandler{
        golfService: suite.mockService,
        logger:      suite.logger,
    }
    
    suite.router = gin.New()
    api := suite.router.Group("/api")
    RegisterGolfRoutes(api, suite.mockService, suite.logger)
}

// Test GetTournaments endpoint
func (suite *GolfHandlerTestSuite) TestGetTournaments() {
    // Arrange
    expectedTournaments := []models.GolfTournament{
        {
            ID:         uuid.New(),
            Name:       "Masters Tournament",
            Status:     models.TournamentScheduled,
            StartDate:  time.Now().AddDate(0, 0, 7),
            CourseName: "Augusta National",
        },
        {
            ID:         uuid.New(),
            Name:       "US Open",
            Status:     models.TournamentInProgress,
            StartDate:  time.Now(),
            CourseName: "Pebble Beach",
        },
    }
    
    suite.mockService.On("GetTournaments", mock.Anything, mock.Anything).
        Return(expectedTournaments, 2, nil)
    
    // Act
    req := httptest.NewRequest("GET", "/api/golf/tournaments?status=in_progress&limit=10", nil)
    w := httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(suite.T(), http.StatusOK, w.Code)
    assert.Equal(suite.T(), "2", w.Header().Get("X-Total-Count"))
    
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(suite.T(), err)
    
    tournaments := response["tournaments"].([]interface{})
    assert.Len(suite.T(), tournaments, 2)
    
    suite.mockService.AssertExpectations(suite.T())
}

// Test OptimizeLineups with validation
func (suite *GolfHandlerTestSuite) TestOptimizeLineups() {
    // Test cases
    testCases := []struct {
        name          string
        request       services.OptimizationRequest
        expectedCode  int
        setupMock     func()
    }{
        {
            name: "Valid request",
            request: services.OptimizationRequest{
                TournamentID:      uuid.New(),
                NumLineups:        20,
                Platform:          "draftkings",
                MinCutProbability: 0.6,
            },
            expectedCode: http.StatusAccepted,
            setupMock: func() {
                suite.mockService.On("SubmitOptimizationJob", mock.Anything, mock.Anything).
                    Return("job-123", nil)
            },
        },
        {
            name: "Invalid platform",
            request: services.OptimizationRequest{
                TournamentID: uuid.New(),
                NumLineups:   20,
                Platform:     "invalid",
            },
            expectedCode: http.StatusBadRequest,
            setupMock:    func() {},
        },
        {
            name: "Too many lineups",
            request: services.OptimizationRequest{
                TournamentID: uuid.New(),
                NumLineups:   500, // Over limit
                Platform:     "draftkings",
            },
            expectedCode: http.StatusBadRequest,
            setupMock:    func() {},
        },
    }
    
    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            // Setup
            tc.setupMock()
            
            body, _ := json.Marshal(tc.request)
            req := httptest.NewRequest("POST", "/api/golf/optimize", bytes.NewBuffer(body))
            req.Header.Set("Content-Type", "application/json")
            
            // Add auth context
            ctx := context.WithValue(req.Context(), "userID", uuid.New())
            req = req.WithContext(ctx)
            
            w := httptest.NewRecorder()
            
            // Act
            suite.router.ServeHTTP(w, req)
            
            // Assert
            assert.Equal(suite.T(), tc.expectedCode, w.Code)
            
            if tc.expectedCode == http.StatusAccepted {
                var response map[string]interface{}
                json.Unmarshal(w.Body.Bytes(), &response)
                assert.Equal(suite.T(), "job-123", response["jobId"])
            }
        })
    }
}

// Test WebSocket connection
func (suite *GolfHandlerTestSuite) TestWebSocketConnection() {
    // Create test server
    server := httptest.NewServer(suite.router)
    defer server.Close()
    
    // Convert http:// to ws://
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/golf/ws/tournament-123"
    
    // Connect
    ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    assert.NoError(suite.T(), err)
    defer ws.Close()
    
    // Test ping/pong
    err = ws.WriteMessage(websocket.PingMessage, []byte{})
    assert.NoError(suite.T(), err)
    
    // Read response
    _, _, err = ws.ReadMessage()
    assert.NoError(suite.T(), err)
}

// Run test suite
func TestGolfHandlerSuite(t *testing.T) {
    suite.Run(t, new(GolfHandlerTestSuite))
}

// Integration tests
func TestGolfIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup real dependencies
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    provider := setupMockProvider()
    projectionSvc := services.NewGolfProjectionService(db, provider)
    golfService := services.NewGolfService(db, provider, projectionSvc, zap.NewNop(), nil)
    
    // Create test data
    tournament := createTestTournament(t, db)
    players := createTestPlayers(t, db, 100)
    createTestEntries(t, db, tournament, players)
    
    // Test optimization flow
    t.Run("Full optimization flow", func(t *testing.T) {
        ctx := context.Background()
        
        // Submit optimization
        req := &services.OptimizationRequest{
            TournamentID:      tournament.ID,
            NumLineups:        10,
            Platform:          "draftkings",
            MinCutProbability: 0.5,
            MaxExposure:       0.5,
        }
        
        lineups, err := golfService.GetOptimizedLineups(ctx, req)
        assert.NoError(t, err)
        assert.Len(t, lineups, 10)
        
        // Validate lineups
        for _, lineup := range lineups {
            assert.Len(t, lineup.Players, 6) // Golf lineups have 6 players
            assert.LessOrEqual(t, lineup.TotalSalary, 50000) // DK salary cap
            
            // Check uniqueness
            playerIDs := make(map[string]bool)
            for _, player := range lineup.Players {
                assert.False(t, playerIDs[player.ID.String()], "Duplicate player in lineup")
                playerIDs[player.ID.String()] = true
            }
        }
    })
}
```

### 3.3 Frontend Testing with React Testing Library
```typescript
// File: frontend/src/components/golf/__tests__/GolfDashboard.test.tsx
import React from 'react';
import { screen, render, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider } from 'react-query';
import { configureStore } from '@reduxjs/toolkit';
import { rest } from 'msw';
import { setupServer } from 'msw/node';

import { GolfDashboard } from '../GolfDashboard';
import golfReducer from '../../../store/slices/golfSlice';
import { mockTournaments, mockPlayers, mockLineups } from '../../../test/mocks/golf';

// Setup MSW server
const server = setupServer(
    rest.get('/api/golf/tournaments', (req, res, ctx) => {
        return res(ctx.json({ tournaments: mockTournaments, total: 2 }));
    }),
    
    rest.get('/api/golf/tournaments/:id', (req, res, ctx) => {
        const { id } = req.params;
        const tournament = mockTournaments.find(t => t.id === id);
        return res(ctx.json(tournament));
    }),
    
    rest.get('/api/golf/tournaments/:id/players', (req, res, ctx) => {
        return res(ctx.json(mockPlayers));
    }),
    
    rest.post('/api/golf/optimize', async (req, res, ctx) => {
        // Simulate optimization delay
        await ctx.delay(1000);
        return res(ctx.json({ jobId: 'job-123' }));
    }),
    
    rest.get('/api/golf/optimize/:jobId', (req, res, ctx) => {
        return res(ctx.json({
            status: 'completed',
            lineups: mockLineups,
            correlations: {},
        }));
    })
);

// Test setup
beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

const renderWithProviders = (component: React.ReactElement) => {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
        },
    });
    
    const store = configureStore({
        reducer: { golf: golfReducer },
    });
    
    return render(
        <QueryClientProvider client={queryClient}>
            <Provider store={store}>
                {component}
            </Provider>
        </QueryClientProvider>
    );
};

describe('GolfDashboard', () => {
    it('renders and loads tournaments', async () => {
        renderWithProviders(<GolfDashboard />);
        
        // Check loading state
        expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
        
        // Wait for tournaments to load
        await waitFor(() => {
            expect(screen.getByText('Masters Tournament')).toBeInTheDocument();
        });
        
        // Check tournament selector
        const selector = screen.getByLabelText('Select Tournament');
        expect(selector).toBeInTheDocument();
    });
    
    it('handles tournament selection and loads players', async () => {
        const user = userEvent.setup();
        renderWithProviders(<GolfDashboard />);
        
        // Wait for tournaments
        await waitFor(() => {
            expect(screen.getByText('Masters Tournament')).toBeInTheDocument();
        });
        
        // Select tournament
        const selector = screen.getByLabelText('Select Tournament');
        await user.selectOptions(selector, 'masters-2024');
        
        // Wait for players to load
        await waitFor(() => {
            expect(screen.getByText('Tiger Woods')).toBeInTheDocument();
            expect(screen.getByText('Rory McIlroy')).toBeInTheDocument();
        });
    });
    
    it('filters players correctly', async () => {
        const user = userEvent.setup();
        renderWithProviders(<GolfDashboard />);
        
        // Setup: Select tournament and wait for players
        await waitFor(() => screen.getByText('Masters Tournament'));
        await user.selectOptions(screen.getByLabelText('Select Tournament'), 'masters-2024');
        await waitFor(() => screen.getByText('Tiger Woods'));
        
        // Test search filter
        const searchInput = screen.getByPlaceholderText('Search players...');
        await user.type(searchInput, 'Tiger');
        
        await waitFor(() => {
            expect(screen.getByText('Tiger Woods')).toBeInTheDocument();
            expect(screen.queryByText('Rory McIlroy')).not.toBeInTheDocument();
        });
        
        // Test cut probability filter
        const cutSlider = screen.getByLabelText('Min Cut Probability');
        fireEvent.change(cutSlider, { target: { value: '0.8' } });
        
        await waitFor(() => {
            const players = screen.getAllByTestId('player-card');
            players.forEach(player => {
                const cutProb = player.querySelector('[data-testid="cut-probability"]');
                expect(parseFloat(cutProb!.textContent!)).toBeGreaterThanOrEqual(0.8);
            });
        });
    });
    
    it('handles lineup optimization', async () => {
        const user = userEvent.setup();
        renderWithProviders(<GolfDashboard />);
        
        // Setup
        await waitFor(() => screen.getByText('Masters Tournament'));
        await user.selectOptions(screen.getByLabelText('Select Tournament'), 'masters-2024');
        await waitFor(() => screen.getByText('Tiger Woods'));
        
        // Configure optimization
        const numLineupsInput = screen.getByLabelText('Number of Lineups');
        await user.clear(numLineupsInput);
        await user.type(numLineupsInput, '20');
        
        // Start optimization
        const optimizeButton = screen.getByText('Optimize Lineups');
        await user.click(optimizeButton);
        
        // Check loading state
        expect(screen.getByText('Optimizing...')).toBeInTheDocument();
        expect(screen.getByRole('progressbar')).toBeInTheDocument();
        
        // Wait for completion
        await waitFor(() => {
            expect(screen.getByText('Lineup 1 of 20')).toBeInTheDocument();
        }, { timeout: 3000 });
        
        // Verify lineup display
        const lineup = screen.getByTestId('lineup-display');
        const players = lineup.querySelectorAll('[data-testid="lineup-player"]');
        expect(players).toHaveLength(6); // Golf lineups have 6 players
    });
    
    it('handles WebSocket live updates', async () => {
        // Mock WebSocket
        const mockWS = {
            send: jest.fn(),
            close: jest.fn(),
            addEventListener: jest.fn(),
            removeEventListener: jest.fn(),
        };
        
        global.WebSocket = jest.fn(() => mockWS) as any;
        
        renderWithProviders(<GolfDashboard />);
        
        // Enable live updates
        const liveToggle = screen.getByLabelText('Enable Live Updates');
        fireEvent.click(liveToggle);
        
        // Simulate incoming message
        const messageHandler = mockWS.addEventListener.mock.calls.find(
            call => call[0] === 'message'
        )[1];
        
        messageHandler({
            data: JSON.stringify({
                type: 'score_update',
                data: {
                    playerId: 'player-1',
                    updates: {
                        currentScore: -5,
                        thruHoles: 12,
                    },
                },
            }),
        });
        
        await waitFor(() => {
            const player = screen.getByTestId('player-player-1');
            expect(player).toHaveTextContent('-5');
            expect(player).toHaveTextContent('Thru 12');
        });
    });
    
    it('exports lineups correctly', async () => {
        const user = userEvent.setup();
        
        // Mock download
        const mockLink = document.createElement('a');
        const clickSpy = jest.spyOn(mockLink, 'click');
        jest.spyOn(document, 'createElement').mockReturnValue(mockLink);
        
        renderWithProviders(<GolfDashboard />);
        
        // Setup and generate lineups
        await waitFor(() => screen.getByText('Masters Tournament'));
        // ... setup steps ...
        
        // Export lineups
        const exportButton = screen.getByText('Export to DraftKings');
        await user.click(exportButton);
        
        expect(clickSpy).toHaveBeenCalled();
        expect(mockLink.download).toBe('golf_lineups_draftkings.csv');
        expect(mockLink.href).toContain('data:text/csv');
    });
});
```

## Phase 4: Deployment & Monitoring

### 4.1 Docker Configuration
```dockerfile
# File: backend/Dockerfile.golf
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o golf-server cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/golf-server .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./golf-server"]
```

### 4.2 Monitoring & Observability
```go
// File: backend/internal/monitoring/golf_metrics.go
package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    GolfOptimizationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "golf_optimization_duration_seconds",
            Help: "Duration of golf lineup optimizations",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
        },
        []string{"tournament", "platform", "num_lineups"},
    )
    
    GolfAPIRequests = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "golf_api_requests_total",
            Help: "Total number of golf API requests",
        },
        []string{"provider", "endpoint", "status"},
    )
    
    GolfActiveTournaments = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "golf_active_tournaments",
            Help: "Number of active golf tournaments",
        },
    )
    
    GolfWebSocketConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "golf_websocket_connections",
            Help: "Number of active WebSocket connections for golf",
        },
    )
)
```

## Implementation Checklist

### Backend Tasks
- [ ] Create database migration files
- [ ] Implement Golf models with GORM
- [ ] Create ESPN Golf provider with caching
- [ ] Implement projection service
- [ ] Build correlation matrix calculator
- [ ] Create optimization engine
- [ ] Add API routes and handlers
- [ ] Implement WebSocket support
- [ ] Add comprehensive tests
- [ ] Setup monitoring/metrics

### Frontend Tasks
- [ ] Add Golf to Redux store
- [ ] Create Golf dashboard component
- [ ] Build player pool component
- [ ] Implement lineup builder
- [ ] Add optimization controls
- [ ] Create real-time update handlers
- [ ] Implement export functionality
- [ ] Add component tests
- [ ] Ensure responsive design

### Integration Tasks
- [ ] Test full optimization flow
- [ ] Verify WebSocket updates
- [ ] Test error scenarios
- [ ] Performance testing
- [ ] Load testing
- [ ] Security audit
- [ ] Documentation
- [ ] Deployment setup

## Success Metrics
1. **Performance**: Optimization < 30s for 150 lineups
2. **Accuracy**: Projections within 10% of actual
3. **Reliability**: 99.9% uptime for API
4. **User Experience**: < 2s page load time
5. **Real-time**: < 500ms WebSocket latency

This enhanced PRP provides comprehensive implementation details with production-ready code patterns, ensuring the agent can deliver a fully functional golf sport addition that integrates seamlessly with the existing DFS optimizer.