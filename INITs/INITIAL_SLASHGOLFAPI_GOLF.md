## FEATURE:

Integrate Live Golf Data API from RapidAPI to replace mock golf data providers in the DFS Lineup Optimizer backend. This will provide real-time tournament data, player information, and leaderboard data for golf contests. The frontend should cache API responses to minimize backend calls and improve performance.

## EXAMPLES:

### API Integration Examples

#### Tournament Data Retrieval
```go
// Get tournament information
url := "https://live-golf-data.p.rapidapi.com/tournament?orgId=1&tournId=475&year=2024"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Player Information
```go
// Get player details
url := "https://live-golf-data.p.rapidapi.com/players?lastName=Morikawa&firstName=Collin&playerId=50525"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Leaderboard Data
```go
// Get tournament leaderboard
url := "https://live-golf-data.p.rapidapi.com/leaderboard?orgId=1&tournId=475&year=2024"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Player Statistics
```go
// Get player statistics for a specific year
url := "https://live-golf-data.p.rapidapi.com/stats?year=2024&statId=186"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Tournament Points
```go
// Get FedEx Cup or other points for a tournament
url := "https://live-golf-data.p.rapidapi.com/points?tournId=475&year=2024"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Tournament Earnings
```go
// Get prize money distribution for a tournament
url := "https://live-golf-data.p.rapidapi.com/earnings?tournId=475&year=2024"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Tournament Schedule
```go
// Get full season schedule
url := "https://live-golf-data.p.rapidapi.com/schedule?orgId=1&year=2024"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Golf Organizations
```go
// Get list of golf organizations (PGA Tour, European Tour, etc.)
url := "https://live-golf-data.p.rapidapi.com/organizations"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

#### Player Scorecard
```go
// Get detailed scorecard for a player in a tournament
url := "https://live-golf-data.p.rapidapi.com/scorecard?orgId=1&tournId=475&year=2024&playerId=47504"
req, _ := http.NewRequest("GET", url, nil)
req.Header.Add("x-rapidapi-key", "YOUR_API_KEY")
req.Header.Add("x-rapidapi-host", "live-golf-data.p.rapidapi.com")
```

### Expected Response Structures
- Tournament data includes: tournament name, dates, course information, prize money
- Player data includes: player stats, recent performance, world ranking
- Leaderboard includes: current standings, scores, player positions
- Stats data includes: various statistical categories (driving distance, putting average, etc.)
- Points data includes: FedEx Cup points, world ranking points
- Earnings data includes: prize money by position, total purse
- Schedule includes: full season calendar with tournament details
- Organizations includes: tour information (PGA, European, Asian tours)
- Scorecard includes: hole-by-hole scores, round totals, eagles/birdies/pars/bogeys

## DOCUMENTATION:

### RapidAPI Live Golf Data Documentation
- Base URL: `https://live-golf-data.p.rapidapi.com`
- Authentication: API Key via headers (`x-rapidapi-key`, `x-rapidapi-host`)
- Available Endpoints:
  - `/tournament` - Tournament details
  - `/players` - Player information
  - `/leaderboard` - Live leaderboard data
  - `/schedule` - Tournament schedule
  - `/stats` - Player statistics (various stat categories)
  - `/points` - FedEx Cup and ranking points
  - `/earnings` - Prize money distribution
  - `/organizations` - Golf tour organizations
  - `/scorecard` - Hole-by-hole player scores

### Integration Points
1. **Backend Provider**: Create new `GolfDataProvider` in `backend/internal/providers/`
2. **Caching Strategy**: Implement Redis caching with TTL:
   - Tournament data: 1 hour
   - Player stats: 30 minutes
   - Leaderboard: 5 minutes (during tournaments)
   - Schedule: 24 hours
   - Organizations: 7 days
   - Points/Earnings: 1 hour during tournaments, 6 hours otherwise
   - Scorecard: 10 minutes during tournaments, 2 hours otherwise
3. **Frontend Caching**: Use React Query with stale-while-revalidate strategy

## OTHER CONSIDERATIONS:

### API Rate Limits
- Check RapidAPI rate limits for the Live Golf Data API
- Implement exponential backoff for failed requests
- Consider implementing a request queue to prevent rate limit violations

### Data Mapping
- Map API response fields to existing DFS optimizer models:
  - Player ID mapping to DFS platform IDs (DraftKings, FanDuel)
  - Salary data will still need to come from DFS platforms
  - Tournament ID mapping for contest creation
  - Stat IDs to meaningful categories (e.g., 186 = Strokes Gained: Total)
  - Organization IDs (1 = PGA Tour, etc.)
  - Points types (FedEx Cup, World Ranking Points)

### Error Handling
- Handle API downtime gracefully with fallback to cached data
- Provide user feedback when real-time data is unavailable
- Log API errors for monitoring

### Performance Optimization
- Batch player requests when possible
- Pre-fetch tournament data on app initialization
- Use WebSocket connections for live leaderboard updates during tournaments

### Security
- Store API key in environment variables
- Never expose API key in frontend code
- Implement backend proxy for all API calls

### Testing
- Mock API responses for unit tests
- Create integration tests with test API key
- Test caching behavior and TTL expiration
- Verify fallback behavior when API is unavailable