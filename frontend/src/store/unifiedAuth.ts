import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { createClient, User, Session } from '@supabase/supabase-js'
import { AuthUser, AuthResponse } from '@/types/auth'

// Combined user type that can handle both phone and Supabase auth
interface UnifiedUser {
  id: string
  phone_number?: string
  email?: string
  first_name?: string
  last_name?: string
  subscription_tier?: string
  subscription_status?: string
  subscription_expires_at?: string
  monthly_optimizations_used?: number
  monthly_simulations_used?: number
  usage_reset_date?: string
  is_active?: boolean
  created_at?: string
  updated_at?: string
  preferences?: any
  // Supabase user properties
  supabaseUser?: User
  // Phone auth user properties
  phoneAuthUser?: AuthUser
}

// Authentication methods
type AuthMethod = 'phone' | 'supabase' | 'magic_link'

interface UnifiedAuthState {
  // Core auth state
  user: UnifiedUser | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
  authMethod: AuthMethod | null

  // Phone auth specific
  phoneToken: string | null
  currentPhoneNumber: string | null
  otpSent: boolean
  verificationInProgress: boolean

  // Supabase auth specific
  supabaseSession: Session | null
  supabaseUser: User | null
  realtimeSubscription: any | null

  // Actions
  loginWithPhone: (phoneNumber: string) => Promise<void>
  verifyPhoneOTP: (phoneNumber: string, code: string) => Promise<void>
  resendPhoneOTP: (phoneNumber: string) => Promise<void>
  
  loginWithMagicLink: (email: string) => Promise<void>
  setSupabaseAuth: (user: User, session: Session) => void
  
  logout: () => Promise<void>
  refreshSession: () => Promise<void>
  getCurrentUser: () => Promise<void>
  
  // Real-time subscription management
  subscribeToUserUpdates: () => void
  unsubscribeFromUserUpdates: () => void
  
  // Error handling
  clearError: () => void
  setError: (error: string | null) => void
  setLoading: (loading: boolean) => void
}

// Initialize Supabase client
const supabaseUrl = import.meta.env.VITE_SUPABASE_URL || 'https://your-project.supabase.co'
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY || 'your-anon-key'
const supabase = createClient(supabaseUrl, supabaseAnonKey)

// Phone auth API functions
const sendPhoneOTP = async (phoneNumber: string) => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  try {
    const loginResponse = await fetch(`${apiUrl}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone_number: phoneNumber })
    })

    if (loginResponse.ok) {
      return await loginResponse.json()
    }
    
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
    
    const error = await loginResponse.json()
    throw new Error(error.error || error.message || 'Authentication failed')
    
  } catch (error) {
    if (error instanceof TypeError && error.message.includes('fetch')) {
      throw new Error('Network error. Please check your connection and try again.')
    }
    throw error
  }
}

const verifyPhoneOTP = async (phoneNumber: string, code: string): Promise<AuthResponse> => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
  const response = await fetch(`${apiUrl}/auth/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone_number: phoneNumber, code })
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || error.message || 'Verification failed')
  }

  return await response.json()
}

const resendPhoneOTP = async (phoneNumber: string) => {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
  
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

const refreshPhoneToken = async (token: string): Promise<AuthResponse> => {
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

  return await response.json()
}

// Automatic token refresh for phone auth
let refreshInterval: NodeJS.Timeout | null = null

const setupPhoneTokenRefresh = (authStore: any) => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
  
  refreshInterval = setInterval(async () => {
    try {
      const { phoneToken, isAuthenticated, authMethod } = authStore
      if (phoneToken && isAuthenticated && authMethod === 'phone') {
        await authStore.refreshSession()
        console.log('Phone token refreshed automatically')
      }
    } catch (error) {
      console.warn('Automatic phone token refresh failed:', error)
    }
  }, 50 * 60 * 1000) // 50 minutes
}

const clearPhoneTokenRefresh = () => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

