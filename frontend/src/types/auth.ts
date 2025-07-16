// Authentication type definitions for multi-method auth system

export type AuthMethod = 'phone' | 'email'
export type AuthStepType = 'welcome' | 'method-selection' | 'phone' | 'email' | 'password' | 'verification' | 'email-verification' | 'success' | 'onboarding'
export type AuthWizardMode = 'login' | 'signup'

export interface PhoneAuthRequest {
  phone_number: string
  country_code?: string
}

export interface VerificationRequest {
  phone_number: string
  verification_code: string
}

export interface LoginRequest {
  phone_number: string
}

export interface ResendRequest {
  phone_number: string
}

export interface EmailAuthRequest {
  email: string
  password: string
}

export interface EmailVerificationRequest {
  email: string
  code: string
}

export interface PasswordResetRequest {
  email: string
}

export interface AuthUser {
  id: number
  phone_number: string
  phone_verified: boolean
  email?: string
  email_verified: boolean
  first_name?: string
  last_name?: string
  subscription_tier: 'free' | 'pro' | 'premium'
  subscription_status: 'active' | 'cancelled' | 'expired'
  subscription_expires_at?: string
  is_active: boolean
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  token: string
  expires_at: string
  user: AuthUser
  is_new_user: boolean
}

export interface PhoneAuthError {
  code: string
  message: string
  details?: any
}

export interface AuthState {
  // State
  user: AuthUser | null
  token: string | null
  isLoading: boolean
  error: string | null
  isAuthenticated: boolean

  // Phone auth flow state
  currentPhoneNumber: string | null
  otpSent: boolean
  verificationInProgress: boolean

  // Actions
  loginWithPhone: (phoneNumber: string) => Promise<void>
  verifyOTP: (phoneNumber: string, code: string) => Promise<void>
  resendOTP: (phoneNumber: string) => Promise<void>
  logout: () => void
  clearError: () => void
  refreshToken: () => Promise<void>
  getCurrentUser: () => Promise<void>

  // Utility actions
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  setCurrentPhoneNumber: (phoneNumber: string | null) => void
  setOtpSent: (sent: boolean) => void
}

export interface OTPInputProps {
  value: string
  onChange: (value: string) => void
  length: number
  disabled?: boolean
  autoFocus?: boolean
  onComplete?: (value: string) => void
  error?: boolean
}

export interface PhoneInputProps {
  value: string
  onChange: (value: string) => void
  onValidate?: (isValid: boolean) => void
  disabled?: boolean
  autoFocus?: boolean
  error?: boolean
  className?: string
}

export interface AuthFormState {
  phoneNumber: string
  verificationCode: string
  isValid: boolean
  errors: {
    phoneNumber?: string
    verificationCode?: string
    general?: string
  }
}

// Supabase-specific types
export interface SupabaseUser {
  id: string
  phone?: string
  email?: string
  created_at: string
  updated_at: string
  phone_confirmed_at?: string
  email_confirmed_at?: string
}

export interface SupabaseSession {
  access_token: string
  refresh_token: string
  expires_at: number
  expires_in: number
  token_type: string
  user: SupabaseUser
}

export interface SupabaseAuthResponse {
  user: SupabaseUser | null
  session: SupabaseSession | null
  error?: PhoneAuthError
}

// Rate limiting and validation
export interface PhoneValidationResult {
  isValid: boolean
  formatted: string
  error?: string
  countryCode?: string
}

export interface RateLimitInfo {
  remainingAttempts: number
  resetTime: string
  blocked: boolean
}

// Auth Provider interfaces
export interface AuthProviderConfig {
  enabled: boolean
  priority: number
  requiresVerification: boolean
}

export interface AuthCredentials {
  method: AuthMethod
  phoneNumber?: string
  email?: string
  password?: string
  verificationCode?: string
  mode?: AuthWizardMode
}

export interface AuthProviderResult {
  success: boolean
  user?: AuthUser
  token?: string
  error?: string
  requiresVerification?: boolean
  verificationSent?: boolean
}

export abstract class AuthProvider {
  abstract method: AuthMethod
  abstract config: AuthProviderConfig

  abstract sendVerification(credentials: AuthCredentials): Promise<AuthProviderResult>
  abstract verifyCredentials(credentials: AuthCredentials): Promise<AuthProviderResult>
  abstract resendVerification(credentials: AuthCredentials): Promise<AuthProviderResult>
  abstract resetPassword?(credentials: AuthCredentials): Promise<AuthProviderResult>

  abstract validateInput(credentials: AuthCredentials): boolean
  abstract formatInput(input: string): string
}
