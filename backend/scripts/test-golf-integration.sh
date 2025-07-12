#!/bin/bash

# Golf Integration Test Script
# This script tests the golf implementation end-to-end

set -e

echo "🏌️ Golf Integration Test Starting..."

# Configuration
API_BASE="http://localhost:8080/api/v1"
DB_NAME="dfs_optimizer"
DB_USER="postgres"
DB_PASS="postgres"
DB_HOST="localhost"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

log_error() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

log_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if server is running
    if ! curl -s -o /dev/null -w "%{http_code}" "$API_BASE/health" | grep -q "200"; then
        log_error "Backend server is not running on port 8080"
    fi
    log_success "Backend server is running"
    
    # Check database connection
    if ! PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c '\q' 2>/dev/null; then
        log_error "Cannot connect to PostgreSQL database"
    fi
    log_success "Database connection successful"
    
    # Check if golf tables exist
    TABLE_COUNT=$(PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name LIKE 'golf_%'")
    if [ "$TABLE_COUNT" -eq 0 ]; then
        log_error "Golf tables not found. Please run migration 004_add_golf_support.sql"
    fi
    log_success "Golf tables exist"
}

# Test 1: Fetch tournaments
test_fetch_tournaments() {
    log_info "Testing tournament fetch..."
    
    RESPONSE=$(curl -s -X GET "$API_BASE/golf/tournaments")
    
    if echo "$RESPONSE" | jq -e '.tournaments' > /dev/null 2>&1; then
        TOURNAMENT_COUNT=$(echo "$RESPONSE" | jq '.tournaments | length')
        log_success "Fetched $TOURNAMENT_COUNT tournaments"
        
        # Save first tournament ID for later tests
        if [ "$TOURNAMENT_COUNT" -gt 0 ]; then
            TOURNAMENT_ID=$(echo "$RESPONSE" | jq -r '.tournaments[0].id')
            export TOURNAMENT_ID
            log_info "Using tournament ID: $TOURNAMENT_ID"
        fi
    else
        log_error "Failed to fetch tournaments"
    fi
}

# Test 2: Get tournament details
test_get_tournament() {
    log_info "Testing tournament details..."
    
    if [ -z "$TOURNAMENT_ID" ]; then
        log_info "No tournament ID available, skipping..."
        return
    fi
    
    RESPONSE=$(curl -s -X GET "$API_BASE/golf/tournaments/$TOURNAMENT_ID")
    
    if echo "$RESPONSE" | jq -e '.name' > /dev/null 2>&1; then
        TOURNAMENT_NAME=$(echo "$RESPONSE" | jq -r '.name')
        log_success "Retrieved tournament: $TOURNAMENT_NAME"
    else
        log_error "Failed to get tournament details"
    fi
}

# Test 3: Tournament Schedule endpoint
test_tournament_schedule() {
    log_info "Testing tournament schedule endpoint..."
    
    RESPONSE=$(curl -s -X GET "$API_BASE/golf/tournaments/schedule")
    
    if echo "$RESPONSE" | jq -e '.tournaments' > /dev/null 2>&1; then
        TOURNAMENT_COUNT=$(echo "$RESPONSE" | jq '.tournaments | length')
        log_success "Tournament schedule returned $TOURNAMENT_COUNT tournaments"
        
        # Check schedule metadata
        if echo "$RESPONSE" | jq -e '.year' > /dev/null 2>&1 && \
           echo "$RESPONSE" | jq -e '.cached_at' > /dev/null 2>&1 && \
           echo "$RESPONSE" | jq -e '.source' > /dev/null 2>&1; then
            log_success "Tournament schedule includes all metadata"
        else
            log_error "Tournament schedule missing metadata"
        fi
        
        # Check year filter
        CURRENT_YEAR=$(date +%Y)
        RESPONSE_WITH_YEAR=$(curl -s -X GET "$API_BASE/golf/tournaments/schedule?year=$CURRENT_YEAR")
        if echo "$RESPONSE_WITH_YEAR" | jq -e '.year' > /dev/null 2>&1; then
            RETURNED_YEAR=$(echo "$RESPONSE_WITH_YEAR" | jq -r '.year')
            if [ "$RETURNED_YEAR" = "$CURRENT_YEAR" ]; then
                log_success "Year filter working correctly"
            else
                log_error "Year filter returned wrong year: $RETURNED_YEAR"
            fi
        fi
    else
        log_error "Failed to get tournament schedule"
        echo "$RESPONSE" | jq '.'
    fi
}

