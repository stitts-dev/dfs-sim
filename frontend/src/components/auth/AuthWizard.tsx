import { useState, useEffect, useCallback } from 'react'
import { AuthLayout, AuthCard, AuthStepIndicator } from './AuthLayout'
import { ModernPhoneInput } from './ModernPhoneInput'
import { EnhancedOTPVerification } from './EnhancedOTPVerification'
import { EmailInput } from './EmailInput'
import { PasswordInput, PasswordStrength } from './PasswordInput'
import { AuthMethodSelector } from './AuthMethodSelector'
import { Button } from '@/components/ui/Button'
import { SparkleIcon } from '@/components/ui/SparkleIcon'
import { usePhoneAuth } from '@/hooks/usePhoneAuth'
import { useAuthStore } from '@/store/auth'
import { AuthMethod, AuthStepType, AuthWizardMode } from '@/types/auth'
import { EmailAuthProvider } from '@/providers/EmailAuthProvider'
import { getEnabledAuthMethods } from '@/config/auth'
import { supabase } from '@/store/unifiedAuth' // fixed path

export type AuthWizardStep = AuthStepType | 'password-reset'

export interface AuthWizardProps {
  onComplete?: (user: any) => void
  onClose?: () => void
  initialMode?: AuthWizardMode
  initialStep?: AuthWizardStep
  enabledMethods?: AuthMethod[]
  className?: string
}

interface WizardStepProps {
  step: AuthWizardStep
  mode: AuthWizardMode
  selectedMethod: AuthMethod
  enabledMethods: AuthMethod[]
  phoneNumber: string
  email: string
  password: string
  verificationCode: string
  isPhoneValid: boolean
  isEmailValid: boolean
  isPasswordValid: boolean
  setSelectedMethod: (method: AuthMethod) => void
  setPhoneNumber: (phone: string) => void
  setEmail: (email: string) => void
  setPassword: (password: string) => void
  setVerificationCode: (code: string) => void
  setIsPhoneValid: (valid: boolean) => void
  setIsEmailValid: (valid: boolean) => void
  setIsPasswordValid: (valid: boolean, strength: PasswordStrength) => void
  onNext: () => void
  onBack: () => void
  onSwitchMode: () => void
  onSetStep: (step: AuthWizardStep) => void // added
  auth: ReturnType<typeof usePhoneAuth>
  user: any
}

function MethodSelectionStep({
  mode,
  selectedMethod,
  enabledMethods,
  setSelectedMethod,
  onNext,
  onBack,
  onSwitchMode
}: Pick<WizardStepProps, 'mode' | 'selectedMethod' | 'enabledMethods' | 'setSelectedMethod' | 'onNext' | 'onBack' | 'onSwitchMode'>) {
  return (
    <AuthCard>
      <AuthMethodSelector
        selectedMethod={selectedMethod}
        onMethodSelect={setSelectedMethod}
        enabledMethods={enabledMethods}
        mode={mode}
      />

      <div className="mt-6 space-y-3">
        <Button
          onClick={onNext}
          disabled={!selectedMethod}
          variant="primary"
          size="lg"
          className="w-full"
          arrow
        >
          Continue with {selectedMethod === 'email' ? 'Email' : 'Phone'}
        </Button>

        <div className="flex justify-between items-center">
          <button
            onClick={onBack}
            className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
          >
            ← Back
          </button>

          <button
            onClick={onSwitchMode}
            className="text-sm text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300 font-medium transition-colors"
          >
            {mode === 'signup' ? 'Sign in instead' : 'Create account'}
          </button>
        </div>
      </div>
    </AuthCard>
  )
}

