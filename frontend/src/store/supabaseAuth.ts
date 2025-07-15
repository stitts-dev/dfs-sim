import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { createClient, User, Session } from '@supabase/supabase-js'

// Types for Supabase-only auth state
interface SupabaseUser {
  id: string
  phone_number: string
  first_name?: string
  last_name?: string
  subscription_tier: string
  subscription_status: string
  subscription_expires_at?: string
  monthly_optimizations_used: number
  monthly_simulations_used: number
  usage_reset_date: string
  is_active: boolean
  created_at: string
  updated_at: string
  preferences?: SupabaseUserPreferences
}

interface SupabaseUserPreferences {
  id: string
  user_id: string
  sport_preferences: string[]
  platform_preferences: string[]
  contest_type_preferences: string[]
  theme: string
  language: string
  notifications_enabled: boolean
  tutorial_completed: boolean
  beginner_mode: boolean
  tooltips_enabled: boolean
  created_at: string
  updated_at: string
}

interface SupabaseAuthState {
  // Auth state
  user: User | null
  session: Session | null
  supabaseUser: SupabaseUser | null
  isLoading: boolean
  error: string | null
  isAuthenticated: boolean

  // Phone auth specific
  currentPhoneNumber: string | null
  otpSent: boolean
  verificationInProgress: boolean

  // Real-time subscriptions
  realtimeSubscription: any | null

  // Actions
  loginWithPhone: (phoneNumber: string) => Promise<void>
  verifyOTP: (phoneNumber: string, code: string) => Promise<void>
  resendOTP: (phoneNumber: string) => Promise<void>
  logout: () => Promise<void>
  refreshSession: () => Promise<void>
  getCurrentUser: () => Promise<void>

  // Real-time subscription management
  subscribeToUserUpdates: () => void
  unsubscribeFromUserUpdates: () => void

  // Error handling
  clearError: () => void
}

// Initialize Supabase client
const supabaseUrl = import.meta.env.VITE_SUPABASE_URL || 'https://your-project.supabase.co'
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY || 'your-anon-key'

const supabase = createClient(supabaseUrl, supabaseAnonKey)

export const useSupabaseAuthStore = create<SupabaseAuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      session: null,
      supabaseUser: null,
      isLoading: false,
      error: null,
      isAuthenticated: false,
      currentPhoneNumber: null,
      otpSent: false,
      verificationInProgress: false,
      realtimeSubscription: null,

      // Phone authentication flow
      loginWithPhone: async (phoneNumber: string) => {
        set({ isLoading: true, error: null, currentPhoneNumber: phoneNumber })
        
        try {
          const { error } = await supabase.auth.signInWithOtp({
            phone: phoneNumber,
          })
          
          if (error) throw error
          
          set({ otpSent: true, isLoading: false })
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
          const { data, error } = await supabase.auth.verifyOtp({
            phone: phoneNumber,
            token: code,
            type: 'sms'
          })
          
          if (error) throw error
          
          if (data.user && data.session) {
            set({
              user: data.user,
              session: data.session,
              isAuthenticated: true,
              verificationInProgress: false,
              otpSent: false,
              currentPhoneNumber: null
            })

            // Fetch user profile from our backend
            await get().getCurrentUser()
            
            // Subscribe to real-time updates
            get().subscribeToUserUpdates()
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Invalid verification code'
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
          const { error } = await supabase.auth.signInWithOtp({
            phone: phoneNumber,
          })
          
          if (error) throw error
          
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
        // Unsubscribe from real-time updates
        get().unsubscribeFromUserUpdates()
        
        // Sign out from Supabase
        await supabase.auth.signOut()
        
        // Reset all state
        set({
          user: null,
          session: null,
          supabaseUser: null,
          isAuthenticated: false,
          currentPhoneNumber: null,
          otpSent: false,
          verificationInProgress: false,
          error: null,
          realtimeSubscription: null
        })
      },

      refreshSession: async () => {
        try {
          const { data, error } = await supabase.auth.refreshSession()
          
          if (error) throw error
          
          if (data.session) {
            set({
              session: data.session,
              user: data.user,
              isAuthenticated: true
            })
          }
        } catch (error) {
          // If refresh fails, logout user
          get().logout()
          throw error
        }
      },

      getCurrentUser: async () => {
        const { session } = get()
        if (!session?.access_token) return

        set({ isLoading: true })
        try {
          // Fetch user profile from our backend API
          const response = await fetch('/api/v1/users/me', {
            headers: {
              'Authorization': `Bearer ${session.access_token}`,
              'Content-Type': 'application/json'
            }
          })

          if (!response.ok) {
            throw new Error('Failed to fetch user profile')
          }

          const userData = await response.json()
          set({ 
            supabaseUser: userData.data || userData,
            isLoading: false 
          })
        } catch (error) {
          console.error('Failed to get current user:', error)
          set({ isLoading: false })
          // Don't logout on profile fetch failure, user might still be authenticated
        }
      },

      // Real-time subscription management
      subscribeToUserUpdates: () => {
        const { user } = get()
        if (!user) return

        console.log('Subscribing to user updates for:', user.id)

        // Subscribe to user data changes
        const subscription = supabase
          .channel(`user:${user.id}`)
          .on('postgres_changes', {
            event: '*',
            schema: 'public',
            table: 'users',
            filter: `id=eq.${user.id}`
          }, (payload) => {
            console.log('User data updated:', payload)
            
            // Update user data in state
            if (payload.new) {
              set({ supabaseUser: payload.new as SupabaseUser })
            }
          })
          .on('postgres_changes', {
            event: '*',
            schema: 'public',
            table: 'user_preferences',
            filter: `user_id=eq.${user.id}`
          }, (payload) => {
            console.log('User preferences updated:', payload)
            
            // Update preferences in user data
            const { supabaseUser } = get()
            if (supabaseUser && payload.new) {
              set({
                supabaseUser: {
                  ...supabaseUser,
                  preferences: payload.new as SupabaseUserPreferences
                }
              })
            }
          })
          .subscribe((status) => {
            console.log('Subscription status:', status)
          })

        // Store subscription for cleanup
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

      clearError: () => set({ error: null }),
    }),
    {
      name: 'supabase-auth-storage',
      version: 1,
      // Only persist essential auth data
      partialize: (state) => ({
        user: state.user,
        session: state.session,
        supabaseUser: state.supabaseUser,
        isAuthenticated: state.isAuthenticated
      }),
      // Restore auth state and setup subscriptions
      onRehydrateStorage: () => (state) => {
        if (state?.session?.access_token) {
          // Verify session is still valid
          setTimeout(async () => {
            try {
              await state?.refreshSession?.()
              await state?.getCurrentUser?.()
              state?.subscribeToUserUpdates?.()
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
  const store = useSupabaseAuthStore.getState()
  
  console.log('Supabase auth state change:', event, session?.user?.id)
  
  switch (event) {
    case 'SIGNED_IN':
      if (session) {
        store.subscribeToUserUpdates()
      }
      break
    case 'SIGNED_OUT':
      store.unsubscribeFromUserUpdates()
      break
    case 'TOKEN_REFRESHED':
      // Update session in store
      useSupabaseAuthStore.setState({ 
        session: session,
        user: session?.user || null 
      })
      break
  }
})

export { supabase }