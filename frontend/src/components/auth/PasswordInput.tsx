import { useState, useEffect } from 'react'
import { EyeIcon, EyeSlashIcon, LockClosedIcon, ExclamationCircleIcon } from '@heroicons/react/24/outline'

interface PasswordInputProps {
  value: string
  onChange: (value: string) => void
  onValidate?: (isValid: boolean, strength: PasswordStrength) => void
  autoFocus?: boolean
  disabled?: boolean
  error?: string
  label?: string
  description?: string
  placeholder?: string
  showStrengthIndicator?: boolean
  requireStrength?: boolean
  className?: string
}

export interface PasswordStrength {
  score: number // 0-4
  feedback: string[]
  isValid: boolean
}

export function PasswordInput({
  value,
  onChange,
  onValidate,
  autoFocus = false,
  disabled = false,
  error,
  label = 'Password',
  description,
  placeholder = 'Enter your password',
  showStrengthIndicator = true,
  requireStrength = false,
  className = ''
}: PasswordInputProps) {
  const [isVisible, setIsVisible] = useState(false)
  const [isFocused, setIsFocused] = useState(false)
  const [hasBlurred, setHasBlurred] = useState(false)
  const [strength, setStrength] = useState<PasswordStrength>({
    score: 0,
    feedback: [],
    isValid: false
  })

  const calculatePasswordStrength = (password: string): PasswordStrength => {
    if (!password) {
      return { score: 0, feedback: ['Password is required'], isValid: false }
    }

    const feedback: string[] = []
    let score = 0

    // Length check
    if (password.length < 8) {
      feedback.push('At least 8 characters')
    } else if (password.length >= 12) {
      score += 2
    } else {
      score += 1
    }

    // Uppercase check
    if (!/[A-Z]/.test(password)) {
      feedback.push('At least one uppercase letter')
    } else {
      score += 1
    }

    // Lowercase check
    if (!/[a-z]/.test(password)) {
      feedback.push('At least one lowercase letter')
    } else {
      score += 1
    }

    // Number check
    if (!/\d/.test(password)) {
      feedback.push('At least one number')
    } else {
      score += 1
    }

    // Special character check
    if (!/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)) {
      feedback.push('At least one special character')
    } else {
      score += 1
    }

    // Common patterns to avoid
    if (/(.)\1{2,}/.test(password)) {
      feedback.push('Avoid repeating characters')
      score = Math.max(0, score - 1)
    }

    if (/123|abc|qwe|password|admin/i.test(password)) {
      feedback.push('Avoid common patterns')
      score = Math.max(0, score - 1)
    }

    const isValid = requireStrength ? score >= 4 : password.length >= 8
    
    return {
      score: Math.min(5, score),
      feedback: feedback.length > 0 ? feedback : ['Strong password'],
      isValid
    }
  }

  useEffect(() => {
    const newStrength = calculatePasswordStrength(value)
    setStrength(newStrength)
    onValidate?.(newStrength.isValid, newStrength)
  }, [value, onValidate, requireStrength])

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.value)
  }

  const toggleVisibility = () => {
    setIsVisible(!isVisible)
  }

  const handleFocus = () => {
    setIsFocused(true)
  }

  const handleBlur = () => {
    setIsFocused(false)
    setHasBlurred(true)
  }

  const showError = error || (hasBlurred && !strength.isValid && value.length > 0)
  const errorMessage = error || (hasBlurred && !strength.isValid && value.length > 0 ? strength.feedback[0] : '')

  const getStrengthColor = (score: number) => {
    if (score <= 1) return 'bg-red-500'
    if (score <= 2) return 'bg-orange-500'
    if (score <= 3) return 'bg-yellow-500'
    if (score <= 4) return 'bg-green-500'
    return 'bg-green-600'
  }

  const getStrengthText = (score: number) => {
    if (score <= 1) return 'Weak'
    if (score <= 2) return 'Fair'
    if (score <= 3) return 'Good'
    if (score <= 4) return 'Strong'
    return 'Very Strong'
  }

  return (
    <div className={`space-y-2 ${className}`}>
      {label && (
        <label className="block text-sm font-medium text-gray-900 dark:text-gray-100">
          {label}
        </label>
      )}
      
      {description && (
        <p className="text-sm text-gray-600 dark:text-gray-400">
          {description}
        </p>
      )}

      <div className="relative">
        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          <LockClosedIcon 
            className={`h-5 w-5 transition-colors ${
              showError 
                ? 'text-red-400' 
                : isFocused 
                ? 'text-sky-500' 
                : 'text-gray-400'
            }`} 
          />
        </div>
        
        <input
          type={isVisible ? 'text' : 'password'}
          value={value}
          onChange={handleChange}
          onFocus={handleFocus}
          onBlur={handleBlur}
          disabled={disabled}
          autoFocus={autoFocus}
          placeholder={placeholder}
          autoComplete="current-password"
          className={`
            block w-full pl-10 pr-12 py-3 border rounded-lg 
            text-gray-900 placeholder-gray-500 
            dark:text-gray-100 dark:placeholder-gray-400 
            dark:bg-gray-800 
            focus:outline-none focus:ring-2 focus:ring-offset-2 
            transition-colors
            ${showError
              ? 'border-red-300 focus:border-red-500 focus:ring-red-500' 
              : strength.isValid && value.length > 0
              ? 'border-green-300 focus:border-green-500 focus:ring-green-500'
              : 'border-gray-300 focus:border-sky-500 focus:ring-sky-500 dark:border-gray-600 dark:focus:border-sky-400'
            }
            ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
          `}
        />

        {/* Visibility toggle */}
        <div className="absolute inset-y-0 right-0 pr-3 flex items-center">
          <button
            type="button"
            onClick={toggleVisibility}
            disabled={disabled}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:outline-none transition-colors"
          >
            {isVisible ? (
              <EyeSlashIcon className="h-5 w-5" />
            ) : (
              <EyeIcon className="h-5 w-5" />
            )}
          </button>
        </div>
      </div>

      {/* Password strength indicator */}
      {showStrengthIndicator && value.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-600 dark:text-gray-400">
              Password strength: 
              <span className={`ml-1 font-medium ${
                strength.score <= 2 ? 'text-red-600 dark:text-red-400' : 
                strength.score <= 3 ? 'text-yellow-600 dark:text-yellow-400' : 
                'text-green-600 dark:text-green-400'
              }`}>
                {getStrengthText(strength.score)}
              </span>
            </span>
          </div>
          
          <div className="flex space-x-1">
            {[1, 2, 3, 4, 5].map((level) => (
              <div
                key={level}
                className={`h-2 flex-1 rounded-full transition-colors ${
                  level <= strength.score 
                    ? getStrengthColor(strength.score)
                    : 'bg-gray-200 dark:bg-gray-700'
                }`}
              />
            ))}
          </div>
        </div>
      )}

      {/* Error message */}
      {showError && (
        <p className="text-sm text-red-600 dark:text-red-400 flex items-center">
          <ExclamationCircleIcon className="h-4 w-4 mr-1" />
          {errorMessage}
        </p>
      )}

      {/* Strength feedback */}
      {!showError && isFocused && strength.feedback.length > 0 && strength.score < 5 && (
        <div className="text-sm text-gray-500 dark:text-gray-400">
          <p className="font-medium mb-1">To improve your password:</p>
          <ul className="space-y-1">
            {strength.feedback.slice(0, 3).map((tip, index) => (
              <li key={index} className="flex items-center">
                <span className="w-1.5 h-1.5 bg-gray-400 rounded-full mr-2" />
                {tip}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}