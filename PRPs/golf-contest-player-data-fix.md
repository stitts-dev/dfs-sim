name: "Golf Contest Player Data Fix - Restore Tournament to Contest Player Population Pipeline"
description: |

## Purpose
Restore the broken DFS golf contest player population pipeline to enable core lineup optimization functionality. The system currently creates tournament contests but fails to populate them with players, making DFS optimization impossible.

## Core Principles
1. **Fix Silent Failures**: Identify and repair broken player import processes
2. **Restore Backup Services**: Reactivate sophisticated player import services from .bak files
3. **Multi-Provider Integration**: Combine RapidAPI, DraftKings, and ESPN data sources
4. **Robust Error Handling**: Implement comprehensive logging and rollback mechanisms
5. **Global Rules**: Adhere to all rules in CLAUDE.md

---

## Goal
Fix the tournament sync service to properly populate golf contests with player data including salaries, projections, and metadata to enable DFS lineup optimization.

## Why
- **Core Functionality**: DFS optimization is impossible without player data in contests
- **User Value**: Enables users to create optimized lineups for golf tournaments
- **Data Completeness**: Provides comprehensive player information for decision-making
- **System Reliability**: Eliminates silent failures in the player import pipeline

## What
Restore the player import pipeline by fixing the `importContestPlayers` method, reactivating backup services, and integrating multiple data providers for comprehensive player information.

### Success Criteria
- [ ] Database query shows `player_count > 0` for all contests
- [ ] Player records contain salary data for DraftKings/FanDuel
- [ ] Tournament sync completes without errors
- [ ] Player import process creates proper contest-player links
- [ ] Lineup optimization works with populated contest data
- [ ] Multi-provider data aggregation provides comprehensive stats

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://rapidapi.com/slashgolf/api/live-golf-data
  why: RapidAPI Live Golf Data API Pro plan features and rate limits
  
- url: https://github.com/SeanDrum/Draft-Kings-API-Documentation
  why: DraftKings unofficial API endpoints for contest and salary data
  
- url: https://gist.github.com/SeanDrum/1e6bde260a4735360376d4d46065d46d
  why: JavaScript examples for DraftKings API integration
  
- file: /Users/jstittsworth/fun/services/sports-data-service/internal/services/golf_tournament_sync.go
  why: Main tournament sync service with broken importContestPlayers method
  
- file: /Users/jstittsworth/fun/services/sports-data-service/internal/services/contest_player_importer.go.bak
  why: Sophisticated player import service that needs restoration
  
- file: /Users/jstittsworth/fun/services/sports-data-service/internal/services/player_matching.go.bak
  why: Advanced player matching logic with fuzzy matching
  
- file: /Users/jstittsworth/fun/services/sports-data-service/internal/services/tournament_enricher.go.bak
  why: Tournament data enrichment with projections and stats
  
- file: /Users/jstittsworth/fun/services/sports-data-service/internal/providers/rapidapi_golf.go
  why: RapidAPI client with rate limiting and caching
  
- file: /Users/jstittsworth/fun/services/sports-data-service/internal/models/base.go
  why: Base models for players, contests, and tournaments
  
- file: /Users/jstittsworth/fun/shared/types/common.go
  why: Shared types for player and contest structures
```

### Current Codebase Analysis
**Database Schema:**
- `contests` table: 23 columns, `tournament_id` links to golf tournaments
- `players` table: 22 columns, `contest_id` links to contests (currently NULL)
- `golf_tournaments` table: 20 columns, successfully populated

**Current State:**
- ✅ Tournament sync creates tournaments and contests
- ✅ RapidAPI fetches tournament and leaderboard data
- ✅ Database schema supports all required relationships
- ❌ `importContestPlayers` method fails silently (lines 311-381)
- ❌ Contest Player Importer service is backed up (.bak file)
- ❌ No salary data from DraftKings/FanDuel APIs
- ❌ Player-contest linking is broken

**Database Evidence:**
```sql
-- All contests show 0 players
SELECT c.name, c.platform, COUNT(p.id) as player_count 
FROM contests c LEFT JOIN players p ON c.id = p.contest_id 
GROUP BY c.id, c.name, c.platform;
-- Result: All show player_count = 0
```

### Known Gotchas & Library Quirks
```go
// CRITICAL: RapidAPI Pro plan has higher limits but still requires careful management
// Current implementation uses Basic plan rate limiting (20 requests/day)
// Pro plan allows more requests but exact limits need verification

