import { AuthProvider, AuthMethod, AuthProviderConfig, AuthCredentials, AuthProviderResult } from '@/types/auth'

export class PhoneAuthProvider extends AuthProvider {
  method: AuthMethod = 'phone'
  config: AuthProviderConfig = {
    enabled: false, // Disabled by default as requested
    priority: 2,
    requiresVerification: true
  }

  validateInput(credentials: AuthCredentials): boolean {
    const { phoneNumber } = credentials
    
    if (!phoneNumber) {
      return false
    }

    // Remove all non-digits
    const cleaned = phoneNumber.replace(/\D/g, '')
    
    // Check if it's a valid US phone number (10 or 11 digits)
    return cleaned.length === 10 || cleaned.length === 11
  }

  formatInput(input: string): string {
    const cleaned = input.replace(/\D/g, '')
    
    // Add country code if missing
    if (cleaned.length === 10) {
      return '+1' + cleaned
    } else if (cleaned.length === 11 && cleaned.startsWith('1')) {
      return '+' + cleaned
    }
    
    return input
  }

  async sendVerification(credentials: AuthCredentials): Promise<AuthProviderResult> {
    const { phoneNumber } = credentials
    
    if (!phoneNumber) {
      return {
        success: false,
        error: 'Phone number is required'
      }
    }

    const formattedPhone = this.formatInput(phoneNumber)

    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
      
      // First try login for existing verified users
      const loginResponse = await fetch(`${apiUrl}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone_number: formattedPhone })
      })

      if (loginResponse.ok) {
        return {
          success: true,
          verificationSent: true,
          requiresVerification: true
        }
      }
      
      // If login fails, try register
      if (loginResponse.status === 404 || loginResponse.status === 400) {
        const registerResponse = await fetch(`${apiUrl}/auth/register`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ phone_number: formattedPhone })
        })
        
        if (!registerResponse.ok) {
          const error = await registerResponse.json()
          return {
            success: false,
            error: error.error || error.message || 'Failed to send verification code'
          }
        }
        
        return {
          success: true,
          verificationSent: true,
          requiresVerification: true
        }
      }
      
      const error = await loginResponse.json()
      return {
        success: false,
        error: error.error || error.message || 'Authentication failed'
      }
    } catch (error) {
      if (error instanceof TypeError && error.message.includes('fetch')) {
        return {
          success: false,
          error: 'Network error. Please check your connection and try again.'
        }
      }
      
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to send verification code'
      }
    }
  }

  async verifyCredentials(credentials: AuthCredentials): Promise<AuthProviderResult> {
    const { phoneNumber, verificationCode } = credentials

    if (!phoneNumber || !verificationCode) {
      return {
        success: false,
        error: 'Phone number and verification code are required'
      }
    }

    const formattedPhone = this.formatInput(phoneNumber)

    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
      
      const response = await fetch(`${apiUrl}/auth/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          phone_number: formattedPhone, 
          code: verificationCode 
        })
      })

      if (!response.ok) {
        const error = await response.json()
        return {
          success: false,
          error: error.error || error.message || 'Verification failed'
        }
      }

      const data = await response.json()
      
      return {
        success: true,
        user: data.user,
        token: data.token
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Verification failed'
      }
    }
  }

  async resendVerification(credentials: AuthCredentials): Promise<AuthProviderResult> {
    const { phoneNumber } = credentials

    if (!phoneNumber) {
      return {
        success: false,
        error: 'Phone number is required'
      }
    }

    const formattedPhone = this.formatInput(phoneNumber)

    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
      
      const response = await fetch(`${apiUrl}/auth/resend`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone_number: formattedPhone })
      })

      if (!response.ok) {
        const error = await response.json()
        return {
          success: false,
          error: error.error || error.message || 'Failed to resend OTP'
        }
      }

      return {
        success: true,
        verificationSent: true
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to resend OTP'
      }
    }
  }

  async resetPassword(_credentials: AuthCredentials): Promise<AuthProviderResult> {
    return {
      success: false,
      error: 'Password reset not supported for phone authentication'
    }
  }
}