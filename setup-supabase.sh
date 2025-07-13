#!/bin/bash

# Supabase Setup Script for DFS Optimizer
# This script helps set up Supabase configuration and run migrations
# Author: Claude Code
# Date: 2025-01-13

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_header() {
    echo -e "${CYAN}🚀 $1${NC}"
    echo "================================================="
}

# Welcome message
clear
log_header "Supabase Setup for DFS Optimizer"
echo ""
log_info "This script will help you set up Supabase for your DFS optimizer."
echo ""

# Step 1: Collect Supabase project information
log_header "Step 1: Supabase Project Information"
echo ""
log_info "First, you need to create a Supabase project at https://supabase.com"
echo ""
log_warning "Make sure you have:"
echo "  1. Created a new Supabase project"
echo "  2. Have your project URL and API keys ready"
echo "  3. Enabled phone authentication in Auth > Providers"
echo ""

read -p "Press Enter when you have created your Supabase project..."
echo ""

# Get project details
log_info "Please enter your Supabase project details:"
echo ""

read -p "🌐 Project URL (https://your-project-id.supabase.co): " SUPABASE_URL
read -p "🔑 Anon Key (eyJ...): " SUPABASE_ANON_KEY
read -s -p "🔐 Service Role Key (eyJ...): " SUPABASE_SERVICE_KEY
echo ""
echo ""

# Validate inputs
if [[ -z "$SUPABASE_URL" || -z "$SUPABASE_ANON_KEY" || -z "$SUPABASE_SERVICE_KEY" ]]; then
    log_error "All fields are required. Please run the script again."
    exit 1
fi

# Validate URL format
if [[ ! "$SUPABASE_URL" =~ ^https://.*\.supabase\.co$ ]]; then
    log_error "Invalid Supabase URL format. Should be: https://your-project-id.supabase.co"
    exit 1
fi

log_success "Project details collected!"
echo ""

# Step 2: Create environment files
log_header "Step 2: Creating Environment Files"
echo ""

# Backend .env
log_info "Creating backend environment file..."
cat > backend/.env << EOF
# Supabase Configuration
SUPABASE_URL=$SUPABASE_URL
SUPABASE_SERVICE_KEY=$SUPABASE_SERVICE_KEY
SUPABASE_ANON_KEY=$SUPABASE_ANON_KEY

# Database Configuration
DATABASE_URL=postgresql://postgres:[password]@db.[project-id].supabase.co:5432/postgres

# Existing Configuration (update as needed)
JWT_SECRET=your_existing_jwt_secret_for_fallback
PORT=8080
CORS_ORIGINS=http://localhost:3000,http://localhost:5173

# SMS Configuration (choose one)
SMS_PROVIDER=supabase
# If using Twilio as backup:
# TWILIO_ACCOUNT_SID=your_twilio_account_sid
# TWILIO_AUTH_TOKEN=your_twilio_auth_token
# TWILIO_FROM_NUMBER=+1234567890

# Environment
ENV=development
EOF

# Frontend .env
log_info "Creating frontend environment file..."
cat > frontend/.env << EOF
# Supabase Configuration
VITE_SUPABASE_URL=$SUPABASE_URL
VITE_SUPABASE_ANON_KEY=$SUPABASE_ANON_KEY

# API Configuration
VITE_API_BASE_URL=http://localhost:8080/api/v1

# Environment
VITE_ENV=development
EOF

log_success "Environment files created!"
echo ""

# Step 3: Database migration instructions
log_header "Step 3: Database Migration"
echo ""
log_info "Now you need to run the database migration in your Supabase dashboard:"
echo ""
echo "1. Go to your Supabase project dashboard"
echo "2. Navigate to 'SQL Editor'"
echo "3. Click 'New Query'"
echo "4. Copy and paste the migration script"
echo "5. Click 'Run' to execute the migration"
echo ""

# Show migration file path
log_info "Migration file location:"
echo "📄 $(pwd)/supabase/migrations/001_initial_user_schema_fixed.sql"
echo ""

# Ask if they want to view the migration
read -p "Would you like to view the migration script now? (y/n): " view_migration
if [[ "$view_migration" =~ ^[Yy]$ ]]; then
    echo ""
    log_info "Migration script content:"
    echo "----------------------------------------"
    cat supabase/migrations/001_initial_user_schema_fixed.sql
    echo "----------------------------------------"
    echo ""
fi

read -p "Press Enter after you have successfully run the migration in Supabase..."
echo ""

# Step 4: Verify setup
log_header "Step 4: Setup Verification"
echo ""

log_info "Testing Supabase connection..."

# Test if we can reach Supabase
if curl -s -f "$SUPABASE_URL/rest/v1/" -H "apikey: $SUPABASE_ANON_KEY" > /dev/null; then
    log_success "✓ Supabase API is reachable"
else
    log_error "✗ Cannot reach Supabase API. Check your URL and keys."
fi

# Step 5: Phone authentication setup reminder
log_header "Step 5: Phone Authentication Setup"
echo ""
log_warning "Don't forget to configure phone authentication:"
echo ""
echo "1. Go to Authentication > Providers in your Supabase dashboard"
echo "2. Enable the Phone provider"
echo "3. Configure your SMS provider (Twilio, MessageBird, etc.)"
echo "4. Set up your phone number and credentials"
echo ""
log_info "Supported SMS providers:"
echo "  • Twilio (recommended for production)"
echo "  • MessageBird"
echo "  • Textlocal"
echo "  • Vonage"
echo ""

# Step 6: Testing
log_header "Step 6: Testing Your Setup"
echo ""
log_info "You can now test your setup:"
echo ""
echo "1. Start your backend server:"
echo "   cd backend && go run cmd/server/main.go"
echo ""
echo "2. Start your frontend:"
echo "   cd frontend && npm run dev"
echo ""
echo "3. Run the integration test:"
echo "   ./test-supabase-migration.sh"
echo ""

# Summary
log_header "Setup Complete! 🎉"
echo ""
log_success "Your Supabase integration is ready!"
echo ""
log_info "Files created:"
echo "  ✓ backend/.env - Backend environment variables"
echo "  ✓ frontend/.env - Frontend environment variables"
echo ""
log_info "Next steps:"
echo "  1. Configure SMS provider in Supabase dashboard"
echo "  2. Test phone authentication with a real phone number"
echo "  3. Deploy to staging environment"
echo "  4. Set up monitoring for real-time connections"
echo ""
log_warning "Security reminders:"
echo "  • Never commit .env files to version control"
echo "  • Keep your service role key secret"
echo "  • Use different projects for development/production"
echo "  • Set up proper CORS origins for production"
echo ""

log_success "Setup script completed successfully!"
exit 0