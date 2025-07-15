import { useState, useEffect } from 'react'
import { AuthLayout, AuthCard, AuthStepIndicator } from './AuthLayout'
import { ModernPhoneInput } from './ModernPhoneInput'
import { EnhancedOTPVerification } from './EnhancedOTPVerification'
import { Button } from '@/components/ui/Button'
import { SparkleIcon } from '@/components/ui/SparkleIcon'
import { usePhoneAuth } from '@/hooks/usePhoneAuth'
import { useAuthStore } from '@/store/auth'

export type AuthWizardStep = 'welcome' | 'phone' | 'verification' | 'success' | 'onboarding'
export type AuthWizardMode = 'login' | 'signup'

export interface AuthWizardProps {
  onComplete?: (user: any) => void
  onClose?: () => void
  initialMode?: AuthWizardMode
  initialStep?: AuthWizardStep
  className?: string
}

interface WizardStepProps {
  step: AuthWizardStep
  mode: AuthWizardMode
  phoneNumber: string
  verificationCode: string
  isPhoneValid: boolean
  setPhoneNumber: (phone: string) => void
  setVerificationCode: (code: string) => void
  setIsPhoneValid: (valid: boolean) => void
  onNext: () => void
  onBack: () => void
  onSwitchMode: () => void
  auth: ReturnType<typeof usePhoneAuth>
  user: any
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
  className = ''
}: AuthWizardProps) {
  const [currentStep, setCurrentStep] = useState<AuthWizardStep>(initialStep)
  const [mode, setMode] = useState<AuthWizardMode>(initialMode)
  const [phoneNumber, setPhoneNumber] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [isPhoneValid, setIsPhoneValid] = useState(false)

  const auth = usePhoneAuth()
  const { user, otpSent } = useAuthStore()

  const steps = mode === 'signup' 
    ? ['welcome', 'phone', 'verification', 'success', 'onboarding']
    : ['welcome', 'phone', 'verification', 'success']

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
    setVerificationCode('')
    auth.clearError()
  }

  const stepProps: WizardStepProps = {
    step: currentStep,
    mode,
    phoneNumber,
    verificationCode,
    isPhoneValid,
    setPhoneNumber,
    setVerificationCode,
    setIsPhoneValid,
    onNext: handleNext,
    onBack: handleBack,
    onSwitchMode: handleSwitchMode,
    auth,
    user
  }

  const renderStep = () => {
    switch (currentStep) {
      case 'welcome':
        return <WelcomeStep {...stepProps} />
      case 'phone':
        return <PhoneStep {...stepProps} />
      case 'verification':
        return <VerificationStep {...stepProps} />
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