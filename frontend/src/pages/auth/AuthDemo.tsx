import { useState } from 'react'
import { AuthLayout, AuthCard } from '@/components/auth/AuthLayout'
import { PhoneInput } from '@/components/auth/PhoneInput'
import { EnhancedOTPVerification } from '@/components/auth/EnhancedOTPVerification'
import { AuthWizard } from '@/components/auth/AuthWizard'
import { Button } from '@/components/ui/Button'
import { StarField } from '@/components/ui/StarField'
import { SparkleIcon } from '@/components/ui/SparkleIcon'
import { Glow } from '@/components/ui/Glow'

export default function AuthDemo() {
  const [demoMode, setDemoMode] = useState<'components' | 'wizard'>('components')
  const [phoneNumber, setPhoneNumber] = useState('')
  const [otpCode, setOtpCode] = useState('')
  const [isPhoneValid, setIsPhoneValid] = useState(false)

  const handleOTPVerify = async (code: string) => {
    console.log('Verifying OTP:', code)
    // Mock verification
    await new Promise(resolve => setTimeout(resolve, 1000))
  }

  const handleOTPResend = async () => {
    console.log('Resending OTP')
    // Mock resend
    await new Promise(resolve => setTimeout(resolve, 500))
  }

  if (demoMode === 'wizard') {
    return (
      <div className="min-h-screen">
        <div className="fixed top-4 left-4 z-50 space-x-2">
          <Button
            onClick={() => setDemoMode('components')}
            variant="outline"
            size="sm"
          >
            ← Back to Components
          </Button>
        </div>
        
        <AuthWizard
          initialMode="signup"
          initialStep="welcome"
          onComplete={(user) => {
            console.log('Wizard completed:', user)
            setDemoMode('components')
          }}
          onClose={() => setDemoMode('components')}
        />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Theme Toggle and Controls */}
      <div className="fixed top-4 right-4 z-50 space-x-2">
        <Button
          onClick={() => setDemoMode('wizard')}
          variant="primary"
          size="sm"
        >
          View Full Wizard
        </Button>
        <Button
          onClick={() => {
            document.documentElement.classList.toggle('dark')
          }}
          variant="outline"
          size="sm"
        >
          Toggle Dark Mode
        </Button>
      </div>

      <div className="py-12 px-4">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-12">
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-4">
              Enhanced Auth Components Demo
            </h1>
            <p className="text-gray-600 dark:text-gray-400">
              Testing responsive design, dark mode, and visual effects
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* AuthLayout Demo */}
            <section className="space-y-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                AuthLayout Component
              </h2>
              
              <div className="h-96 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden">
                <AuthLayout showBranding={true}>
                  <AuthCard
                    title="Demo Card"
                    subtitle="This shows the dual-pane layout with branding"
                  >
                    <div className="space-y-4">
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Left side shows branding with StarField and Glow effects.
                        Right side contains the form content.
                      </p>
                      
                      <Button variant="primary" className="w-full">
                        Sample Button
                      </Button>
                    </div>
                  </AuthCard>
                </AuthLayout>
              </div>
            </section>

            {/* Visual Effects Demo */}
            <section className="space-y-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Visual Effects
              </h2>
              
              <div className="space-y-4">
                {/* StarField Demo */}
                <div className="relative h-40 bg-gray-950 rounded-lg overflow-hidden">
                  <Glow />
                  <StarField className="top-4 -right-20" />
                  <div className="relative z-10 flex items-center justify-center h-full">
                    <div className="text-white text-center">
                      <SparkleIcon className="w-8 h-8 mx-auto mb-2" animated />
                      <p className="text-sm">StarField + Glow Effects</p>
                    </div>
                  </div>
                </div>

                {/* Button Variants */}
                <div className="grid grid-cols-2 gap-4">
                  <Button variant="primary" size="lg">
                    Primary
                  </Button>
                  <Button variant="secondary" size="lg">
                    Secondary
                  </Button>
                  <Button variant="outline" size="lg">
                    Outline
                  </Button>
                  <Button variant="ghost" size="lg">
                    Ghost
                  </Button>
                </div>
              </div>
            </section>

            {/* PhoneInput Demo */}
            <section className="space-y-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Simplified PhoneInput
              </h2>
              
              <AuthCard>
                <div className="space-y-6">
                  <PhoneInput
                    value={phoneNumber}
                    onChange={setPhoneNumber}
                    onValidate={setIsPhoneValid}
                  />
                  
                  <div className="text-sm">
                    <p className="text-gray-600 dark:text-gray-400">
                      Valid: {isPhoneValid ? '✅' : '❌'}
                    </p>
                    <p className="text-gray-600 dark:text-gray-400">
                      Value: {phoneNumber || 'None'}
                    </p>
                  </div>
                </div>
              </AuthCard>
            </section>

            {/* Enhanced OTP Demo */}
            <section className="space-y-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Enhanced OTP Verification
              </h2>
              
              <AuthCard>
                <EnhancedOTPVerification
                  phoneNumber="+1 (555) 123-4567"
                  value={otpCode}
                  onChange={setOtpCode}
                  onVerify={handleOTPVerify}
                  onResend={handleOTPResend}
                  title="Demo OTP Verification"
                  subtitle="Enter any 6 digits to see the success animation"
                />
              </AuthCard>
            </section>

            {/* Responsive Test */}
            <section className="lg:col-span-2 space-y-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Responsive Design Test
              </h2>
              
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                {Array.from({ length: 4 }, (_, i) => (
                  <AuthCard key={i} className="min-h-32">
                    <div className="text-center">
                      <SparkleIcon className="w-6 h-6 mx-auto mb-2 text-sky-500" />
                      <p className="text-sm font-medium text-gray-900 dark:text-white">
                        Card {i + 1}
                      </p>
                      <p className="text-xs text-gray-600 dark:text-gray-400">
                        Responsive grid
                      </p>
                    </div>
                  </AuthCard>
                ))}
              </div>
            </section>

            {/* Dark Mode Specific Elements */}
            <section className="lg:col-span-2 space-y-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Dark Mode Test
              </h2>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                    Light Mode Elements
                  </h3>
                  <div className="bg-white p-4 rounded-lg border">
                    <p className="text-gray-900 mb-2">Light mode content</p>
                    <Button variant="primary" size="sm">Light Button</Button>
                  </div>
                </div>
                
                <div className="space-y-4">
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                    Dark Mode Elements
                  </h3>
                  <div className="bg-gray-800 p-4 rounded-lg border border-gray-700">
                    <p className="text-gray-100 mb-2">Dark mode content</p>
                    <Button variant="outline" size="sm">Dark Button</Button>
                  </div>
                </div>
              </div>
            </section>
          </div>
        </div>
      </div>
    </div>
  )
}