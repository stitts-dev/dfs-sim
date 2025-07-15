## FEATURE:

Complete User Authentication and Data Migration to Supabase

## CONTEXT & MOTIVATION:

The current DFS optimizer uses a custom phone-based authentication system with PostgreSQL user management. To improve scalability, reduce infrastructure complexity, and leverage real-time features, we need to migrate all user-related functionality to Supabase. This migration will provide built-in authentication, real-time user interactions, automatic scaling, and simplified user management while maintaining existing phone-based OTP verification.

**PARALLEL DEVELOPMENT CONTEXT:**
This PRD focuses exclusively on user service migration while golf services are being extracted in parallel (INITIAL_microservices_golf_extraction.md). The migration is designed to be independent and non-blocking to golf service development, with integration happening after both workstreams complete.

**MIGRATION STRATEGY:**
Since no production users exist yet, this is a greenfield migration allowing clean schema design without complex data migration or user ID mapping concerns.

## EXAMPLES:

- Phone number registration sending OTP via Supabase Auth
- User preferences synced in real-time across devices
- Subscription tier management with automatic access control
- Lineup favorites and user settings persisted in Supabase
- Row Level Security automatically isolating user data

## CURRENT STATE ANALYSIS:

**Existing Authentication System:**
- Custom phone-based OTP authentication in `backend/internal/api/handlers/auth.go`
- User models in `backend/internal/models/user.go` with subscription tiers
- Frontend auth store in `frontend/src/store/auth.ts` with hybrid Supabase/backend support
- Database tables: users, user_preferences, subscription_tiers, phone_verification_codes

**Components to Migrate:**
- User authentication and session management
- User preferences and settings storage
- Subscription tier and billing management
- User-owned lineup metadata
- Phone verification and OTP sending

**Current Limitations:**
- Custom JWT implementation requiring maintenance
- Manual user session management
- No real-time user data synchronization
- Complex phone verification flow
- Infrastructure overhead for user management

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [ ] Replace custom auth handlers with Supabase JWT verification
- [ ] Migrate user models to use Supabase UUIDs
- [ ] Create user service with Supabase integration
- [ ] Update user preferences management
- [ ] Implement subscription tier logic with Supabase
- [ ] Remove custom phone verification system
- [ ] Update all user_id references to UUID format

### Frontend Requirements:
- [ ] Simplify auth store to use Supabase exclusively
- [ ] Remove backend API fallback authentication
- [ ] Update user preference management UI
- [ ] Implement real-time user data subscriptions
- [ ] Test phone authentication end-to-end
- [ ] Maintain existing user flow without breaking changes

### Database Requirements:
- [ ] Create complete Supabase schema for user data
- [ ] Design Row Level Security (RLS) policies
- [ ] Set up real-time subscriptions for user data
- [ ] Create database functions for subscription management
- [ ] Implement automatic user onboarding triggers
- [ ] Design user data backup and export procedures

## IMPLEMENTATION APPROACH:

### Phase 1: Supabase Schema Design (2 hours)
- Create users table with proper UUID primary keys
- Design user_preferences table with JSONB storage
- Set up subscription_tiers and user_subscriptions tables
- Implement Row Level Security policies
- Create database functions and triggers

### Phase 2: Backend User Service (3 hours)
- Create new user service with Supabase client
- Implement Supabase JWT verification middleware
- Update user models to use UUID format
- Create user preference management endpoints
- Implement subscription tier logic
- Add user onboarding and profile management

### Phase 3: Frontend Integration (2 hours)
- Update auth store to use Supabase exclusively
- Implement real-time user data subscriptions
- Update user preference components
- Test phone authentication flow
- Add subscription management UI
- Ensure backward compatibility

### Phase 4: Testing & Validation (1 hour)
- Test complete phone OTP authentication flow
- Verify user preferences persistence and real-time sync
- Validate subscription tier access controls
- Test user onboarding and profile management
- End-to-end testing of user workflows

## SUPABASE SCHEMA DESIGN:

### Core Tables:
```sql
-- Users table (leverages Supabase auth.users)
CREATE TABLE public.users (
  id UUID REFERENCES auth.users PRIMARY KEY,
  phone_number TEXT UNIQUE NOT NULL,
  first_name TEXT,
  last_name TEXT,
  subscription_tier TEXT DEFAULT 'free',
  subscription_status TEXT DEFAULT 'active',
  subscription_expires_at TIMESTAMPTZ,
  stripe_customer_id TEXT,
  monthly_optimizations_used INTEGER DEFAULT 0,
  monthly_simulations_used INTEGER DEFAULT 0,
  usage_reset_date DATE DEFAULT CURRENT_DATE,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User preferences with JSONB storage
CREATE TABLE public.user_preferences (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID REFERENCES public.users(id) NOT NULL,
  sport_preferences JSONB DEFAULT '["nba", "nfl", "mlb", "golf"]',
  platform_preferences JSONB DEFAULT '["draftkings", "fanduel"]',
  contest_type_preferences JSONB DEFAULT '["gpp", "cash"]',
  theme TEXT DEFAULT 'light',
  language TEXT DEFAULT 'en',
  notifications_enabled BOOLEAN DEFAULT true,
  tutorial_completed BOOLEAN DEFAULT false,
  beginner_mode BOOLEAN DEFAULT true,
  tooltips_enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Subscription tiers configuration
CREATE TABLE public.subscription_tiers (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  price_cents INTEGER NOT NULL DEFAULT 0,
  currency TEXT DEFAULT 'USD',
  monthly_optimizations INTEGER DEFAULT 10,
  monthly_simulations INTEGER DEFAULT 5,
  ai_recommendations BOOLEAN DEFAULT false,
  bank_verification BOOLEAN DEFAULT false,
  priority_support BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Row Level Security Policies:
```sql
-- Users can only access their own data
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Users can view own profile" ON public.users
  FOR SELECT USING (auth.uid() = id);
