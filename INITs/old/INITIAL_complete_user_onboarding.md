## FEATURE:

Complete User Onboarding & Trial Management Flow

## CONTEXT & MOTIVATION:

The current authentication system has all the technical infrastructure in place but lacks the complete user experience needed for a production DFS platform. Users need:
- Seamless phone registration with real SMS delivery
- Clear trial limits and usage tracking (10 lineups, 5 simulations)
- Guided onboarding to set preferences and understand the platform
- Immediate access to contests and basic functionality

This transforms the working authentication backend into a complete user acquisition and onboarding funnel.

## EXAMPLES:

**User Journey Example:**
1. User visits `/auth/register` → enters phone number
2. Receives SMS OTP via Supabase/Twilio → verifies code
3. OnboardingWizard appears → sets sport preferences, platform preferences
4. Dashboard loads → shows trial usage (0/10 lineups used)
5. User can view contests, create lineups, with usage tracking
6. At 8/10 lineups → warning banner appears
7. At 10/10 lineups → upgrade modal blocks further optimization

## CURRENT STATE ANALYSIS:

**Working Infrastructure (90% complete):**
- ✅ Microservices authentication flow (API Gateway → User Service)
- ✅ Phone registration and OTP verification endpoints
- ✅ JWT token generation and refresh
- ✅ Frontend auth store and session management
- ✅ Database schema for users and preferences

**Missing Components:**
- ❌ Real SMS delivery (currently using MockSMSService)
- ❌ Trial usage tracking and limits enforcement
- ❌ User onboarding wizard
- ❌ Usage display UI components
- ❌ Contest viewing for trial users

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [x] Phone auth API endpoints (already working)
- [ ] Real SMS delivery via Supabase/Twilio integration
- [ ] Usage tracking fields in user model
- [ ] Trial limits enforcement in optimization endpoints
- [ ] Contest viewing endpoints for trial users

### Frontend Requirements:
- [ ] UsageTracker component with progress bars
- [ ] OnboardingWizard multi-step flow
- [ ] Trial warning banners and upgrade modals
- [ ] Enhanced Dashboard with usage display
- [ ] Contest browser for trial users

### Infrastructure Requirements:
- [ ] SMS_PROVIDER environment variable configuration
- [ ] Twilio API credentials (if selected over Supabase)
- [ ] Trial limits configuration (10 lineups, 5 simulations)
- [ ] Usage tracking in Redis/PostgreSQL

## IMPLEMENTATION APPROACH:

### Phase 1: SMS Integration (3 minutes)
- Switch user-service from MockSMSService to real Supabase SMS
- Configure SMS provider selection via environment variables
- Test OTP delivery with real phone numbers

### Phase 2: Trial Management UI (4 minutes)
- Build UsageTracker component showing current usage vs limits
- Add trial warning system with progressive alerts
- Create upgrade modal for limit exceeded scenarios
- Integrate usage display into Dashboard header

### Phase 3: Onboarding Experience (3 minutes)
- Create OnboardingWizard with sport/platform preference setup
- Add welcome messaging and platform introduction
- Implement guided tooltips for first-time features
- Add onboarding completion tracking

## DOCUMENTATION:

- [Supabase Phone Auth Documentation](https://supabase.com/docs/guides/auth/phone-logins)
- [Twilio Programmable SMS API](https://www.twilio.com/docs/sms)
- Current authentication flow in `services/user-service/README.md`
- Frontend auth integration in `frontend/src/services/auth.ts`

## TESTING STRATEGY:

### Unit Tests:
- [ ] SMS service provider switching logic
- [ ] Usage tracking calculations
- [ ] Trial limits enforcement

### Integration Tests:
- [ ] End-to-end phone registration with real SMS
- [ ] Onboarding wizard completion flow
- [ ] Trial usage tracking across optimization calls

### E2E Tests:
- [ ] Complete user journey from registration to first lineup
- [ ] Trial limit enforcement and upgrade prompts
- [ ] Cross-device onboarding experience

## POTENTIAL CHALLENGES & RISKS:

**SMS Delivery:**
- Rate limits on SMS providers
- Phone number validation for international users
- SMS delivery delays or failures

**Trial Management:**
- Usage tracking accuracy across microservices
- Race conditions in usage limit checks
- User expectations around trial limitations

**Performance:**
- Real-time usage updates without database overhead
- Onboarding wizard loading performance

## SUCCESS CRITERIA:

- [ ] User can register with phone number and receive SMS OTP within 30 seconds
- [ ] Onboarding wizard completion rate > 80%
- [ ] Trial usage tracking accurate to within 1 optimization
- [ ] Upgrade conversion funnel ready for payment integration
- [ ] Complete user flow from registration to first lineup in < 5 minutes

## OTHER CONSIDERATIONS:

**User Experience:**
- Clear communication of trial benefits and limitations
- Smooth transition from trial to paid subscription
- Accessible design for users with disabilities

**Business Logic:**
- Trial period duration (30 days vs usage-based)
- Grace period for users approaching limits
- Feature restrictions vs usage restrictions

## MONITORING & OBSERVABILITY:

**Key Metrics:**
- Registration completion rate
- SMS delivery success rate
- Onboarding wizard abandonment points
- Trial to paid conversion rate
- Average time to first lineup creation

**Logging Requirements:**
- SMS delivery attempts and failures
- Usage tracking events
- Onboarding step completion
- Trial limit hit events

## ROLLBACK PLAN:

**Safe Rollback Strategy:**
- Keep MockSMSService as fallback option
- Feature flags for onboarding wizard
- Database migration rollback for usage tracking fields
- Frontend component lazy loading for gradual rollout

**Rollback Triggers:**
- SMS delivery failure rate > 10%
- Registration completion rate drops > 20%
- Critical errors in usage tracking
- Performance degradation in auth flow