// GOTCHA: DraftKings unofficial API patterns
// API endpoint: https://www.draftkings.com/lobby/getcontests?sport=GOLF
// Player data: https://api.draftkings.com/draftgroups/v1/draftgroups/{groupId}/draftables
// No authentication required but rate limits are unpredictable

// CRITICAL: Player matching across providers requires fuzzy logic
// RapidAPI returns: "Tiger Woods"
// DraftKings returns: "T. Woods"
// ESPN returns: "Tiger Woods"
// Need sophisticated matching from player_matching.go.bak

// GOTCHA: Golf tournaments have unique challenges
// - Cut lines affect player availability mid-tournament
// - Withdrawals/injuries need real-time handling
// - Salary updates frequently before tournament start
```

## Implementation Blueprint

### Data Models and Structure
```go
// Core models need proper relationships
type Player struct {
    ID         uuid.UUID `json:"id"`
    SportID    uuid.UUID `json:"sport_id"`
    ContestID  *uuid.UUID `json:"contest_id"`  // CRITICAL: Currently NULL
    ExternalID string    `json:"external_id"`
    Name       string    `json:"name"`
    Position   string    `json:"position"`
    SalaryDK   *int      `json:"salary_dk"`   // CRITICAL: Currently NULL
    SalaryFD   *int      `json:"salary_fd"`   // CRITICAL: Currently NULL
    // ... other fields
}

type Contest struct {
    ID           uuid.UUID `json:"id"`
    TournamentID *uuid.UUID `json:"tournament_id"`
    Platform     string    `json:"platform"`    // "draftkings" or "fanduel"
    ContestType  string    `json:"contest_type"` // "gpp" or "cash"
    // ... other fields
}

type ContestPlayerImporter struct {
    db          *sql.DB
    rapidClient *RapidAPIGolfClient
    dkClient    *DraftKingsClient    // NEW: Need to create
    fdClient    *FanDuelClient       // NEW: Need to create
    logger      *slog.Logger
}
```

### List of Tasks to Complete (In Order)

```yaml
Task 1: Fix Current Tournament Sync Service
  MODIFY services/sports-data-service/internal/services/golf_tournament_sync.go:
    - FIND method: "importContestPlayers" (lines 311-381)
    - REPLACE broken player creation logic
    - ADD proper contest-player linking with UUID references
    - ADD comprehensive error handling and logging
    - PRESERVE existing tournament and contest creation logic

Task 2: Restore Contest Player Importer Service  
  RESTORE services/sports-data-service/internal/services/contest_player_importer.go.bak:
    - MOVE .bak file to active service
    - UPDATE import paths and dependencies
    - MODIFY to work with current database schema
    - ADD transaction handling for atomic operations
    - INTEGRATE with existing golf_tournament_sync.go

Task 3: Create DraftKings API Client
  CREATE services/sports-data-service/internal/providers/draftkings_client.go:
    - MIRROR pattern from: rapidapi_golf.go
    - IMPLEMENT unofficial DraftKings API endpoints
    - ADD rate limiting and retry logic
    - INCLUDE salary data extraction for golf contests
    - HANDLE JSON response parsing for contest/player data

Task 4: Restore Player Matching Service
  RESTORE services/sports-data-service/internal/services/player_matching.go.bak:
    - MOVE .bak file to active service
    - UPDATE for multi-provider matching (RapidAPI + DraftKings)
    - IMPLEMENT fuzzy name matching with confidence scores
    - ADD manual match creation for edge cases
    - INTEGRATE with player import pipeline

