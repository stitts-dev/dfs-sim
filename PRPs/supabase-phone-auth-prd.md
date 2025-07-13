name: "Supabase + Twilio Phone Authentication with Plaid-like UX"
description: |
  Production-ready phone number authentication using Supabase Auth with Twilio SMS delivery, 
  featuring a modern Plaid-inspired user experience with proper user-lineup-AI recommendation relationships.

## Goal
Implement production-ready phone number authentication that replaces the current mock SMS service and provides seamless user onboarding with a Plaid-inspired UX. The system must establish proper user relationships for lineups and AI recommendations while providing frictionless authentication matching modern fintech standards.

## Why
- **Production Requirement**: Current mock SMS service is development-only and unsuitable for production
- **User Experience**: Phone-based auth provides better security and UX than email authentication
- **Modern Standards**: Plaid-inspired UX matches industry standards users expect from fintech apps
- **Foundation for Features**: Proper user relationships needed for lineups, AI recommendations, and subscription tiers
- **Reliability**: Supabase + Twilio provides scalable, reliable SMS delivery with fallback systems

## What
A complete phone authentication system that handles user registration and login with SMS verification codes, featuring auto-formatting phone inputs, 6-digit OTP verification, and seamless progression through signup flows.

### Success Criteria
- [ ] Users can register with phone number and receive SMS within 30 seconds (95% success rate)
- [ ] Phone number formatting works for US and international numbers (E.164 compliance)
- [ ] Existing users can login with phone authentication flow
- [ ] Rate limiting prevents abuse (max 3 SMS per phone per hour)
- [ ] Fallback to Twilio works when Supabase is unavailable
- [ ] Mobile-first responsive design with accessibility compliance (WCAG 2.1 AA)
- [ ] Authentication API responses under 500ms (95th percentile)
- [ ] Error recovery and user feedback within 2 seconds

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://supabase.com/docs/guides/auth/phone-login
  why: Core Supabase Auth phone authentication implementation patterns
  
- url: https://supabase.com/docs/guides/auth/phone-login/twilio
  why: Supabase SMS provider configuration for Twilio integration
  
- url: https://supabase.com/docs/reference/javascript/auth-signinwithotp
  why: JavaScript client reference for signInWithOtp implementation
  
- url: https://github.com/twilio/twilio-go
  why: Official Twilio Go SDK for SMS API implementation
  
- url: https://dev.to/he110/circuitbreaker-pattern-in-go-43cn
  why: Circuit breaker pattern implementation for reliable SMS service fallbacks
  
- url: https://ui.shadcn.com/docs/components/input-otp
  why: Modern OTP input component patterns for React implementation
  
- file: backend/internal/api/handlers/auth.go
  why: Existing auth handler patterns to follow, phone validation, OTP generation
  
- file: backend/internal/services/sms.go
  why: SMS service interface pattern - TwilioSMSService and SupabaseSMSService stubs exist
  
- file: backend/internal/models/user.go
  why: User model already supports phone authentication with phone_verified field
  
- file: frontend/src/catalyst/
  why: Catalyst UI Kit component patterns for Button, Input, Dialog, Field, Label
  
- file: frontend/src/store/preferences.ts
  why: Zustand state management patterns with persistence middleware
  
- file: frontend/src/services/api.ts
  why: Axios configuration with auth token interceptors
```

### Current Codebase Tree (Relevant Sections)
```bash
backend/
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   └── auth.go              # ✅ Complete phone auth handlers with OTP
│   │   └── middleware/
│   │       └── auth.go              # ✅ JWT validation middleware
│   ├── models/
│   │   └── user.go                  # ✅ User model with phone_number, phone_verified
│   ├── services/
│   │   └── sms.go                   # ✅ SMS interface with Twilio/Supabase stubs
│   └── pkg/
│       └── config/
│           └── config.go            # ✅ Viper config with environment binding

