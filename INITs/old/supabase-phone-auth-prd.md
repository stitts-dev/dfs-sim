## FEATURE:

**Supabase + Twilio Phone Authentication with Plaid-like UX**

Implement production-ready phone number authentication using Supabase Auth with Twilio SMS delivery, featuring a modern Plaid-inspired user experience. This system will replace the current mock SMS service and provide seamless user onboarding with proper user-lineup-AI recommendation relationships.

## CONTEXT & MOTIVATION:

**Why is this feature needed?**
- Current system uses mock SMS service unsuitable for production
- Need frictionless user authentication matching modern fintech UX standards
- Phone-based auth provides better security and user experience than email
- Supabase + Twilio integration offers reliable, scalable SMS delivery
- Proper user relationships needed for lineups and AI recommendations

**What problem does it solve?**
- Eliminates development-only authentication barrier
- Provides production-ready user onboarding
- Establishes clean user data relationships for lineups and AI features
- Enables subscription tier management and usage tracking

**What value does it provide?**
- Professional user experience matching industry standards (Plaid, Robinhood, etc.)
- Reliable SMS delivery with fallback systems
- Seamless connection between users, their lineups, and AI recommendations
- Foundation for monetization through subscription tiers

## EXAMPLES:

**Reference Implementation: Plaid Phone Auth Flow**
1. Single input field with auto-formatting: "(123) 456-7890"
2. SMS verification with 6-digit code entry
3. Auto-advance between input fields
4. Clear error states and resend functionality
5. Clean, minimalist design with strong visual hierarchy

**User Flow Examples:**
- New User: Phone → SMS → Verify → Subscription Selection → Dashboard
- Existing User: Phone → SMS → Verify → Dashboard (skip subscription)
- Error Cases: Invalid phone, expired code, rate limiting

## CURRENT STATE ANALYSIS:

**What exists currently:**
- Complete User model with phone authentication structure
- Subscription tiers (Free/Pro/Premium) with usage tracking
- Mock SMS service for development
- JWT-based authentication system
- Clean database relationships: User → Lineups, User → AI Recommendations

**What components will this interact with:**
- Backend: AuthHandler, SMS services, User models, subscription middleware
- Frontend: Authentication components, API client, state management
- External: Supabase Auth API, Twilio SMS API
- Database: Users, PhoneVerificationCode, SubscriptionTier tables

**What constraints exist:**
- Supabase rate limits for SMS sends
- Twilio messaging costs and delivery rates
- Phone number validation (E.164 format)
- GDPR/privacy considerations for phone storage
- Circuit breaker patterns for external API reliability

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [ ] Complete SupabaseSMSService implementation with Supabase Auth API
- [ ] Implement TwilioSMSService as fallback with Twilio REST API
- [ ] Add Supabase configuration: project URL, service key, anon key
- [ ] Add Twilio configuration: account SID, auth token, phone number
- [ ] Update AuthHandler to use production SMS service instead of mock
- [ ] Implement SMS service factory with environment-based selection
- [ ] Add comprehensive error handling and circuit breaker patterns
- [ ] Add rate limiting for SMS sends (per phone number and global)

### Frontend Requirements:
- [ ] Create PhoneInput component with auto-formatting and validation
- [ ] Build OTPVerification component with 6-digit code entry
- [ ] Implement SignupFlow component with phone → SMS → verify progression
- [ ] Create LoginFlow component for existing users
- [ ] Add ResendCode functionality with countdown timer
- [ ] Implement authentication state management (replace mock auth)
- [ ] Create user dashboard showing lineups and subscription status
- [ ] Add proper error states and loading indicators
- [ ] Ensure responsive design for mobile-first experience

### Infrastructure Requirements:
- [ ] Add Supabase environment variables: SUPABASE_URL, SUPABASE_SERVICE_KEY, SUPABASE_ANON_KEY
- [ ] Add Twilio environment variables: TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_PHONE_NUMBER
- [ ] Configure SMS service selection: SMS_PROVIDER (supabase/twilio/mock)
- [ ] Set up proper secrets management for production deployment
- [ ] Update Docker configuration with new environment variables
- [ ] Configure Supabase Auth settings: SMS templates, rate limits, security
- [ ] Set up Twilio messaging service and phone number provisioning

## IMPLEMENTATION APPROACH:

### Phase 1: Foundation (Backend SMS Integration)
**Core components and basic functionality**
- Complete SupabaseSMSService with Supabase Auth API integration
- Implement TwilioSMSService with Twilio REST API
- Add environment configuration and service factory
- Update AuthHandler to use production SMS services
- Add comprehensive error handling and logging
- Test SMS delivery with real phone numbers

### Phase 2: Integration (Frontend Components)
**Connect components and establish data flow**
- Create Plaid-inspired phone input component with formatting
- Build OTP verification UI with proper UX patterns
- Implement complete authentication flows (signup/login)
- Replace mock auth service with real JWT token management
- Add proper error handling and user feedback
- Test complete user flows end-to-end

### Phase 3: Enhancement (Polish and Production Features)
**Add advanced features, optimization, edge cases**
- Add rate limiting UI feedback and countdown timers
- Implement advanced error states (network issues, service outages)
- Add user dashboard with lineups and AI recommendations
- Optimize for mobile experience and accessibility
- Add analytics and monitoring for authentication flows
- Performance optimization and error recovery

## DOCUMENTATION:

