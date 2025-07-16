import { useEffect, useState } from 'react'
import { useNavigate, useLocation, useSearchParams } from 'react-router-dom'
import { supabase } from '@/store/unifiedAuth'
import { useUnifiedAuthStore } from '@/store/unifiedAuth'
import { AuthLayout, AuthCard } from '@/components/auth/AuthLayout'
import { PasswordInput } from '@/components/auth/PasswordInput'
import { Button } from '@/components/ui/Button'
import { SparkleIcon } from '@/components/ui/SparkleIcon'

export default function PasswordResetPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [newPassword, setNewPassword] = useState('')
  const [isPasswordValid, setIsPasswordValid] = useState(false)
  const [isValidResetSession, setIsValidResetSession] = useState(false)
  const { setSupabaseAuth } = useUnifiedAuthStore()

  useEffect(() => {
    const handlePasswordResetSession = async () => {
      try {
        // Check if we have hash parameters (from email link)
        const hashParams = new URLSearchParams(location.hash.substring(1))
        const type = hashParams.get('type')

        // Also check URL search parameters as backup
        const accessToken = hashParams.get('access_token') || searchParams.get('access_token')
        const refreshToken = hashParams.get('refresh_token') || searchParams.get('refresh_token')

        if (type === 'recovery' && accessToken && refreshToken) {
          // Set the session from the tokens
          const { data, error } = await supabase.auth.setSession({
            access_token: accessToken,
            refresh_token: refreshToken
          })

          if (error) {
            throw error
          }

          if (data.session) {
            setIsValidResetSession(true)
            setIsLoading(false)
            return
          }
        }

        // If no valid tokens, check if we already have a session
        const { data: sessionData, error: sessionError } = await supabase.auth.getSession()

        if (sessionError) {
          throw sessionError
        }

        if (sessionData.session && sessionData.session.user) {
          // We have a valid session, allow password reset
          setIsValidResetSession(true)
          setIsLoading(false)
        } else {
          // No valid session, redirect to login
          setError('Invalid or expired password reset link')
          setIsLoading(false)
          setTimeout(() => {
            navigate('/auth/login')
          }, 3000)
        }
      } catch (err) {
        console.error('Password reset session error:', err)
        setError(err instanceof Error ? err.message : 'Invalid password reset link')
        setIsLoading(false)

        setTimeout(() => {
          navigate('/auth/login')
        }, 3000)
      }
    }

    handlePasswordResetSession()
  }, [navigate, location, searchParams])

  const handleResetPassword = async () => {
    if (!isPasswordValid) return

    setIsLoading(true)
    setError('')

    try {
      const { data, error } = await supabase.auth.updateUser({
        password: newPassword
      })

      if (error) {
        throw error
      }

      if (data.user) {
        // Get fresh session after password update
        const { data: sessionData } = await supabase.auth.getSession()

        if (sessionData.session) {
          setSupabaseAuth(data.user, sessionData.session)
        }

        setSuccess(true)

        // Redirect to dashboard after success
        setTimeout(() => {
          navigate('/dashboard')
        }, 2000)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update password')
    } finally {
      setIsLoading(false)
    }
  }

  if (isLoading) {
    return (
      <AuthLayout showBranding={true}>
        <AuthCard>
          <div className="text-center space-y-6">
            <div className="mx-auto w-16 h-16 bg-gradient-to-br from-sky-400 to-sky-600 rounded-full flex items-center justify-center">
              <div className="animate-spin rounded-full h-8 w-8 border-2 border-white border-t-transparent"></div>
            </div>

            <div>
              <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
                Verifying Reset Link
              </h3>
              <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                Please wait while we verify your password reset request...
              </p>
            </div>
          </div>
        </AuthCard>
      </AuthLayout>
    )
  }

  if (error) {
    return (
      <AuthLayout showBranding={true}>
        <AuthCard>
          <div className="text-center space-y-6">
            <div className="mx-auto w-16 h-16 bg-gradient-to-br from-red-400 to-red-600 rounded-full flex items-center justify-center">
              <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </div>

            <div>
              <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
                Password Reset Failed
              </h3>
              <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                {error}
              </p>
            </div>

            <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4">
              <p className="text-sm text-red-700 dark:text-red-300">
                Redirecting to login page...
              </p>
            </div>

            <Button
              onClick={() => navigate('/auth/login')}
              variant="outline"
              size="lg"
              className="w-full"
            >
              Return to Login
            </Button>
          </div>
        </AuthCard>
      </AuthLayout>
    )
  }

  if (success) {
    return (
      <AuthLayout showBranding={true}>
        <AuthCard>
          <div className="text-center space-y-6">
            <div className="mx-auto w-20 h-20 bg-gradient-to-br from-green-400 to-green-600 rounded-full flex items-center justify-center">
              <SparkleIcon className="w-10 h-10 text-white" animated />
            </div>

            <div>
              <h3 className="text-2xl font-semibold text-gray-900 dark:text-white">
                Password Updated!
              </h3>
              <p className="mt-2 text-gray-600 dark:text-gray-400">
                Your password has been successfully updated
              </p>
            </div>

            <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-4">
              <p className="text-sm text-green-700 dark:text-green-300">
                Redirecting to your dashboard...
              </p>
            </div>
          </div>
        </AuthCard>
      </AuthLayout>
    )
  }

  if (isValidResetSession) {
    return (
      <AuthLayout showBranding={true}>
        <AuthCard
          title="Set New Password"
          subtitle="Enter a strong password to secure your account"
        >
          <div className="space-y-6">
            <PasswordInput
              value={newPassword}
              onChange={setNewPassword}
              onValidate={(valid) => setIsPasswordValid(valid)}
              autoFocus={true}
              error={error}
              label="New Password"
              showStrengthIndicator={true}
              requireStrength={true}
              description="Choose a strong password that you haven't used before"
            />

            <div className="space-y-4">
              <Button
                onClick={handleResetPassword}
                disabled={!isPasswordValid || isLoading}
                loading={isLoading}
                variant="primary"
                size="lg"
                className="w-full"
              >
                {isLoading ? 'Updating Password...' : 'Update Password'}
              </Button>

              <div className="text-center">
                <button
                  onClick={() => navigate('/auth/login')}
                  className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
                >
                  Cancel and return to login
                </button>
              </div>
            </div>
          </div>
        </AuthCard>
      </AuthLayout>
    )
  }

  return null
}
