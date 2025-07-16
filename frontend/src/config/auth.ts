import { AuthMethod } from '@/types/auth'

export interface AuthConfig {
  enabledMethods: AuthMethod[]
  defaultMethod: AuthMethod
  emailAuth: {
    enabled: boolean
    requiresVerification: boolean
    passwordStrengthRequired: boolean
    allowPasswordReset: boolean
  }
  phoneAuth: {
    enabled: boolean
    requiresVerification: boolean
    countryCode: string
  }
  features: {
    multiMethodSelection: boolean
    rememberLastMethod: boolean
    socialAuth: boolean
    twoFactorAuth: boolean
  }
}

// Default configuration - Email only as requested
export const defaultAuthConfig: AuthConfig = {
  enabledMethods: ['email'],
  defaultMethod: 'email',
  emailAuth: {
    enabled: true,
    requiresVerification: true,
    passwordStrengthRequired: true,
    allowPasswordReset: true
  },
  phoneAuth: {
    enabled: false, // Disabled by default as requested
    requiresVerification: true,
    countryCode: '+1'
  },
  features: {
    multiMethodSelection: false, // Since only email is enabled
    rememberLastMethod: false,
    socialAuth: false,
    twoFactorAuth: false
  }
}

// Configuration for development/testing with multiple methods
export const devAuthConfig: AuthConfig = {
  enabledMethods: ['email', 'phone'],
  defaultMethod: 'email',
  emailAuth: {
    enabled: true,
    requiresVerification: true,
    passwordStrengthRequired: true,
    allowPasswordReset: true
  },
  phoneAuth: {
    enabled: true,
    requiresVerification: true,
    countryCode: '+1'
  },
  features: {
    multiMethodSelection: true,
    rememberLastMethod: true,
    socialAuth: false,
    twoFactorAuth: false
  }
}

// Get auth config based on environment
export const getAuthConfig = (): AuthConfig => {
  const isDevelopment = import.meta.env.MODE === 'development'
  const enableMultiAuth = import.meta.env.VITE_ENABLE_MULTI_AUTH === 'true'
  
  if (isDevelopment && enableMultiAuth) {
    return devAuthConfig
  }
  
  return defaultAuthConfig
}

// Feature flag helpers
export const isAuthMethodEnabled = (method: AuthMethod): boolean => {
  const config = getAuthConfig()
  return config.enabledMethods.includes(method)
}

export const isFeatureEnabled = (feature: keyof AuthConfig['features']): boolean => {
  const config = getAuthConfig()
  return config.features[feature]
}

export const getEnabledAuthMethods = (): AuthMethod[] => {
  const config = getAuthConfig()
  return config.enabledMethods
}

export const getDefaultAuthMethod = (): AuthMethod => {
  const config = getAuthConfig()
  return config.defaultMethod
}

// Email auth config helpers
export const isEmailAuthEnabled = (): boolean => {
  const config = getAuthConfig()
  return config.emailAuth.enabled
}

export const isEmailVerificationRequired = (): boolean => {
  const config = getAuthConfig()
  return config.emailAuth.requiresVerification
}

export const isPasswordStrengthRequired = (): boolean => {
  const config = getAuthConfig()
  return config.emailAuth.passwordStrengthRequired
}

export const isPasswordResetAllowed = (): boolean => {
  const config = getAuthConfig()
  return config.emailAuth.allowPasswordReset
}

// Phone auth config helpers
export const isPhoneAuthEnabled = (): boolean => {
  const config = getAuthConfig()
  return config.phoneAuth.enabled
}

export const isPhoneVerificationRequired = (): boolean => {
  const config = getAuthConfig()
  return config.phoneAuth.requiresVerification
}

export const getPhoneCountryCode = (): string => {
  const config = getAuthConfig()
  return config.phoneAuth.countryCode
}

export default getAuthConfig