Task 5: Restore Tournament Data Enricher
  RESTORE services/sports-data-service/internal/services/tournament_enricher.go.bak:
    - MOVE .bak file to active service
    - UPDATE to work with current data flow
    - IMPLEMENT projection calculations (floor, ceiling, expected)
    - ADD rollback capabilities for failed enhancements
    - INTEGRATE with contest player importer

Task 6: Update RapidAPI Client for Pro Plan
  MODIFY services/sports-data-service/internal/providers/rapidapi_golf.go:
    - UPDATE rate limiting for Pro plan limits
    - ADD new endpoints for enhanced player data
    - IMPLEMENT more aggressive caching strategies
    - ADD comprehensive player field data fetching
    - PRESERVE existing caching and fallback logic

Task 7: Database Schema Enhancements
  CREATE migration for improved indexing:
    - ADD index on players.contest_id for faster queries
    - ADD index on players.external_id for matching
    - ADD index on contests.tournament_id for joins
    - VERIFY foreign key constraints are properly set
    - OPTIMIZE for frequent player lookup queries

Task 8: Integration and Testing
  MODIFY services/sports-data-service/internal/api/handlers/golf.go:
    - ADD new endpoints for player data validation
    - IMPLEMENT debugging endpoints for import status
    - ADD tournament sync status endpoints
    - PRESERVE existing golf tournament endpoints
    - INCLUDE comprehensive error reporting
```

### Per Task Pseudocode

```go
// Task 1: Fix importContestPlayers method
func (s *GolfTournamentSyncService) importContestPlayers(tournament *types.GolfTournament, contest *types.Contest) error {
    // PATTERN: Use existing error handling from rapidapi_golf.go
    players, err := s.rapidClient.GetPlayers(tournament.ExternalID)
    if err != nil {
        return fmt.Errorf("failed to fetch players for tournament %s: %w", tournament.Name, err)
    }
    
    // CRITICAL: Create proper contest-player links
    for _, player := range players {
        playerRecord := &types.Player{
            ID:         uuid.New(),
            SportID:    s.golfSportID,
            ContestID:  &contest.ID,  // CRITICAL: Link to contest
            ExternalID: player.ExternalID,
            Name:       player.Name,
            Position:   player.Position,
            // GOTCHA: Default salaries until DraftKings integration
            SalaryDK:   calculateDefaultSalary(player.Ranking, "draftkings"),
            SalaryFD:   calculateDefaultSalary(player.Ranking, "fanduel"),
        }
        
        // PATTERN: Use database transaction for atomic operations
        if err := s.db.CreatePlayer(playerRecord); err != nil {
            return fmt.Errorf("failed to create player %s: %w", player.Name, err)
        }
    }
    
    return nil
}

// Task 3: DraftKings API Client
type DraftKingsClient struct {
    httpClient *http.Client
    rateLimiter *rate.Limiter
    logger     *slog.Logger
}

func (c *DraftKingsClient) GetContests(sport string) ([]DKContest, error) {
    // PATTERN: Follow rapidapi_golf.go rate limiting
    if err := c.rateLimiter.Wait(context.Background()); err != nil {
        return nil, err
    }
    
    url := fmt.Sprintf("https://www.draftkings.com/lobby/getcontests?sport=%s", sport)
    // CRITICAL: Handle unofficial API response format
    resp, err := c.httpClient.Get(url)
    // ... handle response parsing
}

func (c *DraftKingsClient) GetPlayers(draftGroupID string) ([]DKPlayer, error) {
    // PATTERN: Use existing retry logic from rapidapi_golf.go
    url := fmt.Sprintf("https://api.draftkings.com/draftgroups/v1/draftgroups/%s/draftables", draftGroupID)
    // ... implement player data extraction
}
```

### Integration Points
```yaml
DATABASE:
  - migration: "CREATE INDEX idx_players_contest_id ON players(contest_id)"
  - migration: "CREATE INDEX idx_players_external_id ON players(external_id)"
  - migration: "ALTER TABLE players ADD CONSTRAINT fk_players_contest_id FOREIGN KEY (contest_id) REFERENCES contests(id)"
  
