import { AuthProvider, AuthMethod, AuthProviderConfig, AuthCredentials, AuthProviderResult } from '@/types/auth'
import { supabase, useUnifiedAuthStore } from '@/store/unifiedAuth'

export class EmailAuthProvider extends AuthProvider {
  method: AuthMethod = 'email'
  config: AuthProviderConfig = {
    enabled: true,
    priority: 1,
    requiresVerification: true
  }

  validateInput(credentials: AuthCredentials): boolean {
    const { email, password } = credentials

    if (!email || !password) {
      return false
    }

    // Basic email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(email)) {
      return false
    }

    // Basic password validation (minimum 8 characters)
    if (password.length < 8) {
      return false
    }

    return true
  }

  formatInput(input: string): string {
    return input.trim().toLowerCase()
  }

  async sendVerification(credentials: AuthCredentials): Promise<AuthProviderResult> {
    const { email } = credentials

    if (!email) {
      return {
        success: false,
        error: 'Email is required'
      }
    }

    try {
      // Use signInWithOtp for magic link + OTP support
      const { error } = await supabase.auth.signInWithOtp({
        email: this.formatInput(email),
        options: {
          shouldCreateUser: true,
          // Send both magic link and OTP for flexibility
          emailRedirectTo: `${window.location.origin}/auth/callback`
        }
      })

      if (error) {
        return {
          success: false,
          error: error.message
        }
      }

      return {
        success: true,
        verificationSent: true,
        requiresVerification: true
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to send verification email'
      }
    }
  }

  async verifyCredentials(credentials: AuthCredentials): Promise<AuthProviderResult> {
    const { email, password, verificationCode, mode } = credentials

    if (!email) {
      return {
        success: false,
        error: 'Email is required'
      }
    }

    try {
      // If verification code is provided, verify OTP (for email signup/magic link users)
      if (verificationCode) {
        const otpType = mode === 'signup' ? 'signup' : 'magiclink'
        const { data, error } = await supabase.auth.verifyOtp({
          email: this.formatInput(email),
          token: verificationCode,
          type: otpType
        })

        if (error) {
          return {
            success: false,
            error: error.message
          }
        }

        if (data.user && data.session) {
          // Update the auth store with the new session
          this.updateAuthStore(data.user, data.session)

          return {
            success: true,
            user: this.mapSupabaseUser(data.user),
            token: data.session.access_token
          }
        }
      }

      // If password is provided, try password authentication
      if (password) {
        const { data, error } = await supabase.auth.signInWithPassword({
          email: this.formatInput(email),
          password
        })

        if (error) {
          if (error.message.includes('Email not confirmed')) {
            const otpResult = await this.sendVerification(credentials)
            if (otpResult.success) {
              return {
                success: false,
                error: 'Email not confirmed. Please check your email for a verification link.',
                requiresVerification: true,
                verificationSent: true
              }
            }
          }

          return {
            success: false,
            error: error.message
          }
        }

        if (data.user && data.session) {
          // Update the auth store with the new session
          this.updateAuthStore(data.user, data.session)

          return {
            success: true,
            user: this.mapSupabaseUser(data.user),
            token: data.session.access_token
          }
        }
      }

      return {
        success: false,
        error: 'Invalid credentials'
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Authentication failed'
      }
    }
  }

  async resendVerification(credentials: AuthCredentials): Promise<AuthProviderResult> {
    return this.sendVerification(credentials)
  }

  async resetPassword(credentials: AuthCredentials): Promise<AuthProviderResult> {
    const { email } = credentials

    if (!email) {
      return {
        success: false,
        error: 'Email is required'
      }
    }

    try {
      const { error } = await supabase.auth.resetPasswordForEmail(
        this.formatInput(email),
        {
          redirectTo: `${window.location.origin}/auth/callback`
        }
      )

      if (error) {
        return {
          success: false,
          error: error.message
        }
      }

      return {
        success: true,
        verificationSent: true
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to send reset email'
      }
    }
  }

  async checkUserExists(email: string): Promise<{ exists: boolean; hasPassword: boolean; error?: string }> {
    try {
      // Attempt a password reset to check if user exists
      // This is a safe way to check user existence without exposing user data
      const { error } = await supabase.auth.resetPasswordForEmail(
        this.formatInput(email),
        {
          redirectTo: `${window.location.origin}/auth/callback`
        }
      )

      if (error) {
        // If error is "User not found" or similar, user doesn't exist
        if (error.message.includes('User not found') || error.message.includes('not found')) {
          return { exists: false, hasPassword: false }
        }

        // Other errors might indicate rate limiting or other issues
        return { exists: true, hasPassword: true, error: error.message }
      }

      // If no error, user exists and likely has password setup
      return { exists: true, hasPassword: true }
    } catch (error) {
      return {
        exists: false,
        hasPassword: false,
        error: error instanceof Error ? error.message : 'Failed to check user'
      }
    }
  }

  async smartSignIn(email: string): Promise<AuthProviderResult> {
    const userCheck = await this.checkUserExists(email)

    if (!userCheck.exists) {
      // User doesn't exist, send magic link for signup
      return this.sendVerification({ method: 'email', email })
    }

    // User exists, send OTP for magic link authentication
    return this.sendVerification({ method: 'email', email })
  }

  private mapSupabaseUser(supabaseUser: any): any {
    return {
      id: supabaseUser.id,
      email: supabaseUser.email,
      email_verified: supabaseUser.email_confirmed_at ? true : false,
      first_name: supabaseUser.user_metadata?.first_name,
      last_name: supabaseUser.user_metadata?.last_name,
      subscription_tier: 'free',
      subscription_status: 'active',
      is_active: true,
      created_at: supabaseUser.created_at,
      updated_at: supabaseUser.updated_at
    }
  }

  private updateAuthStore(user: any, session: any) {
    try {
      // Update the unified auth store with the Supabase session
      const authStore = useUnifiedAuthStore.getState()
      authStore.setSupabaseAuth(user, session)
    } catch (error) {
      console.error('Failed to update auth store:', error)
    }
  }
}
