/**
 * Authentication Components
 * 
 * Production-ready phone authentication components using Supabase and Twilio
 * with modern Plaid-inspired UX patterns and Tailwind Plus design enhancements.
 */

// Core input components
export { PhoneInput } from './PhoneInput'
export { OTPInput, OTPVerification, OTPDialog } from './OTPVerification'

// Enhanced components with improved styling and visual effects (preferred)
export { EnhancedOTPInput, EnhancedOTPVerification } from './EnhancedOTPVerification'

// Layout and wizard components
export { AuthLayout, AuthCard, AuthStepIndicator } from './AuthLayout'
export { AuthWizard } from './AuthWizard'

// Authentication flows
export { SignupFlow, SignupPage, InlineSignupForm } from './SignupFlow'
export { LoginFlow, AuthFlow, LoginPage, QuickLoginButton } from './LoginFlow'

// Types for component props
export type { 
  SignupStep
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
export { useUnifiedAuthStore } from '@/store/unifiedAuth'