# Test 4: Create golf contest
test_create_contest() {
    log_info "Testing contest creation..."
    
    CONTEST_DATA='{
        "name": "Test Golf GPP",
        "sport": "golf",
        "platform": "draftkings",
        "contest_type": "gpp",
        "entry_fee": 20,
        "prize_pool": 10000,
        "max_entries": 500,
        "salary_cap": 50000,
        "start_time": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"
    }'
    
    RESPONSE=$(curl -s -X POST "$API_BASE/contests" \
        -H "Content-Type: application/json" \
        -d "$CONTEST_DATA")
    
    if echo "$RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
        CONTEST_ID=$(echo "$RESPONSE" | jq -r '.id')
        export CONTEST_ID
        log_success "Created contest with ID: $CONTEST_ID"
    else
        log_error "Failed to create contest: $RESPONSE"
    fi
}

# Test 4: Get tournament players
test_get_players() {
    log_info "Testing player fetch..."
    
    if [ -z "$TOURNAMENT_ID" ]; then
        log_info "No tournament ID available, skipping..."
        return
    fi
    
    RESPONSE=$(curl -s -X GET "$API_BASE/golf/tournaments/$TOURNAMENT_ID/players?platform=draftkings")
    
    if echo "$RESPONSE" | jq -e '.players' > /dev/null 2>&1; then
        PLAYER_COUNT=$(echo "$RESPONSE" | jq '.players | length')
        log_success "Fetched $PLAYER_COUNT players"
        
        # Verify all players are golfers
        NON_GOLFERS=$(echo "$RESPONSE" | jq '[.players[] | select(.position != "G")] | length')
        if [ "$NON_GOLFERS" -eq 0 ]; then
            log_success "All players have position 'G' (Golfer)"
        else
            log_error "Found $NON_GOLFERS players with incorrect position"
        fi
    else
        log_error "Failed to fetch players"
    fi
}

# Test 5: Get projections
test_get_projections() {
    log_info "Testing projections..."
    
    if [ -z "$TOURNAMENT_ID" ]; then
        log_info "No tournament ID available, skipping..."
        return
    fi
    
    RESPONSE=$(curl -s -X GET "$API_BASE/golf/tournaments/$TOURNAMENT_ID/projections")
    
    if echo "$RESPONSE" | jq -e '.projections' > /dev/null 2>&1; then
        PROJECTION_COUNT=$(echo "$RESPONSE" | jq '.projections | length')
        log_success "Generated projections for $PROJECTION_COUNT players"
        
        # Check projection values
        FIRST_PROJECTION=$(echo "$RESPONSE" | jq '.projections | to_entries[0].value')
        if [ ! -z "$FIRST_PROJECTION" ]; then
            CUT_PROB=$(echo "$FIRST_PROJECTION" | jq '.cut_probability')
            if (( $(echo "$CUT_PROB >= 0 && $CUT_PROB <= 1" | bc -l) )); then
                log_success "Cut probabilities are valid (0-1 range)"
            else
                log_error "Invalid cut probability: $CUT_PROB"
            fi
        fi
    else
        log_error "Failed to get projections"
    fi
}

