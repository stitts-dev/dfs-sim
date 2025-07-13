import { createClient, SupabaseClient } from '@supabase/supabase-js'
import { PhoneAuthRequest, VerificationRequest, SupabaseAuthResponse } from '@/types/auth'

// Supabase configuration
const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY

if (!supabaseUrl || !supabaseAnonKey) {
  console.warn('Supabase configuration missing. Phone auth will use backend API fallback.')
}

// Create Supabase client (optional if environment variables are not set)
let supabase: SupabaseClient | null = null
if (supabaseUrl && supabaseAnonKey) {
  supabase = createClient(supabaseUrl, supabaseAnonKey, {
    auth: {
      autoRefreshToken: true,
      persistSession: true,
      detectSessionInUrl: false
    }
  })
}

/**
 * Send OTP to phone number using Supabase Auth
 */
export const sendOTPWithSupabase = async (phoneNumber: string): Promise<SupabaseAuthResponse> => {
  if (!supabase) {
    throw new Error('Supabase not configured')
  }

  try {
    const { data, error } = await supabase.auth.signInWithOtp({
      phone: phoneNumber
    })

    if (error) {
      return {
        user: null,
        session: null,
        error: {
          code: error.name || 'auth_error',
          message: error.message
        }
      }
    }

    return {
      user: data.user,
      session: data.session,
      error: undefined
    }
  } catch (error) {
    return {
      user: null,
      session: null,
      error: {
        code: 'network_error',
        message: error instanceof Error ? error.message : 'Network error occurred'
      }
    }
  }
}

/**
 * Verify OTP code using Supabase Auth
 */
export const verifyOTPWithSupabase = async (
  phoneNumber: string, 
  token: string
): Promise<SupabaseAuthResponse> => {
  if (!supabase) {
    throw new Error('Supabase not configured')
  }

  try {
    const { data, error } = await supabase.auth.verifyOtp({
      phone: phoneNumber,
      token,
      type: 'sms'
    })

    if (error) {
      return {
        user: null,
        session: null,
        error: {
          code: error.name || 'verification_error',
          message: error.message
        }
      }
    }

    return {
      user: data.user,
      session: data.session,
      error: undefined
    }
  } catch (error) {
    return {
      user: null,
      session: null,
      error: {
        code: 'network_error',
        message: error instanceof Error ? error.message : 'Network error occurred'
      }
    }
  }
}

/**
 * Sign out user
 */
export const signOutWithSupabase = async (): Promise<{ error?: any }> => {
  if (!supabase) {
    return { error: null }
  }

  try {
    const { error } = await supabase.auth.signOut()
    return { error }
  } catch (error) {
    return { error }
  }
}

/**
 * Get current session
 */
export const getCurrentSession = async () => {
  if (!supabase) {
    return { data: { session: null }, error: null }
  }

  try {
    return await supabase.auth.getSession()
  } catch (error) {
    return { 
      data: { session: null }, 
      error: error instanceof Error ? error : new Error('Session error')
    }
  }
}

/**
 * Listen to auth state changes
 */
export const onAuthStateChange = (callback: (event: string, session: any) => void) => {
  if (!supabase) {
    return { data: { subscription: null } }
  }

  return supabase.auth.onAuthStateChange(callback)
}

/**
 * Check if Supabase is available
 */
export const isSupabaseAvailable = (): boolean => {
  return supabase !== null
}

/**
 * Get Supabase client instance
 */
export const getSupabaseClient = (): SupabaseClient | null => {
  return supabase
}

// Utility functions for phone number validation
export const normalizePhoneNumber = (phone: string): string => {
  // Remove all non-digit characters except +
  let cleaned = phone.replace(/[^\d+]/g, '')
  
  // Add + if not present
  if (!cleaned.startsWith('+')) {
    // Assume US number if no country code and exactly 10 digits
    if (/^\d{10}$/.test(cleaned)) {
      cleaned = '+1' + cleaned
    } else if (/^\d{11}$/.test(cleaned) && cleaned.startsWith('1')) {
      cleaned = '+' + cleaned
    } else {
      // Default to +1 for incomplete numbers
      cleaned = '+1' + cleaned
    }
  }
  
  return cleaned
}

export const validatePhoneNumber = (phone: string): boolean => {
  const normalized = normalizePhoneNumber(phone)
  // E.164 format validation
  return /^\+[1-9]\d{1,14}$/.test(normalized)
}

export const formatPhoneNumber = (phone: string): string => {
  const cleaned = phone.replace(/\D/g, '')
  
  if (cleaned.length === 0) return ''
  if (cleaned.length <= 3) return cleaned
  if (cleaned.length <= 6) return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3)}`
  
  return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6, 10)}`
}