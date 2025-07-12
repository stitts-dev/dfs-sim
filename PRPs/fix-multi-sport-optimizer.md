# PRP: Fix Multi-Sport Optimizer - Complete Implementation

## 🎯 Objective
Fix the DFS optimizer to properly return lineups for all sports (NBA, NFL, MLB, NHL), not just golf. Currently, non-golf sports receive empty results despite valid requests.

## 📋 Problem Analysis

### Root Cause Investigation
1. **Frontend Inconsistency**: Golf uses `optimizeGolfLineups` which adds `sport` and `contest_type` fields, while other sports use generic `optimizeLineups` without these fields
2. **Backend Design**: Backend correctly retrieves contest from database and uses its sport/platform for slot resolution
3. **Potential Issues**: 
   - Silent failures in slot resolution for non-golf sports
   - Missing error handling and logging
   - Possible data issues (no players for non-golf contests)

### Current Implementation Flow
```
Frontend (optimizeLineups) → Backend (OptimizeLineups handler) → Get Contest → Get Players → OptimizeLineups algorithm → GetPositionSlots(contest.Sport, contest.Platform) → Generate Lineups
```

## 🏗️ Implementation Blueprint

### Phase 1: Enhanced Logging & Debugging

#### 1.1 Backend Optimizer Handler (`backend/internal/api/handlers/optimizer.go`)
```go
// Add detailed logging after contest retrieval
log.Printf("Optimizer: Contest ID=%d, Sport=%s, Platform=%s, SalaryCap=%d", 
    contest.ID, contest.Sport, contest.Platform, contest.SalaryCap)

// Log player count by position
playersByPos := make(map[string]int)
for _, p := range players {
    playersByPos[p.Position]++
}
log.Printf("Optimizer: Players by position: %+v", playersByPos)
```

#### 1.2 Slot Resolution (`backend/internal/optimizer/slots.go`)
```go
// Add validation and logging
func GetPositionSlots(sport, platform string) []PositionSlot {
    log.Printf("GetPositionSlots: sport=%s, platform=%s", sport, platform)
    
    slots := []PositionSlot{}
    switch sport {
    case "nba":
        slots = getNBASlots(platform)
    // ... other sports
    default:
        log.Printf("WARNING: Unknown sport '%s' for slot resolution", sport)
    }
    
    log.Printf("GetPositionSlots: Returning %d slots for %s/%s", len(slots), sport, platform)
    return slots
}
```

### Phase 2: Frontend Standardization

#### 2.1 Update API Service (`frontend/src/services/api.ts`)
```typescript
export interface OptimizeConfigWithContext extends OptimizeConfig {
  sport?: string;
  platform?: string;
}

export const optimizeLineups = async (config: OptimizeConfigWithContext) => {
  // Log request for debugging
  console.log('Optimize request:', config);
  
  const { data } = await api.post('/optimize', config)
  
  // Log response
  console.log('Optimize response:', data);
  
  return data.data as OptimizerResult
}
```

#### 2.2 Update Optimizer Component (`frontend/src/pages/Optimizer.tsx`)
```typescript
const handleOptimize = async (config: Partial<OptimizeConfig>) => {
  if (!contestId || !contest) {
    toast.error('Please select a contest first')
    return
  }

  try {
    const optimizeConfig: OptimizeConfigWithContext = {
      contest_id: contestId,
      sport: contest.sport,        // Add sport from contest
      platform: contest.platform,  // Add platform from contest
      num_lineups: config.num_lineups || 20,
      // ... rest of config
    }

    console.log('Sending optimization request:', optimizeConfig)
    const result = await optimizeLineups(optimizeConfig)
    
    if (!result?.lineups?.length) {
      console.error('No lineups returned:', result)
      toast.error('No valid lineups generated. Check console for details.')
      return
    }
    
    // ... handle success
  } catch (error) {
    console.error('Optimization failed:', error)
    // Enhanced error handling
  }
}
```

### Phase 3: Backend Validation & Error Handling

