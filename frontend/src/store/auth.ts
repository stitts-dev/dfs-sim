import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { AuthState, AuthUser, AuthResponse } from '@/types/auth'

// API endpoints for phone authentication via microservices
const sendPhoneOTP = async (phoneNumber: string) => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  // First try login for existing verified users (fast path)
  try {
    const loginResponse = await fetch(`${apiUrl}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone_number: phoneNumber })
    })

    if (loginResponse.ok) {
      return await loginResponse.json()
    }
    
    // If login fails with 404 (not registered) or 400 (not verified), 
    // fallback to register endpoint
    if (loginResponse.status === 404 || loginResponse.status === 400) {
      const registerResponse = await fetch(`${apiUrl}/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone_number: phoneNumber })
      })
      
      if (!registerResponse.ok) {
        const error = await registerResponse.json()
        throw new Error(error.error || error.message || 'Failed to send verification code')
      }
      
      return await registerResponse.json()
    }
    
    // For other login errors (not 404/400), throw the original error
    const error = await loginResponse.json()
    throw new Error(error.error || error.message || 'Authentication failed')
    
  } catch (error) {
    // If it's a network error, provide helpful message
    if (error instanceof TypeError && error.message.includes('fetch')) {
      throw new Error('Network error. Please check your connection and try again.')
    }
    
    // Re-throw other errors as-is
    throw error
  }
}

const verifyPhoneOTP = async (phoneNumber: string, code: string): Promise<AuthResponse> => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  // Use API Gateway -> User Service flow for OTP verification
  const response = await fetch(`${apiUrl}/auth/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone_number: phoneNumber, code })
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || error.message || 'Verification failed')
  }

  const data = await response.json()
  return data
}

const resendPhoneOTP = async (phoneNumber: string) => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  // Use API Gateway -> User Service flow for OTP resend
  const response = await fetch(`${apiUrl}/auth/resend`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone_number: phoneNumber })
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || error.message || 'Failed to resend OTP')
  }

  return await response.json()
}

const refreshAuthToken = async (token: string): Promise<AuthResponse> => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  const response = await fetch(`${apiUrl}/auth/refresh`, {
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
  return data
}

const getCurrentUserFromAPI = async (token: string): Promise<AuthUser> => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  const response = await fetch(`${apiUrl}/auth/me`, {
    method: 'GET',
    headers: { 
      'Authorization': `Bearer ${token}`
    }
  })

  if (!response.ok) {
    throw new Error('Failed to get current user')
  }

  const data = await response.json()
  return data
}

// Automatic token refresh setup
let refreshInterval: NodeJS.Timeout | null = null

const setupTokenRefresh = (authStore: any) => {
  // Clear existing interval
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
  
  // Set up automatic refresh every 50 minutes (tokens expire in 1 hour)
  refreshInterval = setInterval(async () => {
    try {
      const { token, isAuthenticated } = authStore
      if (token && isAuthenticated) {
        await authStore.refreshToken()
        console.log('Token refreshed automatically')
      }
    } catch (error) {
      console.warn('Automatic token refresh failed:', error)
      // If refresh fails, user will be logged out automatically by the refresh method
    }
  }, 50 * 60 * 1000) // 50 minutes
}

const clearTokenRefresh = () => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
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
          
          // Set up automatic token refresh for this session
          setupTokenRefresh(get())
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
        const { token } = get()
        
        // Call logout endpoint if we have a token
        if (token) {
          try {
            const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
            await fetch(`${apiUrl}/auth/logout`, {
              method: 'POST',
              headers: { 
                'Authorization': `Bearer ${token}`
              }
            })
          } catch (error) {
            console.warn('Logout API call failed:', error)
          }
        }

        // Clear local storage
        localStorage.removeItem('auth_token')
        
        // Clear automatic token refresh
        clearTokenRefresh()
        
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
          
          // Set up automatic token refresh
          setupTokenRefresh(state)
        }
      }
    }
  )
)