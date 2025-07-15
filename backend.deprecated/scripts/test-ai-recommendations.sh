#!/bin/bash

# Test AI Recommendations API
# This script tests the AI recommendation endpoints

API_URL="http://localhost:8080/api/v1"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Testing AI Recommendations API${NC}"
echo "================================"

# Check if ANTHROPIC_API_KEY is set
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo -e "${RED}Error: ANTHROPIC_API_KEY environment variable is not set${NC}"
    echo "Please run: export ANTHROPIC_API_KEY='your-key-here'"
    exit 1
fi

# For testing, we'll need a JWT token. In production, this would come from login
# For now, you can set it manually or implement a login endpoint
JWT_TOKEN="${JWT_TOKEN:-your-jwt-token-here}"

if [ "$JWT_TOKEN" == "your-jwt-token-here" ]; then
    echo -e "${RED}Warning: Using placeholder JWT token. Set JWT_TOKEN environment variable for authenticated requests.${NC}"
    echo "For development testing, you may need to temporarily move AI routes outside the auth group in router.go"
    echo ""
fi

# Test 1: Get Player Recommendations for GPP
echo -e "${GREEN}Test 1: Get Player Recommendations (GPP)${NC}"
curl -s -X POST "$API_URL/ai/recommend-players" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 1,
    "contest_type": "GPP",
    "sport": "NBA",
    "remaining_budget": 20000,
    "current_lineup": [],
    "positions_needed": ["PG", "SG", "SF", "PF", "C"],
    "beginner_mode": true,
    "optimize_for": "ceiling"
  }' | jq '.' || echo "Failed to get recommendations"

echo -e "\n${GREEN}Test 2: Get Player Recommendations (Cash Game)${NC}"
curl -s -X POST "$API_URL/ai/recommend-players" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 1,
    "contest_type": "Cash",
    "sport": "NBA",
    "remaining_budget": 15000,
    "current_lineup": [1, 2, 3],
    "positions_needed": ["PG", "C"],
    "beginner_mode": false,
    "optimize_for": "floor"
  }' | jq '.' || echo "Failed to get recommendations"

# Test 3: Analyze a lineup
echo -e "\n${GREEN}Test 3: Analyze Lineup${NC}"
curl -s -X POST "$API_URL/ai/analyze-lineup" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "lineup_id": 1,
    "contest_type": "GPP",
    "sport": "NBA"
  }' | jq '.' || echo "Failed to analyze lineup"

# Test 4: Get recommendation history
echo -e "\n${GREEN}Test 4: Get Recommendation History${NC}"
curl -s -X GET "$API_URL/ai/recommendations/history?limit=5" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq '.' || echo "Failed to get history"

# Test 5: Test rate limiting
echo -e "\n${GREEN}Test 5: Testing Rate Limiting${NC}"
echo "Making 6 rapid requests to test rate limiting (limit is 5 per minute)..."
for i in {1..6}; do
    echo -n "Request $i: "
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/ai/recommend-players" \
      -H "Authorization: Bearer $JWT_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "contest_id": 1,
        "contest_type": "GPP",
        "sport": "NBA",
        "remaining_budget": 10000,
        "positions_needed": ["PG"],
        "optimize_for": "ceiling"
      }')
    
    http_code=$(echo "$response" | tail -n 1)
    if [ "$http_code" == "429" ]; then
        echo -e "${RED}Rate limited (429)${NC}"
    elif [ "$http_code" == "200" ]; then
        echo -e "${GREEN}Success (200)${NC}"
    else
        echo -e "${RED}Error ($http_code)${NC}"
    fi
    
    # Small delay between requests
    sleep 0.5
done

echo -e "\n${BLUE}AI Recommendations API Tests Complete${NC}"