**Supabase Documentation:**
- [Supabase Auth Phone Documentation](https://supabase.com/docs/guides/auth/phone-login)
- [Supabase SMS Provider Configuration](https://supabase.com/docs/guides/auth/phone-login/twilio)
- [Supabase JavaScript Client Reference](https://supabase.com/docs/reference/javascript/auth-signinwithotp)

**Twilio Documentation:**
- [Twilio REST API SMS Documentation](https://www.twilio.com/docs/sms/api)
- [Twilio Node.js SDK](https://www.twilio.com/docs/libraries/node)
- [SMS Best Practices and Rate Limiting](https://www.twilio.com/docs/sms/best-practices)

**UX Pattern References:**
- Plaid Link phone authentication flow
- Robinhood signup experience
- Modern fintech authentication patterns

## TESTING STRATEGY:

### Unit Tests:
- [ ] SMS service implementations (mock external APIs)
- [ ] Phone number validation and formatting
- [ ] OTP generation and validation logic
- [ ] User model authentication methods
- [ ] JWT token generation and validation

### Integration Tests:
- [ ] Complete authentication API endpoint flows
- [ ] Database user creation and verification code management
- [ ] SMS service integration with mocked external APIs
- [ ] Rate limiting and circuit breaker behavior
- [ ] User-lineup relationship validation

### E2E Tests:
- [ ] Complete user signup flow (phone → SMS → verify → dashboard)
- [ ] Existing user login flow
- [ ] Error cases: invalid phone, expired code, rate limiting
- [ ] Mobile device testing for responsive design
- [ ] Accessibility testing with screen readers

## POTENTIAL CHALLENGES & RISKS:

**Technical challenges:**
- SMS delivery reliability and latency varies by carrier
- Supabase and Twilio API rate limits during high traffic
- Phone number validation across international formats
- Circuit breaker implementation for external API failures

**Dependencies:**
- Supabase service availability and API stability
- Twilio messaging service reliability
- Mobile carrier SMS delivery capabilities
- Frontend component library compatibility

**Performance concerns:**
- SMS delivery can take 5-30 seconds in some regions
- Database queries for user lookup and verification codes
- Frontend component rendering performance on mobile devices

**Breaking changes:**
- Replacing mock authentication may affect development workflows
- Environment variable changes require deployment coordination
- Database schema is already compatible, no breaking changes expected

## SUCCESS CRITERIA:

**Functional Requirements:**
- [ ] Users can successfully register with phone number and receive SMS
- [ ] SMS codes arrive within 30 seconds in 95% of cases
- [ ] Phone number formatting works correctly for US and international numbers
- [ ] Existing users can login with phone number authentication
- [ ] Rate limiting prevents abuse (max 3 SMS per phone per hour)
- [ ] Fallback to Twilio works when Supabase is unavailable

**Performance Requirements:**
- [ ] Authentication API responses under 500ms (95th percentile)
- [ ] Frontend components render in under 100ms
- [ ] SMS delivery success rate above 95%
- [ ] Error recovery and user feedback within 2 seconds

**User Experience Requirements:**
- [ ] Plaid-like UX with smooth transitions and clear feedback
- [ ] Mobile-first responsive design works on all device sizes
- [ ] Accessibility compliance (WCAG 2.1 AA)
- [ ] Clear error messages and recovery paths

## OTHER CONSIDERATIONS:

**Security Considerations:**
- Phone numbers stored in hashed format where possible
- OTP codes expire after 10 minutes maximum
- Rate limiting prevents brute force attacks
- Circuit breakers prevent service abuse

**Privacy and Compliance:**
- GDPR compliance for phone number storage and processing
- User consent for SMS communications
- Data retention policies for verification codes
- Clear privacy policy updates

**Cost Management:**
- Monitor Twilio SMS costs (approximately $0.0075 per SMS)
- Supabase usage tracking for monthly limits
- Implement usage alerts and cost controls

**AI Coding Assistant Gotchas:**
- Phone number validation must handle international formats correctly
- SMS service circuit breakers need proper state management
- Frontend phone input components require careful event handling
- Environment variable loading order affects service initialization

## MONITORING & OBSERVABILITY:

**Logging Requirements:**
- SMS send attempts and delivery status
- Authentication attempt success/failure rates
- API response times for Supabase and Twilio
- User registration and login flows
- Error rates and types by component

**Metrics to Track:**
- SMS delivery success rate by provider
- Authentication flow completion rates
- API response time percentiles
- Error rates by error type and component
- User signup conversion rates

**Alerts to Set Up:**
- SMS delivery failure rate above 5%
- Authentication API response time above 1 second
- Supabase or Twilio API unavailability
- Unusual pattern detection (potential abuse)

## ROLLBACK PLAN:

**Safe Rollback Strategy:**
1. **Feature Flags:** Use environment variables to switch between mock and production SMS
2. **Database Compatibility:** User model already supports phone auth, no schema changes
3. **Frontend Fallback:** Keep mock auth service as fallback option
4. **Staged Deployment:** Deploy backend first, then frontend components
5. **Monitoring:** Watch success rates and error logs during rollout

**Emergency Procedures:**
- Immediate rollback: Set `SMS_PROVIDER=mock` environment variable
- Database rollback: No schema changes, safe to revert application code
- User Impact: Existing authenticated users remain logged in during rollback
- Communication: Clear messaging about temporary authentication issues

**Rollback Triggers:**
- SMS delivery success rate below 90%
- Authentication API error rate above 5%
- User complaints about authentication failures
- External service outages affecting core functionality