# Manual Test Checklist - Multi-Sport Optimizer with API Data

## Prerequisites
- [ ] Backend server running (`docker-compose up` or `go run cmd/server/main.go`)
- [ ] Frontend dev server running (`npm run dev`)
- [ ] Database migrated with seed data (`go run cmd/migrate/main.go up`)
- [ ] Redis running (for caching)
- [ ] Valid API keys configured in `.env`:
  - [ ] `BALLDONTLIE_API_KEY` (get from https://www.balldontlie.io/)
  - [ ] `THESPORTSDB_API_KEY=4191544` (free tier)

## Initial Data Check (Seed Data)
Run the data validation script first:
```bash
cd backend
go run cmd/check-data/main.go
```

- [ ] Verify contests exist for all sports (NBA, NFL, MLB, NHL, Golf)
- [ ] Note initial player counts (seed data only)

## API Data Fetching Test

### Check Data Status
```bash
# Check NBA contest data status
curl http://localhost:8080/api/v1/contests/1/data-status | jq '.'
```
- [ ] Note: `total_players`, `is_stale`, `recommended_action`

### Trigger Data Fetch for NBA
```bash
# Fetch fresh NBA data from BallDontLie API
curl -X POST http://localhost:8080/api/v1/contests/1/fetch-data | jq '.'
```
- [ ] Verify: "Data fetch triggered successfully" message
- [ ] Wait 10-15 seconds for data to load

### Verify Fresh Data
```bash
# Check data status again
curl http://localhost:8080/api/v1/contests/1/data-status | jq '.'

# Check players
curl http://localhost:8080/api/v1/contests/1/players | jq '.data | length'
```
- [ ] Verify: More players than seed data
- [ ] Verify: Position distribution includes PG, SG, SF, PF, C
- [ ] Verify: Realistic salaries ($3,000-$12,000)

## NBA Optimization Test
1. [ ] Open the app and navigate to Optimizer page
2. [ ] Select an NBA contest (e.g., "NBA Main Slate")
3. [ ] Verify contest details show:
   - Sport: NBA
   - Platform: draftkings/fanduel
   - Salary Cap: $50,000
4. [ ] Click "Optimize" with default settings
5. [ ] Open browser console (F12) and check:
   - [ ] `Optimize request:` log shows sport="nba", platform set
   - [ ] `Optimize response:` log shows returned lineups
6. [ ] Verify UI shows generated lineups
7. [ ] For each lineup, verify:
   - [ ] 8 players total (PG, SG, SF, PF, C, G, F, UTIL)
   - [ ] Total salary ≤ $50,000
   - [ ] All positions filled correctly

## NFL Optimization Test
1. [ ] Select an NFL contest (e.g., "NFL Sunday Main")
2. [ ] Verify contest details show:
   - Sport: NFL
   - Platform: draftkings/fanduel
3. [ ] Click "Optimize" with default settings
4. [ ] Check browser console for request/response logs
5. [ ] Verify generated lineups have:
   - [ ] 9 players total (QB, 2 RB, 3 WR, TE, FLEX, DST)
   - [ ] Total salary ≤ $50,000
   - [ ] FLEX filled with RB/WR/TE only

## MLB Optimization Test
1. [ ] Select an MLB contest
2. [ ] Verify contest details show:
   - Sport: MLB
   - Platform: draftkings/fanduel
   - Salary Cap: $35,000 (note: different from others)
3. [ ] Click "Optimize" with default settings
4. [ ] Verify generated lineups have:
   - [ ] 10 players total (2 P, C, 1B, 2B, 3B, SS, 3 OF)
   - [ ] Total salary ≤ $35,000

## NHL Optimization Test
1. [ ] Select an NHL contest
2. [ ] Click "Optimize" with default settings
3. [ ] Verify generated lineups have:
   - [ ] 9 players total (2 C, 3 W, 2 D, G, UTIL)
   - [ ] UTIL filled with C/W/D only (not G)

## Golf Optimization Test
1. [ ] Select a Golf contest
2. [ ] Click "Optimize" with default settings
3. [ ] Verify generated lineups have:
   - [ ] 6 golfers total
   - [ ] All positions shown as "G"

## Constraint Testing
### Locked Players
1. [ ] Select NBA contest
2. [ ] Lock 2 players using the lock icon
3. [ ] Click "Optimize"
4. [ ] Verify all generated lineups contain the locked players

### Excluded Players
1. [ ] Select NFL contest
2. [ ] Exclude 3 players using the exclude icon
3. [ ] Click "Optimize"
4. [ ] Verify no generated lineups contain the excluded players

### Stacking Rules (NFL)
1. [ ] Select NFL contest
2. [ ] Enable "QB Stack" in optimizer controls
3. [ ] Click "Optimize"
4. [ ] Verify lineups have QB + at least 1 pass catcher from same team

## Error Handling
### No Players Available
1. [ ] If any sport returns no lineups:
   - [ ] Check browser console for error messages
   - [ ] Check backend logs for detailed errors
   - [ ] Run data validation script to diagnose

### Backend Connection Issues
1. [ ] Stop the backend server
2. [ ] Try to optimize
3. [ ] Verify error message: "Cannot connect to backend"
4. [ ] Restart backend and verify it works again

## Performance Testing
1. [ ] Select NBA contest
2. [ ] Set "Number of Lineups" to 150 (max)
3. [ ] Click "Optimize"
4. [ ] Verify:
   - [ ] Optimization completes within 2 seconds
   - [ ] All 150 lineups are valid
   - [ ] UI remains responsive

## Backend Log Verification
Check backend logs for each sport optimization:
- [ ] `Optimizer: Contest ID=X, Sport=Y, Platform=Z` logged
- [ ] `Optimizer: Players by position:` shows correct counts
- [ ] `GetPositionSlots: sport=Y, platform=Z` logged
- [ ] `GetPositionSlots: Returning N slots` shows correct count
- [ ] No ERROR logs during optimization

## API Testing with cURL
Test each sport via API directly:

### NBA
```bash
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 1,
    "num_lineups": 5
  }' | jq '.'
```
- [ ] Returns lineups array with 5 items

### NFL
```bash
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 2,
    "num_lineups": 5
  }' | jq '.'
```
- [ ] Returns lineups array with 5 items

## Regression Testing
- [ ] Golf optimization still works (was working before)
- [ ] Lineup drag-and-drop still works
- [ ] Export functionality works for all sports

## Final Verification
- [ ] All sports generate valid lineups
- [ ] No console errors in frontend
- [ ] No ERROR logs in backend
- [ ] Response times < 2 seconds
- [ ] UI displays lineups correctly

## Notes
- Record any issues found:
  - Sport: ___________
  - Issue: ___________
  - Error message: ___________
  - Steps to reproduce: ___________