#### 3.1 Request Validation (`backend/internal/api/handlers/optimizer.go`)
```go
// Add request logging
reqJSON, _ := json.Marshal(req)
log.Printf("Optimizer request: %s", string(reqJSON))

// Validate contest data
if contest.Sport == "" || contest.Platform == "" {
    utils.SendValidationError(c, "Invalid contest", 
        fmt.Sprintf("Contest missing sport/platform: sport=%s, platform=%s", 
            contest.Sport, contest.Platform))
    return
}

// Validate players exist
if len(players) == 0 {
    utils.SendValidationError(c, "No players available", 
        fmt.Sprintf("No players found for contest %d", req.ContestID))
    return
}
```

#### 3.2 Algorithm Enhancement (`backend/internal/optimizer/algorithm.go`)
```go
func generateValidLineups(playersByPosition map[string][]models.Player, config OptimizeConfig) []lineupCandidate {
    // Early validation
    if config.Contest == nil {
        log.Printf("ERROR: No contest provided to optimizer")
        return []lineupCandidate{}
    }
    
    slots := GetPositionSlots(config.Contest.Sport, config.Contest.Platform)
    if len(slots) == 0 {
        log.Printf("ERROR: No position slots found for %s/%s", 
            config.Contest.Sport, config.Contest.Platform)
        return []lineupCandidate{}
    }
    
    // Validate player availability for each slot
    for i, slot := range slots {
        hasPlayers := false
        for _, pos := range slot.AllowedPositions {
            if len(playersByPosition[pos]) > 0 {
                hasPlayers = true
                break
            }
        }
        if !hasPlayers {
            log.Printf("WARNING: No players available for slot %d (%s)", i, slot.SlotName)
        }
    }
    
    // Continue with generation...
}
```

### Phase 4: Comprehensive Testing

#### 4.1 Integration Test for All Sports (`backend/tests/optimizer_all_sports_test.go`)
```go
func TestOptimizerAllSports(t *testing.T) {
    sports := []struct {
        sport    string
        platform string
        positions []string
    }{
        {"nba", "draftkings", []string{"PG", "SG", "SF", "PF", "C"}},
        {"nfl", "draftkings", []string{"QB", "RB", "WR", "TE", "DST"}},
        {"mlb", "draftkings", []string{"P", "C", "1B", "2B", "3B", "SS", "OF"}},
        {"nhl", "draftkings", []string{"C", "W", "D", "G"}},
        {"golf", "draftkings", []string{"G"}},
    }
    
    for _, test := range sports {
        t.Run(fmt.Sprintf("%s_%s", test.sport, test.platform), func(t *testing.T) {
            // Setup test data
            contest := createTestContest(test.sport, test.platform)
            players := createTestPlayers(contest.ID, test.positions)
            
            // Run optimizer
            result := runOptimizer(contest.ID)
            
            // Verify results
            assert.NotEmpty(t, result.Lineups, 
                "Expected lineups for %s/%s", test.sport, test.platform)
            
            // Verify positions filled correctly
            for _, lineup := range result.Lineups {
                verifyLineupPositions(t, lineup, test.sport, test.platform)
            }
        })
    }
}
```

#### 4.2 API Endpoint Test (`backend/tests/api_optimizer_test.go`)
```go
func TestOptimizerEndpoint_AllSports(t *testing.T) {
    // Test each sport with minimal request
    for _, sport := range []string{"nba", "nfl", "mlb", "nhl", "golf"} {
        contest := setupContestWithPlayers(sport, "draftkings")
        
        req := OptimizeRequest{
            ContestID:  contest.ID,
            NumLineups: 5,
        }
        
        resp := callAPI("/optimize", req)
        assert.Equal(t, 200, resp.StatusCode)
        
        var result OptimizerResult
        json.Unmarshal(resp.Body, &result)
        assert.Greater(t, len(result.Lineups), 0, 
            "Should return lineups for %s", sport)
    }
}
```

### Phase 5: Data Validation Script