function WelcomeStep({ mode, onNext, onSwitchMode }: Pick<WizardStepProps, 'mode' | 'onNext' | 'onSwitchMode'>) {
  return (
    <AuthCard
      title={mode === 'signup' ? 'Welcome to DFS Pro' : 'Welcome Back'}
      subtitle={mode === 'signup'
        ? 'Join thousands of successful DFS players using professional optimization tools'
        : 'Sign in to access your advanced DFS optimization tools'
      }
    >
      <div className="space-y-6">
        {/* Features highlight for signup */}
        {mode === 'signup' && (
          <div className="space-y-4">
            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-6 h-6 bg-sky-100 dark:bg-sky-900 rounded-full flex items-center justify-center mt-0.5">
                <SparkleIcon className="w-3 h-3 text-sky-600 dark:text-sky-400" />
              </div>
              <div>
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">Advanced Optimization</h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">Monte Carlo simulations with correlation matrices</p>
              </div>
            </div>

            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-6 h-6 bg-sky-100 dark:bg-sky-900 rounded-full flex items-center justify-center mt-0.5">
                <SparkleIcon className="w-3 h-3 text-sky-600 dark:text-sky-400" />
              </div>
              <div>
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">Smart Stacking</h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">Game stacks, team stacks, and correlation-based lineup building</p>
              </div>
            </div>

            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-6 h-6 bg-sky-100 dark:bg-sky-900 rounded-full flex items-center justify-center mt-0.5">
                <SparkleIcon className="w-3 h-3 text-sky-600 dark:text-sky-400" />
              </div>
              <div>
                <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">Real-time Data</h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">Live updates from multiple professional data providers</p>
              </div>
            </div>
          </div>
        )}

        <Button
          onClick={onNext}
          variant="primary"
          size="lg"
          className="w-full"
          arrow
        >
          {mode === 'signup' ? 'Get Started' : 'Sign In'}
        </Button>

        <div className="text-center">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {mode === 'signup' ? 'Already have an account?' : "Don't have an account?"}{' '}
            <button
              onClick={onSwitchMode}
              className="text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300 font-medium transition-colors"
            >
              {mode === 'signup' ? 'Sign in' : 'Sign up'}
            </button>
          </p>
        </div>
      </div>
    </AuthCard>
  )
}

