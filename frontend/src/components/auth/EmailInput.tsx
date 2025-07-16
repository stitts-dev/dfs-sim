import { useState, useEffect } from 'react'
import { EnvelopeIcon, ExclamationCircleIcon } from '@heroicons/react/24/outline'

interface EmailInputProps {
  value: string
  onChange: (value: string) => void
  onValidate?: (isValid: boolean) => void
  autoFocus?: boolean
  disabled?: boolean
  error?: string
  label?: string
  description?: string
  placeholder?: string
  className?: string
}

export function EmailInput({
  value,
  onChange,
  onValidate,
  autoFocus = false,
  disabled = false,
  error,
  label = 'Email Address',
  description,
  placeholder = 'Enter your email address',
  className = ''
}: EmailInputProps) {
  const [isValid, setIsValid] = useState(false)
  const [isFocused, setIsFocused] = useState(false)
  const [hasBlurred, setHasBlurred] = useState(false)

  // Email validation regex
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

  useEffect(() => {
    const valid = emailRegex.test(value)
    setIsValid(valid)
    onValidate?.(valid)
  }, [value, onValidate])

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value
    onChange(newValue)
  }

  const handleFocus = () => {
    setIsFocused(true)
  }

  const handleBlur = () => {
    setIsFocused(false)
    setHasBlurred(true)
  }

  const showError = error || (hasBlurred && !isValid && value.length > 0)
  const errorMessage = error || (hasBlurred && !isValid && value.length > 0 ? 'Please enter a valid email address' : '')

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
          <EnvelopeIcon 
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
          type="email"
          value={value}
          onChange={handleChange}
          onFocus={handleFocus}
          onBlur={handleBlur}
          disabled={disabled}
          autoFocus={autoFocus}
          placeholder={placeholder}
          autoComplete="email"
          className={`
            block w-full pl-10 pr-3 py-3 border rounded-lg 
            text-gray-900 placeholder-gray-500 
            dark:text-gray-100 dark:placeholder-gray-400 
            dark:bg-gray-800 
            focus:outline-none focus:ring-2 focus:ring-offset-2 
            transition-colors
            ${showError
              ? 'border-red-300 focus:border-red-500 focus:ring-red-500' 
              : isValid && value.length > 0
              ? 'border-green-300 focus:border-green-500 focus:ring-green-500'
              : 'border-gray-300 focus:border-sky-500 focus:ring-sky-500 dark:border-gray-600 dark:focus:border-sky-400'
            }
            ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
          `}
        />

        {/* Success indicator */}
        {isValid && value.length > 0 && !showError && (
          <div className="absolute inset-y-0 right-0 pr-3 flex items-center">
            <svg className="h-5 w-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
        )}

        {/* Error indicator */}
        {showError && (
          <div className="absolute inset-y-0 right-0 pr-3 flex items-center">
            <ExclamationCircleIcon className="h-5 w-5 text-red-500" />
          </div>
        )}
      </div>

      {/* Error message */}
      {showError && (
        <p className="text-sm text-red-600 dark:text-red-400 flex items-center">
          <ExclamationCircleIcon className="h-4 w-4 mr-1" />
          {errorMessage}
        </p>
      )}

      {/* Validation hint */}
      {!showError && isFocused && value.length === 0 && (
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Enter a valid email address (e.g., user@example.com)
        </p>
      )}
    </div>
  )
}