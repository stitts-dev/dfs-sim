import { createClient, SupabaseClient } from '@supabase/supabase-js'
import { SupabaseAuthResponse } from '@/types/auth'
import { parsePhoneNumber } from 'libphonenumber-js/min'

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
      user: data.user ? {
        id: (data.user as any).id,
        phone: (data.user as any).phone,
        email: (data.user as any).email,
        created_at: (data.user as any).created_at,
        updated_at: (data.user as any).updated_at || (data.user as any).created_at,
        phone_confirmed_at: (data.user as any).phone_confirmed_at,
        email_confirmed_at: (data.user as any).email_confirmed_at
      } : null,
      session: data.session ? {
        access_token: (data.session as any).access_token,
        refresh_token: (data.session as any).refresh_token,
        expires_at: (data.session as any).expires_at || 0,
        expires_in: (data.session as any).expires_in,
        token_type: (data.session as any).token_type,
        user: (data.session as any).user ? {
          id: (data.session as any).user.id,
          phone: (data.session as any).user.phone,
          email: (data.session as any).user.email,
          created_at: (data.session as any).user.created_at,
          updated_at: (data.session as any).user.updated_at || (data.session as any).user.created_at,
          phone_confirmed_at: (data.session as any).user.phone_confirmed_at,
          email_confirmed_at: (data.session as any).user.email_confirmed_at
        } : {
          id: '',
          created_at: '',
          updated_at: ''
        }
      } : null,
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
      user: data.user ? {
        id: (data.user as any).id,
        phone: (data.user as any).phone,
        email: (data.user as any).email,
        created_at: (data.user as any).created_at,
        updated_at: (data.user as any).updated_at || (data.user as any).created_at,
        phone_confirmed_at: (data.user as any).phone_confirmed_at,
        email_confirmed_at: (data.user as any).email_confirmed_at
      } : null,
      session: data.session ? {
        access_token: (data.session as any).access_token,
        refresh_token: (data.session as any).refresh_token,
        expires_at: (data.session as any).expires_at || 0,
        expires_in: (data.session as any).expires_in,
        token_type: (data.session as any).token_type,
        user: (data.session as any).user ? {
          id: (data.session as any).user.id,
          phone: (data.session as any).user.phone,
          email: (data.session as any).user.email,
          created_at: (data.session as any).user.created_at,
          updated_at: (data.session as any).user.updated_at || (data.session as any).user.created_at,
          phone_confirmed_at: (data.session as any).user.phone_confirmed_at,
          email_confirmed_at: (data.session as any).user.email_confirmed_at
        } : {
          id: '',
          created_at: '',
          updated_at: ''
        }
      } : null,
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

// Enhanced utility functions for phone number validation using libphonenumber-js
export const normalizePhoneNumber = (phone: string): string => {
  if (!phone) return ''
  
  try {
    // Try to parse with libphonenumber-js
    const phoneNumber = parsePhoneNumber(phone)
    if (phoneNumber && phoneNumber.isValid()) {
      return phoneNumber.format('E.164')
    }
  } catch (error) {
    // Fall back to manual parsing
  }
  
  // Fallback to original logic for incomplete numbers
  let cleaned = phone.replace(/[^\d+]/g, '')
  
  // Add + if not present
  if (!cleaned.startsWith('+')) {
    // Assume US number if no country code and exactly 10 digits
    if (/^\d{10}$/.test(cleaned)) {
      cleaned = '+1' + cleaned
    } else if (/^\d{11}$/.test(cleaned) && cleaned.startsWith('1')) {
      cleaned = '+' + cleaned
    } else {
      // Return as-is for incomplete numbers - don't force +1
      // cleaned = cleaned (this is a no-op)
    }
  }
  
  return cleaned
}

export const validatePhoneNumber = (phone: string): boolean => {
  if (!phone) return false
  
  try {
    // Use libphonenumber-js for accurate validation
    const phoneNumber = parsePhoneNumber(phone)
    return phoneNumber ? phoneNumber.isValid() : false
  } catch (error) {
    // Fallback to regex validation
    const normalized = normalizePhoneNumber(phone)
    return /^\+[1-9]\d{1,14}$/.test(normalized)
  }
}

export const formatPhoneNumber = (phone: string): string => {
  if (!phone) return ''
  
  try {
    // Use libphonenumber-js for international formatting
    const phoneNumber = parsePhoneNumber(phone)
    if (phoneNumber && phoneNumber.isValid()) {
      return phoneNumber.formatNational()
    }
  } catch (error) {
    // Fall back to manual formatting
  }
  
  // Fallback to original US formatting logic
  const cleaned = phone.replace(/\D/g, '')
  
  if (cleaned.length === 0) return ''
  
  // Handle 11-digit US numbers (starting with 1)
  let numberToFormat = cleaned
  if (cleaned.length === 11 && cleaned.startsWith('1')) {
    // Strip the country code "1" and format the 10-digit number
    numberToFormat = cleaned.slice(1)
  }
  
  if (numberToFormat.length <= 3) return numberToFormat
  if (numberToFormat.length <= 6) return `(${numberToFormat.slice(0, 3)}) ${numberToFormat.slice(3)}`
  
  return `(${numberToFormat.slice(0, 3)}) ${numberToFormat.slice(3, 6)}-${numberToFormat.slice(6, 10)}`
}