frontend/
├── src/
│   ├── catalyst/                    # ✅ UI Kit components (Button, Input, Dialog)
│   ├── components/
│   │   └── auth/                    # ❌ MISSING - needs phone auth components
│   ├── services/
│   │   ├── api.ts                   # ✅ Axios client with auth interceptors
│   │   └── mockAuth.ts              # ⚠️ TO REPLACE - mock auth service
│   ├── store/
│   │   └── preferences.ts           # ✅ Zustand patterns to follow
│   └── types/                       # ❌ MISSING - needs auth type definitions
```

### Desired Codebase Tree (Files to Add)
```bash
backend/
├── internal/
│   ├── services/
│   │   ├── supabase_sms.go          # NEW: Complete SupabaseSMSService implementation
│   │   └── twilio_sms.go            # NEW: Complete TwilioSMSService implementation
│   └── pkg/
│       └── config/
│           └── config.go            # MODIFY: Add Supabase/Twilio config fields

frontend/
├── src/
│   ├── components/
│   │   └── auth/
│   │       ├── PhoneInput.tsx       # NEW: Auto-formatting phone input component
│   │       ├── OTPVerification.tsx  # NEW: 6-digit OTP verification component
│   │       ├── SignupFlow.tsx       # NEW: Phone → SMS → verify progression
│   │       └── LoginFlow.tsx        # NEW: Existing user login flow
│   ├── hooks/
│   │   └── usePhoneAuth.ts          # NEW: React Query mutations for phone auth
│   ├── services/
│   │   └── supabase.ts              # NEW: Supabase client configuration
│   ├── store/
│   │   └── auth.ts                  # NEW: Authentication state management
│   └── types/
│       └── auth.ts                  # NEW: TypeScript interfaces for auth
```

### Known Gotchas & Library Quirks
```go
// CRITICAL: Supabase Auth requires specific environment variables
// SUPABASE_URL, SUPABASE_SERVICE_KEY (server-side), SUPABASE_ANON_KEY (client-side)

// CRITICAL: Twilio Go SDK requires this exact import pattern
import "github.com/twilio/twilio-go"
import twilioApi "github.com/twilio/twilio-go/rest/api/v2010"

// CRITICAL: Phone numbers MUST be E.164 format for international compatibility
// Use existing validation in auth.go: normalizePhoneNumber() function

// CRITICAL: OTP codes expire after 1 hour in Supabase (3600 seconds)
// Rate limit: Only 1 OTP request per 60 seconds per phone number

// CRITICAL: React Query needs proper error typing
interface PhoneAuthError {
  code: string
  message: string
  details?: any
}

// CRITICAL: Catalyst UI components require specific import structure
import { Button, Input, Dialog, Field, Label } from '@/catalyst'
// Not individual imports like '@/catalyst/Button'

// CRITICAL: Circuit breaker state must persist across requests
// Use Redis for circuit breaker state management, not in-memory

// CRITICAL: SMS costs approximately $0.0075 per message
// Implement usage tracking and cost alerts
```

## Implementation Blueprint

### Data Models and Structure
```go
// Backend: Update config.go with Supabase/Twilio settings
type Config struct {
    // Existing fields...
    
    // Supabase Configuration
    SupabaseURL        string `mapstructure:"SUPABASE_URL"`
    SupabaseServiceKey string `mapstructure:"SUPABASE_SERVICE_KEY"`
    SupabaseAnonKey    string `mapstructure:"SUPABASE_ANON_KEY"`
    
    // Twilio Configuration
    TwilioAccountSID string `mapstructure:"TWILIO_ACCOUNT_SID"`
    TwilioAuthToken  string `mapstructure:"TWILIO_AUTH_TOKEN"`
    TwilioFromNumber string `mapstructure:"TWILIO_FROM_NUMBER"`
    
    // SMS Provider Selection
    SMSProvider string `mapstructure:"SMS_PROVIDER"` // "supabase", "twilio", "mock"
}
```

```typescript
// Frontend: TypeScript interfaces for authentication
interface PhoneAuthRequest {
  phone_number: string
  country_code?: string
}

interface VerificationRequest {
  phone_number: string
  verification_code: string
}

interface AuthUser {
  id: string
  phone_number: string
  phone_verified: boolean
  subscription_tier: 'free' | 'pro' | 'premium'
  created_at: string
}