CREATE POLICY "Users can update own profile" ON public.users
  FOR UPDATE USING (auth.uid() = id);

-- User preferences access control
ALTER TABLE public.user_preferences ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Users can manage own preferences" ON public.user_preferences
  FOR ALL USING (auth.uid() = user_id);

-- Subscription tiers are publicly readable
ALTER TABLE public.subscription_tiers ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Subscription tiers are publicly readable" ON public.subscription_tiers
  FOR SELECT USING (true);
```

## DOCUMENTATION:

- Supabase project configuration and environment variables
- User authentication flow documentation
- Phone verification setup with Supabase Auth
- Row Level Security policy explanations
- User service API documentation
- Frontend auth integration guide
- User preference management guide
- Subscription tier configuration
- Real-time subscription setup
- User onboarding flow documentation

## TESTING STRATEGY:

### Unit Tests:
- [ ] User service CRUD operations
- [ ] Supabase JWT verification
- [ ] User preference management
- [ ] Subscription tier logic
- [ ] Phone authentication flow

### Integration Tests:
- [ ] Supabase Auth integration
- [ ] User data persistence
- [ ] Real-time subscription updates
- [ ] Row Level Security enforcement
- [ ] Phone OTP verification

### E2E Tests:
- [ ] Complete user registration flow
- [ ] Phone authentication and login
- [ ] User preference management
- [ ] Subscription tier access controls
- [ ] Real-time user data synchronization

## POTENTIAL CHALLENGES & RISKS:

**Technical Challenges:**
- UUID format conversion throughout codebase
- Supabase Auth configuration complexity
- Real-time subscription management
- Phone authentication setup with Supabase

**Dependencies:**
- Supabase project configuration and API keys
- Phone authentication provider setup
- Frontend Supabase client configuration
- Row Level Security policy validation

**Migration Concerns:**
- User model format changes affecting other services
- Authentication token validation changes
- Frontend auth flow modifications
- Database schema migration coordination

**Performance Considerations:**
- Real-time subscription overhead
- Supabase API rate limiting
- User data synchronization latency
- Phone OTP delivery reliability

## SUCCESS CRITERIA:

- [ ] Phone authentication works seamlessly with Supabase
- [ ] User preferences persist and sync in real-time
- [ ] Subscription tier management functional
- [ ] All user data properly isolated via RLS
- [ ] User registration completes in <3 seconds
- [ ] Frontend auth flows unchanged from user perspective
- [ ] User service ready for integration with golf services
- [ ] No custom authentication code remaining
- [ ] Real-time user updates working correctly
- [ ] User onboarding flow tested and validated

## OTHER CONSIDERATIONS:

**Integration with Golf Services:**
- User service provides authentication for golf API calls
- Golf services will receive authenticated user UUIDs
- Lineup ownership links to Supabase user IDs
- Cross-service user validation through API gateway

**Future Enhancements:**
- Social authentication integration
- Advanced user analytics with Supabase
- Real-time user activity feeds
- Advanced subscription management features

**Development Workflow:**
- Independent development from golf service extraction
- Supabase project setup and configuration
- Environment variable management
- Testing with mock user data

## MONITORING & OBSERVABILITY:

**Supabase Metrics:**
- User authentication success/failure rates
- Phone OTP delivery and verification rates
- User data synchronization performance
- Real-time subscription connection health

**User Service Metrics:**
- User registration and onboarding completion rates
- Preference management operation times
- Subscription tier access patterns
- User session duration and activity

**Integration Metrics:**
- Cross-service user authentication success
- User UUID resolution performance
- API gateway user validation times
- User data consistency across services

## ROLLBACK PLAN:

**Immediate Rollback:**
- Switch back to custom authentication system
- Restore original user models and database
- Revert frontend auth store to hybrid mode
- Maintain user data consistency

**Graceful Migration:**
- Parallel operation of both auth systems
- Gradual user migration to Supabase
- Fallback authentication for service interruptions
- Data synchronization between systems during transition

**Prevention Measures:**
- Comprehensive testing before production deployment
- Staged rollout with monitoring
- Automated health checks and rollback triggers
- User data backup and recovery procedures