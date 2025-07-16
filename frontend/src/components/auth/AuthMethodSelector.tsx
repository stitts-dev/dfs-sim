import { AuthMethod } from '@/types/auth'
import { EnvelopeIcon, DevicePhoneMobileIcon } from '@heroicons/react/24/outline'

interface AuthMethodSelectorProps {
  selectedMethod: AuthMethod
  onMethodSelect: (method: AuthMethod) => void
  enabledMethods: AuthMethod[]
  mode: 'login' | 'signup'
  className?: string
}

const methodConfig = {
  email: {
    icon: EnvelopeIcon,
    title: 'Email',
    description: 'Sign in with your email and password',
    signupDescription: 'Create account with email and password'
  },
  phone: {
    icon: DevicePhoneMobileIcon,
    title: 'Phone',
    description: 'Sign in with phone number verification',
    signupDescription: 'Create account with phone number'
  }
}

export function AuthMethodSelector({
  selectedMethod,
  onMethodSelect,
  enabledMethods,
  mode,
  className = ''
}: AuthMethodSelectorProps) {
  // If only one method is enabled, don't show selector
  if (enabledMethods.length <= 1) {
    return null
  }

  return (
    <div className={`space-y-4 ${className}`}>
      <div className="text-center">
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
          Choose your {mode === 'signup' ? 'sign up' : 'sign in'} method
        </h3>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Select how you'd like to {mode === 'signup' ? 'create your account' : 'access your account'}
        </p>
      </div>

      <div className="grid gap-3">
        {enabledMethods.map((method) => {
          const config = methodConfig[method]
          const Icon = config.icon
          const isSelected = selectedMethod === method
          
          return (
            <button
              key={method}
              onClick={() => onMethodSelect(method)}
              className={`
                relative p-4 rounded-lg border-2 text-left transition-all
                focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-sky-500
                ${isSelected
                  ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20'
                  : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500 bg-white dark:bg-gray-800'
                }
              `}
            >
              <div className="flex items-center space-x-3">
                <div className={`
                  flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center
                  ${isSelected
                    ? 'bg-sky-500 text-white'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                  }
                `}>
                  <Icon className="w-5 h-5" />
                </div>
                
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2">
                    <h4 className={`text-sm font-medium ${
                      isSelected 
                        ? 'text-sky-700 dark:text-sky-300' 
                        : 'text-gray-900 dark:text-gray-100'
                    }`}>
                      {config.title}
                    </h4>
                    
                    {isSelected && (
                      <div className="flex-shrink-0">
                        <svg className="w-4 h-4 text-sky-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                      </div>
                    )}
                  </div>
                  
                  <p className={`text-sm mt-1 ${
                    isSelected 
                      ? 'text-sky-600 dark:text-sky-400' 
                      : 'text-gray-600 dark:text-gray-400'
                  }`}>
                    {mode === 'signup' ? config.signupDescription : config.description}
                  </p>
                </div>
              </div>

              {/* Premium badge for certain methods */}
              {method === 'phone' && (
                <div className="absolute top-2 right-2">
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800 dark:bg-amber-900/20 dark:text-amber-400">
                    Coming Soon
                  </span>
                </div>
              )}
            </button>
          )
        })}
      </div>

      {/* Security note */}
      <div className="text-center">
        <p className="text-xs text-gray-500 dark:text-gray-400">
          All methods use industry-standard security and encryption
        </p>
      </div>
    </div>
  )
}