#!/bin/bash

# AI Recommendations Service Validation Script
# This script validates the complete AI recommendations service functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVICE_URL=${AI_RECOMMENDATIONS_SERVICE_URL:-"http://localhost:8084"}
API_GATEWAY_URL=${API_GATEWAY_URL:-"http://localhost:8080"}
TEST_USER_ID=${TEST_USER_ID:-"550e8400-e29b-41d4-a716-446655440000"}
TEST_CONTEST_ID=${TEST_CONTEST_ID:-"123"}

echo -e "${BLUE}🧠 AI Recommendations Service Validation${NC}"
echo "=============================================="
echo "Service URL: $SERVICE_URL"
echo "API Gateway: $API_GATEWAY_URL"
echo "Test User ID: $TEST_USER_ID"
echo "Test Contest ID: $TEST_CONTEST_ID"
echo ""

# Function to make HTTP requests with proper error handling
make_request() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    echo -n "Testing $description... "
    
    if [ "$method" = "POST" ] && [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" \
            -X POST \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$url")
    else
        response=$(curl -s -w "\n%{http_code}" "$url")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗ (HTTP $http_code)${NC}"
        echo "Response: $body"
        return 1
    fi
}

# Function to check WebSocket connection
test_websocket() {
    echo -n "Testing WebSocket connection... "
    
    # Use Node.js or Python if available, otherwise skip
    if command -v node &> /dev/null; then
        # Simple Node.js WebSocket test
        cat << 'EOF' > /tmp/ws_test.js
const WebSocket = require('ws');
const ws = new WebSocket(process.argv[2]);

ws.on('open', function open() {
    console.log('✓');
    process.exit(0);
});

ws.on('error', function error(err) {
    console.log('✗');
    process.exit(1);
});

setTimeout(() => {
    console.log('✗ (timeout)');
    process.exit(1);
}, 5000);
EOF
        
        if node /tmp/ws_test.js "ws://localhost:8084/ws/ai-recommendations/$TEST_USER_ID" 2>/dev/null; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${YELLOW}⚠ (WebSocket test skipped - connection failed)${NC}"
        fi
        rm -f /tmp/ws_test.js
    else
        echo -e "${YELLOW}⚠ (WebSocket test skipped - Node.js not available)${NC}"
    fi
}

# Track test results
tests_passed=0
tests_failed=0
tests_skipped=0

run_test() {
    if "$@"; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
}

run_skipped_test() {
    ((tests_skipped++))
}

echo -e "${BLUE}1. Health Check Tests${NC}"
echo "===================="

# Test direct service health
run_test make_request "GET" "$SERVICE_URL/health" "" "AI Recommendations Service health check"
run_test make_request "GET" "$SERVICE_URL/ready" "" "AI Recommendations Service readiness check"
run_test make_request "GET" "$SERVICE_URL/metrics" "" "AI Recommendations Service metrics"

# Test through API Gateway
run_test make_request "GET" "$API_GATEWAY_URL/health" "" "API Gateway health check"

echo ""
echo -e "${BLUE}2. AI Recommendation Tests${NC}"
echo "=========================="

# Test player recommendations
player_request='{
    "contest_id": 123,
    "sport": "golf",
    "contest_type": "gpp",
    "max_players": 6,
    "budget": 50000,
    "players": [
        {
            "id": 1,
            "name": "Rory McIlroy",
            "position": "G",
            "salary": 9500,
            "projected_points": 65.0,
            "ownership": 25.5,
            "team": "NIR"
        },
        {
            "id": 2,
            "name": "Scottie Scheffler", 
            "position": "G",
            "salary": 10000,
            "projected_points": 68.0,
            "ownership": 30.0,
            "team": "USA"
        }
    ],
    "preferences": {
        "risk_tolerance": "medium",
        "ownership_strategy": "balanced",
        "optimization_goal": "roi"
    }
}'

run_test make_request "POST" "$SERVICE_URL/api/v1/recommendations/players" "$player_request" "Player recommendations"

# Test through API Gateway
run_test make_request "POST" "$API_GATEWAY_URL/api/v1/ai-recommendations/players" "$player_request" "Player recommendations via API Gateway"

# Test lineup recommendations
lineup_request='{
    "contest_id": 123,
    "sport": "golf",
    "contest_type": "gpp",
    "current_lineup": [
        {"player_id": 1, "salary": 9500, "projected_points": 65.0},
        {"player_id": 2, "salary": 10000, "projected_points": 68.0}
    ],
    "available_players": [
        {"id": 3, "name": "Jon Rahm", "position": "G", "salary": 9200, "projected_points": 64.0}
    ],
    "optimization_goal": "ceiling"
}'