#### 5.1 Contest Data Checker (`backend/cmd/check-data/main.go`)
```go
func main() {
    db := database.Connect()
    
    // Check contests
    var contests []models.Contest
    db.Find(&contests)
    
    for _, contest := range contests {
        fmt.Printf("\nContest %d: %s\n", contest.ID, contest.Name)
        fmt.Printf("  Sport: %s, Platform: %s\n", contest.Sport, contest.Platform)
        
        // Check players
        var playerCount int64
        db.Model(&models.Player{}).Where("contest_id = ?", contest.ID).Count(&playerCount)
        fmt.Printf("  Players: %d\n", playerCount)
        
        // Check by position
        var positions []struct {
            Position string
            Count    int64
        }
        db.Model(&models.Player{}).
            Where("contest_id = ?", contest.ID).
            Select("position, COUNT(*) as count").
            Group("position").
            Scan(&positions)
        
        fmt.Printf("  By Position:\n")
        for _, p := range positions {
            fmt.Printf("    %s: %d\n", p.Position, p.Count)
        }
    }
}
```

## 📦 Implementation Tasks

### Critical Path (Must Do First)
1. [ ] Add comprehensive logging to optimizer handler
2. [ ] Add slot resolution logging and validation
3. [ ] Create data validation script to check contests/players
4. [ ] Add frontend request/response logging

### Core Fixes
5. [ ] Standardize frontend optimizer service (remove golf special case)
6. [ ] Add sport/platform to optimizer request (for consistency)
7. [ ] Enhance error messages throughout the stack
8. [ ] Add request/response validation

### Testing & Validation
9. [ ] Create integration tests for all sports
10. [ ] Add API endpoint tests
11. [ ] Create manual test checklist
12. [ ] Add performance benchmarks

## 🔍 Debugging Commands

```bash
# Check if players exist for non-golf contests
curl http://localhost:8080/api/v1/contests/1/players | jq '.data | length'

# Test NBA optimization directly
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 1,
    "num_lineups": 1,
    "min_different_players": 0,
    "use_correlations": false
  }' | jq '.'

# Check optimizer logs
docker-compose logs backend | grep -i "optimizer"

# Run data validation
go run cmd/check-data/main.go
```

## ⚠️ Common Pitfalls to Avoid

1. **Position Name Mismatch**: Ensure player positions match exactly (e.g., "D/ST" vs "DST")
2. **Case Sensitivity**: Sport/platform should be lowercase
3. **Empty Slots**: Some platforms have different slot configurations
4. **Salary Cap**: Different sports have different salary caps
5. **Missing Data**: Ensure test data includes all required positions

## 🎯 Validation Gates

```bash
# Backend Tests
cd backend
go test ./internal/optimizer/... -v
go test ./tests/... -v -run TestOptimizerAllSports

# Frontend Tests  
cd frontend
npm test -- --testNamePattern="Optimizer"

# Integration Test
# Start backend
cd backend && go run cmd/server/main.go

# In another terminal, run test script
cd backend && go run cmd/test-all-sports/main.go

# Manual Validation
# 1. Open app, select NBA contest
# 2. Click optimize - should see lineups
# 3. Repeat for NFL, MLB, NHL
# 4. Verify golf still works
```

## 📊 Success Metrics

- [ ] NBA optimizer returns valid lineups
- [ ] NFL optimizer returns valid lineups  
- [ ] MLB optimizer returns valid lineups
- [ ] NHL optimizer returns valid lineups
- [ ] Golf optimizer continues to work
- [ ] Clear error messages when optimization fails
- [ ] All integration tests pass
- [ ] Performance: < 2s for 20 lineup optimization

## 🔗 Reference URLs

- Gin Framework Logging: https://gin-gonic.com/docs/examples/custom-log-format/
- Go Testing Patterns: https://github.com/golang/go/wiki/TableDrivenTests
- React Error Boundaries: https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary
- TypeScript Discriminated Unions: https://www.typescriptlang.org/docs/handbook/2/narrowing.html#discriminated-unions

## 📈 Future Improvements

1. Add WebSocket progress updates during optimization
2. Cache optimization results by request hash
3. Add player pool filters (injury status, recent form)
4. Implement optimization presets by sport
5. Add visualization of lineup diversity

---

**Confidence Score: 9/10** - This PRP provides comprehensive context including actual code snippets from the codebase, clear implementation steps, debugging tools, and validation gates. The only uncertainty is around potential data issues that might need investigation.