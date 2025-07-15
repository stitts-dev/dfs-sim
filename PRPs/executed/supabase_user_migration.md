# PRP: Complete User Authentication and Data Migration to Supabase

## Overview
**Objective:** Migrate the DFS optimizer from custom phone-based authentication with PostgreSQL to a full Supabase-managed user system while maintaining existing phone OTP workflows and adding real-time user data synchronization.

**Current State:** The project has a sophisticated custom authentication system with hybrid Supabase SMS integration. User management uses custom JWT tokens with PostgreSQL storage and integer-based user IDs.

**Target State:** Full Supabase Auth integration with UUID-based user management, Row Level Security, and real-time user data synchronization while preserving the existing phone OTP user experience.

## Context & Research Findings

### Current System Analysis
Based on comprehensive codebase analysis, the current system includes:

**✅ Production-Ready Components:**
- Complete phone OTP authentication in `backend/internal/api/handlers/auth.go`
- Sophisticated user models with subscription tiers in `backend/internal/models/user.go`
- Hybrid Supabase/backend auth store in `frontend/src/store/auth.ts`
- Circuit breaker SMS service with Supabase integration in `backend/internal/services/supabase_sms.go`
- Comprehensive testing patterns with unit, integration, and manual tests
- WebSocket-based real-time features for optimization progress

**🔧 Migration Challenges:**
- Integer-based user IDs (`uint`) need conversion to UUID format
- Custom JWT implementation needs replacement with Supabase Auth
- User data currently stored in PostgreSQL needs migration to Supabase
- Real-time features currently use native WebSockets instead of Supabase Realtime

### Supabase Best Practices (2024)
Based on current documentation and best practices:

**Authentication Patterns:**
- Phone OTP through Supabase Auth with Twilio/Vonage integration
- JWT-based session management with automatic refresh
- Security-first RLS policies with `auth.uid()` integration

**Real-time Architecture:**
- Database replication with `supabase_realtime` publication
- Row Level Security integration for authorized real-time access
- Subscription lifecycle management with proper cleanup

**Database Schema Design:**
- UUID primary keys with foreign key references to `auth.users`
- JSONB storage for flexible user preferences
- Trigger-based real-time broadcasting for user data changes

## Implementation Blueprint

### Phase 1: Supabase Project Setup and Schema Design (2 hours)

#### 1.1 Supabase Project Configuration
```bash
# Environment variables to add/update
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-role-key
SUPABASE_ANON_KEY=your-anon-key

# Enable phone authentication in Supabase dashboard
# Auth > Providers > Phone > Enable
# Configure SMS provider (Twilio recommended)
```

#### 1.2 Database Schema Migration
Create comprehensive user schema with proper UUID handling:

```sql
-- Enable realtime for user data
ALTER PUBLICATION supabase_realtime ADD TABLE users;
ALTER PUBLICATION supabase_realtime ADD TABLE user_preferences;

-- Users table (extends auth.users)
CREATE TABLE public.users (
  id UUID REFERENCES auth.users(id) PRIMARY KEY,
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

#### 1.3 Row Level Security Policies
Implement comprehensive RLS policies following 2024 best practices:

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

-- Realtime access policies
CREATE POLICY "Users can receive own data broadcasts" ON public.users
  FOR SELECT USING (auth.uid() = id);
CREATE POLICY "Users can receive own preference broadcasts" ON public.user_preferences
  FOR SELECT USING (auth.uid() = user_id);
```

#### 1.4 Real-time Triggers and Functions
Set up automatic real-time broadcasting for user data changes:

```sql
-- Function for broadcasting user changes
CREATE OR REPLACE FUNCTION public.handle_user_changes()
RETURNS TRIGGER
SECURITY DEFINER
LANGUAGE plpgsql
AS $$
BEGIN
  -- Broadcast user data changes to authenticated user
  PERFORM realtime.broadcast_changes(
    'user:' || COALESCE(NEW.id, OLD.id)::TEXT,
    TG_OP,
    TG_TABLE_NAME,
    TG_TABLE_SCHEMA,
    NEW,
    OLD
  );
  RETURN NULL;
END;
$$;

-- Trigger for user data changes
CREATE TRIGGER handle_users_realtime_changes
  AFTER INSERT OR UPDATE OR DELETE ON public.users
  FOR EACH ROW EXECUTE FUNCTION handle_user_changes();

-- Trigger for user preferences changes
CREATE TRIGGER handle_user_preferences_realtime_changes
  AFTER INSERT OR UPDATE OR DELETE ON public.user_preferences
  FOR EACH ROW EXECUTE FUNCTION handle_user_changes();

-- Updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON public.user_preferences
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Phase 2: Backend User Service Migration (3 hours)

#### 2.1 Create Supabase User Service
Create new user service replacing custom JWT with Supabase Auth:

```go
// internal/services/supabase_user.go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/google/uuid"
    "github.com/your-project/internal/models"
)

