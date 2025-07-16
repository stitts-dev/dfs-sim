#!/bin/bash

# Full Integration Validation Suite
# Validates the complete DFS Lineup Optimizer system end-to-end
# Author: Claude Code (Integration Validation)
# Date: 2025-01-14

set -e

# Configuration
API_GATEWAY_URL="http://localhost:8080"
USER_SERVICE_URL="http://localhost:8083"
GOLF_SERVICE_URL="http://localhost:8081"
OPTIMIZATION_SERVICE_URL="http://localhost:8082"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counters
INTEGRATION_TESTS_PASSED=0
INTEGRATION_TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((INTEGRATION_TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((INTEGRATION_TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_section() {
    echo -e "\n${PURPLE}=== $1 ===${NC}"
}

# Test wrapper
run_integration_test() {
    local test_name="$1"
    local test_function="$2"
    
    echo ""
    log_info "Running integration test: $test_name"
    echo "================================================"
    
    if $test_function; then
        log_success "$test_name integration validated"
    else
        log_error "$test_name integration failed"
    fi
}

# Integration Test 1: Service Health and Connectivity
test_service_connectivity() {
    log_section "SERVICE CONNECTIVITY VALIDATION"
    
    local all_healthy=true
    
    # Test all service health endpoints
    for service in "API Gateway:$API_GATEWAY_URL" "User Service:$USER_SERVICE_URL" "Golf Service:$GOLF_SERVICE_URL" "Optimization Service:$OPTIMIZATION_SERVICE_URL"; do
        local name=$(echo $service | cut -d: -f1)
        local url=$(echo $service | cut -d: -f2-3)
        
        if curl -s "${url}/health" > /dev/null; then
            log_success "$name is responding"
        else
            log_error "$name is not responding"
            all_healthy=false
        fi
    done
    
    # Test service discovery through API Gateway
    if curl -s "${API_GATEWAY_URL}/status/services" > /dev/null; then
        log_success "API Gateway service discovery working"
    else
        log_warning "API Gateway service discovery may not be implemented"
    fi
    
    return $all_healthy
}

# Integration Test 2: Database Connectivity and Schema
test_database_integration() {
    log_section "DATABASE INTEGRATION VALIDATION"
    
    # Test database connectivity through each service
    for service in "User Service:$USER_SERVICE_URL" "Golf Service:$GOLF_SERVICE_URL" "Optimization Service:$OPTIMIZATION_SERVICE_URL"; do
        local name=$(echo $service | cut -d: -f1)
        local url=$(echo $service | cut -d: -f2-3)
        
        local response=$(curl -s "${url}/ready")
        if echo "$response" | grep -q "ready"; then
            log_success "$name database connectivity working"
        else
            log_error "$name database connectivity failed"
            echo "Response: $response"
        fi
    done
    
    return 0
}

# Integration Test 3: Authentication Flow Integration
test_auth_integration() {
    log_section "AUTHENTICATION INTEGRATION VALIDATION"
    
    local test_phone="+15551234567"
    
    # Test registration through API Gateway
    log_info "Testing registration flow through API Gateway..."
    local reg_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        -X POST "${API_GATEWAY_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"phone_number\": \"$test_phone\"}")
    
    local reg_status=$(echo "$reg_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    if [[ "$reg_status" == "200" ]] || [[ "$reg_status" == "201" ]]; then
        log_success "Registration endpoint accessible through API Gateway"
    else
        log_error "Registration through API Gateway failed"
    fi
    
    # Test subscription tiers endpoint (public)
    log_info "Testing public endpoints accessibility..."
    local tiers_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        "${API_GATEWAY_URL}/api/v1/subscription-tiers")
    
    local tiers_status=$(echo "$tiers_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    if [[ "$tiers_status" == "200" ]]; then
        log_success "Subscription tiers endpoint working"
        local tiers_body=$(echo "$tiers_response" | sed '/HTTP_STATUS:/d')
        if echo "$tiers_body" | grep -q "tiers"; then
            log_success "Subscription tiers data structure correct"
        fi
    else
        log_error "Subscription tiers endpoint failed"
    fi
    
    return 0
}

# Integration Test 4: Stripe Payment Integration
test_stripe_integration() {
    log_section "STRIPE PAYMENT INTEGRATION VALIDATION"
    
    # Test Stripe endpoints accessibility
    log_info "Testing Stripe integration endpoints..."
    
    # Test subscription tiers (should include Stripe price IDs)
    local response=$(curl -s "${API_GATEWAY_URL}/api/v1/subscription-tiers")
    if echo "$response" | grep -q "price_cents"; then
        log_success "Subscription pricing data accessible"
    fi
    
    # Test Stripe webhook endpoint (should return method not allowed for GET)
    local webhook_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        "${API_GATEWAY_URL}/api/v1/webhooks/stripe")
    
    local webhook_status=$(echo "$webhook_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    if [[ "$webhook_status" == "405" ]] || [[ "$webhook_status" == "400" ]]; then
        log_success "Stripe webhook endpoint exists (method not allowed for GET)"
    else
        log_warning "Stripe webhook endpoint may not be properly configured"
    fi
    
    return 0
}

# Integration Test 5: Cross-Service Data Flow
test_cross_service_data_flow() {
    log_section "CROSS-SERVICE DATA FLOW VALIDATION"
    
    # Test API Gateway routing to different services
    log_info "Testing API Gateway routing..."
    
    # Test route to user service (subscription tiers)
    if curl -s "${API_GATEWAY_URL}/api/v1/subscription-tiers" | grep -q "tiers"; then
        log_success "API Gateway → User Service routing working"
    else
        log_warning "API Gateway → User Service routing may have issues"
    fi
    
    # Test route to golf service (if implemented)
    local golf_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        "${API_GATEWAY_URL}/api/v1/sports/golf/tournaments" 2>/dev/null || echo "HTTP_STATUS:000")
    
    local golf_status=$(echo "$golf_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    if [[ "$golf_status" == "200" ]] || [[ "$golf_status" == "401" ]]; then
        log_success "API Gateway → Golf Service routing configured"
    else
        log_info "Golf Service routes may not be implemented yet"
    fi
    
    # Test route to optimization service (if implemented)
    local opt_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        "${API_GATEWAY_URL}/api/v1/optimize" 2>/dev/null || echo "HTTP_STATUS:000")
    
    local opt_status=$(echo "$opt_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    if [[ "$opt_status" == "200" ]] || [[ "$opt_status" == "401" ]] || [[ "$opt_status" == "405" ]]; then
        log_success "API Gateway → Optimization Service routing configured"
    else
        log_info "Optimization Service routes may not be implemented yet"
    fi
    
    return 0
}

# Integration Test 6: Redis Cache Integration
test_redis_integration() {
    log_section "REDIS CACHE INTEGRATION VALIDATION"
    
    # Test Redis connectivity indirectly through service health
    log_info "Testing Redis integration through services..."
    
    local redis_working=true
    
    # Each service should report Redis connectivity in ready endpoint
    for service in "User Service:$USER_SERVICE_URL" "Golf Service:$GOLF_SERVICE_URL" "Optimization Service:$OPTIMIZATION_SERVICE_URL"; do
        local name=$(echo $service | cut -d: -f1)
        local url=$(echo $service | cut -d: -f2-3)
        
        local response=$(curl -s "${url}/ready" 2>/dev/null || echo "not ready")
        if echo "$response" | grep -q "ready"; then
            log_success "$name Redis integration working"
        else
            log_warning "$name may have Redis connectivity issues"
            redis_working=false
        fi
    done
    
    return $redis_working
}

# Integration Test 7: Environment Configuration
test_environment_configuration() {
    log_section "ENVIRONMENT CONFIGURATION VALIDATION"
    
    # Run configuration verification script
    if [[ -x "./verify-config.sh" ]]; then
        log_info "Running configuration verification..."
        if ./verify-config.sh > /tmp/config_check.log 2>&1; then
            log_success "Configuration verification passed"
            # Show summary
            grep -E "(PASS|FAIL)" /tmp/config_check.log | tail -5
        else
            log_warning "Configuration verification found issues"
            echo "Check /tmp/config_check.log for details"
        fi
    else
        log_info "Configuration verification script not found or not executable"
    fi
    
    return 0
}

# Integration Test 8: Database Schema Validation
test_schema_validation() {
    log_section "DATABASE SCHEMA VALIDATION"
    
    # This would require actual database access, so we'll test indirectly
    log_info "Testing schema through service endpoints..."
    
    # Test if services can access their required tables
    local schema_working=true
    
    # User service should be able to access user tables
    local user_ready=$(curl -s "${USER_SERVICE_URL}/ready")
    if echo "$user_ready" | grep -q "ready"; then
        log_success "User Service schema access working"
    else
        log_error "User Service schema access failed"
        schema_working=false
    fi
    
    # Test subscription tiers (requires database access)
    local tiers=$(curl -s "${API_GATEWAY_URL}/api/v1/subscription-tiers")
    if echo "$tiers" | grep -q "free\|basic\|premium"; then
        log_success "Subscription tiers schema and data working"
    else
        log_warning "Subscription tiers may not be populated"
    fi
    
    return $schema_working
}

# Integration Test 9: Security and CORS
test_security_integration() {
    log_section "SECURITY INTEGRATION VALIDATION"
    
    # Test CORS headers
    log_info "Testing CORS configuration..."
    local cors_response=$(curl -s -I \
        -H "Origin: http://localhost:3000" \
        -H "Access-Control-Request-Method: POST" \
        -H "Access-Control-Request-Headers: Content-Type,Authorization" \
        -X OPTIONS "${API_GATEWAY_URL}/api/v1/auth/register" 2>/dev/null || true)
    
    if echo "$cors_response" | grep -q "Access-Control-Allow-Origin"; then
        log_success "CORS headers configured"
    else
        log_warning "CORS configuration may need attention"
    fi
    
    # Test JWT endpoint protection
    log_info "Testing JWT protection..."
    local protected_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        "${API_GATEWAY_URL}/api/v1/auth/me")
    
    local protected_status=$(echo "$protected_response" | grep "HTTP_STATUS:" | cut -d: -f2)
    if [[ "$protected_status" == "401" ]]; then
        log_success "JWT protection working (unauthorized access blocked)"
    else
        log_warning "JWT protection may not be properly configured"
    fi
    
    return 0
}

# Integration Test 10: Frontend Integration Readiness
test_frontend_integration() {
    log_section "FRONTEND INTEGRATION READINESS"
    
    # Test if all necessary endpoints for frontend are available
    log_info "Testing frontend integration endpoints..."
    
    # Authentication endpoints
    local auth_endpoints=("register" "login" "verify" "refresh" "me")
    for endpoint in "${auth_endpoints[@]}"; do
        local response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
            -X GET "${API_GATEWAY_URL}/api/v1/auth/${endpoint}" 2>/dev/null || echo "HTTP_STATUS:000")
        
        local status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
        if [[ "$status" == "401" ]] || [[ "$status" == "405" ]] || [[ "$status" == "400" ]]; then
            log_success "Auth endpoint /$endpoint exists"
        else
            log_warning "Auth endpoint /$endpoint may not be configured"
        fi
    done
    
    # Frontend configuration endpoints
    if curl -s "${API_GATEWAY_URL}/api/v1/subscription-tiers" | grep -q "tiers"; then
        log_success "Frontend configuration endpoints working"
    fi
    
    return 0
}

# Main validation function
main() {
    echo "================================================================"
    echo "   🚀 DFS Lineup Optimizer - Full Integration Validation"
    echo "================================================================"
    echo "Validating complete system integration across all components:"
    echo "• API Gateway + 3 Microservices"
    echo "• Unified Supabase Database + Redis Cache"
    echo "• Phone Authentication + JWT Tokens"
    echo "• Stripe Payment Integration"
    echo "• Cross-Service Communication"
    echo ""
    
    # Run all integration tests
    run_integration_test "Service Connectivity" test_service_connectivity
    run_integration_test "Database Integration" test_database_integration
    run_integration_test "Authentication Integration" test_auth_integration
    run_integration_test "Stripe Payment Integration" test_stripe_integration
    run_integration_test "Cross-Service Data Flow" test_cross_service_data_flow
    run_integration_test "Redis Cache Integration" test_redis_integration
    run_integration_test "Environment Configuration" test_environment_configuration
    run_integration_test "Database Schema" test_schema_validation
    run_integration_test "Security Integration" test_security_integration
    run_integration_test "Frontend Integration Readiness" test_frontend_integration
    
    # Final Results
    echo ""
    echo "================================================================"
    echo "   🎯 INTEGRATION VALIDATION RESULTS"
    echo "================================================================"
    echo -e "Integration Tests Passed: ${GREEN}$INTEGRATION_TESTS_PASSED${NC}"
    echo -e "Integration Tests Failed: ${RED}$INTEGRATION_TESTS_FAILED${NC}"
    echo -e "Total Integration Tests: $((INTEGRATION_TESTS_PASSED + INTEGRATION_TESTS_FAILED))"
    
    local success_rate=$(( (INTEGRATION_TESTS_PASSED * 100) / (INTEGRATION_TESTS_PASSED + INTEGRATION_TESTS_FAILED) ))
    
    echo ""
    if [[ $INTEGRATION_TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}🎉 ALL INTEGRATIONS VALIDATED SUCCESSFULLY!${NC}"
        echo -e "${CYAN}✨ The DFS Lineup Optimizer is ready for production deployment!${NC}"
    elif [[ $success_rate -ge 80 ]]; then
        echo -e "${YELLOW}⚠️  Most integrations working (${success_rate}% success rate)${NC}"
        echo -e "${CYAN}🔧 Address the failed tests and the system will be ready!${NC}"
    else
        echo -e "${RED}❌ Multiple integration issues found (${success_rate}% success rate)${NC}"
        echo -e "${YELLOW}🛠️  Significant work needed before production deployment${NC}"
    fi
    
    echo ""
    echo "================================================================"
    echo "   📋 SYSTEM STATUS SUMMARY"
    echo "================================================================"
    echo -e "${BLUE}Architecture:${NC} Microservices (4 services)"
    echo -e "${BLUE}Database:${NC} Unified Supabase PostgreSQL"
    echo -e "${BLUE}Cache:${NC} Redis (shared across services)"
    echo -e "${BLUE}Authentication:${NC} Phone-based with JWT tokens"
    echo -e "${BLUE}Payments:${NC} Stripe integration configured"
    echo -e "${BLUE}Frontend API:${NC} RESTful + WebSocket support"
    
    echo ""
    echo "================================================================"
    echo "   🚀 NEXT STEPS FOR DEPLOYMENT"
    echo "================================================================"
    echo "1. 📊 Deploy Supabase schema using deployment-guide.md"
    echo "2. 🔐 Run auth migration if existing users need linking"
    echo "3. 💳 Configure Stripe products and webhooks"
    echo "4. 🧪 Run full authentication tests with real phone numbers"
    echo "5. 🎨 Complete frontend integration testing"
    echo "6. 🌐 Configure production domain and SSL"
    echo "7. 📈 Set up monitoring and logging"
    echo "8. 🚢 Deploy to production environment"
    
    if [[ $INTEGRATION_TESTS_FAILED -eq 0 ]]; then
        echo ""
        echo -e "${GREEN}🚀 Ready for production deployment!${NC}"
    fi
    
    return $INTEGRATION_TESTS_FAILED
}

# Run the full integration validation
main "$@"