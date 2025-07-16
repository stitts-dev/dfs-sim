import { useState } from 'react'
import { useMutation, useQueryClient } from 'react-query'
import { useUnifiedAuthStore } from '@/store/unifiedAuth'
// import { PhoneAuthError } from '@/types/auth'

/**
 * Hook for phone authentication with React Query integration
 */
export const usePhoneAuth = () => {
  const queryClient = useQueryClient()
  const {
    loginWithPhone,
    verifyPhoneOTP,
    resendPhoneOTP,
    logout,
    clearError,
    user,
    phoneToken,
    isAuthenticated,
    isLoading,
    error,
    currentPhoneNumber,
    otpSent,
    verificationInProgress
  } = useUnifiedAuthStore()
  
  // Provide backward compatibility
  const verifyOTP = verifyPhoneOTP
  const resendOTP = resendPhoneOTP
  const token = phoneToken

  // Send OTP mutation
  const sendOTPMutation = useMutation(
    (phoneNumber: string) => loginWithPhone(phoneNumber),
    {
      onSuccess: () => {
        // Invalidate any user-related queries
        queryClient.invalidateQueries(['user'])
      },
      onError: (error: Error) => {
        console.error('Send OTP failed:', error)
      }
    }
  )

  // Verify OTP mutation
  const verifyOTPMutation = useMutation(
    ({ phoneNumber, code }: { phoneNumber: string; code: string }) => 
      verifyOTP(phoneNumber, code),
    {
      onSuccess: () => {
        // Invalidate all queries and refetch user data
        queryClient.invalidateQueries()
        queryClient.refetchQueries(['user'])
      },
      onError: (error: Error) => {
        console.error('Verify OTP failed:', error)
      }
    }
  )

  // Resend OTP mutation
  const resendOTPMutation = useMutation(
    (phoneNumber: string) => resendOTP(phoneNumber),
    {
      onError: (error: Error) => {
        console.error('Resend OTP failed:', error)
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
        // Clear all cached data
        queryClient.clear()
      }
    }
  )

  // Helper function to handle phone number formatting
  const formatPhoneNumber = (phone: string): string => {
    const cleaned = phone.replace(/\D/g, '')
    
    if (cleaned.length === 0) return ''
    if (cleaned.length <= 3) return cleaned
    if (cleaned.length <= 6) return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3)}`
    
    return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6, 10)}`
  }

  // Helper function to validate phone number
  const validatePhoneNumber = (phone: string): boolean => {
    const cleaned = phone.replace(/\D/g, '')
    return cleaned.length === 10 || cleaned.length === 11
  }

  // Helper function to normalize phone number to E.164 format
  const normalizePhoneNumber = (phone: string): string => {
    let cleaned = phone.replace(/\D/g, '')
    
    // Add country code if missing
    if (cleaned.length === 10) {
      cleaned = '1' + cleaned
    }
    
    return '+' + cleaned
  }

  return {
    // State
    user,
    token,
    isAuthenticated,
    currentPhoneNumber,
    otpSent,
    
    // Loading states
    isLoading: isLoading || verificationInProgress,
    isSendingOTP: sendOTPMutation.isLoading,
    isVerifyingOTP: verifyOTPMutation.isLoading,
    isResendingOTP: resendOTPMutation.isLoading,
    isLoggingOut: logoutMutation.isLoading,
    
    // Error states
    error: error || 
           (sendOTPMutation.error as any)?.message || 
           (verifyOTPMutation.error as any)?.message || 
           (resendOTPMutation.error as any)?.message || 
           (logoutMutation.error as any)?.message,
    
    // Actions
    sendOTP: (phoneNumber: string) => {
      const normalized = normalizePhoneNumber(phoneNumber)
      return sendOTPMutation.mutateAsync(normalized)
    },
    
    verifyCode: (phoneNumber: string, code: string) => {
      const normalized = normalizePhoneNumber(phoneNumber)
      return verifyOTPMutation.mutateAsync({ phoneNumber: normalized, code })
    },
    
    resendCode: (phoneNumber: string) => {
      const normalized = normalizePhoneNumber(phoneNumber)
      return resendOTPMutation.mutateAsync(normalized)
    },
    
    signOut: () => logoutMutation.mutateAsync(),
    
    clearError,
    
    // Utility functions
    formatPhoneNumber,
    validatePhoneNumber,
    normalizePhoneNumber,
    
    // Reset mutations
    resetSendOTP: () => sendOTPMutation.reset(),
    resetVerifyOTP: () => verifyOTPMutation.reset(),
    resetResendOTP: () => resendOTPMutation.reset(),
  }
}

/**
 * Hook to check authentication status and redirect if needed
 */
export const useAuthGuard = (redirectTo: string = '/auth') => {
  const { isAuthenticated, isLoading } = useAuthStore()
  
  // This would typically be used with a router
  // For now, just return the status
  return {
    isAuthenticated,
    isLoading,
    shouldRedirect: !isLoading && !isAuthenticated,
    redirectTo
  }
}

/**
 * Hook for managing authentication form state
 */
export const useAuthForm = () => {
  const [phoneNumber, setPhoneNumber] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [errors, setErrors] = useState<{
    phoneNumber?: string
    verificationCode?: string
  }>({})

  const { formatPhoneNumber, validatePhoneNumber } = usePhoneAuth()

  const handlePhoneChange = (value: string) => {
    // Format as user types
    const formatted = formatPhoneNumber(value)
    setPhoneNumber(formatted)
    
    // Clear phone number error when user starts typing
    if (errors.phoneNumber) {
      setErrors(prev => ({ ...prev, phoneNumber: undefined }))
    }
  }

  const handleCodeChange = (value: string) => {
    // Only allow digits and limit to 6 characters
    const cleaned = value.replace(/\D/g, '').slice(0, 6)
    setVerificationCode(cleaned)
    
    // Clear verification code error when user starts typing
    if (errors.verificationCode) {
      setErrors(prev => ({ ...prev, verificationCode: undefined }))
    }
  }

  const validateForm = () => {
    const newErrors: typeof errors = {}
    
    if (!phoneNumber) {
      newErrors.phoneNumber = 'Phone number is required'
    } else if (!validatePhoneNumber(phoneNumber)) {
      newErrors.phoneNumber = 'Please enter a valid phone number'
    }
    
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const validateVerificationCode = () => {
    const newErrors: typeof errors = {}
    
    if (!verificationCode) {
      newErrors.verificationCode = 'Verification code is required'
    } else if (verificationCode.length !== 6) {
      newErrors.verificationCode = 'Verification code must be 6 digits'
    }
    
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  return {
    phoneNumber,
    verificationCode,
    errors,
    handlePhoneChange,
    handleCodeChange,
    validateForm,
    validateVerificationCode,
    setErrors,
    clearErrors: () => setErrors({})
  }
}