interface AuthState {
  user: AuthUser | null
  token: string | null
  isLoading: boolean
  error: string | null
  // Actions
  loginWithPhone: (phoneNumber: string) => Promise<void>
  verifyOTP: (code: string) => Promise<void>
  logout: () => void
  clearError: () => void
}
```

### List of Tasks (Implementation Order)

```yaml
Task 1: Backend SMS Service Implementation
MODIFY backend/internal/services/sms.go:
  - COMPLETE TwilioSMSService.SendOTP() implementation using Twilio Go SDK
  - ADD circuit breaker pattern around Twilio API calls
  - IMPLEMENT SupabaseSMSService.SendOTP() using Supabase REST API
  - ADD comprehensive error handling and logging

CREATE backend/internal/services/twilio_sms.go:
  - IMPLEMENT full Twilio SMS service with rate limiting
  - ADD webhook handling for delivery status (optional)
  - FOLLOW existing service patterns from codebase

CREATE backend/internal/services/supabase_sms.go:
  - IMPLEMENT Supabase Auth phone authentication
  - ADD proper error mapping for Supabase responses
  - INCLUDE JWT token handling for Supabase sessions

Task 2: Backend Configuration Updates
MODIFY backend/pkg/config/config.go:
  - ADD Supabase configuration fields (URL, service key, anon key)
  - ADD Twilio configuration fields (SID, token, from number)
  - ADD SMS_PROVIDER selection field with validation

MODIFY backend/internal/api/handlers/auth.go:
  - UPDATE SMS service initialization to use real providers based on config
  - REPLACE MockSMSService with factory pattern (Supabase/Twilio/Mock)
  - PRESERVE existing auth flow and validation patterns

Task 3: Frontend Authentication Infrastructure
CREATE frontend/src/services/supabase.ts:
  - INITIALIZE Supabase client with proper configuration
  - EXPORT auth methods: signInWithOtp, verifyOtp, signOut
  - HANDLE session management and token refresh

CREATE frontend/src/types/auth.ts:
  - DEFINE all authentication TypeScript interfaces
  - INCLUDE error types for proper error handling
  - FOLLOW existing type patterns from the codebase

CREATE frontend/src/store/auth.ts:
  - IMPLEMENT Zustand auth store following patterns from preferences.ts
  - ADD persistence with localStorage for token management
  - INCLUDE loading states and error handling

Task 4: Frontend Phone Input Component
CREATE frontend/src/components/auth/PhoneInput.tsx:
  - USE Catalyst Input component as base
  - IMPLEMENT auto-formatting for US/international numbers
  - ADD real-time validation with visual feedback
  - INCLUDE country code selector (optional enhancement)

CREATE frontend/src/hooks/usePhoneAuth.ts:
  - IMPLEMENT React Query mutation for phone auth
  - FOLLOW existing API patterns from api.ts
  - ADD proper error typing and handling

Task 5: Frontend OTP Verification Component
CREATE frontend/src/components/auth/OTPVerification.tsx:
  - IMPLEMENT 6-digit OTP input with auto-advance
  - USE Catalyst styling patterns and components
  - ADD paste functionality for SMS codes
  - INCLUDE resend code functionality with countdown timer

Task 6: Frontend Authentication Flows
CREATE frontend/src/components/auth/SignupFlow.tsx:
  - IMPLEMENT phone → SMS → verify → dashboard progression
  - USE Catalyst Dialog/Modal components for flow steps
  - ADD proper loading states and error handling
  - INTEGRATE with auth store for state management

CREATE frontend/src/components/auth/LoginFlow.tsx:
  - IMPLEMENT existing user login flow
  - REUSE PhoneInput and OTPVerification components
  - ADD "Remember Me" functionality (optional)

Task 7: Environment Configuration
UPDATE .env.example:
  - ADD all Supabase environment variables with examples
  - ADD all Twilio environment variables with examples
  - ADD SMS_PROVIDER selection with options

UPDATE docker-compose.yml:
  - ADD new environment variables to backend service
  - ENSURE proper secrets management for production

Task 8: Integration Testing and Validation
CREATE backend/tests/sms_integration_test.go:
  - TEST both Supabase and Twilio SMS services
  - MOCK external API calls for unit tests
  - VALIDATE error handling and circuit breaker behavior

