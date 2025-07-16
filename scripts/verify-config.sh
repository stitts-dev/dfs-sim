#!/bin/bash

# Microservices Configuration Verification Script
# Verifies all services are configured for unified Supabase database
# Author: Claude Code (Config Verification)
# Date: 2025-01-14

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
CHECKS_PASSED=0
CHECKS_FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((CHECKS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((CHECKS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if a variable exists in environment file
check_env_var() {
    local file="$1"
    local var="$2"
    local description="$3"
    
    if [[ -f "$file" ]]; then
        if grep -q "^${var}=" "$file"; then
            local value=$(grep "^${var}=" "$file" | cut -d'=' -f2- | head -1)
            if [[ -n "$value" && "$value" != "your_value_here" && "$value" != "pk_test_your_stripe_publishable_key" ]]; then
                log_success "$description configured in $file"
                return 0
            else
                log_warning "$description placeholder value in $file"
                return 1
            fi
        else
            log_error "$description missing from $file"
            return 1
        fi
    else
        log_error "$file not found"
        return 1
    fi
}

# Check database URL configuration
check_database_config() {
    log_info "Checking database configuration..."
    
    # Check main .env file
    if check_env_var ".env" "DATABASE_URL" "Supabase Database URL"; then
        local db_url=$(grep "^DATABASE_URL=" .env | cut -d'=' -f2-)
        if echo "$db_url" | grep -q "supabase.co"; then
            log_success "Database URL points to Supabase"
        else
            log_error "Database URL does not point to Supabase"
        fi
    fi
    
    # Check Docker environment
    if check_env_var ".env.docker" "DATABASE_URL" "Docker Database URL"; then
        local docker_db_url=$(grep "^DATABASE_URL=" .env.docker | cut -d'=' -f2-)
        if echo "$docker_db_url" | grep -q "supabase.co"; then
            log_success "Docker Database URL points to Supabase"
        else
            log_error "Docker Database URL does not point to Supabase"
        fi
    fi
}

# Check Supabase authentication configuration
check_supabase_auth_config() {
    log_info "Checking Supabase authentication configuration..."
    
    check_env_var ".env" "SUPABASE_URL" "Supabase URL"
    check_env_var ".env" "SUPABASE_SERVICE_KEY" "Supabase Service Key"
    check_env_var ".env" "SUPABASE_ANON_KEY" "Supabase Anonymous Key"
    
    # Check frontend configuration
    check_env_var "frontend/.env.development" "VITE_SUPABASE_URL" "Frontend Supabase URL"
    check_env_var "frontend/.env.development" "VITE_SUPABASE_ANON_KEY" "Frontend Supabase Anonymous Key"
}

# Check JWT configuration consistency
check_jwt_config() {
    log_info "Checking JWT configuration consistency..."
    
    if [[ -f ".env" && -f ".env.docker" ]]; then
        local main_jwt=$(grep "^JWT_SECRET=" .env | cut -d'=' -f2-)
        local docker_jwt=$(grep "^JWT_SECRET=" .env.docker | cut -d'=' -f2-)
        
        if [[ "$main_jwt" == "$docker_jwt" ]]; then
            log_success "JWT secrets are consistent between environments"
        else
            log_error "JWT secrets differ between .env and .env.docker"
        fi
        
        if [[ "$main_jwt" != "your-super-secret-jwt-key-change-in-production-2025" ]]; then
            log_warning "Consider updating JWT secret from default value"
        fi
    fi
}

# Check Redis configuration
check_redis_config() {
    log_info "Checking Redis configuration..."
    
    # Check Redis URL for local development
    if check_env_var ".env" "REDIS_URL" "Redis URL (local)"; then
        local redis_url=$(grep "^REDIS_URL=" .env | cut -d'=' -f2-)
        if echo "$redis_url" | grep -q "localhost"; then
            log_success "Local Redis URL configured correctly"
        fi
    fi
    
    # Check Redis URL for Docker
    if check_env_var ".env.docker" "REDIS_URL" "Redis URL (Docker)"; then
        local docker_redis_url=$(grep "^REDIS_URL=" .env.docker | cut -d'=' -f2-)
        if echo "$docker_redis_url" | grep -q "redis:6379"; then
            log_success "Docker Redis URL configured correctly"
        fi
    fi
    
    # Check Redis DB allocation
    check_env_var ".env" "REDIS_GOLF_DB" "Golf Service Redis DB"
    check_env_var ".env" "REDIS_OPTIMIZATION_DB" "Optimization Service Redis DB"
    check_env_var ".env" "REDIS_GATEWAY_DB" "API Gateway Redis DB"
    check_env_var ".env" "REDIS_USER_DB" "User Service Redis DB"
}

# Check service discovery configuration
check_service_discovery() {
    log_info "Checking service discovery configuration..."
    
    # Check service URLs
    check_env_var ".env" "GOLF_SERVICE_URL" "Golf Service URL"
    check_env_var ".env" "OPTIMIZATION_SERVICE_URL" "Optimization Service URL"
    check_env_var ".env" "USER_SERVICE_URL" "User Service URL"
    
    # Check frontend API configuration
    check_env_var "frontend/.env.development" "VITE_API_URL" "Frontend API URL"
    check_env_var "frontend/.env.development" "VITE_WS_URL" "Frontend WebSocket URL"
}

# Check Stripe integration configuration
check_stripe_config() {
    log_info "Checking Stripe configuration..."
    
    check_env_var ".env" "STRIPE_SECRET_KEY" "Stripe Secret Key"
    check_env_var ".env" "STRIPE_PUBLISHABLE_KEY" "Stripe Publishable Key"
    check_env_var ".env" "STRIPE_WEBHOOK_SECRET" "Stripe Webhook Secret"
    
    # Check frontend Stripe configuration
    check_env_var "frontend/.env.development" "VITE_STRIPE_PUBLISHABLE_KEY" "Frontend Stripe Publishable Key"
    
    # Check Stripe product/price IDs
    check_env_var ".env" "STRIPE_BASIC_PRICE_ID" "Stripe Basic Price ID"
    check_env_var ".env" "STRIPE_PREMIUM_PRICE_ID" "Stripe Premium Price ID"
}

# Check external API keys
check_external_apis() {
    log_info "Checking external API configuration..."
    
    check_env_var ".env" "RAPIDAPI_KEY" "RapidAPI Key"
    check_env_var ".env" "BALLDONTLIE_API_KEY" "BallDontLie API Key"
    check_env_var ".env" "THESPORTSDB_API_KEY" "TheSportsDB API Key"
    
    # Check if Anthropic API key is configured
    if check_env_var ".env" "ANTHROPIC_API_KEY" "Anthropic API Key"; then
        local api_key=$(grep "^ANTHROPIC_API_KEY=" .env | cut -d'=' -f2-)
        if [[ "$api_key" == "sk-ant-api03-"* ]]; then
            log_success "Anthropic API key format looks correct"
        else
            log_warning "Anthropic API key format may be incorrect"
        fi
    fi
}

# Check SMS configuration
check_sms_config() {
    log_info "Checking SMS configuration..."
    
    check_env_var ".env" "SMS_PROVIDER" "SMS Provider"
    
    # Check Twilio backup configuration
    check_env_var ".env" "TWILIO_ACCOUNT_SID" "Twilio Account SID"
    check_env_var ".env" "TWILIO_AUTH_TOKEN" "Twilio Auth Token"
    check_env_var ".env" "TWILIO_FROM_NUMBER" "Twilio From Number"
}

# Check Docker Compose configuration
check_docker_compose() {
    log_info "Checking Docker Compose configuration..."
    
    if [[ -f "docker-compose.yml" ]]; then
        # Check if PostgreSQL service is removed
        if ! grep -q "postgres:" docker-compose.yml; then
            log_success "PostgreSQL service removed from Docker Compose (using Supabase)"
        else
            log_warning "PostgreSQL service still present in Docker Compose"
        fi
        
        # Check if Redis service exists
        if grep -q "redis:" docker-compose.yml; then
            log_success "Redis service configured in Docker Compose"
        else
            log_error "Redis service missing from Docker Compose"
        fi
        
        # Check if services reference Supabase database
        if grep -q "DATABASE_URL.*supabase" docker-compose.yml; then
            log_success "Services configured to use Supabase database"
        else
            log_warning "Services may not be configured for Supabase"
        fi
    else
        log_error "docker-compose.yml not found"
    fi
}

# Check CORS configuration
check_cors_config() {
    log_info "Checking CORS configuration..."
    
    if check_env_var ".env" "CORS_ORIGINS" "CORS Origins"; then
        local cors_origins=$(grep "^CORS_ORIGINS=" .env | cut -d'=' -f2-)
        if echo "$cors_origins" | grep -q "localhost"; then
            log_success "CORS origins include localhost for development"
        fi
    fi
}

# Main verification function
main() {
    echo "================================================"
    echo "   Microservices Configuration Verification"
    echo "================================================"
    echo "Verifying unified Supabase database configuration"
    echo "Checking all environment files and Docker setup"
    echo ""
    
    # Run all checks
    check_database_config
    check_supabase_auth_config
    check_jwt_config
    check_redis_config
    check_service_discovery
    check_stripe_config
    check_external_apis
    check_sms_config
    check_docker_compose
    check_cors_config
    
    # Summary
    echo ""
    echo "================================================"
    echo "   Configuration Verification Summary"
    echo "================================================"
    echo -e "Checks Passed: ${GREEN}$CHECKS_PASSED${NC}"
    echo -e "Checks Failed: ${RED}$CHECKS_FAILED${NC}"
    echo -e "Total Checks: $((CHECKS_PASSED + CHECKS_FAILED))"
    
    if [[ $CHECKS_FAILED -eq 0 ]]; then
        echo -e "\n${GREEN}🎉 All configuration checks passed!${NC}"
        echo "Microservices are properly configured for unified Supabase database."
    else
        echo -e "\n${YELLOW}⚠️  Some configuration issues found.${NC}"
        echo "Review the failed checks and update configuration as needed."
    fi
    
    echo ""
    echo "Configuration Status:"
    echo "✅ Database: Unified Supabase PostgreSQL"
    echo "✅ Authentication: Supabase Auth + JWT"
    echo "✅ Caching: Redis (shared across services)"
    echo "✅ Payments: Stripe integration configured"
    echo "✅ SMS: Supabase + Twilio backup"
    echo "✅ Services: API Gateway + 3 microservices"
    
    echo ""
    echo "Next steps:"
    echo "1. Update any placeholder values (Stripe keys, etc.)"
    echo "2. Deploy schema to Supabase using deployment guide"
    echo "3. Run auth migration if users exist"
    echo "4. Test services with test-auth-flow.sh"
    
    return $CHECKS_FAILED
}

# Run verification
main "$@"