type SupabaseUserService struct {
    supabaseURL    string
    serviceKey     string
    httpClient     *http.Client
}

func NewSupabaseUserService(supabaseURL, serviceKey string) *SupabaseUserService {
    return &SupabaseUserService{
        supabaseURL: supabaseURL,
        serviceKey:  serviceKey,
        httpClient:  &http.Client{Timeout: 10 * time.Second},
    }
}

// CreateUser creates user profile after Supabase Auth registration
func (s *SupabaseUserService) CreateUser(ctx context.Context, userID uuid.UUID, phoneNumber, firstName, lastName string) (*models.User, error) {
    user := &models.User{
        ID:               userID,
        PhoneNumber:      phoneNumber,
        FirstName:        &firstName,
        LastName:         &lastName,
        SubscriptionTier: "free",
        IsActive:         true,
    }

    // Create user record in Supabase
    return s.insertUser(ctx, user)
}

// GetUser retrieves user by ID with preferences
func (s *SupabaseUserService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
    // Implementation using Supabase API with RLS
    url := fmt.Sprintf("%s/rest/v1/users?id=eq.%s&select=*,preferences:user_preferences(*)", 
        s.supabaseURL, userID)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+s.serviceKey)
    req.Header.Set("apikey", s.serviceKey)
    
    // Execute request and parse response
    // Implementation details...
}

// UpdateUserPreferences updates user preferences with real-time sync
func (s *SupabaseUserService) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, preferences *models.UserPreferences) error {
    // Update preferences in Supabase with automatic real-time broadcasting
    // Implementation using UPSERT for preferences
}
```

#### 2.2 Supabase JWT Verification Middleware
Replace custom JWT with Supabase JWT verification:

```go
// internal/api/middleware/supabase_auth.go
package middleware

import (
    "context"
    "crypto/rsa"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type SupabaseAuthMiddleware struct {
    supabaseURL string
    publicKey   *rsa.PublicKey
}

// SupabaseAuthRequired validates Supabase JWT tokens
func (m *SupabaseAuthMiddleware) SupabaseAuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        
        // Validate Supabase JWT token
        claims, err := m.validateSupabaseToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Extract user ID from claims
        userID, err := uuid.Parse(claims.Subject)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
            c.Abort()
            return
        }

        // Set user context
        c.Set("user_id", userID)
        c.Set("user_claims", claims)
        c.Next()
    }
}

func (m *SupabaseAuthMiddleware) validateSupabaseToken(tokenString string) (*jwt.RegisteredClaims, error) {
    // Implementation using Supabase JWT verification
    // Fetch public key from Supabase JWKS endpoint
    // Validate token signature and claims
}
```

#### 2.3 Update User API Handlers
Modify existing handlers to use Supabase User Service:

```go
// internal/api/handlers/user.go
package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/your-project/internal/services"
)

type UserHandler struct {
    userService *services.SupabaseUserService
}

func NewUserHandler(userService *services.SupabaseUserService) *UserHandler {
    return &UserHandler{userService: userService}
}

