## FEATURE: Fix Multi-Sport Optimizer - Currently Only Returns Golf Results

**Problem Statement:**
The DFS optimizer endpoint (`/api/v1/optimize`) is designed to support multiple sports (NBA, NFL, MLB, NHL, Golf) but currently only returns results for golf. Other sports receive no lineups despite valid requests being sent.

**Root Cause Analysis:**

1. **Request Structure Mismatch**:
   - Golf optimizer explicitly sends `sport: 'golf'` and `contest_type: 'golf'` in request body
   - Other sports DO NOT include these fields in the request
   - Backend expects these fields to determine correct slot/position logic

2. **Frontend Service Inconsistency**:
   ```typescript
   // Golf (WORKING):
   optimizeGolfLineups(params) {
     return api.post('/optimize', {
       contest_type: 'golf',
       sport: 'golf',
       ...params
     });
   }
   
   // Other Sports (BROKEN):
   optimizeLineups(params) {
     return api.post('/optimize', params); // Missing sport/contest_type!
   }
   ```

3. **Backend Slot Resolution**:
   - `backend/internal/optimizer/slots.go` relies on sport/platform to determine positions
   - Without these fields, slot resolution fails silently
   - Golf works because it has simple slot structure (all positions are "G")

**Requirements for Resolution:**

### 1. Frontend Fix (Priority: HIGH)
**File:** `frontend/src/services/api.ts`
```typescript
// Current (Broken):
optimizeLineups(contestId: number, config: OptimizeConfig) {
  return api.post(`/optimize`, { contest_id: contestId, ...config });
}

// Fixed:
optimizeLineups(contestId: number, config: OptimizeConfig, contest: Contest) {
  return api.post(`/optimize`, {
    contest_id: contestId,
    sport: contest.sport.toLowerCase(),
    platform: contest.platform,
    ...config
  });
}
```

### 2. Backend Validation (Priority: HIGH)
**File:** `backend/internal/api/handlers/optimizer.go`
- Add validation for required fields
- Return clear error if sport/platform missing
- Log warnings for debugging

### 3. Unified Request Interface
**File:** `backend/internal/models/optimizer.go`
```go
type OptimizeRequest struct {
    ContestID int     `json:"contest_id" binding:"required"`
    Sport     string  `json:"sport" binding:"required"`
    Platform  string  `json:"platform" binding:"required"`
    // ... other fields
}
```

### 4. Remove Golf Special Case
- Consolidate `optimizeGolfLineups` into generic `optimizeLineups`
- Ensure all sports use same code path
- Keep golf-specific logic only where necessary (scoring, positions)

## EXAMPLES:

### Example 1: Fixed Frontend Service Call
```typescript
// frontend/src/pages/Optimizer.tsx
const handleOptimize = async () => {
  const result = await optimizeLineups(
    contestId,
    optimizerConfig,
    selectedContest // Pass contest object
  );
};
```

### Example 2: Backend Validation
```go
// backend/internal/api/handlers/optimizer.go
func (h *Handler) OptimizeLineups(c *gin.Context) {
    var req OptimizeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Missing required fields: sport, platform"})
        return
    }
    
    // Validate sport is supported
    if !isValidSport(req.Sport) {
        c.JSON(400, gin.H{"error": fmt.Sprintf("Unsupported sport: %s", req.Sport)})
        return
    }
}
```

### Example 3: Integration Test
```go
// backend/tests/optimizer_integration_test.go
func TestOptimizerAllSports(t *testing.T) {
    sports := []string{"nba", "nfl", "mlb", "nhl", "golf"}
    
    for _, sport := range sports {
        t.Run(sport, func(t *testing.T) {
            req := OptimizeRequest{
                ContestID: 1,
                Sport:     sport,
                Platform:  "draftkings",
                MaxSalary: 50000,
            }
            
            resp := callOptimizer(req)
            assert.NotEmpty(t, resp.Lineups, "Expected lineups for %s", sport)
        })
    }
}
```

## DOCUMENTATION:

### API Contract Update
All optimizer requests MUST include:
- `contest_id`: The contest to optimize for
- `sport`: The sport type (nba, nfl, mlb, nhl, golf)
- `platform`: The DFS platform (draftkings, fanduel)

### Error Responses
- `400 Bad Request`: Missing required fields
- `400 Bad Request`: Unsupported sport
- `404 Not Found`: Contest not found
- `500 Internal Error`: Optimizer failure

### Testing Checklist
- [ ] NBA optimizer returns lineups
- [ ] NFL optimizer returns lineups  
- [ ] MLB optimizer returns lineups
- [ ] NHL optimizer returns lineups
- [ ] Golf optimizer still works
- [ ] Clear errors for missing fields
- [ ] Integration tests pass

## OTHER CONSIDERATIONS:

1. **Backward Compatibility**:
   - Keep golf service working during transition
   - Add deprecation notice for golf-specific endpoint
   - Migrate golf UI to use generic optimizer

2. **Performance Impact**:
   - No performance degradation expected
   - Same optimization algorithms used
   - Request size increase minimal (2 fields)

3. **Monitoring**:
   - Add metrics for optimizer calls by sport
   - Track error rates by sport
   - Alert on high failure rates

4. **Migration Strategy**:
   - Fix frontend first (immediate impact)
   - Deploy backend validation
   - Remove golf special case in v2

5. **Root Cause Prevention**:
   - Add TypeScript types for all API requests
   - Implement request/response validation
   - Add automated API contract tests

## Quick Fix Steps:

1. **Immediate Fix** (5 minutes):
   ```typescript
   // frontend/src/services/api.ts
   export const optimizeLineups = async (
     contestId: number,
     config: OptimizeConfig,
     contest: Contest // ADD THIS
   ): Promise<OptimizeResponse> => {
     const response = await api.post('/api/v1/optimize', {
       contest_id: contestId,
       sport: contest.sport.toLowerCase(), // ADD THIS
       platform: contest.platform,          // ADD THIS
       ...config
     });
     return response.data;
   };
   ```

2. **Update Component** (2 minutes):
   ```typescript
   // frontend/src/pages/Optimizer.tsx
   // Find where optimizeLineups is called
   // Pass selectedContest as third parameter
   ```

3. **Test All Sports** (10 minutes):
   - Test NBA optimization
   - Test NFL optimization
   - Verify golf still works

This focused approach will resolve the multi-sport optimizer issue with minimal code changes and maximum impact.