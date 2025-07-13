import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { AuthState, AuthUser, AuthResponse, PhoneAuthError } from '@/types/auth'
import * as api from '@/services/api'
import { sendOTPWithSupabase, verifyOTPWithSupabase, signOutWithSupabase, isSupabaseAvailable } from '@/services/supabase'

// API endpoints for phone authentication
const sendPhoneOTP = async (phoneNumber: string) => {
  // Try Supabase first if available, fallback to backend API
  if (isSupabaseAvailable()) {
    try {
      const response = await sendOTPWithSupabase(phoneNumber)
      if (response.error) {
        throw new Error(response.error.message)
      }
      return { success: true, message: 'OTP sent via Supabase' }
    } catch (error) {
      console.warn('Supabase OTP failed, falling back to backend API:', error)
    }
  }

  // Fallback to backend API
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone_number: phoneNumber })
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.message || 'Failed to send OTP')
  }

  return await response.json()
}

const verifyPhoneOTP = async (phoneNumber: string, code: string): Promise<AuthResponse> => {
  // Try Supabase first if available, fallback to backend API
  if (isSupabaseAvailable()) {
    try {
      const response = await verifyOTPWithSupabase(phoneNumber, code)
      if (response.error) {
        throw new Error(response.error.message)
      }
      
      // Convert Supabase response to our AuthResponse format
      if (response.session && response.user) {
        return {
          token: response.session.access_token,
          expires_at: new Date(response.session.expires_at * 1000).toISOString(),
          user: {
            id: parseInt(response.user.id), // Convert string to number
            phone_number: response.user.phone || phoneNumber,
            phone_verified: !!response.user.phone_confirmed_at,
            email: response.user.email,
            email_verified: !!response.user.email_confirmed_at,
            subscription_tier: 'free', // Default for new users
            subscription_status: 'active',
            is_active: true,
            created_at: response.user.created_at,
            updated_at: response.user.updated_at
          } as AuthUser,
          is_new_user: false // Supabase doesn't provide this info directly
        }
      }
    } catch (error) {
      console.warn('Supabase verification failed, falling back to backend API:', error)
    }
  }

  // Fallback to backend API
  const response = await fetch('/api/v1/auth/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone_number: phoneNumber, code })
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.message || 'Verification failed')
  }

  const data = await response.json()
  return data.data || data
}

const resendPhoneOTP = async (phoneNumber: string) => {
  // Always use backend API for resend to maintain rate limiting
  const response = await fetch('/api/v1/auth/resend', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone_number: phoneNumber })
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.message || 'Failed to resend OTP')
  }

  return await response.json()
}

const refreshAuthToken = async (token: string): Promise<AuthResponse> => {
  const response = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    }
  })

  if (!response.ok) {
    throw new Error('Token refresh failed')
  }

  const data = await response.json()
  return data.data || data
}

const getCurrentUserFromAPI = async (token: string): Promise<AuthUser> => {
  const response = await fetch('/api/v1/auth/me', {
    method: 'GET',
    headers: { 
      'Authorization': `Bearer ${token}`
    }
  })

  if (!response.ok) {
    throw new Error('Failed to get current user')
  }

  const data = await response.json()
  return data.data || data
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      token: null,
      isLoading: false,
      error: null,
      isAuthenticated: false,
      currentPhoneNumber: null,
      otpSent: false,
      verificationInProgress: false,

      // Actions
      loginWithPhone: async (phoneNumber: string) => {
        set({ isLoading: true, error: null, currentPhoneNumber: phoneNumber })
        
        try {
          await sendPhoneOTP(phoneNumber)
          set({ 
            otpSent: true, 
            isLoading: false, 
            currentPhoneNumber: phoneNumber 
          })
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Failed to send OTP'
          set({ 
            error: errorMessage, 
            isLoading: false, 
            otpSent: false 
          })
          throw error
        }
      },

      verifyOTP: async (phoneNumber: string, code: string) => {
        set({ verificationInProgress: true, error: null })
        
        try {
          const authResponse = await verifyPhoneOTP(phoneNumber, code)
          
          // Store token in localStorage for API interceptor
          localStorage.setItem('auth_token', authResponse.token)
          
          set({
            user: authResponse.user,
            token: authResponse.token,
            isAuthenticated: true,
            verificationInProgress: false,
            otpSent: false,
            currentPhoneNumber: null,
            error: null
          })
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Verification failed'
          set({ 
            error: errorMessage, 
            verificationInProgress: false 
          })
          throw error
        }
      },

      resendOTP: async (phoneNumber: string) => {
        set({ isLoading: true, error: null })
        
        try {
          await resendPhoneOTP(phoneNumber)
          set({ 
            isLoading: false,
            currentPhoneNumber: phoneNumber,
            otpSent: true
          })
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Failed to resend OTP'
          set({ 
            error: errorMessage, 
            isLoading: false 
          })
          throw error
        }
      },

      logout: async () => {
        try {
          // Try to sign out from Supabase if available
          if (isSupabaseAvailable()) {
            await signOutWithSupabase()
          }
        } catch (error) {
          console.warn('Supabase signout failed:', error)
        }

        // Clear local storage
        localStorage.removeItem('auth_token')
        
        // Reset auth state
        set({
          user: null,
          token: null,
          isAuthenticated: false,
          currentPhoneNumber: null,
          otpSent: false,
          verificationInProgress: false,
          error: null
        })
      },

      refreshToken: async () => {
        const { token } = get()
        if (!token) return

        try {
          const authResponse = await refreshAuthToken(token)
          
          // Update token in localStorage
          localStorage.setItem('auth_token', authResponse.token)
          
          set({
            token: authResponse.token,
            user: authResponse.user,
            isAuthenticated: true
          })
        } catch (error) {
          // If refresh fails, logout user
          get().logout()
          throw error
        }
      },

      getCurrentUser: async () => {
        const { token } = get()
        if (!token) return

        set({ isLoading: true })
        try {
          const user = await getCurrentUserFromAPI(token)
          set({ 
            user, 
            isAuthenticated: true,
            isLoading: false 
          })
        } catch (error) {
          // If getting user fails, logout
          get().logout()
          set({ isLoading: false })
          throw error
        }
      },

      clearError: () => set({ error: null }),
      
      setLoading: (loading: boolean) => set({ isLoading: loading }),
      
      setError: (error: string | null) => set({ error }),
      
      setCurrentPhoneNumber: (phoneNumber: string | null) => 
        set({ currentPhoneNumber: phoneNumber }),
      
      setOtpSent: (sent: boolean) => set({ otpSent: sent }),
    }),
    {
      name: 'dfs-auth',
      version: 1,
      // Only persist user and token, not loading states
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        isAuthenticated: state.isAuthenticated
      }),
      // Restore isAuthenticated state based on token presence
      onRehydrateStorage: () => (state) => {
        if (state?.token) {
          localStorage.setItem('auth_token', state.token)
          // Verify token is still valid by getting current user
          setTimeout(() => {
            state?.getCurrentUser?.()
          }, 100)
        }
      }
    }
  )
)