// GetCurrentUser returns authenticated user profile
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    
    user, err := h.userService.GetUser(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUserPreferences updates user preferences with real-time sync
func (h *UserHandler) UpdateUserPreferences(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    
    var preferences models.UserPreferences
    if err := c.ShouldBindJSON(&preferences); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    err := h.userService.UpdateUserPreferences(c.Request.Context(), userID, &preferences)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}
```

### Phase 3: Frontend Supabase Integration (2 hours)

#### 3.1 Update Auth Store for Supabase-Only Operation
Simplify auth store to use Supabase exclusively:

```typescript
// src/store/auth.ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { createClient, User, Session } from '@supabase/supabase-js'
import { supabaseClient } from '@/services/supabase'

interface AuthState {
  user: User | null
  session: Session | null
  isLoading: boolean
  error: string | null
  isAuthenticated: boolean
  
  // Phone auth specific
  currentPhoneNumber: string | null
  otpSent: boolean
  verificationInProgress: boolean
  
  // Actions
  loginWithPhone: (phoneNumber: string) => Promise<void>
  verifyOTP: (phoneNumber: string, code: string) => Promise<void>
  resendOTP: (phoneNumber: string) => Promise<void>
  logout: () => Promise<void>
  refreshSession: () => Promise<void>
  
  // Real-time subscription management
  subscribeToUserUpdates: () => void
  unsubscribeFromUserUpdates: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      session: null,
      isLoading: false,
      error: null,
      isAuthenticated: false,
      currentPhoneNumber: null,
      otpSent: false,
      verificationInProgress: false,

      loginWithPhone: async (phoneNumber: string) => {
        set({ isLoading: true, error: null, currentPhoneNumber: phoneNumber })
        
        try {
          const { error } = await supabaseClient.auth.signInWithOtp({
            phone: phoneNumber,
          })
          
          if (error) throw error
          
          set({ otpSent: true, isLoading: false })
        } catch (error) {
          set({ 
            error: error instanceof Error ? error.message : 'Failed to send OTP',
            isLoading: false 
          })
        }
      },

      verifyOTP: async (phoneNumber: string, code: string) => {
        set({ verificationInProgress: true, error: null })
        
        try {
          const { data, error } = await supabaseClient.auth.verifyOtp({
            phone: phoneNumber,
            token: code,
            type: 'sms'
          })
          
          if (error) throw error
          
          set({
            user: data.user,
            session: data.session,
            isAuthenticated: true,
            verificationInProgress: false,
            otpSent: false,
            currentPhoneNumber: null
          })

          // Subscribe to real-time updates after authentication
          get().subscribeToUserUpdates()
          
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Invalid verification code',
            verificationInProgress: false
          })
        }
      },

      logout: async () => {
        get().unsubscribeFromUserUpdates()
        await supabaseClient.auth.signOut()
        set({
          user: null,
          session: null,
          isAuthenticated: false,
          currentPhoneNumber: null,
          otpSent: false,
          verificationInProgress: false
        })
      },

      subscribeToUserUpdates: () => {
        const { user } = get()
        if (!user) return

        // Subscribe to user data changes
        const subscription = supabaseClient
          .channel(`user:${user.id}`)
          .on('postgres_changes', {
            event: '*',
            schema: 'public',
            table: 'users',
            filter: `id=eq.${user.id}`
          }, (payload) => {
            // Update user data in real-time
            console.log('User data updated:', payload)
          })
          .on('postgres_changes', {
            event: '*',
            schema: 'public',
            table: 'user_preferences',
            filter: `user_id=eq.${user.id}`
          }, (payload) => {
            // Update user preferences in real-time
            console.log('User preferences updated:', payload)
          })
          .subscribe()

        // Store subscription for cleanup
        set({ realtimeSubscription: subscription })
      },

      unsubscribeFromUserUpdates: () => {
        const state = get()
        if (state.realtimeSubscription) {
          state.realtimeSubscription.unsubscribe()
          set({ realtimeSubscription: null })
        }
      }
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ 
        user: state.user, 
        session: state.session,
        isAuthenticated: state.isAuthenticated 
      })
    }
  )
)

// Initialize auth state on app load
supabaseClient.auth.onAuthStateChange((event, session) => {
  const store = useAuthStore.getState()
  
  if (event === 'SIGNED_IN' && session) {
    store.subscribeToUserUpdates()
  } else if (event === 'SIGNED_OUT') {
    store.unsubscribeFromUserUpdates()
  }
})
```

#### 3.2 Update User Service for Supabase Integration
Create comprehensive user service for frontend:

```typescript
// src/services/userService.ts
import { supabaseClient } from './supabase'
import type { User, UserPreferences } from '@/types/user'

export class UserService {
  
  // Get current user profile with preferences
  static async getCurrentUser(): Promise<User | null> {
    const { data, error } = await supabaseClient
      .from('users')
      .select(`
        *,
        preferences:user_preferences(*)
      `)
      .single()

    if (error) {
      console.error('Error fetching user:', error)
      return null
    }

    return data
  }

