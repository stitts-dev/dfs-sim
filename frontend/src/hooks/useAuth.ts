import { useState, useMemo } from 'react'
import { useMutation, useQueryClient } from 'react-query'
import { AuthMethod, AuthCredentials, AuthProviderResult, AuthProvider } from '@/types/auth'
import { EmailAuthProvider } from '@/providers/EmailAuthProvider'
import { PhoneAuthProvider } from '@/providers/PhoneAuthProvider'
import { useUnifiedAuthStore } from '@/store/unifiedAuth'
import { usePhoneAuth } from './usePhoneAuth'

interface UseAuthConfig {
  enabledMethods?: AuthMethod[]
  defaultMethod?: AuthMethod
}

/**
 * Unified authentication hook that provides a clean interface for components
 * Supports multiple authentication methods with provider pattern
 */
export const useAuth = (config: UseAuthConfig = {}) => {
  const queryClient = useQueryClient()
  const {
    user,
    phoneToken,
    supabaseSession,
    authMethod,
    isAuthenticated,
    isLoading,
    error,
    currentPhoneNumber,
    otpSent,
    verificationInProgress,
    logout,
    clearError
  } = useUnifiedAuthStore()
  
  // Get the appropriate token based on auth method
  const token = authMethod === 'phone' ? phoneToken : supabaseSession?.access_token

  const phoneAuth = usePhoneAuth()

  const {
    enabledMethods = ['email'],
    defaultMethod = enabledMethods[0] || 'email'
  } = config

  const [currentMethod, setCurrentMethod] = useState<AuthMethod>(defaultMethod)

  // Initialize providers
  const providers = useMemo<Record<AuthMethod, AuthProvider>>(() => {
    return {
      email: new EmailAuthProvider(),
      phone: new PhoneAuthProvider()
    }
  }, [])

  const currentProvider = providers[currentMethod]

  // Send verification mutation
  const sendVerificationMutation = useMutation(
    (credentials: AuthCredentials) => currentProvider.sendVerification(credentials),
    {
      onSuccess: (result: AuthProviderResult) => {
        if (result.success) {
          queryClient.invalidateQueries(['user'])
        }
      },
      onError: (error: Error) => {
        console.error('Send verification failed:', error)
      }
    }
  )

  // Verify credentials mutation
  const verifyCredentialsMutation = useMutation(
    (credentials: AuthCredentials) => currentProvider.verifyCredentials(credentials),
    {
      onSuccess: (result: AuthProviderResult) => {
        if (result.success && result.user && result.token) {
          // Update auth store with successful authentication
          queryClient.invalidateQueries()
          queryClient.refetchQueries(['user'])
        }
      },
      onError: (error: Error) => {
        console.error('Verify credentials failed:', error)
      }
    }
  )

  // Resend verification mutation
  const resendVerificationMutation = useMutation(
    (credentials: AuthCredentials) => currentProvider.resendVerification(credentials),
    {
      onError: (error: Error) => {
        console.error('Resend verification failed:', error)
      }
    }
  )

  // Reset password mutation (email only)
  const resetPasswordMutation = useMutation(
    (credentials: AuthCredentials) => {
      if (currentMethod === 'email' && 'resetPassword' in currentProvider) {
        return currentProvider.resetPassword!(credentials)
      }
      return Promise.reject(new Error('Password reset not supported for this method'))
    },
    {
      onError: (error: Error) => {
        console.error('Reset password failed:', error)
      }
    }
  )

  // Logout mutation
  const logoutMutation = useMutation(
    async () => {
      await logout()
    },
    {
      onSuccess: () => {
        queryClient.clear()
      }
    }
  )

  // Validation and formatting helpers
  const validateInput = (credentials: AuthCredentials): boolean => {
    return currentProvider.validateInput(credentials)
  }

  const formatInput = (input: string): string => {
    return currentProvider.formatInput(input)
  }

  // Check if method is enabled
  const isMethodEnabled = (method: AuthMethod): boolean => {
    return enabledMethods.includes(method) && providers[method].config.enabled
  }

  // Get available methods
  const getAvailableMethods = (): AuthMethod[] => {
    return enabledMethods.filter(method => isMethodEnabled(method))
  }

  // Switch authentication method
  const switchMethod = (method: AuthMethod) => {
    if (isMethodEnabled(method)) {
      setCurrentMethod(method)
      clearError()
    }
  }

  return {
    // State
    user,
    token,
    isAuthenticated,
    currentMethod,
    enabledMethods: getAvailableMethods(),
    currentPhoneNumber,
    otpSent,
    
    // Loading states
    isLoading: isLoading || verificationInProgress,
    isSendingVerification: sendVerificationMutation.isLoading,
    isVerifyingCredentials: verifyCredentialsMutation.isLoading,
    isResendingVerification: resendVerificationMutation.isLoading,
    isResettingPassword: resetPasswordMutation.isLoading,
    isLoggingOut: logoutMutation.isLoading,
    
    // Error states
    error: error || 
           (sendVerificationMutation.error as any)?.message || 
           (verifyCredentialsMutation.error as any)?.message || 
           (resendVerificationMutation.error as any)?.message || 
           (resetPasswordMutation.error as any)?.message ||
           (logoutMutation.error as any)?.message,
    
    // Actions
    sendVerification: (credentials: Omit<AuthCredentials, 'method'>) => {
      return sendVerificationMutation.mutateAsync({
        ...credentials,
        method: currentMethod
      })
    },
    
    verifyCredentials: (credentials: Omit<AuthCredentials, 'method'>) => {
      return verifyCredentialsMutation.mutateAsync({
        ...credentials,
        method: currentMethod
      })
    },
    
    resendVerification: (credentials: Omit<AuthCredentials, 'method'>) => {
      return resendVerificationMutation.mutateAsync({
        ...credentials,
        method: currentMethod
      })
    },
    
    resetPassword: (credentials: Omit<AuthCredentials, 'method'>) => {
      return resetPasswordMutation.mutateAsync({
        ...credentials,
        method: currentMethod
      })
    },
    
    signOut: () => logoutMutation.mutateAsync(),
    logout: () => logoutMutation.mutateAsync(),
    
    clearError,
    switchMethod,
    
    // Utility functions
    validateInput,
    formatInput,
    isMethodEnabled,
    getAvailableMethods,
    
    // Provider info
    currentProvider: {
      method: currentProvider.method,
      config: currentProvider.config
    },
    
    // Legacy phone auth support for backward compatibility
    phoneAuth,
    
    // Reset mutations
    resetSendVerification: () => sendVerificationMutation.reset(),
    resetVerifyCredentials: () => verifyCredentialsMutation.reset(),
    resetResendVerification: () => resendVerificationMutation.reset(),
    resetResetPassword: () => resetPasswordMutation.reset(),
  }
}