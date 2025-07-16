import { AuthWizard } from './AuthWizard'
import { getEnabledAuthMethods } from '@/config/auth'
import type { AuthMethod } from '@/types/auth'

/**
 * Example component showing how to use the extended AuthWizard
 * with email authentication (phone disabled as requested)
 */
export function AuthWizardExample() {
  const enabledMethods = getEnabledAuthMethods() // Returns ['email'] by default

  const handleComplete = (user: any) => {
    console.log('Authentication completed:', user)
    // Handle successful authentication
  }

  const handleClose = () => {
    console.log('Auth wizard closed')
    // Handle wizard close
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12">
      <div className="max-w-md mx-auto">
        <AuthWizard
          initialMode="login"
          enabledMethods={enabledMethods}
          onComplete={handleComplete}
          onClose={handleClose}
        />
      </div>
    </div>
  )
}

/**
 * Example for signup mode
 */
export function SignupExample() {
  const enabledMethods = getEnabledAuthMethods()

  const handleComplete = (user: any) => {
    console.log('Signup completed:', user)
    // Handle successful signup
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12">
      <div className="max-w-md mx-auto">
        <AuthWizard
          initialMode="signup"
          enabledMethods={enabledMethods}
          onComplete={handleComplete}
        />
      </div>
    </div>
  )
}

/**
 * Example showing how to enable multiple auth methods
 * (for development or future use)
 */
export function MultiAuthExample() {
  // This would be enabled via environment variables in development
  const enabledMethods: AuthMethod[] = ['email', 'phone']

  const handleComplete = (user: any) => {
    console.log('Multi-auth completed:', user)
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-12">
      <div className="max-w-md mx-auto">
        <AuthWizard
          initialMode="login"
          enabledMethods={enabledMethods}
          onComplete={handleComplete}
        />
      </div>
    </div>
  )
}

export default AuthWizardExample