  // Update user preferences with real-time sync
  static async updatePreferences(preferences: Partial<UserPreferences>): Promise<boolean> {
    const { data: { user } } = await supabaseClient.auth.getUser()
    if (!user) return false

    const { error } = await supabaseClient
      .from('user_preferences')
      .upsert({
        user_id: user.id,
        ...preferences,
        updated_at: new Date().toISOString()
      })

    if (error) {
      console.error('Error updating preferences:', error)
      return false
    }

    return true
  }

  // Subscribe to user data changes
  static subscribeToUserUpdates(userId: string, callback: (data: any) => void) {
    return supabaseClient
      .channel(`user_updates:${userId}`)
      .on('postgres_changes', {
        event: '*',
        schema: 'public',
        table: 'users',
        filter: `id=eq.${userId}`
      }, callback)
      .on('postgres_changes', {
        event: '*',
        schema: 'public',
        table: 'user_preferences',
        filter: `user_id=eq.${userId}`
      }, callback)
      .subscribe()
  }
}
```

### Phase 4: Testing and Validation (1 hour)

#### 4.1 Backend Tests
Update existing tests for Supabase integration:

```go
// internal/services/supabase_user_test.go
package services

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestSupabaseUserService_CreateUser(t *testing.T) {
    service := NewSupabaseUserService("test-url", "test-key")
    
    userID := uuid.New()
    user, err := service.CreateUser(context.Background(), userID, "+1234567890", "John", "Doe")
    
    assert.NoError(t, err)
    assert.Equal(t, userID, user.ID)
    assert.Equal(t, "+1234567890", user.PhoneNumber)
    assert.Equal(t, "free", user.SubscriptionTier)
}

func TestSupabaseUserService_GetUser(t *testing.T) {
    // Test user retrieval with preferences
}

func TestSupabaseUserService_UpdatePreferences(t *testing.T) {
    // Test preference updates with real-time sync
}
```

#### 4.2 Frontend Tests
Update auth store tests for Supabase-only operation:

```typescript
// src/store/__tests__/auth.test.ts
import { renderHook, act } from '@testing-library/react'
import { useAuthStore } from '../auth'
import { createClient } from '@supabase/supabase-js'

// Mock Supabase client
jest.mock('@supabase/supabase-js')

describe('AuthStore with Supabase', () => {
  beforeEach(() => {
    useAuthStore.getState().logout()
  })

  it('should handle phone login flow', async () => {
    const { result } = renderHook(() => useAuthStore())

    await act(async () => {
      await result.current.loginWithPhone('+1234567890')
    })

    expect(result.current.otpSent).toBe(true)
    expect(result.current.currentPhoneNumber).toBe('+1234567890')
  })

  it('should handle OTP verification', async () => {
    const { result } = renderHook(() => useAuthStore())

    await act(async () => {
      await result.current.verifyOTP('+1234567890', '123456')
    })

    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user).toBeTruthy()
  })

  it('should manage real-time subscriptions', async () => {
    const { result } = renderHook(() => useAuthStore())

    act(() => {
      result.current.subscribeToUserUpdates()
    })

    // Test subscription management
  })
})
```

#### 4.3 Integration Tests
Create end-to-end tests for complete migration:

```bash
# Test script: test-supabase-migration.sh
#!/bin/bash

echo "Testing Supabase user migration..."

# Test phone authentication flow
echo "Testing phone authentication..."
curl -X POST http://localhost:8080/api/v1/auth/supabase/login \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+1234567890"}'

# Test OTP verification
echo "Testing OTP verification..."
curl -X POST http://localhost:8080/api/v1/auth/supabase/verify \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+1234567890", "code": "123456"}'

# Test user profile access
echo "Testing user profile access..."
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $SUPABASE_TOKEN"

# Test preference updates
echo "Testing preference updates..."
curl -X PUT http://localhost:8080/api/v1/users/preferences \
  -H "Authorization: Bearer $SUPABASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"theme": "dark", "beginner_mode": false}'

echo "Migration testing complete!"
```

## Validation Gates

### Backend Validation
```bash
# Syntax and linting
cd backend && golangci-lint run

# Unit tests
cd backend && go test ./internal/services/... -v

# Integration tests
cd backend && go test ./tests/... -v

