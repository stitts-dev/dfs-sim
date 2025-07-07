# Golf API Testing Guide

## Prerequisites
- Backend server running on port 8080
- PostgreSQL database with golf tables (run migration 004_add_golf_support.sql)
- Redis running for caching

## Test Scenarios

### 1. Fetch Golf Tournaments
```bash
curl -X GET http://localhost:8080/api/v1/golf/tournaments \
  -H "Content-Type: application/json"
```

Expected response:
```json
{
  "tournaments": [
    {
      "id": "uuid",
      "name": "Masters Tournament",
      "status": "scheduled",
      "start_date": "2024-04-11T00:00:00Z",
      "course_name": "Augusta National",
      ...
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### 2. Get Tournament Details
```bash
curl -X GET http://localhost:8080/api/v1/golf/tournaments/{tournamentId} \
  -H "Content-Type: application/json"
```

### 3. Get Tournament Leaderboard
```bash
curl -X GET http://localhost:8080/api/v1/golf/tournaments/{tournamentId}/leaderboard \
  -H "Content-Type: application/json"
```

Expected response:
```json
{
  "tournament": { ... },
  "entries": [
    {
      "player_id": 123,
      "current_position": 1,
      "total_score": -8,
      "thru_holes": 18,
      "player": {
        "name": "Scottie Scheffler",
        "team": "USA"
      }
    }
  ],
  "cut_line": 2,
  "updated_at": "2024-01-20T15:30:00Z"
}
```

### 4. Get Tournament Players with Salaries
```bash
# For DraftKings
curl -X GET http://localhost:8080/api/v1/golf/tournaments/{tournamentId}/players?platform=draftkings \
  -H "Content-Type: application/json"

# For FanDuel
curl -X GET http://localhost:8080/api/v1/golf/tournaments/{tournamentId}/players?platform=fanduel \
  -H "Content-Type: application/json"
```

### 5. Get Golf Projections
```bash
curl -X GET http://localhost:8080/api/v1/golf/tournaments/{tournamentId}/projections \
  -H "Content-Type: application/json"
```

Expected response:
```json
{
  "tournament": { ... },
  "projections": {
    "player_123": {
      "player_id": "123",
      "expected_score": 280.5,
      "cut_probability": 0.75,
      "top10_probability": 0.20,
      "dk_points": 85.5,
      "fd_points": 82.3
    }
  },
  "correlations": {
    "123": {
      "124": 0.15,  // Same country correlation
      "125": 0.08   // Similar skill level
    }
  }
}
```

### 6. Create Golf Contest
```bash
curl -X POST http://localhost:8080/api/v1/contests \
  -H "Content-Type: application/json" \
  -d '{
    "name": "PGA Championship GPP",
    "sport": "golf",
    "platform": "draftkings",
    "contest_type": "gpp",
    "entry_fee": 20,
    "prize_pool": 100000,
    "max_entries": 10000,
    "salary_cap": 50000,
    "start_time": "2024-05-16T07:00:00Z"
  }'
```

### 7. Optimize Golf Lineup
```bash
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": "{contestId}",
    "num_lineups": 20,
    "constraints": {
      "min_cut_probability": 0.6,
      "max_exposure": 0.5,
      "locked_player_ids": [],
      "excluded_player_ids": []
    },
    "use_correlations": true,
    "unique_multiplier": 0.8
  }'
```

Expected response:
```json
{
  "lineups": [
    {
      "players": [
        {
          "id": 123,
          "name": "Scottie Scheffler",
          "position": "G",
          "salary": 11500,
          "projected_points": 95.5
        },
        // ... 5 more golfers
      ],
      "total_salary": 49800,
      "projected_points": 425.5,
      "correlation_score": 0.85
    }
  ]
}
```

### 8. Sync Tournament Data (Admin)
```bash
curl -X POST http://localhost:8080/api/v1/golf/tournaments/{tournamentId}/sync \
  -H "Content-Type: application/json"
```

### 9. Get Player Course History
```bash
curl -X GET http://localhost:8080/api/v1/golf/players/{playerId}/history?course_id=augusta \
  -H "Content-Type: application/json"
```

## Testing Workflow

1. **Initial Setup**
   ```bash
   # Run migrations
   cd backend
   go run cmd/migrate/main.go up
   
   # Start server
   go run cmd/server/main.go
   ```

2. **Create Test Data**
   ```bash
   # Sync current PGA tournament
   curl -X POST http://localhost:8080/api/v1/golf/tournaments/current/sync
   ```

3. **Test Optimization Flow**
   - Get active tournaments
   - Select a tournament
   - Create a contest for that tournament
   - Fetch players with projections
   - Run optimization
   - Validate lineups meet golf constraints (6 players, under salary cap)

## Common Issues & Solutions

### No Tournament Data
- ESPN API may be temporarily unavailable
- Try syncing data manually using the sync endpoint
- Check server logs for API errors

### Empty Projections
- Ensure players have been synced for the tournament
- Check that the tournament status is not "completed"
- Verify course history data is available

### Optimization Failures
- Check minimum cut probability isn't too high (>0.8)
- Ensure enough players meet the constraints
- Verify salary cap is appropriate for the platform

## Performance Testing

```bash
# Test optimization performance with different lineup counts
for i in 10 20 50 100 150; do
  echo "Testing $i lineups..."
  time curl -X POST http://localhost:8080/api/v1/optimize \
    -H "Content-Type: application/json" \
    -d "{\"contest_id\": \"$CONTEST_ID\", \"num_lineups\": $i}"
done
```

## Validation Checklist

- [ ] All lineups have exactly 6 golfers
- [ ] All players have position "G"
- [ ] Lineup salaries are under the cap ($50k DK, $60k FD)
- [ ] Cut probabilities are calculated correctly
- [ ] Correlations include country and skill level factors
- [ ] Stacking rules produce valid combinations
- [ ] Real-time updates work for live tournaments
- [ ] Export format is compatible with DFS platforms