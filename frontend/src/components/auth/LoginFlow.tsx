import React, { useState, useEffect } from 'react'
import { Dialog, DialogTitle, DialogBody, DialogActions } from '@/catalyst'
import { Button } from '@/components/ui/Button'
import { PhoneInput } from './PhoneInput'
import { OTPVerification } from './OTPVerification'
import { usePhoneAuth } from '@/hooks/usePhoneAuth'
import { useAuthStore } from '@/store/auth'
import clsx from 'clsx'

export type LoginStep = 'phone' | 'verification' | 'complete'

export interface LoginFlowProps {
  isOpen: boolean
  onClose: () => void
  onComplete?: (user: any) => void
  onSwitchToSignup?: () => void
  initialStep?: LoginStep
  className?: string
}

export const LoginFlow: React.FC<LoginFlowProps> = ({
  isOpen,
  onClose,
  onComplete,
  onSwitchToSignup,
  initialStep = 'phone',
  className
}) => {
  const [currentStep, setCurrentStep] = useState<LoginStep>(initialStep)
  const [phoneNumber, setPhoneNumber] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [isPhoneValid, setIsPhoneValid] = useState(false)

  const {
    sendOTP,
    verifyCode,
    resendCode,
    isSendingOTP,
    isVerifyingOTP,
    isResendingOTP,
    error,
    clearError,
    normalizePhoneNumber
  } = usePhoneAuth()

  const { user, currentPhoneNumber, otpSent } = useAuthStore()

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (isOpen) {
      setCurrentStep(initialStep)
      setVerificationCode('')
      clearError()
      if (!currentPhoneNumber) {
        setPhoneNumber('')
      } else {
        setPhoneNumber(currentPhoneNumber)
      }
    }
  }, [isOpen, initialStep, currentPhoneNumber, clearError])

  // Auto-advance to verification step if OTP was sent
  useEffect(() => {
    if (otpSent && currentStep === 'phone') {
      setCurrentStep('verification')
    }
  }, [otpSent, currentStep])

  // Handle successful authentication
  useEffect(() => {
    if (user && isOpen) {
      setCurrentStep('complete')
      if (onComplete) {
        onComplete(user)
      }
      // Auto-close after a brief delay
      setTimeout(() => {
        onClose()
      }, 1500)
    }
  }, [user, isOpen, onComplete, onClose])

  const handleSendOTP = async () => {
    if (!isPhoneValid) return

    try {
      const normalized = normalizePhoneNumber(phoneNumber)
      await sendOTP(normalized)
      setCurrentStep('verification')
    } catch (error) {
      console.error('Failed to send OTP:', error)
      // Check if error indicates user doesn't exist
      if (error instanceof Error && error.message.includes('not registered')) {
        // Could switch to signup flow or show error
      }
    }
  }

  const handleVerifyOTP = async (code: string) => {
    try {
      const normalized = normalizePhoneNumber(phoneNumber)
      await verifyCode(normalized, code)
      // User state will update and trigger useEffect above
    } catch (error) {
      console.error('Failed to verify OTP:', error)
    }
  }

  const handleResendOTP = async () => {
    try {
      const normalized = normalizePhoneNumber(phoneNumber)
      await resendCode(normalized)
    } catch (error) {
      console.error('Failed to resend OTP:', error)
    }
  }

  const handleBack = () => {
    if (currentStep === 'verification') {
      setCurrentStep('phone')
      setVerificationCode('')
      clearError()
    }
  }

  const renderPhoneStep = () => (
    <div className="space-y-6">
      <div className="text-center">
        <DialogTitle>Welcome Back</DialogTitle>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Enter your phone number to sign in to your account
        </p>
      </div>

      <PhoneInput
        value={phoneNumber}
        onChange={setPhoneNumber}
        onValidate={setIsPhoneValid}
        autoFocus={true}
        error={error}
      />

      <div className="flex flex-col gap-3">
        <Button
          onClick={handleSendOTP}
          disabled={!isPhoneValid || isSendingOTP}
          className="w-full"
        >
          {isSendingOTP ? 'Sending Code...' : 'Send Verification Code'}
        </Button>

        {onSwitchToSignup && (
          <div className="text-center">
            <p className="text-sm text-zinc-600 dark:text-zinc-400">
              Don't have an account?{' '}
              <button
                onClick={onSwitchToSignup}
                className="text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300 font-medium transition-colors"
              >
                Sign up
              </button>
            </p>
          </div>
        )}
      </div>
    </div>
  )

  const renderVerificationStep = () => (
    <div className="space-y-6">
      <div className="text-center">
        <DialogTitle>Verify Your Identity</DialogTitle>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Enter the verification code to sign in
        </p>
      </div>

      <OTPVerification
        phoneNumber={normalizePhoneNumber(phoneNumber)}
        value={verificationCode}
        onChange={setVerificationCode}
        onVerify={handleVerifyOTP}
        onResend={handleResendOTP}
        isVerifying={isVerifyingOTP}
        isResending={isResendingOTP}
        error={error}
        autoFocus={true}
      />

      <div className="flex justify-center">
        <button
          onClick={handleBack}
          className="text-sm text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200 transition-colors"
        >
          ← Change phone number
        </button>
      </div>
    </div>
  )

  const renderCompleteStep = () => (
    <div className="space-y-6 text-center">
      <div className="mx-auto w-16 h-16 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center">
        <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
        </svg>
      </div>

      <div>
        <DialogTitle>Welcome Back!</DialogTitle>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          You've been signed in successfully
        </p>
      </div>

      <div className="bg-zinc-50 dark:bg-zinc-800 rounded-lg p-4">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Redirecting to your dashboard...
        </p>
      </div>
    </div>
  )

  const getStepContent = () => {
    switch (currentStep) {
      case 'phone':
        return renderPhoneStep()
      case 'verification':
        return renderVerificationStep()
      case 'complete':
        return renderCompleteStep()
      default:
        return renderPhoneStep()
    }
  }

  return (
    <Dialog open={isOpen} onClose={onClose} className={className}>
      <DialogBody>
        {getStepContent()}
      </DialogBody>

      {currentStep !== 'complete' && (
        <DialogActions>
          <Button onClick={onClose} variant="outline">
            Cancel
          </Button>
        </DialogActions>
      )}

      {/* Progress indicator */}
      <div className="absolute top-4 right-4">
        <div className="flex space-x-2">
          {['phone', 'verification', 'complete'].map((step, index) => (
            <div
              key={step}
              className={clsx(
                'w-2 h-2 rounded-full transition-colors',
                step === currentStep
                  ? 'bg-blue-600 dark:bg-blue-400'
                  : index < ['phone', 'verification', 'complete'].indexOf(currentStep)
                  ? 'bg-green-600 dark:bg-green-400'
                  : 'bg-zinc-300 dark:bg-zinc-600'
              )}
            />
          ))}
        </div>
      </div>
    </Dialog>
  )
}