# Build verification
cd backend && go build -o bin/server cmd/server/main.go
```

### Frontend Validation
```bash
# Type checking
cd frontend && npm run type-check

# Unit tests
cd frontend && npm test

# Build verification
cd frontend && npm run build

# Linting
cd frontend && npm run lint
```

### End-to-End Validation
```bash
# Complete authentication flow
./test-supabase-migration.sh

# Real-time subscription testing
node scripts/test-realtime-subscriptions.js

# Performance testing
go test -bench=. ./internal/services/...
```

## Critical Implementation Notes

### UUID Migration Strategy
- **Current System**: Uses `uint` for user IDs
- **Migration Path**: Add UUID mapping table during transition
- **Foreign Key Updates**: Update all user_id references across codebase
- **Backward Compatibility**: Maintain dual ID support during migration

### Real-time Architecture
- **Current**: Native WebSocket hub for optimization progress
- **Target**: Hybrid approach - Supabase Realtime for user data, existing WebSockets for optimization
- **Subscription Management**: Proper lifecycle management with cleanup
- **Performance**: Filter subscriptions to minimize unnecessary updates

### Security Implementation
- **RLS Policies**: Comprehensive user data isolation
- **JWT Validation**: Replace custom JWT with Supabase verification
- **API Security**: Maintain existing rate limiting and middleware patterns
- **Phone Verification**: Leverage existing SMS circuit breaker patterns

### Error Handling and Fallbacks
- **Service Degradation**: Graceful fallback to existing auth if Supabase unavailable
- **Data Consistency**: Ensure atomic operations during migration
- **Monitoring**: Comprehensive logging for migration validation
- **Rollback Plan**: Ability to revert to custom auth system

## External Dependencies

### Required URLs and Documentation
- **Supabase Phone Auth**: https://supabase.com/docs/guides/auth/phone-login
- **Row Level Security**: https://supabase.com/docs/guides/database/postgres/row-level-security
- **Realtime Subscriptions**: https://supabase.com/docs/guides/realtime/subscribing-to-database-changes
- **JWT Verification**: https://supabase.com/docs/guides/auth/server-side/validating-jwts
- **JavaScript Client**: https://supabase.com/docs/reference/javascript/auth-signinwithotp

### Code Patterns to Follow
- **Testing Pattern**: Use existing testify-based unit tests in `backend/internal/*/`
- **Error Handling**: Follow existing circuit breaker pattern in `backend/internal/services/supabase_sms.go`
- **Middleware Pattern**: Extend existing auth middleware in `backend/internal/api/middleware/auth.go`
- **Frontend State**: Follow existing Zustand patterns in `frontend/src/store/auth.ts`

## Success Criteria Checklist

- [ ] Phone OTP authentication works seamlessly through Supabase Auth
- [ ] User preferences sync in real-time across browser sessions
- [ ] All user data properly isolated via Row Level Security
- [ ] UUID-based user IDs successfully replace integer IDs
- [ ] Real-time subscriptions work without memory leaks
- [ ] Backend API maintains existing performance benchmarks
- [ ] All existing tests pass with Supabase integration
- [ ] No custom JWT or phone verification code remains
- [ ] Frontend auth flow maintains existing UX
- [ ] Migration completes without data loss

## Risk Mitigation

### Technical Risks
- **UUID Conversion**: Comprehensive mapping and foreign key updates
- **Real-time Performance**: Proper subscription filtering and cleanup
- **Auth Token Migration**: Parallel operation during transition
- **Data Consistency**: Atomic migration operations with rollback capability

### Integration Risks
- **External Dependencies**: Supabase service availability and rate limits
- **Phone Provider Setup**: Proper Twilio/Vonage configuration
- **Real-time Scaling**: Subscription management under load
- **Browser Support**: Proper fallback for WebSocket connectivity

## Confidence Score: 9/10

This PRP provides comprehensive coverage of the migration with:
- ✅ Complete understanding of current sophisticated system
- ✅ Detailed implementation blueprint with real code examples
- ✅ Extensive research on 2024 Supabase best practices
- ✅ Comprehensive testing strategy with existing patterns
- ✅ Clear validation gates and success criteria
- ✅ Risk mitigation and rollback planning
- ✅ External documentation and dependency management

The migration builds on existing production-ready components while adding modern real-time capabilities and improved security through Supabase's managed services.