function EmailStep({
  mode,
  email,
  isEmailValid,
  setEmail,
  setIsEmailValid,
  onNext,
  onBack,
  onSwitchMode
}: Pick<WizardStepProps, 'mode' | 'email' | 'isEmailValid' | 'setEmail' | 'setIsEmailValid' | 'onNext' | 'onBack' | 'onSwitchMode'>) {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const emailProvider = new EmailAuthProvider()

  // No-op: user existence is checked on handleNext only

  const handleNext = async () => {
    if (!isEmailValid) return
    setIsLoading(true)
    setError('')
    try {
      // Check user existence again for safety
      const result = await emailProvider.checkUserExists(email)
      if (mode === 'login') {
        if (result.exists) {
          // User exists, proceed to password step
          onNext()
        } else {
          setError('No account found with this email. Would you like to sign up?')
        }
      } else if (mode === 'signup') {
        if (result.exists) {
          setError('An account with this email already exists. Please sign in instead.')
        } else {
          onNext()
        }
      }
    } catch (err) {
      setError('Failed to check user. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <AuthCard
      title={mode === 'signup' ? 'Create Your Account' : 'Sign In'}
      subtitle={mode === 'signup'
        ? 'Enter your email address to create your DFS Pro account'
        : 'Enter your email address to access your account'
      }
    >
      <div className="space-y-6">
        <EmailInput
          value={email}
          onChange={setEmail}
          onValidate={setIsEmailValid}
          autoFocus={true}
          error={error}
          label={mode === 'signup' ? 'Your Email Address' : 'Email Address'}
          description={mode === 'signup'
            ? 'You\'ll set a password in the next step'
            : 'Enter your email address to sign in'}
        />
        <div className="space-y-3">
          <Button
            onClick={handleNext}
            disabled={!isEmailValid || isLoading}
            loading={isLoading}
            variant="primary"
            size="lg"
            className="w-full"
          >
            {isLoading
              ? 'Continuing...'
              : 'Continue'}
          </Button>
          <div className="flex justify-between items-center">
            <button
              onClick={onBack}
              className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
            >
              ← Back
            </button>
            <button
              onClick={onSwitchMode}
              className="text-sm text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300 font-medium transition-colors"
            >
              {mode === 'signup' ? 'Sign in instead' : 'Create account'}
            </button>
          </div>
        </div>
      </div>
    </AuthCard>
  )
}

function PasswordStep({
  mode,
  email,
  password,
  isPasswordValid,
  setPassword,
  setIsPasswordValid,
  onNext,
  onBack,
  onSwitchMode,
  onSetStep
}: Pick<WizardStepProps, 'mode' | 'email' | 'password' | 'isPasswordValid' | 'setPassword' | 'setIsPasswordValid' | 'onNext' | 'onBack' | 'onSwitchMode' | 'onSetStep'>) {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const emailProvider = new EmailAuthProvider()

  const handleNext = async () => {
    if (!isPasswordValid) return

    setIsLoading(true)
    setError('')

    try {
      if (mode === 'signup') {
        const { error } = await supabase.auth.signUp({
          email,
          password,
          options: {
            emailRedirectTo: `${window.location.origin}/auth/callback`
          }
        })

        if (error) {
          setError(error.message)
          if (error.message.includes('User already registered')) {
            onSwitchMode()
          }
          return
        }

        onNext()
      } else {
        const result = await emailProvider.verifyCredentials({
          method: 'email',
          email,
          password,
          mode
        })

        if (result.success) {
          onNext()
        } else if (result.requiresVerification && result.verificationSent) {
          setError(result.error || 'Verification email sent.')
          onSetStep('email-verification')
        } else {
          setError(result.error || 'Invalid email or password')
        }
      }
    } catch (err) {
      setError('Authentication failed. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }

  const handleForgotPassword = async () => {
    setIsLoading(true)
    setError('')

    try {
      const result = await emailProvider.resetPassword({ method: 'email', email })

      if (result.success) {
        onSetStep('password-reset')
      } else {
        setError(result.error || 'Failed to send password reset email')
      }
    } catch (err) {
      setError('Failed to send password reset email. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <AuthCard
      title={mode === 'signup' ? 'Set Your Password' : 'Enter Your Password'}
      subtitle={mode === 'signup'
        ? 'Create a strong password to secure your account'
        : 'Enter your password to access your account'
      }
    >
      <div className="space-y-6">
        <PasswordInput
          value={password}
          onChange={setPassword}
          onValidate={setIsPasswordValid}
          autoFocus={true}
          error={error}
          label="Password"
          showStrengthIndicator={mode === 'signup'}
          requireStrength={mode === 'signup'}
        />

        <div className="space-y-3">
          <Button
            onClick={handleNext}
            disabled={!isPasswordValid || isLoading}
            loading={isLoading}
            variant="primary"
            size="lg"
            className="w-full"
          >
            {isLoading
              ? 'Authenticating...'
              : mode === 'signup'
              ? 'Create Account'
              : 'Sign In'
            }
          </Button>

          <div className="flex justify-between items-center">
            <button
              onClick={onBack}
              className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
            >
              ← Back
            </button>

            {mode === 'login' && (
              <button
                onClick={handleForgotPassword}
                className="text-sm text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300 font-medium transition-colors"
              >
                Forgot password?
              </button>
            )}

            <button
              onClick={onSwitchMode}
              className="text-sm text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300 font-medium transition-colors"
            >
              {mode === 'signup' ? 'Sign in instead' : 'Create account'}
            </button>
          </div>
        </div>
      </div>
    </AuthCard>
  )
}

function PhoneStep({
  mode,
  phoneNumber,
  isPhoneValid,
  setPhoneNumber,
  setIsPhoneValid,
  onNext,
  onBack,
  onSwitchMode,
  auth
}: Pick<WizardStepProps, 'mode' | 'phoneNumber' | 'isPhoneValid' | 'setPhoneNumber' | 'setIsPhoneValid' | 'onNext' | 'onBack' | 'onSwitchMode' | 'auth'>) {
  const handleSendOTP = async () => {
    if (!isPhoneValid) return

    try {
      await auth.sendOTP(phoneNumber)
      onNext()
    } catch (error) {
      console.error('Failed to send OTP:', error)
    }
  }

  return (
    <AuthCard
      title={mode === 'signup' ? 'Create Your Account' : 'Sign In'}
      subtitle={mode === 'signup'
        ? 'Enter your phone number to create your DFS Pro account'
        : 'Enter your phone number to access your account'
      }
    >
      <div className="space-y-6">
        <ModernPhoneInput
          value={phoneNumber}
          onChange={setPhoneNumber}
          onValidate={setIsPhoneValid}
          autoFocus={true}
          error={auth.error}
          label={mode === 'signup' ? 'Your Phone Number' : 'Phone Number'}
          description={mode === 'signup'
            ? 'We\'ll send a verification code to create your account'
            : 'Enter your phone number to sign in'}
        />

        <div className="space-y-3">
          <Button
            onClick={handleSendOTP}
            disabled={!isPhoneValid || auth.isSendingOTP}
            loading={auth.isSendingOTP}
            variant="primary"
            size="lg"
            className="w-full"
          >
            {auth.isSendingOTP
              ? 'Sending Code...'
              : mode === 'signup'
              ? 'Create Account'
              : 'Send Verification Code'
            }
          </Button>

          <div className="flex justify-between items-center">
            <button
              onClick={onBack}
              className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
            >
              ← Back
            </button>

            <button
              onClick={onSwitchMode}
              className="text-sm text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300 font-medium transition-colors"
            >
              {mode === 'signup' ? 'Sign in instead' : 'Create account'}
            </button>
          </div>
        </div>
      </div>
    </AuthCard>
  )
}

function VerificationStep({
  mode,
  phoneNumber,
  verificationCode,
  setVerificationCode,
  onNext,
  onBack,
  auth
}: Pick<WizardStepProps, 'mode' | 'phoneNumber' | 'verificationCode' | 'setVerificationCode' | 'onNext' | 'onBack' | 'auth'>) {
  const handleVerifyOTP = async (code: string) => {
    try {
      const normalized = auth.normalizePhoneNumber(phoneNumber)
      await auth.verifyCode(normalized, code)
      onNext()
    } catch (error) {
      console.error('Failed to verify OTP:', error)
    }
  }

  const handleResendOTP = async () => {
    try {
      const normalized = auth.normalizePhoneNumber(phoneNumber)
      await auth.resendCode(normalized)
    } catch (error) {
      console.error('Failed to resend OTP:', error)
    }
  }

  return (
    <AuthCard>
      <EnhancedOTPVerification
        phoneNumber={auth.normalizePhoneNumber(phoneNumber)}
        value={verificationCode}
        onChange={setVerificationCode}
        onVerify={handleVerifyOTP}
        onResend={handleResendOTP}
        isVerifying={auth.isVerifyingOTP}
        isResending={auth.isResendingOTP}
        error={auth.error}
        autoFocus={true}
        title="Verify Your Phone Number"
        subtitle={mode === 'signup'
          ? "We'll send you a verification code to complete your account setup"
          : "Enter the verification code to access your account"
        }
      />

      <div className="mt-6 flex justify-center">
        <button
          onClick={onBack}
          className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
        >
          ← Change phone number
        </button>
      </div>
    </AuthCard>
  )
}

function EmailVerificationStep({
  mode,
  email,
  verificationCode,
  setVerificationCode,
  onNext,
  onBack
}: Pick<WizardStepProps, 'mode' | 'email' | 'verificationCode' | 'setVerificationCode' | 'onNext' | 'onBack'>) {
  const [isVerifying, setIsVerifying] = useState(false)
  const [isResending, setIsResending] = useState(false)
  const [error, setError] = useState('')
  const [showOTPInput, setShowOTPInput] = useState(false)

  const emailProvider = new EmailAuthProvider()

  const handleVerifyEmail = async (code: string) => {
    setIsVerifying(true)
    setError('')

    try {
      const result = await emailProvider.verifyCredentials({
        method: 'email',
        email,
        verificationCode: code,
        mode // added
      })

      if (result.success) {
        onNext()
      } else {
        setError(result.error || 'Invalid verification code')
      }
    } catch (err) {
      setError('Verification failed. Please try again.')
    } finally {
      setIsVerifying(false)
    }
  }

  const handleResendCode = async () => {
    setIsResending(true)
    setError('')

    try {
      const result = await emailProvider.resendVerification({
        method: 'email',
        email
      })

      if (!result.success) {
        setError(result.error || 'Failed to resend verification code')
      }
    } catch (err) {
      setError('Failed to resend verification code. Please try again.')
    } finally {
      setIsResending(false)
    }
  }

  return (
    <AuthCard>
      <div className="text-center space-y-6">
        <div className="mx-auto w-16 h-16 bg-gradient-to-br from-sky-400 to-sky-600 rounded-full flex items-center justify-center">
          <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
        </div>

        <div>
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
            Check Your Email
          </h3>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            We've sent a verification link to <span className="font-medium">{email}</span>
          </p>
        </div>

        <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
          <div className="flex items-start space-x-3">
            <div className="flex-shrink-0">
              <svg className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="text-sm">
              <p className="text-blue-700 dark:text-blue-300">
                Click the link in your email to verify your account. The link will expire in 24 hours.
              </p>
            </div>
          </div>
        </div>

        {/* Option to enter OTP code instead */}
        {!showOTPInput ? (
          <div className="space-y-4">
            <button
              onClick={() => setShowOTPInput(true)}
              className="text-sm text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300 font-medium transition-colors"
            >
              Enter verification code instead
            </button>

            <div className="flex justify-center">
              <button
                onClick={handleResendCode}
                disabled={isResending}
                className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
              >
                {isResending ? 'Resending...' : 'Resend email'}
              </button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="text-left">
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                Or enter the 6-digit code from your email:
              </p>
              <EnhancedOTPVerification
                phoneNumber={email}
                value={verificationCode}
                onChange={setVerificationCode}
                onVerify={handleVerifyEmail}
                onResend={handleResendCode}
                isVerifying={isVerifying}
                isResending={isResending}
                error={error}
                autoFocus={true}
                title=""
                subtitle=""
              />
            </div>

            <button
              onClick={() => setShowOTPInput(false)}
              className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
            >
              ← Back to email link
            </button>
          </div>
        )}

        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4">
            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
          </div>
        )}
      </div>

      <div className="mt-6 flex justify-center">
        <button
          onClick={onBack}
          className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
        >
          ← Change email address
        </button>
      </div>
    </AuthCard>
  )
}

function PasswordResetStep({
  email,
  onBack
}: Pick<WizardStepProps, 'email' | 'onBack'>) {
  const [isResending, setIsResending] = useState(false)
  const [error, setError] = useState('')

  const emailProvider = new EmailAuthProvider()

  const handleResend = async () => {
    setIsResending(true)
    setError('')
    try {
      const result = await emailProvider.resetPassword({ method: 'email', email })
      if (!result.success) {
        setError(result.error || 'Failed to resend reset email')
      }
    } catch (err) {
      setError('Failed to resend reset email. Please try again.')
    } finally {
      setIsResending(false)
    }
  }

  return (
    <AuthCard>
      <div className="text-center space-y-6">
        <div className="mx-auto w-16 h-16 bg-gradient-to-br from-sky-400 to-sky-600 rounded-full flex items-center justify-center">
          <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
        </div>

        <div>
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
            Check Your Email
          </h3>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            We've sent a password reset link to <span className="font-medium">{email}</span>
          </p>
        </div>

        <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
          <div className="flex items-start space-x-3">
            <div className="flex-shrink-0">
              <svg className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="text-sm">
              <p className="text-blue-700 dark:text-blue-300">
                Click the link in your email to set your password. The link will expire in 1 hour.
              </p>
            </div>
          </div>
        </div>

        <div className="flex justify-center">
          <button
            onClick={handleResend}
            disabled={isResending}
            className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
          >
            {isResending ? 'Resending...' : 'Resend reset email'}
          </button>
        </div>

        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4">
            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
          </div>
        )}
      </div>

      <div className="mt-6 flex justify-center">
        <button
          onClick={onBack}
          className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
        >
          ← Change email address
        </button>
      </div>
    </AuthCard>
  )
}

function SuccessStep({ mode, user, onNext }: Pick<WizardStepProps, 'mode' | 'user' | 'onNext'>) {
  useEffect(() => {
    // Auto-advance to onboarding after 2 seconds for new users
    if (mode === 'signup') {
      const timer = setTimeout(() => {
        onNext()
      }, 2000)
      return () => clearTimeout(timer)
    }
  }, [mode, onNext])

  return (
    <AuthCard>
      <div className="text-center space-y-6">
        {/* Success Icon */}
        <div className="mx-auto w-20 h-20 bg-gradient-to-br from-green-400 to-green-600 rounded-full flex items-center justify-center">
          <SparkleIcon className="w-10 h-10 text-white" animated />
        </div>

        <div>
          <h3 className="text-2xl font-semibold text-gray-900 dark:text-white">
            {mode === 'signup' ? 'Account Created!' : 'Welcome Back!'}
          </h3>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            {mode === 'signup'
              ? "You've successfully created your DFS Pro account"
              : "You've been signed in successfully"
            }
          </p>
        </div>

        {/* User Info */}
        <div className="bg-gray-50 dark:bg-gray-800 rounded-xl p-4">
          <div className="text-sm">
            <span className="text-gray-600 dark:text-gray-400">Signed in as:</span>
            <span className="ml-2 font-medium text-gray-900 dark:text-gray-100">
              {user?.phone || 'User'}
            </span>
          </div>
        </div>

        {mode === 'signup' ? (
          <div className="space-y-3">
            <Button
              onClick={onNext}
              variant="primary"
              size="lg"
              className="w-full"
              arrow
            >
              Complete Setup
            </Button>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Next: Let's set up your preferences and get you started
            </p>
          </div>
        ) : (
          <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-4">
            <p className="text-sm text-green-700 dark:text-green-300">
              Redirecting to your dashboard...
            </p>
          </div>
        )}
      </div>
    </AuthCard>
  )
}

function OnboardingStep({ user, onComplete }: { user: any; onComplete?: (user: any) => void }) {
  const [preferences, setPreferences] = useState({
    primarySport: '',
    experienceLevel: '',
    budgetRange: '',
    notifications: true
  })

  const handleComplete = () => {
    // Save preferences and complete onboarding
    console.log('Saving preferences:', preferences)
    onComplete?.(user)
  }

  return (
    <AuthCard
      title="Welcome to DFS Pro!"
      subtitle="Let's personalize your experience to help you build winning lineups"
    >
      <div className="space-y-6">
        {/* Primary Sport Selection */}
        <div>
          <label className="block text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">
            What's your primary sport?
          </label>
          <div className="grid grid-cols-2 gap-3">
            {['NFL', 'NBA', 'MLB', 'Golf'].map((sport) => (
              <button
                key={sport}
                onClick={() => setPreferences(p => ({ ...p, primarySport: sport }))}
                className={`p-3 rounded-lg border-2 transition-all ${
                  preferences.primarySport === sport
                    ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20 text-sky-700 dark:text-sky-300'
                    : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
                }`}
              >
                <div className="text-sm font-medium">{sport}</div>
              </button>
            ))}
          </div>
        </div>

        {/* Experience Level */}
        <div>
          <label className="block text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">
            How would you describe your DFS experience?
          </label>
          <div className="space-y-2">
            {[
              { id: 'beginner', label: 'Beginner', desc: 'New to DFS or just getting started' },
              { id: 'intermediate', label: 'Intermediate', desc: 'Some experience, looking to improve' },
              { id: 'advanced', label: 'Advanced', desc: 'Experienced player seeking optimization tools' }
            ].map((level) => (
              <button
                key={level.id}
                onClick={() => setPreferences(p => ({ ...p, experienceLevel: level.id }))}
                className={`w-full p-4 rounded-lg border-2 text-left transition-all ${
                  preferences.experienceLevel === level.id
                    ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20'
                    : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
                }`}
              >
                <div className="font-medium text-gray-900 dark:text-gray-100">{level.label}</div>
                <div className="text-sm text-gray-600 dark:text-gray-400">{level.desc}</div>
              </button>
            ))}
          </div>
        </div>

        <Button
          onClick={handleComplete}
          disabled={!preferences.primarySport || !preferences.experienceLevel}
          variant="primary"
          size="lg"
          className="w-full"
        >
          Complete Setup
        </Button>
      </div>
    </AuthCard>
  )
}

export function AuthWizard({
  onComplete,
  onClose,
  initialMode = 'login',
  initialStep = 'welcome',
  enabledMethods = getEnabledAuthMethods(), // Use config-driven enabled methods
  className = ''
}: AuthWizardProps) {
  const [currentStep, setCurrentStep] = useState<AuthWizardStep>(initialStep)
  const [mode, setMode] = useState<AuthWizardMode>(initialMode)
  const [selectedMethod, setSelectedMethod] = useState<AuthMethod>(enabledMethods[0] || 'email')
  const [phoneNumber, setPhoneNumber] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [isPhoneValid, setIsPhoneValid] = useState(false)
  const [isEmailValid, setIsEmailValid] = useState(false)
  const [isPasswordValid, setIsPasswordValid] = useState(false)
  const [, setPasswordStrength] = useState<PasswordStrength>({ score: 0, feedback: [], isValid: false })

  const auth = usePhoneAuth()
  const { user, otpSent } = useAuthStore()

  // Define steps based on auth method and mode
  const getSteps = (): AuthWizardStep[] => {
    const baseSteps: AuthWizardStep[] = ['welcome']

    // Add method selection if multiple methods are enabled
    if (enabledMethods.length > 1) {
      baseSteps.push('method-selection')
    }

    // Add method-specific steps
    if (selectedMethod === 'email') {
      baseSteps.push('email', 'password')
      if (mode === 'signup') {
        baseSteps.push('email-verification')
      }
    } else if (selectedMethod === 'phone') {
      baseSteps.push('phone', 'verification')
    }

    // Add common final steps
    baseSteps.push('success')
    if (mode === 'signup') {
      baseSteps.push('onboarding')
    }

    return baseSteps
  }

  const steps = getSteps()

  // Handle password strength validation
  const handlePasswordValidation = useCallback((isValid: boolean, strength: PasswordStrength) => {
    setIsPasswordValid(isValid)
    setPasswordStrength(strength)
  }, [setIsPasswordValid, setPasswordStrength])

  const completedSteps = steps.slice(0, steps.indexOf(currentStep))

  // Auto-advance when OTP is sent
  useEffect(() => {
    if (otpSent && currentStep === 'phone') {
      setCurrentStep('verification')
    }
  }, [otpSent, currentStep])

  // Auto-advance when user is authenticated
  useEffect(() => {
    if (user && currentStep === 'verification') {
      setCurrentStep('success')
    }
  }, [user, currentStep])

  const handleNext = () => {
    const currentIndex = steps.indexOf(currentStep)
    if (currentIndex < steps.length - 1) {
      setCurrentStep(steps[currentIndex + 1] as AuthWizardStep)
    } else if (currentStep === 'onboarding') {
      onComplete?.(user)
    } else {
      onComplete?.(user)
    }
  }

  const handleBack = () => {
    const currentIndex = steps.indexOf(currentStep)
    if (currentIndex > 0) {
      setCurrentStep(steps[currentIndex - 1] as AuthWizardStep)
    }
  }

  const handleSwitchMode = () => {
    setMode(mode === 'login' ? 'signup' : 'login')
    setCurrentStep('welcome')
    setPhoneNumber('')
    setEmail('')
    setPassword('')
    setVerificationCode('')
    setIsPhoneValid(false)
    setIsEmailValid(false)
    setIsPasswordValid(false)
    auth.clearError()
  }

  const stepProps: WizardStepProps = {
    step: currentStep,
    mode,
    selectedMethod,
    enabledMethods,
    phoneNumber,
    email,
    password,
    verificationCode,
    isPhoneValid,
    isEmailValid,
    isPasswordValid,
    setSelectedMethod,
    setPhoneNumber,
    setEmail,
    setPassword,
    setVerificationCode,
    setIsPhoneValid,
    setIsEmailValid,
    setIsPasswordValid: handlePasswordValidation,
    onNext: handleNext,
    onBack: handleBack,
    onSwitchMode: handleSwitchMode,
    onSetStep: setCurrentStep,
    auth,
    user
  }

  const renderStep = () => {
    switch (currentStep) {
      case 'welcome':
        return <WelcomeStep {...stepProps} />
      case 'method-selection':
        return <MethodSelectionStep {...stepProps} />
      case 'email':
        return <EmailStep {...stepProps} />
      case 'password':
        return <PasswordStep {...stepProps} />
      case 'phone':
        return <PhoneStep {...stepProps} />
      case 'verification':
        return <VerificationStep {...stepProps} />
      case 'email-verification':
        return <EmailVerificationStep {...stepProps} />
      case 'password-reset': // added
        return <PasswordResetStep {...stepProps} />
      case 'success':
        return <SuccessStep {...stepProps} />
      case 'onboarding':
        return <OnboardingStep user={user} onComplete={onComplete} />
      default:
        return <WelcomeStep {...stepProps} />
    }
  }

  return (
    <div className={className}>
      <AuthLayout showBranding={true}>
        <div className="space-y-6">
          {/* Step Indicator */}
          {currentStep !== 'welcome' && (
            <AuthStepIndicator
              steps={steps}
              currentStep={currentStep}
              completedSteps={completedSteps}
            />
          )}

          {/* Close Button */}
          {onClose && (
            <div className="flex justify-end">
              <button
                onClick={onClose}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          )}

          {/* Current Step */}
          {renderStep()}
        </div>
      </AuthLayout>
    </div>
  )
}