CREATE frontend/src/components/auth/__tests__/:
  - TEST PhoneInput formatting and validation
  - TEST OTPVerification component behavior
  - TEST complete authentication flows
  - USE React Testing Library patterns
```

### Per Task Pseudocode

```go
// Task 1: TwilioSMSService Implementation
func (s *TwilioSMSService) SendOTP(phoneNumber, code string) error {
    // PATTERN: Use circuit breaker from existing patterns
    if s.circuitBreaker.State() == circuit.Open {
        return ErrServiceUnavailable
    }
    
    // CRITICAL: Initialize Twilio client with proper auth
    client := twilio.NewRestClientWithParams(twilio.ClientParams{
        Username: s.config.TwilioAccountSID,
        Password: s.config.TwilioAuthToken,
    })
    
    // GOTCHA: Phone numbers must be E.164 format
    normalizedNumber := normalizePhoneNumber(phoneNumber)
    
    params := &twilioApi.CreateMessageParams{}
    params.SetTo(normalizedNumber)
    params.SetFrom(s.config.TwilioFromNumber)
    params.SetBody(fmt.Sprintf("Your verification code is: %s", code))
    
    // PATTERN: Rate limiting with Redis (see existing patterns)
    if err := s.rateLimiter.Allow(phoneNumber); err != nil {
        return ErrRateLimited
    }
    
    // CRITICAL: Handle Twilio-specific errors
    resp, err := client.Api.CreateMessage(params)
    if err != nil {
        s.circuitBreaker.RecordFailure()
        return s.mapTwilioError(err)
    }
    
    s.circuitBreaker.RecordSuccess()
    s.logger.Info("SMS sent successfully", "sid", *resp.Sid)
    return nil
}
```

```typescript
// Task 4: PhoneInput Component Implementation
export const PhoneInput: React.FC<PhoneInputProps> = ({ 
  value, 
  onChange, 
  onValidate,
  className 
}) => {
  // PATTERN: Use Catalyst Field/Input components
  const [formatted, setFormatted] = useState('')
  const [isValid, setIsValid] = useState(false)
  
  // CRITICAL: Auto-format as user types
  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const input = e.target.value.replace(/\D/g, '') // Remove non-digits
    const formatted = formatPhoneNumber(input) // Format: (123) 456-7890
    
    setFormatted(formatted)
    onChange?.(input) // Send raw digits to parent
    
    // PATTERN: Real-time validation feedback
    const valid = validatePhoneNumber(input)
    setIsValid(valid)
    onValidate?.(valid)
  }
  
  return (
    <Field>
      <Label>Phone Number</Label>
      <Input
        type="tel"
        value={formatted}
        onChange={handleChange}
        placeholder="+1 (555) 123-4567"
        className={cn(
          'w-full',
          !isValid && value && 'border-red-500',
          className
        )}
      />
      {!isValid && value && (
        <div className="text-sm text-red-600 mt-1">
          Please enter a valid phone number
        </div>
      )}
    </Field>
  )
}
```

### Integration Points
```yaml
DATABASE:
  - existing: User model already has phone_number and phone_verified fields
  - optional: Add supabase_user_id field for Supabase integration tracking
  
CONFIG:
  - add to: backend/pkg/config/config.go
  - pattern: Environment variable binding with mapstructure tags
  
ROUTES:
  - existing: auth.go handlers already support phone authentication
  - modify: Update SMS service initialization to use real providers
  
FRONTEND_STATE:
  - replace: mockAuth.ts with real Supabase authentication
  - add: auth store with Zustand following preferences.ts patterns
  
UI_COMPONENTS:
  - base: Use existing Catalyst components (Button, Input, Dialog)
  - new: Create auth-specific components in components/auth/
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Backend validation
cd backend
go mod tidy                          # Ensure dependencies are clean
go vet ./...                         # Static analysis
golangci-lint run                    # Comprehensive linting
go build ./...                       # Compilation check

# Frontend validation  
cd frontend
npm install                          # Install dependencies
npm run lint                         # ESLint checks
npm run type-check                   # TypeScript validation
npm run build                        # Build verification