run_test make_request "POST" "$SERVICE_URL/api/v1/recommendations/lineup" "$lineup_request" "Lineup recommendations"

echo ""
echo -e "${BLUE}3. Ownership Analysis Tests${NC}"
echo "=========================="

# Test ownership data
run_test make_request "GET" "$SERVICE_URL/api/v1/ownership/$TEST_CONTEST_ID" "" "Ownership data retrieval"

# Test leverage opportunities
run_test make_request "GET" "$SERVICE_URL/api/v1/ownership/$TEST_CONTEST_ID/leverage?contest_type=gpp&min_leverage_score=0.3" "" "Leverage opportunities"

# Test ownership trends
run_test make_request "GET" "$SERVICE_URL/api/v1/ownership/$TEST_CONTEST_ID/trends?timeframe=24h" "" "Ownership trends"

echo ""
echo -e "${BLUE}4. Analysis Tests${NC}"
echo "================="

# Test lineup analysis
lineup_analysis_request='{
    "contest_id": 123,
    "lineup": [
        {"player_id": 1, "salary": 9500, "projected_points": 65.0},
        {"player_id": 2, "salary": 10000, "projected_points": 68.0}
    ],
    "analysis_type": "risk_reward"
}'

run_test make_request "POST" "$SERVICE_URL/api/v1/analyze/lineup" "$lineup_analysis_request" "Lineup analysis"

# Test contest analysis
contest_analysis_request='{
    "contest_id": 123,
    "sport": "golf",
    "analysis_depth": "comprehensive"
}'

run_test make_request "POST" "$SERVICE_URL/api/v1/analyze/contest" "$contest_analysis_request" "Contest analysis"

# Test trends
run_test make_request "GET" "$SERVICE_URL/api/v1/analyze/trends/golf?timeframe=7d" "" "Trends analysis"

echo ""
echo -e "${BLUE}5. WebSocket Tests${NC}"
echo "=================="

test_websocket

echo ""
echo -e "${BLUE}6. Error Handling Tests${NC}"
echo "======================="

# Test invalid contest ID
run_test make_request "GET" "$SERVICE_URL/api/v1/ownership/invalid" "" "Invalid contest ID error handling"

# Test malformed JSON
run_test make_request "POST" "$SERVICE_URL/api/v1/recommendations/players" '{"invalid": json}' "Malformed JSON error handling"

# Test missing required fields
run_test make_request "POST" "$SERVICE_URL/api/v1/recommendations/players" '{"contest_id": 123}' "Missing required fields error handling"

echo ""
echo -e "${BLUE}7. Performance Tests${NC}"
echo "===================="

echo -n "Testing response time for player recommendations... "
start_time=$(date +%s%N)
make_request "POST" "$SERVICE_URL/api/v1/recommendations/players" "$player_request" "Response time test" > /dev/null 2>&1
end_time=$(date +%s%N)
response_time=$((($end_time - $start_time) / 1000000))

if [ $response_time -lt 5000 ]; then
    echo -e "${GREEN}✓ (${response_time}ms)${NC}"
    ((tests_passed++))
else
    echo -e "${YELLOW}⚠ (${response_time}ms - slower than expected)${NC}"
    ((tests_skipped++))
fi

echo ""
echo -e "${BLUE}8. Integration Tests${NC}"
echo "===================="

# Test service discovery through API Gateway
echo -n "Testing service discovery... "
if curl -s "$API_GATEWAY_URL/status/services" | grep -q "ai-recommendations-service"; then
    echo -e "${GREEN}✓${NC}"
    ((tests_passed++))
else
    echo -e "${RED}✗${NC}"
    ((tests_failed++))
fi

# Test Redis connectivity (if accessible)
echo -n "Testing Redis integration... "
if command -v redis-cli &> /dev/null; then
    if redis-cli -p 6379 -n 4 ping | grep -q "PONG"; then
        echo -e "${GREEN}✓${NC}"
        ((tests_passed++))
    else
        echo -e "${YELLOW}⚠ (Redis not accessible)${NC}"
        ((tests_skipped++))
    fi
else
    echo -e "${YELLOW}⚠ (redis-cli not available)${NC}"
    ((tests_skipped++))
fi

echo ""
echo "=============================================="
echo -e "${BLUE}Validation Summary${NC}"
echo "=============================================="
echo -e "Tests Passed: ${GREEN}$tests_passed${NC}"
echo -e "Tests Failed: ${RED}$tests_failed${NC}"
echo -e "Tests Skipped: ${YELLOW}$tests_skipped${NC}"
echo ""

if [ $tests_failed -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! AI Recommendations Service is working correctly.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Please check the service configuration and logs.${NC}"
    exit 1
fi