export const useUnifiedAuthStore = create<UnifiedAuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
      authMethod: null,

      // Phone auth state
      phoneToken: null,
      currentPhoneNumber: null,
      otpSent: false,
      verificationInProgress: false,

      // Supabase auth state
      supabaseSession: null,
      supabaseUser: null,
      realtimeSubscription: null,

      // Phone authentication actions
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

      verifyPhoneOTP: async (phoneNumber: string, code: string) => {
        set({ verificationInProgress: true, error: null })
        
        try {
          const authResponse = await verifyPhoneOTP(phoneNumber, code)
          
          // Store token in localStorage for API interceptor
          localStorage.setItem('auth_token', authResponse.token)
          
          const unifiedUser: UnifiedUser = {
            id: authResponse.user.id,
            phone_number: authResponse.user.phone_number,
            first_name: authResponse.user.first_name,
            last_name: authResponse.user.last_name,
            subscription_tier: authResponse.user.subscription_tier,
            subscription_status: authResponse.user.subscription_status,
            subscription_expires_at: authResponse.user.subscription_expires_at,
            monthly_optimizations_used: authResponse.user.monthly_optimizations_used,
            monthly_simulations_used: authResponse.user.monthly_simulations_used,
            usage_reset_date: authResponse.user.usage_reset_date,
            is_active: authResponse.user.is_active,
            created_at: authResponse.user.created_at,
            updated_at: authResponse.user.updated_at,
            phoneAuthUser: authResponse.user
          }
          
          set({
            user: unifiedUser,
            phoneToken: authResponse.token,
            isAuthenticated: true,
            authMethod: 'phone',
            verificationInProgress: false,
            otpSent: false,
            currentPhoneNumber: null,
            error: null
          })
          
          // Set up automatic token refresh
          setupPhoneTokenRefresh(get())
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Verification failed'
          set({ 
            error: errorMessage, 
            verificationInProgress: false 
          })
          throw error
        }
      },

      resendPhoneOTP: async (phoneNumber: string) => {
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

      // Magic link authentication actions
      loginWithMagicLink: async (email: string) => {
        set({ isLoading: true, error: null })
        
        try {
          const { error } = await supabase.auth.signInWithOtp({
            email,
            options: {
              shouldCreateUser: true,
            }
          })
          
          if (error) throw error
          
          set({ isLoading: false })
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Failed to send magic link'
          set({ 
            error: errorMessage, 
            isLoading: false 
          })
          throw error
        }
      },

      setSupabaseAuth: (user: User, session: Session) => {
        const unifiedUser: UnifiedUser = {
          id: user.id,
          email: user.email,
          phone_number: user.phone,
          supabaseUser: user
        }
        
        set({
          user: unifiedUser,
          supabaseUser: user,
          supabaseSession: session,
          isAuthenticated: true,
          authMethod: 'magic_link',
          error: null
        })
        
        // Subscribe to real-time updates
        get().subscribeToUserUpdates()
      },

      logout: async () => {
        const { authMethod, phoneToken, realtimeSubscription } = get()
        
        // Unsubscribe from real-time updates
        if (realtimeSubscription) {
          get().unsubscribeFromUserUpdates()
        }
        
        // Phone auth logout
        if (authMethod === 'phone' && phoneToken) {
          try {
            const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
            await fetch(`${apiUrl}/auth/logout`, {
              method: 'POST',
              headers: { 
                'Authorization': `Bearer ${phoneToken}`
              }
            })
          } catch (error) {
            console.warn('Phone auth logout API call failed:', error)
          }
          
          localStorage.removeItem('auth_token')
          clearPhoneTokenRefresh()
        }
        
        // Supabase auth logout
        if (authMethod === 'magic_link' || authMethod === 'supabase') {
          await supabase.auth.signOut()
        }
        
        // Reset all state
        set({
          user: null,
          isAuthenticated: false,
          authMethod: null,
          phoneToken: null,
          currentPhoneNumber: null,
          otpSent: false,
          verificationInProgress: false,
          supabaseSession: null,
          supabaseUser: null,
          realtimeSubscription: null,
          error: null
        })
      },

      refreshSession: async () => {
        const { authMethod, phoneToken, supabaseSession } = get()
        
        if (authMethod === 'phone' && phoneToken) {
          try {
            const authResponse = await refreshPhoneToken(phoneToken)
            
            localStorage.setItem('auth_token', authResponse.token)
            
            const unifiedUser: UnifiedUser = {
              ...get().user!,
              phoneAuthUser: authResponse.user
            }
            
            set({
              phoneToken: authResponse.token,
              user: unifiedUser,
              isAuthenticated: true
            })
          } catch (error) {
            get().logout()
            throw error
          }
        } else if ((authMethod === 'magic_link' || authMethod === 'supabase') && supabaseSession) {
          try {
            const { data, error } = await supabase.auth.refreshSession()
            
            if (error) throw error
            
            if (data.session) {
              set({
                supabaseSession: data.session,
                supabaseUser: data.user,
                isAuthenticated: true
              })
            }
          } catch (error) {
            get().logout()
            throw error
          }
        }
      },

      getCurrentUser: async () => {
        const { authMethod, phoneToken, supabaseSession } = get()
        
        if (authMethod === 'phone' && phoneToken) {
          set({ isLoading: true })
          try {
            const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
            const response = await fetch(`${apiUrl}/auth/me`, {
              method: 'GET',
              headers: { 
                'Authorization': `Bearer ${phoneToken}`
              }
            })

            if (!response.ok) {
              throw new Error('Failed to get current user')
            }

            const userData = await response.json()
            const unifiedUser: UnifiedUser = {
              ...get().user!,
              phoneAuthUser: userData
            }
            
            set({ 
              user: unifiedUser, 
              isAuthenticated: true,
              isLoading: false 
            })
          } catch (error) {
            get().logout()
            set({ isLoading: false })
            throw error
          }
        } else if ((authMethod === 'magic_link' || authMethod === 'supabase') && supabaseSession) {
          set({ isLoading: true })
          try {
            const response = await fetch('/api/v1/users/me', {
              headers: {
                'Authorization': `Bearer ${supabaseSession.access_token}`,
                'Content-Type': 'application/json'
              }
            })

            if (response.ok) {
              const userData = await response.json()
              const unifiedUser: UnifiedUser = {
                ...get().user!,
                ...userData.data || userData
              }
              
              set({ 
                user: unifiedUser,
                isLoading: false 
              })
            } else {
              set({ isLoading: false })
            }
          } catch (error) {
            console.error('Failed to get current user:', error)
            set({ isLoading: false })
          }
        }
      },

      // Real-time subscription management
      subscribeToUserUpdates: () => {
        const { supabaseUser } = get()
        if (!supabaseUser) return

        console.log('Subscribing to user updates for:', supabaseUser.id)

        const subscription = supabase
          .channel(`user:${supabaseUser.id}`)
          .on('postgres_changes', {
            event: '*',
            schema: 'public',
            table: 'users',
            filter: `id=eq.${supabaseUser.id}`
          }, (payload) => {
            console.log('User data updated:', payload)
            
            if (payload.new) {
              const currentUser = get().user
              if (currentUser) {
                set({ 
                  user: {
                    ...currentUser,
                    ...payload.new
                  }
                })
              }
            }
          })
          .subscribe((status) => {
            console.log('Subscription status:', status)
          })

        set({ realtimeSubscription: subscription })
      },

      unsubscribeFromUserUpdates: () => {
        const { realtimeSubscription } = get()
        if (realtimeSubscription) {
          console.log('Unsubscribing from user updates')
          realtimeSubscription.unsubscribe()
          set({ realtimeSubscription: null })
        }
      },

      // Utility actions
      clearError: () => set({ error: null }),
      setError: (error: string | null) => set({ error }),
      setLoading: (loading: boolean) => set({ isLoading: loading }),
    }),
    {
      name: 'unified-auth-storage',
      version: 1,
      // Only persist essential auth data
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        authMethod: state.authMethod,
        phoneToken: state.phoneToken,
        supabaseSession: state.supabaseSession,
        supabaseUser: state.supabaseUser
      }),
      // Restore auth state and setup subscriptions
      onRehydrateStorage: () => (state) => {
        if (state?.isAuthenticated) {
          setTimeout(async () => {
            try {
              if (state.authMethod === 'phone' && state.phoneToken) {
                localStorage.setItem('auth_token', state.phoneToken)
                await state?.getCurrentUser?.()
                setupPhoneTokenRefresh(state)
              } else if (state.authMethod === 'magic_link' && state.supabaseSession) {
                await state?.refreshSession?.()
                await state?.getCurrentUser?.()
                state?.subscribeToUserUpdates?.()
              }
            } catch (error) {
              console.error('Failed to restore auth session:', error)
              state?.logout?.()
            }
          }, 100)
        }
      }
    }
  )
)

// Listen to Supabase auth state changes
supabase.auth.onAuthStateChange((event, session) => {
  const store = useUnifiedAuthStore.getState()
  
  console.log('Supabase auth state change:', event, session?.user?.id)
  
  switch (event) {
    case 'SIGNED_IN':
      if (session && session.user) {
        store.setSupabaseAuth(session.user, session)
      }
      break
    case 'SIGNED_OUT':
      // Only logout if currently using Supabase auth
      if (store.authMethod === 'magic_link' || store.authMethod === 'supabase') {
        store.logout()
      }
      break
    case 'TOKEN_REFRESHED':
      if (session && (store.authMethod === 'magic_link' || store.authMethod === 'supabase')) {
        useUnifiedAuthStore.setState({ 
          supabaseSession: session,
          supabaseUser: session?.user || null 
        })
      }
      break
  }
})

export { supabase }