CONFIG:
  - add to: .env
  - pattern: "DRAFTKINGS_REQUEST_DELAY=1000  # ms between requests"
  - pattern: "FANDUEL_REQUEST_DELAY=1000     # ms between requests"
  
ROUTES:
  - add to: services/sports-data-service/internal/api/handlers/golf.go
  - pattern: "GET /api/golf/contests/{id}/players - List players for contest"
  - pattern: "POST /api/golf/tournaments/{id}/sync - Trigger tournament sync"
  - pattern: "GET /api/golf/import/status - Check import pipeline status"
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST - fix any errors before proceeding
cd services/sports-data-service
go mod tidy
go fmt ./...
go vet ./...

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests
```go
// CREATE test_golf_tournament_sync_test.go
func TestImportContestPlayers(t *testing.T) {
    // Test happy path - players imported successfully
    tournament := &types.GolfTournament{ExternalID: "test-tournament"}
    contest := &types.Contest{ID: uuid.New(), Platform: "draftkings"}
    
    err := syncService.importContestPlayers(tournament, contest)
    assert.NoError(t, err)
    
    // Verify players were created and linked
    players, err := db.GetPlayersByContestID(contest.ID)
    assert.NoError(t, err)
    assert.Greater(t, len(players), 0)
}

func TestDraftKingsClientIntegration(t *testing.T) {
    // Test DraftKings API client
    client := NewDraftKingsClient()
    contests, err := client.GetContests("GOLF")
    assert.NoError(t, err)
    assert.Greater(t, len(contests), 0)
}
```

```bash
# Run and iterate until passing:
go test ./internal/services/... -v
go test ./internal/providers/... -v
# If failing: Read error, understand root cause, fix code, re-run
```

### Level 3: Integration Test
```bash
# Start the service
go run cmd/server/main.go

# Test tournament sync with player import
curl -X POST http://localhost:8080/api/golf/tournaments/sync \
  -H "Content-Type: application/json" \
  -d '{"tournament_id": "test-tournament"}'

# Verify players were created
curl http://localhost:8080/api/golf/contests/STATUS_CHECK

# Expected: {"status": "success", "contests_with_players": 4}
```

## Final Validation Checklist
- [ ] All tests pass: `go test ./... -v`
- [ ] No compilation errors: `go build ./...`
- [ ] Database shows players linked to contests: `SELECT COUNT(*) FROM players WHERE contest_id IS NOT NULL`
- [ ] Tournament sync creates players: Manual sync test successful
- [ ] DraftKings API integration works: Salary data populated
- [ ] Error handling graceful: Failed imports don't break sync
- [ ] Logs are informative: Import status clearly visible
- [ ] Performance acceptable: Large tournaments (150+ players) import in <30s

---

## Anti-Patterns to Avoid
- ❌ Don't ignore silent failures - add comprehensive logging
- ❌ Don't skip transaction handling - use database transactions
- ❌ Don't hardcode salaries - integrate real DraftKings/FanDuel data
- ❌ Don't create new database patterns - follow existing schema
- ❌ Don't bypass rate limiting - respect API limits
- ❌ Don't ignore player matching - implement fuzzy matching
- ❌ Don't skip rollback mechanisms - handle partial failures
- ❌ Don't ignore cut lines - handle tournament-specific rules

## Confidence Score: 8/10

This PRP provides comprehensive context for one-pass implementation success:
- **Strengths**: Detailed codebase analysis, existing backup services, clear database schema
- **Challenges**: DraftKings unofficial API stability, player matching complexity
- **Mitigation**: Fallback providers, transaction handling, comprehensive testing

The backup services (.bak files) contain sophisticated implementations that just need restoration and integration, making this achievable in one pass with proper attention to the existing patterns and error handling.