name: "Fix Missing AI Recommendations Table - Database Migration Issue"
description: |

## Purpose
Fix the missing `ai_recommendations` table in the PostgreSQL database that's causing AI recommendation feature failures. This PRP provides comprehensive context and validation loops to ensure the table is properly created and the AI recommendations feature works correctly.

## Core Principles
1. **Context is King**: Include ALL necessary documentation, examples, and caveats about GORM AutoMigrate
2. **Validation Loops**: Provide executable tests the AI can run to verify the fix
3. **Information Dense**: Use patterns from existing migration files in the codebase
4. **Progressive Success**: Start with table creation, validate, then test the feature
5. **Global rules**: Follow all rules in CLAUDE.md for Go backend development

---

## Goal
Create the missing `ai_recommendations` table in the PostgreSQL database and ensure the AI recommendations feature works properly without "relation does not exist" errors.

## Why
- **User Impact**: Users cannot get AI-generated player recommendations due to database errors
- **Feature Integration**: AI recommendations are a core feature of the DFS optimizer
- **Data Analytics**: The table stores recommendation history for improving AI performance
- **Business Value**: AI recommendations help users make better lineup decisions

## What
Fix the database schema issue where GORM AutoMigrate failed to create the `ai_recommendations` table despite the model being defined and included in the migration process.

### Success Criteria
- [ ] `ai_recommendations` table exists in the database with proper schema
- [ ] All required indexes are created for performance
- [ ] AI recommendations API endpoint works without database errors
- [ ] User can successfully request AI player recommendations
- [ ] Recommendation history is properly stored in the database

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://gorm.io/docs/migration.html
  why: Official GORM AutoMigrate documentation and troubleshooting
  section: Common migration issues and solutions
  
- url: https://github.com/go-gorm/gorm/issues/4164
  why: Foreign key constraint creation issues in GORM
  critical: GORM silently fails to create foreign keys in some cases
  
- url: https://gorm.io/docs/constraints.html
  why: Understanding GORM foreign key constraints
  section: OnDelete and OnUpdate constraint handling
  
- file: /Users/jstittsworth/fun/backend/migrations/003_add_ai_recommendations.sql
  why: Exact SQL schema that should be created
  
- file: /Users/jstittsworth/fun/backend/internal/models/metadata.go:45-57
  why: AIRecommendation model definition with GORM tags
  
- file: /Users/jstittsworth/fun/backend/cmd/migrate/main.go:76
  why: Migration tool that includes AIRecommendation in AutoMigrate
  
- file: /Users/jstittsworth/fun/backend/internal/services/ai_recommendations.go:188
  why: Where the database error occurs during INSERT operations
  
- file: /Users/jstittsworth/fun/backend/migrations/002_user_preferences.sql
  why: Pattern for creating tables with proper indexes and triggers
  
- file: /Users/jstittsworth/fun/backend/migrations/008_add_contest_discovery_fields.sql
  why: Pattern for adding columns and indexes to existing tables
```

### Current Codebase Tree (relevant sections)
```bash
backend/
├── cmd/
│   └── migrate/main.go              # Migration tool with AutoMigrate
├── internal/
│   ├── models/
│   │   └── metadata.go              # AIRecommendation model definition
│   └── services/
│       └── ai_recommendations.go    # Service that fails with DB error
├── migrations/
│   ├── 002_user_preferences.sql     # Example working migration
│   ├── 003_add_ai_recommendations.sql # Target migration SQL
│   └── 008_add_contest_discovery_fields.sql # Recent working migration
└── tests/
    └── MANUAL_TEST_CHECKLIST.md    # Testing procedures
```

### Desired State After Fix
```bash
# Database should contain:
- ai_recommendations table with proper schema
- All required indexes for performance
- Foreign key constraint to contests table
- Proper GORM model mapping working

