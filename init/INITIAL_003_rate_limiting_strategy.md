## FEATURE:

Implement comprehensive rate limiting strategy for RapidAPI Golf Data API (20 requests/day limit)

## EXAMPLES:

Current implementation issues in `backend/internal/providers/rapidapi_golf.go`:
```go
// Missing request queuing
resp, err := p.makeRequest(ctx, endpoint, params)

// Basic rate limiting without queue
if p.rateLimiter != nil {
    if err := p.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limiter error: %w", err)
    }
}

// No cache warming strategy
// No intelligent request prioritization
```

Improved implementation should include:
```go
type RequestQueue struct {
    high   chan Request  // Live scoring updates
    medium chan Request  // Player stats
    low    chan Request  // Historical data
}

type CacheWarmingStrategy struct {
    // Pre-fetch tournament data at 00:01 UTC
    // Cache player stats for likely lineups
    // Prioritize active tournament data
}

func (p *RapidAPIGolfProvider) ProcessRequestQueue(ctx context.Context) {
    for {
        select {
        case req := <-p.queue.high:
            // Process high priority
        case req := <-p.queue.medium:
            // Process if daily limit allows
        case req := <-p.queue.low:
            // Process only if well under limit
        }
    }
}
```

Cache warming schedule:
```yaml
cache_warming:
  - time: "00:01 UTC"
    requests:
      - tournament_schedule  # 1 request
      - current_leaderboard  # 1 request
  - time: "06:00 UTC"
    requests:
      - player_stats_top_50  # 5 requests batched
  - time: "tournament_start - 2h"
    requests:
      - final_field_list     # 1 request
      - weather_conditions   # 1 request
```

## DOCUMENTATION:

- RapidAPI rate limiting: https://rapidapi.com/docs/rate-limiting
- Redis queue patterns: https://redis.io/docs/manual/patterns/
- Golang rate limiting: https://pkg.go.dev/golang.org/x/time/rate
- Priority queue implementations

## OTHER CONSIDERATIONS:

- Only 20 requests per day with Basic plan ($10/month)
- No request queuing currently implemented
- Cache not being warmed strategically
- ESPN fallback happens too late (after limit hit)
- No request prioritization (live data vs historical)
- Missing daily request counter persistence
- No alerting when approaching limit
- Should batch similar requests when possible
- Need graceful degradation strategy
- Consider upgrading to Pro plan (500 req/day) if needed
- Track most valuable API calls for optimization
- Implement circuit breaker pattern for API failures