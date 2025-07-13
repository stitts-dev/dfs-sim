/**
 * Authentication Components
 * 
 * Production-ready phone authentication components using Supabase and Twilio
 * with modern Plaid-inspired UX patterns.
 */

// Core input components
export { PhoneInput, PhoneInputWithCountryCode, CountryCodeSelector } from './PhoneInput'
export { OTPInput, OTPVerification, OTPDialog } from './OTPVerification'

// Authentication flows
export { SignupFlow, SignupPage, InlineSignupForm } from './SignupFlow'
export { LoginFlow, AuthFlow, LoginPage, QuickLoginButton } from './LoginFlow'

// Types for component props
export type { 
  SignupStep,
  LoginStep 
} from './SignupFlow'

export type {
  SignupFlowProps
} from './SignupFlow'

export type {
  LoginFlowProps
} from './LoginFlow'

// Re-export auth types for convenience
export type {
  PhoneAuthRequest,
  VerificationRequest,
  AuthUser,
  AuthResponse,
  PhoneAuthError,
  OTPInputProps,
  PhoneInputProps,
  AuthFormState,
  PhoneValidationResult,
  RateLimitInfo
} from '@/types/auth'

// Re-export auth hook for convenience
export { usePhoneAuth, useAuthGuard } from '@/hooks/usePhoneAuth'
export { useAuthStore } from '@/store/auth'