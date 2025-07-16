import { useEffect, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom' // add useLocation
import { supabase } from '@/store/unifiedAuth'
import { useUnifiedAuthStore } from '@/store/unifiedAuth'
import { AuthLayout, AuthCard } from '@/components/auth/AuthLayout'
import { SparkleIcon } from '@/components/ui/SparkleIcon'
import { PasswordInput, PasswordStrength } from '@/components/auth/PasswordInput' // add
import { Button } from '@/components/ui/Button' // add

export default function AuthCallback() {
  const navigate = useNavigate()
  const location = useLocation()
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [authType, setAuthType] = useState<'recovery' | 'signup' | null>(null)
  const [newPassword, setNewPassword] = useState('')
  const [isPasswordValid, setIsPasswordValid] = useState(false)
  const { setSupabaseAuth } = useUnifiedAuthStore()

  useEffect(() => {
    const handleAuthCallback = async () => {
      try {
        // Parse URL hash for auth params
        const hashParams = new URLSearchParams(location.hash.substring(1))
        const type = hashParams.get('type')
        setAuthType(type as 'recovery' | 'signup' | null)

        const { data, error } = await supabase.auth.getSession()

        if (error) {
          throw error
        }

        if (data.session && data.session.user) {
          // Update the unified auth store with Supabase session
          setSupabaseAuth(data.session.user, data.session)

          setSuccess(true)
          setIsLoading(false)

          // Redirect to dashboard after a short delay
          setTimeout(() => {
            navigate('/dashboard')
          }, 2000)
        } else if (type === 'recovery') {
          // For recovery, session should be temporary, proceed to password form
          setIsLoading(false)
        } else {
          // If no session, try to handle the auth callback
          const { data: userData, error: authError } = await supabase.auth.getUser()

          if (authError) {
            throw authError
          }

          if (userData.user) {
            // Get fresh session for this user
            const { data: sessionData } = await supabase.auth.getSession()

            if (sessionData.session) {
              setSupabaseAuth(userData.user, sessionData.session)
            } else {
              throw new Error('No session found for authenticated user')
            }

            setSuccess(true)
            setIsLoading(false)

            setTimeout(() => {
              navigate('/dashboard')
            }, 2000)
          } else {
            throw new Error('No user found after authentication')
          }
        }
      } catch (err) {
        console.error('Auth callback error:', err)
        setError(err instanceof Error ? err.message : 'Authentication failed')
        setIsLoading(false)

        // Redirect to login after error
        setTimeout(() => {
          navigate('/login')
        }, 3000)
      }
    }

    handleAuthCallback()
  }, [navigate, location])

  const handleResetPassword = async () => {
    if (!isPasswordValid) return

    try {
      setIsLoading(true)
      const { error } = await supabase.auth.updateUser({
        password: newPassword
      })

      if (error) {
        setError(error.message)
        return
      }

      setSuccess(true)
      setTimeout(() => {
        navigate('/dashboard')
      }, 2000)
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
                Verifying Your Account
              </h3>
              <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                Please wait while we complete your authentication...
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
                Authentication Failed
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
          </div>
        </AuthCard>
      </AuthLayout>
    )
  }

  if (authType === 'recovery') {
    return (
      <AuthLayout showBranding={true}>
        <AuthCard
          title="Reset Your Password"
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
            />

            <Button
              onClick={handleResetPassword}
              disabled={!isPasswordValid || isLoading}
              loading={isLoading}
              variant="primary"
              size="lg"
              className="w-full"
            >
              {isLoading ? 'Updating...' : 'Set Password'}
            </Button>
          </div>
        </AuthCard>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout showBranding={true}>
      <AuthCard>
        <div className="text-center space-y-6">
          <div className="mx-auto w-20 h-20 bg-gradient-to-br from-green-400 to-green-600 rounded-full flex items-center justify-center">
            <SparkleIcon className="w-10 h-10 text-white" animated />
          </div>

          <div>
            <h3 className="text-2xl font-semibold text-gray-900 dark:text-white">
              Account Verified!
            </h3>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              Your email has been successfully verified
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