// Combined auth flow that can switch between login and signup
export const AuthFlow: React.FC<{
  isOpen: boolean
  onClose: () => void
  onComplete?: (user: any) => void
  initialMode?: 'login' | 'signup'
  className?: string
}> = ({
  isOpen,
  onClose,
  onComplete,
  initialMode = 'login',
  className
}) => {
  const [mode, setMode] = useState<'login' | 'signup'>(initialMode)

  useEffect(() => {
    if (isOpen) {
      setMode(initialMode)
    }
  }, [isOpen, initialMode])

  if (mode === 'signup') {
    return (
      <div>
        {/* Import and use SignupFlow */}
        <div className="text-center mb-4">
          <p className="text-sm text-zinc-600 dark:text-zinc-400">
            Already have an account?{' '}
            <button
              onClick={() => setMode('login')}
              className="text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300 font-medium transition-colors"
            >
              Sign in
            </button>
          </p>
        </div>
        {/* SignupFlow component would go here */}
      </div>
    )
  }

  return (
    <LoginFlow
      isOpen={isOpen}
      onClose={onClose}
      onComplete={onComplete}
      onSwitchToSignup={() => setMode('signup')}
      className={className}
    />
  )
}

// Standalone login page component
export const LoginPage: React.FC<{
  onComplete?: (user: any) => void
  className?: string
}> = ({ onComplete, className }) => {
  return (
    <div className={clsx('min-h-screen flex items-center justify-center p-4', className)}>
      <div className="w-full max-w-md">
        <div className="bg-white dark:bg-zinc-800 rounded-lg shadow-lg p-6">
          <AuthFlow
            isOpen={true}
            onClose={() => {}}
            onComplete={onComplete}
            initialMode="login"
          />
        </div>
      </div>
    </div>
  )
}

// Quick login button for existing users
export const QuickLoginButton: React.FC<{
  phoneNumber?: string
  onComplete?: (user: any) => void
  className?: string
}> = ({ phoneNumber, onComplete, className }) => {
  const [showLogin, setShowLogin] = useState(false)
  const { sendOTP, isSendingOTP } = usePhoneAuth()

  const handleQuickLogin = async () => {
    if (phoneNumber) {
      try {
        await sendOTP(phoneNumber)
        setShowLogin(true)
      } catch (error) {
        console.error('Quick login failed:', error)
        setShowLogin(true) // Show full login flow as fallback
      }
    } else {
      setShowLogin(true)
    }
  }

  return (
    <>
      <Button
        onClick={handleQuickLogin}
        disabled={isSendingOTP}
        className={clsx('w-full', className)}
        variant="outline"
      >
        {isSendingOTP ? 'Sending Code...' : phoneNumber ? `Sign in as ${phoneNumber}` : 'Sign In'}
      </Button>

      <LoginFlow
        isOpen={showLogin}
        onClose={() => setShowLogin(false)}
        onComplete={onComplete}
        initialStep={phoneNumber ? 'verification' : 'phone'}
      />
    </>
  )
}