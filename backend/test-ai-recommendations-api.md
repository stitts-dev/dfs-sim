# AI Recommendations API Test Guide

This guide demonstrates how to test the AI recommendation endpoints.

## Prerequisites

1. Set the `ANTHROPIC_API_KEY` environment variable:
```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

2. Ensure the server is running:
```bash
cd backend
go run cmd/server/main.go
```

3. Run the migration to create the AI recommendations table:
```bash
go run cmd/migrate/main.go up
```

## API Endpoints

All AI endpoints require authentication. You'll need a valid JWT token in the Authorization header.

### 1. Get Player Recommendations

**Endpoint:** `POST /api/v1/ai/recommend-players`

**Headers:**
```
Authorization: Bearer <your-jwt-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "contest_id": 1,
  "contest_type": "GPP",
  "sport": "NBA",
  "remaining_budget": 20000,
  "current_lineup": [1, 2, 3],
  "positions_needed": ["PG", "SG", "C"],
  "beginner_mode": true,
  "optimize_for": "ceiling"
}
```

**Example cURL:**
```bash
curl -X POST http://localhost:8080/api/v1/ai/recommend-players \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 1,
    "contest_type": "GPP",
    "sport": "NBA",
    "remaining_budget": 20000,
    "current_lineup": [1, 2, 3],
    "positions_needed": ["PG", "SG", "C"],
    "beginner_mode": true,
    "optimize_for": "ceiling"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "recommendations": [
      {
        "player_id": 10,
        "player_name": "Stephen Curry",
        "position": "PG",
        "team": "GSW",
        "salary": 9500,
        "projected_points": 45.5,
        "confidence": 0.85,
        "reasoning": "Curry has an excellent matchup against a weak defensive team...",
        "beginner_tip": "Point guards who face teams that play at a fast pace typically score more fantasy points",
        "stack_with": ["Klay Thompson", "Andrew Wiggins"],
        "avoid_with": ["LeBron James"]
      }
    ],
    "request": {
      "contest_id": 1,
      "contest_type": "GPP",
      "sport": "NBA",
      "remaining_budget": 20000,
      "optimize_for": "ceiling",
      "positions_needed": ["PG", "SG", "C"]
    }
  }
}
```

### 2. Analyze Lineup

**Endpoint:** `POST /api/v1/ai/analyze-lineup`

**Headers:**
```
Authorization: Bearer <your-jwt-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "lineup_id": 1,
  "contest_type": "GPP",
  "sport": "NBA"
}
```

**Example cURL:**
```bash
curl -X POST http://localhost:8080/api/v1/ai/analyze-lineup \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "lineup_id": 1,
    "contest_type": "GPP",
    "sport": "NBA"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "analysis": {
      "overall_score": 82.5,
      "strengths": [
        "Strong correlation between PG and SG from same team",
        "Good mix of high-floor and high-ceiling players",
        "Solid value plays that allow for star players"
      ],
      "weaknesses": [
        "Heavy exposure to one game",
        "Missing leverage plays for GPP contests"
      ],
      "improvements": [
        "Consider adding a contrarian play at center position",
        "Diversify game exposure to reduce risk"
      ],
      "stacking_analysis": {
        "game_stacks": ["GSW vs LAL"],
        "team_stacks": ["GSW (3 players)"],
        "correlation_score": 0.75
      },
      "risk_level": "medium",
      "beginner_insights": [
        "Game stacking means selecting multiple players from the same game",
        "This strategy works because high-scoring games benefit multiple players"
      ]
    },
    "lineup_id": 1
  }
}
```

### 3. Get Recommendation History

**Endpoint:** `GET /api/v1/ai/recommendations/history`

**Headers:**
```
Authorization: Bearer <your-jwt-token>
```

**Query Parameters:**
- `limit` (optional): Number of recommendations to return (default: 20, max: 100)

**Example cURL:**
```bash
curl -X GET "http://localhost:8080/api/v1/ai/recommendations/history?limit=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "recommendations": [
      {
        "id": 1,
        "user_id": 1,
        "contest_id": 1,
        "request": {
          "contest_type": "GPP",
          "sport": "NBA",
          "remaining_budget": 20000
        },
        "response": {
          "recommendations": [...]
        },
        "confidence": 0.82,
        "was_used": true,
        "lineup_result": 285.5,
        "created_at": "2024-01-15T10:30:00Z"
      }
    ],
    "count": 10,
    "limit": 10
  }
}
```

## Error Responses

### Rate Limit Exceeded
```json
{
  "success": false,
  "error": "AI rate limit exceeded, please try again later"
}
```
Status Code: 429

### Invalid Request
```json
{
  "success": false,
  "error": "Contest ID is required"
}
```
Status Code: 400

### Unauthorized
```json
{
  "success": false,
  "error": "User not authenticated"
}
```
Status Code: 401

## Configuration

The AI service can be configured using environment variables:

- `ANTHROPIC_API_KEY`: Your Anthropic API key (required)
- `AI_RATE_LIMIT`: Maximum requests per minute per user (default: 5)
- `AI_CACHE_EXPIRATION`: Cache duration in seconds (default: 3600)

## Rate Limiting

The AI recommendation service implements rate limiting to prevent abuse:
- Default: 5 requests per minute per user
- Cached responses don't count against the rate limit
- Rate limit is applied per user ID

## Caching

To improve performance and reduce API costs:
- Recommendations are cached for 1 hour by default
- Cache key includes user ID, contest ID, and optimization type
- Identical requests within the cache period return cached results

## Best Practices

1. **Use Beginner Mode**: Set `beginner_mode: true` for new DFS players to get educational explanations

2. **Specify Optimization Goals**:
   - `"ceiling"` for GPP tournaments (high risk, high reward)
   - `"floor"` for cash games (consistent, safer plays)
   - `"balanced"` for a mix of both

3. **Provide Context**: Include current lineup players to get complementary recommendations

4. **Check History**: Use the history endpoint to track which recommendations performed well

5. **Handle Errors Gracefully**: Implement retry logic for rate limit errors

## Testing Without Authentication (Development Only)

For development testing, you can temporarily move the AI routes outside the auth group in `router.go`. Remember to move them back before deploying to production!