# Files that may need updates:
- No code changes needed (model already exists)
- Only database schema needs fixing
```

### Known Gotchas of GORM & PostgreSQL
```go
// CRITICAL: GORM AutoMigrate issues that cause table creation failures
// 1. Foreign key constraint creation can fail silently
// 2. Migration order matters - referenced tables must exist first
// 3. GORM may skip table creation if foreign key references don't exist
// 4. PostgreSQL foreign key constraints must match exact column types

// GORM AutoMigrate behavior:
// - Creates tables, missing columns, foreign keys, constraints, indexes
// - WON'T delete unused columns to protect data
// - May fail silently on foreign key constraint issues
// - Requires proper migration order for referenced tables

// Common failure patterns:
// - Table not created when foreign key references non-existent table
// - Silent failures with no error messages
// - GORM config issues preventing table creation
```

## Implementation Blueprint

### Root Cause Analysis
The `ai_recommendations` table was not created by GORM AutoMigrate despite:
1. Model definition exists in `internal/models/metadata.go`
2. Migration includes `&models.AIRecommendation{}` in AutoMigrate call
3. SQL migration file exists with proper schema

Likely causes:
- Foreign key constraint to `contests` table creation failed
- GORM silently failed during AutoMigrate
- Migration order issues during initial setup

### Task List (In Order)
```yaml
Task 1: Verify Database State
  - Check current database tables and confirm ai_recommendations is missing
  - Verify contests table exists (required for foreign key)
  - Check migration history and AutoMigrate logs
  
Task 2: Create AI Recommendations Table Manually
  - Execute SQL from 003_add_ai_recommendations.sql directly
  - Create all required indexes
  - Verify table structure matches GORM model
  
Task 3: Test GORM Model Mapping
  - Run simple database operations with AIRecommendation model
  - Verify foreign key constraint works properly
  - Test CRUD operations
  
Task 4: Test AI Recommendations Service
  - Test the failing INSERT operation from ai_recommendations.go:188
  - Verify service can store recommendations
  - Test API endpoint end-to-end
  
Task 5: Update Migration Process (If Needed)
  - Investigate why AutoMigrate failed
  - Update migration tool if necessary
  - Document the fix for future reference
```

### Per Task Implementation Details

#### Task 1: Database State Verification
```sql
-- Check if ai_recommendations table exists
SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename = 'ai_recommendations';

-- Check if contests table exists (required for foreign key)
SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename = 'contests';

-- Check table structure if it exists
\d ai_recommendations
```

#### Task 2: Manual Table Creation
```sql
-- Execute the exact SQL from 003_add_ai_recommendations.sql
-- PATTERN: Use IF NOT EXISTS to avoid conflicts
CREATE TABLE IF NOT EXISTS ai_recommendations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    contest_id INTEGER REFERENCES contests(id) ON DELETE SET NULL,
    request JSONB NOT NULL,
    response JSONB NOT NULL,
    confidence DOUBLE PRECISION DEFAULT 0.0,
    was_used BOOLEAN DEFAULT FALSE,
    lineup_result DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create all indexes for performance
CREATE INDEX idx_ai_recommendations_user_id ON ai_recommendations(user_id);
CREATE INDEX idx_ai_recommendations_contest_id ON ai_recommendations(contest_id);
CREATE INDEX idx_ai_recommendations_created_at ON ai_recommendations(created_at DESC);
CREATE INDEX idx_ai_recommendations_confidence ON ai_recommendations(confidence);
```

#### Task 3: GORM Model Testing
```go
// Test basic GORM operations
func TestAIRecommendationModel(t *testing.T) {
    // Create test record
    aiRec := models.AIRecommendation{
        UserID:     1,
        ContestID:  1,
        Request:    datatypes.JSON(`{"test": "data"}`),
        Response:   datatypes.JSON(`{"result": "success"}`),
        Confidence: 0.85,
    }
    
    // Test Create
    err := db.Create(&aiRec).Error
    assert.NoError(t, err)
    
    // Test Read
    var retrieved models.AIRecommendation
    err = db.First(&retrieved, aiRec.ID).Error
    assert.NoError(t, err)
    
    // Test foreign key constraint
    err = db.Model(&retrieved).Association("Contest").Find(&models.Contest{})
    assert.NoError(t, err)
}
```

### Integration Points
```yaml
DATABASE:
  - table: ai_recommendations
  - foreign_key: contest_id -> contests(id) ON DELETE SET NULL
  - indexes: user_id, contest_id, created_at DESC, confidence
  
