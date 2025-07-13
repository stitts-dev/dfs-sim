import React, { useState, useEffect } from 'react'
import { Button, Dialog, DialogTitle, DialogBody, DialogActions } from '@/catalyst'
import { PhoneInput } from './PhoneInput'
import { OTPVerification } from './OTPVerification'
import { usePhoneAuth } from '@/hooks/usePhoneAuth'
import { useAuthStore } from '@/store/auth'
import clsx from 'clsx'

export type SignupStep = 'phone' | 'verification' | 'complete'

export interface SignupFlowProps {
  isOpen: boolean
  onClose: () => void
  onComplete?: (user: any) => void
  initialStep?: SignupStep
  className?: string
}

export const SignupFlow: React.FC<SignupFlowProps> = ({
  isOpen,
  onClose,
  onComplete,
  initialStep = 'phone',
  className
}) => {
  const [currentStep, setCurrentStep] = useState<SignupStep>(initialStep)
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
      }, 2000)
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
        <DialogTitle>Create Your Account</DialogTitle>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Enter your phone number to get started with DFS Optimizer
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

        <p className="text-xs text-center text-zinc-500 dark:text-zinc-400">
          By continuing, you agree to our Terms of Service and Privacy Policy
        </p>
      </div>
    </div>
  )

  const renderVerificationStep = () => (
    <div className="space-y-6">
      <div className="text-center">
        <DialogTitle>Verify Your Phone</DialogTitle>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Complete your account setup by verifying your phone number
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
        <DialogTitle>Welcome to DFS Optimizer!</DialogTitle>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Your account has been created successfully
        </p>
      </div>

      <div className="bg-zinc-50 dark:bg-zinc-800 rounded-lg p-4">
        <h3 className="font-medium text-zinc-900 dark:text-zinc-100 mb-2">
          What's next?
        </h3>
        <ul className="text-sm text-zinc-600 dark:text-zinc-400 space-y-1 text-left">
          <li>• Explore daily fantasy contests</li>
          <li>• Build and optimize your lineups</li>
          <li>• Run Monte Carlo simulations</li>
          <li>• Get AI-powered recommendations</li>
        </ul>
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

// Standalone signup page component
export const SignupPage: React.FC<{
  onComplete?: (user: any) => void
  className?: string
}> = ({ onComplete, className }) => {
  const [currentStep, setCurrentStep] = useState<SignupStep>('phone')

  return (
    <div className={clsx('min-h-screen flex items-center justify-center p-4', className)}>
      <div className="w-full max-w-md">
        <div className="bg-white dark:bg-zinc-800 rounded-lg shadow-lg p-6">
          <SignupFlow
            isOpen={true}
            onClose={() => {}}
            onComplete={onComplete}
            initialStep={currentStep}
          />
        </div>
      </div>
    </div>
  )
}

// Compact signup form for embedded use
export const InlineSignupForm: React.FC<{
  onComplete?: (user: any) => void
  className?: string
}> = ({ onComplete, className }) => {
  const [phoneNumber, setPhoneNumber] = useState('')
  const [isPhoneValid, setIsPhoneValid] = useState(false)
  const [showVerification, setShowVerification] = useState(false)

  const {
    sendOTP,
    isSendingOTP,
    error,
    otpSent,
    currentPhoneNumber
  } = usePhoneAuth()

  useEffect(() => {
    if (otpSent) {
      setShowVerification(true)
    }
  }, [otpSent])

  const handleSendOTP = async () => {
    if (!isPhoneValid) return

    try {
      await sendOTP(phoneNumber)
    } catch (error) {
      console.error('Failed to send OTP:', error)
    }
  }

  if (showVerification) {
    return (
      <SignupFlow
        isOpen={true}
        onClose={() => setShowVerification(false)}
        onComplete={onComplete}
        initialStep="verification"
        className={className}
      />
    )
  }

  return (
    <div className={clsx('space-y-4', className)}>
      <div className="text-center">
        <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
          Get Started
        </h2>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Create your account in seconds
        </p>
      </div>

      <PhoneInput
        value={phoneNumber}
        onChange={setPhoneNumber}
        onValidate={setIsPhoneValid}
        error={error}
      />

      <Button
        onClick={handleSendOTP}
        disabled={!isPhoneValid || isSendingOTP}
        className="w-full"
      >
        {isSendingOTP ? 'Creating Account...' : 'Create Account'}
      </Button>
    </div>
  )
}