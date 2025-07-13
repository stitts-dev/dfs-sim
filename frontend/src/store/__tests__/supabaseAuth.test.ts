import { renderHook, act } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useSupabaseAuthStore } from '../supabaseAuth'

// Mock Supabase client
const mockSupabase = {
  auth: {
    signInWithOtp: vi.fn(),
    verifyOtp: vi.fn(),
    signOut: vi.fn(),
    refreshSession: vi.fn(),
    getSession: vi.fn(),
    onAuthStateChange: vi.fn()
  },
  channel: vi.fn(() => ({
    on: vi.fn().mockReturnThis(),
    subscribe: vi.fn().mockReturnThis(),
    unsubscribe: vi.fn()
  }))
}

// Mock the supabase module
vi.mock('@supabase/supabase-js', () => ({
  createClient: () => mockSupabase
}))

// Mock environment variables
vi.mock('import.meta', () => ({
  env: {
    VITE_SUPABASE_URL: 'https://test.supabase.co',
    VITE_SUPABASE_ANON_KEY: 'test-anon-key'
  }
}))

describe('useSupabaseAuthStore', () => {
  beforeEach(() => {
    // Reset the store before each test
    useSupabaseAuthStore.getState().logout()
    vi.clearAllMocks()
  })

  describe('Phone Authentication Flow', () => {
    it('should handle phone login successfully', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      mockSupabase.auth.signInWithOtp.mockResolvedValueOnce({ 
        error: null 
      })

      await act(async () => {
        await result.current.loginWithPhone('+1234567890')
      })

      expect(mockSupabase.auth.signInWithOtp).toHaveBeenCalledWith({
        phone: '+1234567890'
      })
      expect(result.current.otpSent).toBe(true)
      expect(result.current.currentPhoneNumber).toBe('+1234567890')
      expect(result.current.isLoading).toBe(false)
      expect(result.current.error).toBe(null)
    })

    it('should handle phone login failure', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const error = new Error('Invalid phone number')
      mockSupabase.auth.signInWithOtp.mockResolvedValueOnce({ 
        error 
      })

      await act(async () => {
        try {
          await result.current.loginWithPhone('+invalid')
        } catch (e) {
          // Expected to throw
        }
      })

      expect(result.current.otpSent).toBe(false)
      expect(result.current.error).toBe('Invalid phone number')
      expect(result.current.isLoading).toBe(false)
    })

    it('should handle OTP verification successfully', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const mockUser = {
        id: '550e8400-e29b-41d4-a716-446655440000',
        phone: '+1234567890'
      }
      const mockSession = {
        access_token: 'test-token',
        user: mockUser
      }

      mockSupabase.auth.verifyOtp.mockResolvedValueOnce({
        data: {
          user: mockUser,
          session: mockSession
        },
        error: null
      })

      // Mock fetch for getCurrentUser
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          data: {
            id: mockUser.id,
            phone_number: '+1234567890',
            subscription_tier: 'free'
          }
        })
      })

      await act(async () => {
        await result.current.verifyOTP('+1234567890', '123456')
      })

      expect(mockSupabase.auth.verifyOtp).toHaveBeenCalledWith({
        phone: '+1234567890',
        token: '123456',
        type: 'sms'
      })
      expect(result.current.isAuthenticated).toBe(true)
      expect(result.current.user).toEqual(mockUser)
      expect(result.current.session).toEqual(mockSession)
      expect(result.current.verificationInProgress).toBe(false)
      expect(result.current.otpSent).toBe(false)
    })

    it('should handle OTP verification failure', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const error = new Error('Invalid OTP code')
      mockSupabase.auth.verifyOtp.mockResolvedValueOnce({
        data: null,
        error
      })

      await act(async () => {
        try {
          await result.current.verifyOTP('+1234567890', '000000')
        } catch (e) {
          // Expected to throw
        }
      })

      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.error).toBe('Invalid OTP code')
      expect(result.current.verificationInProgress).toBe(false)
    })

    it('should handle OTP resend successfully', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      mockSupabase.auth.signInWithOtp.mockResolvedValueOnce({ 
        error: null 
      })

      await act(async () => {
        await result.current.resendOTP('+1234567890')
      })

      expect(mockSupabase.auth.signInWithOtp).toHaveBeenCalledWith({
        phone: '+1234567890'
      })
      expect(result.current.otpSent).toBe(true)
      expect(result.current.currentPhoneNumber).toBe('+1234567890')
      expect(result.current.isLoading).toBe(false)
    })
  })

  describe('Session Management', () => {
    it('should handle logout successfully', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      // Set up authenticated state
      act(() => {
        useSupabaseAuthStore.setState({
          isAuthenticated: true,
          user: { id: 'test-user' } as any,
          session: { access_token: 'test-token' } as any,
          realtimeSubscription: { unsubscribe: vi.fn() }
        })
      })

      mockSupabase.auth.signOut.mockResolvedValueOnce({ error: null })

      await act(async () => {
        await result.current.logout()
      })

      expect(mockSupabase.auth.signOut).toHaveBeenCalled()
      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.user).toBe(null)
      expect(result.current.session).toBe(null)
      expect(result.current.supabaseUser).toBe(null)
    })

    it('should handle session refresh successfully', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const newSession = {
        access_token: 'new-token',
        user: { id: 'test-user' }
      }

      mockSupabase.auth.refreshSession.mockResolvedValueOnce({
        data: { session: newSession },
        error: null
      })

      await act(async () => {
        await result.current.refreshSession()
      })

      expect(mockSupabase.auth.refreshSession).toHaveBeenCalled()
      expect(result.current.session).toEqual(newSession)
      expect(result.current.isAuthenticated).toBe(true)
    })

    it('should logout user when session refresh fails', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      // Set up authenticated state
      act(() => {
        useSupabaseAuthStore.setState({
          isAuthenticated: true,
          session: { access_token: 'expired-token' } as any
        })
      })

      const error = new Error('Session expired')
      mockSupabase.auth.refreshSession.mockResolvedValueOnce({
        data: null,
        error
      })
      mockSupabase.auth.signOut.mockResolvedValueOnce({ error: null })

      await act(async () => {
        try {
          await result.current.refreshSession()
        } catch (e) {
          // Expected to throw
        }
      })

      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.session).toBe(null)
    })
  })

  describe('User Profile Management', () => {
    it('should fetch current user profile successfully', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const mockUserProfile = {
        id: 'test-user',
        phone_number: '+1234567890',
        subscription_tier: 'premium',
        preferences: {
          theme: 'dark',
          beginner_mode: false
        }
      }

      // Set up authenticated state
      act(() => {
        useSupabaseAuthStore.setState({
          session: { access_token: 'test-token' } as any
        })
      })

      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: mockUserProfile })
      })

      await act(async () => {
        await result.current.getCurrentUser()
      })

      expect(global.fetch).toHaveBeenCalledWith('/api/v1/users/me', {
        headers: {
          'Authorization': 'Bearer test-token',
          'Content-Type': 'application/json'
        }
      })
      expect(result.current.supabaseUser).toEqual(mockUserProfile)
      expect(result.current.isLoading).toBe(false)
    })

    it('should handle getCurrentUser API failure gracefully', async () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      // Set up authenticated state
      act(() => {
        useSupabaseAuthStore.setState({
          session: { access_token: 'test-token' } as any
        })
      })

      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: false,
        status: 404
      })

      await act(async () => {
        await result.current.getCurrentUser()
      })

      expect(result.current.supabaseUser).toBe(null)
      expect(result.current.isLoading).toBe(false)
      // Should not logout user on profile fetch failure
      expect(result.current.session).not.toBe(null)
    })
  })

  describe('Real-time Subscriptions', () => {
    it('should subscribe to user updates when user is authenticated', () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const mockSubscription = {
        on: vi.fn().mockReturnThis(),
        subscribe: vi.fn().mockReturnThis(),
        unsubscribe: vi.fn()
      }

      mockSupabase.channel.mockReturnValueOnce(mockSubscription)

      // Set up authenticated state
      act(() => {
        useSupabaseAuthStore.setState({
          user: { id: 'test-user' } as any
        })
      })

      act(() => {
        result.current.subscribeToUserUpdates()
      })

      expect(mockSupabase.channel).toHaveBeenCalledWith('user:test-user')
      expect(mockSubscription.on).toHaveBeenCalledTimes(2) // users and user_preferences tables
      expect(mockSubscription.subscribe).toHaveBeenCalled()
      expect(result.current.realtimeSubscription).toBe(mockSubscription)
    })

    it('should unsubscribe from user updates', () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const mockSubscription = {
        unsubscribe: vi.fn()
      }

      // Set up subscription state
      act(() => {
        useSupabaseAuthStore.setState({
          realtimeSubscription: mockSubscription
        })
      })

      act(() => {
        result.current.unsubscribeFromUserUpdates()
      })

      expect(mockSubscription.unsubscribe).toHaveBeenCalled()
      expect(result.current.realtimeSubscription).toBe(null)
    })

    it('should not subscribe when user is not authenticated', () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      act(() => {
        result.current.subscribeToUserUpdates()
      })

      expect(mockSupabase.channel).not.toHaveBeenCalled()
    })
  })

  describe('Error Handling', () => {
    it('should clear error state', () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      act(() => {
        useSupabaseAuthStore.setState({ error: 'Test error' })
      })

      expect(result.current.error).toBe('Test error')

      act(() => {
        result.current.clearError()
      })

      expect(result.current.error).toBe(null)
    })
  })

  describe('State Persistence', () => {
    it('should persist essential auth data', () => {
      const { result } = renderHook(() => useSupabaseAuthStore())

      const testUser = { id: 'test-user' } as any
      const testSession = { access_token: 'test-token' } as any
      const testSupabaseUser = { id: 'test-user', subscription_tier: 'free' } as any

      act(() => {
        useSupabaseAuthStore.setState({
          user: testUser,
          session: testSession,
          supabaseUser: testSupabaseUser,
          isAuthenticated: true,
          // These should not be persisted
          isLoading: true,
          error: 'test error',
          otpSent: true
        })
      })

      // Get the persisted state
      const store = useSupabaseAuthStore.persist
      const persistedState = store.getOptions().partialize?.(result.current)

      expect(persistedState).toEqual({
        user: testUser,
        session: testSession,
        supabaseUser: testSupabaseUser,
        isAuthenticated: true
      })
    })
  })
})