# Test 6: Optimize lineup
test_optimize_lineup() {
    log_info "Testing lineup optimization..."
    
    if [ -z "$CONTEST_ID" ]; then
        log_info "No contest ID available, skipping..."
        return
    fi
    
    OPTIMIZE_DATA='{
        "contest_id": "'$CONTEST_ID'",
        "num_lineups": 5,
        "constraints": {
            "min_cut_probability": 0.5
        },
        "use_correlations": true
    }'
    
    RESPONSE=$(curl -s -X POST "$API_BASE/optimize" \
        -H "Content-Type: application/json" \
        -d "$OPTIMIZE_DATA")
    
    if echo "$RESPONSE" | jq -e '.lineups' > /dev/null 2>&1; then
        LINEUP_COUNT=$(echo "$RESPONSE" | jq '.lineups | length')
        log_success "Generated $LINEUP_COUNT lineups"
        
        # Validate first lineup
        FIRST_LINEUP=$(echo "$RESPONSE" | jq '.lineups[0]')
        PLAYER_COUNT=$(echo "$FIRST_LINEUP" | jq '.players | length')
        TOTAL_SALARY=$(echo "$FIRST_LINEUP" | jq '.total_salary')
        
        if [ "$PLAYER_COUNT" -eq 6 ]; then
            log_success "Lineup has correct number of players (6)"
        else
            log_error "Lineup has $PLAYER_COUNT players, expected 6"
        fi
        
        if [ "$TOTAL_SALARY" -le 50000 ]; then
            log_success "Lineup meets salary cap ($TOTAL_SALARY <= 50000)"
        else
            log_error "Lineup exceeds salary cap ($TOTAL_SALARY > 50000)"
        fi
    else
        log_error "Failed to optimize lineup: $RESPONSE"
    fi
}

# Test 7: Test correlations
test_correlations() {
    log_info "Testing golf correlations..."
    
    if [ -z "$TOURNAMENT_ID" ]; then
        log_info "No tournament ID available, skipping..."
        return
    fi
    
    RESPONSE=$(curl -s -X GET "$API_BASE/golf/tournaments/$TOURNAMENT_ID/projections")
    
    if echo "$RESPONSE" | jq -e '.correlations' > /dev/null 2>&1; then
        # Check if correlations exist and are in valid range
        CORRELATION_COUNT=$(echo "$RESPONSE" | jq '.correlations | to_entries | length')
        if [ "$CORRELATION_COUNT" -gt 0 ]; then
            log_success "Generated correlations for players"
            
            # Validate correlation values are between -1 and 1
            INVALID_CORR=$(echo "$RESPONSE" | jq '[.correlations | to_entries[] | .value | to_entries[] | select(.value < -1 or .value > 1)] | length')
            if [ "$INVALID_CORR" -eq 0 ]; then
                log_success "All correlations are in valid range [-1, 1]"
            else
                log_error "Found $INVALID_CORR invalid correlation values"
            fi
        fi
    else
        log_info "No correlations found in response"
    fi
}

# Test 8: Database integrity
test_database_integrity() {
    log_info "Testing database integrity..."
    
    # Check foreign key relationships
    ORPHANED_ENTRIES=$(PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "
        SELECT COUNT(*) FROM golf_player_entries 
        WHERE tournament_id NOT IN (SELECT id FROM golf_tournaments)
    ")
    
    if [ "$ORPHANED_ENTRIES" -eq 0 ]; then
        log_success "No orphaned player entries found"
    else
        log_error "Found $ORPHANED_ENTRIES orphaned player entries"
    fi
    
    # Check data consistency
    INVALID_POSITIONS=$(PGPASSWORD=$DB_PASS psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "
        SELECT COUNT(*) FROM players 
        WHERE sport = 'golf' AND position != 'G'
    ")
    
    if [ "$INVALID_POSITIONS" -eq 0 ]; then
        log_success "All golf players have correct position"
    else
        log_error "Found $INVALID_POSITIONS golf players with incorrect position"
    fi
}

# Clean up test data
cleanup() {
    log_info "Cleaning up test data..."
    
    if [ ! -z "$CONTEST_ID" ]; then
        curl -s -X DELETE "$API_BASE/contests/$CONTEST_ID" > /dev/null 2>&1
        log_success "Cleaned up test contest"
    fi
}

# Main execution
main() {
    echo "================================================"
    echo "      Golf Integration Test Suite"
    echo "================================================"
    echo ""
    
    check_prerequisites
    
    echo ""
    echo "Running tests..."
    echo ""
    
    test_fetch_tournaments
    test_get_tournament
    test_tournament_schedule
    test_create_contest
    test_get_players
    test_get_projections
    test_optimize_lineup
    test_correlations
    test_database_integrity
    
    cleanup
    
    echo ""
    echo "================================================"
    log_success "All tests completed successfully! 🏌️"
    echo "================================================"
}

# Run main function
main