SERVICE:
  - file: internal/services/ai_recommendations.go:188
  - operation: db.Create(&aiRec).Error
  - pattern: Store recommendation history for analytics
  
API:
  - endpoint: POST /api/v1/ai/recommend-players
  - handler: internal/api/handlers/ai_recommendations.go
  - pattern: Return recommendations and store history
```

## Validation Loop

### Level 1: Database Structure
```bash
# Verify table exists and has correct structure
docker exec -i dfs_postgres psql -U postgres -d dfs_optimizer -c "\d ai_recommendations"

# Expected: Table with all columns matching the model
# If error: Table doesn't exist or has wrong structure
```

### Level 2: GORM Model Testing
```bash
# Navigate to backend directory
cd backend

# Run specific test for AI recommendations model
go test -run TestAIRecommendationModel ./internal/models/...

# Create simple test file if needed
# Expected: No errors, successful CRUD operations
```

### Level 3: Service Integration Test
```bash
# Test the AI recommendations service directly
go test -run TestAIRecommendationService ./internal/services/...

# Test database operations work
# Expected: No "relation does not exist" errors
```

### Level 4: API Endpoint Test
```bash
# Start the backend server
go run cmd/server/main.go

# Test the AI recommendations endpoint
curl -X POST http://localhost:8080/api/v1/ai/recommend-players \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 3,
    "contest_type": "GPP",
    "sport": "golf",
    "remaining_budget": 50000,
    "current_lineup": [],
    "positions_needed": ["G","G","G","G","G","G"],
    "beginner_mode": false,
    "optimize_for": "ceiling"
  }'

# Expected: JSON response with recommendations, no database errors
# Check logs for "Failed to store AI recommendation" errors
```

### Level 5: Database Verification
```bash
# Check that recommendations are being stored
docker exec -i dfs_postgres psql -U postgres -d dfs_optimizer -c "SELECT COUNT(*) FROM ai_recommendations;"

# Expected: Count > 0 after API test
# Check recent entries
docker exec -i dfs_postgres psql -U postgres -d dfs_optimizer -c "SELECT id, user_id, contest_id, confidence, created_at FROM ai_recommendations ORDER BY created_at DESC LIMIT 5;"
```

## Final Validation Checklist
- [ ] Table exists: `\d ai_recommendations` shows correct schema
- [ ] All indexes created: Check pg_indexes for ai_recommendations
- [ ] Foreign key constraint works: Can reference contests table
- [ ] GORM model operations work: No ORM errors
- [ ] Service layer works: No "relation does not exist" errors
- [ ] API endpoint works: Returns recommendations successfully
- [ ] Data persistence: Recommendations stored in database
- [ ] No regressions: Other features still work

## Anti-Patterns to Avoid
- ❌ Don't drop and recreate the table if it has data
- ❌ Don't ignore foreign key constraint failures
- ❌ Don't skip index creation for performance
- ❌ Don't assume AutoMigrate will fix itself
- ❌ Don't modify the model structure without updating SQL
- ❌ Don't forget to restart services after database changes

## Success Metrics
- Zero "relation 'ai_recommendations' does not exist" errors
- AI recommendations API returns successful responses
- Database contains stored recommendation history
- All validation tests pass
- Service logs show successful recommendation storage

---

**PRP Quality Score: 9/10**

This PRP provides comprehensive context about GORM AutoMigrate issues, exact SQL schemas, validation loops, and step-by-step implementation guidance. The AI agent has all necessary information to diagnose and fix the missing table issue in one pass.