# Expected: No errors. If errors exist, READ and fix before proceeding.
```

### Level 2: Unit Tests
```go
// Backend: Create comprehensive test suite
func TestTwilioSMSService_SendOTP(t *testing.T) {
    tests := []struct {
        name        string
        phoneNumber string
        code        string
        wantErr     bool
        mockStatus  int
    }{
        {
            name:        "successful_send",
            phoneNumber: "+1234567890",
            code:        "123456",
            wantErr:     false,
            mockStatus:  201,
        },
        {
            name:        "invalid_phone_number",
            phoneNumber: "invalid",
            code:        "123456", 
            wantErr:     true,
        },
        {
            name:        "twilio_api_error",
            phoneNumber: "+1234567890",
            code:        "123456",
            wantErr:     true,
            mockStatus:  400,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation with mocked Twilio client
        })
    }
}
```

```typescript
// Frontend: Component and hook testing
describe('PhoneInput', () => {
  it('formats phone number as user types', () => {
    render(<PhoneInput onChange={mockOnChange} />)
    
    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '1234567890' } })
    
    expect(input.value).toBe('(123) 456-7890')
    expect(mockOnChange).toHaveBeenCalledWith('1234567890')
  })
  
  it('shows validation error for invalid number', () => {
    render(<PhoneInput value="123" />)
    
    expect(screen.getByText('Please enter a valid phone number')).toBeInTheDocument()
  })
})
```

```bash
# Run and iterate until passing
cd backend && go test ./... -v
cd frontend && npm test

# If failing: Read error messages, understand root cause, fix code, re-run
# NEVER mock failures away - fix the underlying issue
```

### Level 3: Integration Testing
```bash
# Start backend with real environment variables
cd backend
export SMS_PROVIDER=mock  # Use mock for testing
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/dfs_optimizer_test
go run cmd/server/main.go

# Test phone authentication flow
curl -X POST http://localhost:8080/api/v1/auth/phone \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+1234567890"}'

# Expected: {"success": true, "message": "OTP sent"}

# Test OTP verification
curl -X POST http://localhost:8080/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+1234567890", "verification_code": "123456"}'

# Expected: {"success": true, "token": "jwt_token_here", "user": {...}}

# Start frontend and test complete flow
cd frontend
npm run dev
# Manually test: Register → Enter phone → Receive SMS → Verify → Dashboard
```

## Final Validation Checklist
- [ ] All tests pass: `cd backend && go test ./... -v`
- [ ] All tests pass: `cd frontend && npm test`
- [ ] No linting errors: `cd backend && golangci-lint run`
- [ ] No linting errors: `cd frontend && npm run lint`
- [ ] No type errors: `cd frontend && npm run type-check`
- [ ] Manual auth flow works: Phone input → SMS send → OTP verify → Login
- [ ] Error cases handled: Invalid phone, expired OTP, rate limiting
- [ ] Mobile responsive design tested on real devices
- [ ] Accessibility tested with screen reader
- [ ] Environment variables documented in .env.example
- [ ] Circuit breaker behavior tested with API failures
- [ ] SMS delivery tested with real phone numbers (Twilio trial account)

---

## Anti-Patterns to Avoid
- ❌ Don't hardcode Twilio/Supabase credentials - use environment variables
- ❌ Don't skip phone number validation - international numbers need E.164 format
- ❌ Don't ignore rate limiting - SMS abuse can be expensive
- ❌ Don't mock SMS delivery in production - test with real services
- ❌ Don't forget circuit breaker state persistence - use Redis not memory
- ❌ Don't assume SMS delivery is instant - provide proper user feedback
- ❌ Don't skip accessibility - phone auth must work with assistive technology
- ❌ Don't ignore mobile UX - most users will authenticate on mobile devices

---

## Confidence Score: 9/10
This PRP provides comprehensive context, specific implementation patterns from the existing codebase, detailed error handling strategies, and thorough validation loops. The existing authentication infrastructure provides an excellent foundation, requiring primarily the implementation of real SMS services and modern React components following established patterns.