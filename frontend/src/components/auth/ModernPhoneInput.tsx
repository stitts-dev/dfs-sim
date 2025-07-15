import React, { useState, useEffect, useRef } from 'react'
import { Field, Input, Description, ErrorMessage } from '@/catalyst'
import { PhoneInputProps } from '@/types/phoneInput'
import { formatPhoneNumber, validatePhoneNumber } from '@/services/supabase'
import { CountrySelector } from './CountrySelector'

export const ModernPhoneInput: React.FC<PhoneInputProps> = ({
  value,
  onChange,
  onValidate,
  disabled = false,
  autoFocus = false,
  error = false,
  className,
  label = "Phone Number",
  description = "Enter your phone number to receive a verification code",
  placeholder = "+1 (555) 123-4567"
}) => {
  const [validationError, setValidationError] = useState<string | null>(null)
  const [isFocused, setIsFocused] = useState(false)
  const [isFloating, setIsFloating] = useState(false)
  const [countryCode, setCountryCode] = useState('US')
  const inputRef = useRef<HTMLInputElement>(null)

  // Format and validate phone number when value changes
  useEffect(() => {
    if (value) {
      const isValidNumber = validatePhoneNumber(value)
      setValidationError(isValidNumber ? null : 'Please enter a valid phone number')
      onValidate?.(isValidNumber)
    } else {
      setValidationError(null)
      onValidate?.(false)
    }
  }, [value, onValidate])

  // Handle floating label state
  useEffect(() => {
    setIsFloating(isFocused || !!value)
  }, [isFocused, value])

  // Simple digit extraction only - single source of truth
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const digitsOnly = e.target.value.replace(/\D/g, '')
    const limitedDigits = digitsOnly.slice(0, 11) // US: 1 + 10 digits
    onChange?.(limitedDigits)
  }

  // Separate paste handler to handle formatted input
  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault()
    const pastedText = e.clipboardData.getData('text')
    const digitsOnly = pastedText.replace(/\D/g, '')
    const limitedDigits = digitsOnly.slice(0, 11)
    onChange?.(limitedDigits)
  }

  const handleFocus = () => {
    setIsFocused(true)
  }

  const handleBlur = () => {
    setIsFocused(false)
  }

  const handleCountryChange = (newCountryCode: string) => {
    setCountryCode(newCountryCode)
    if (inputRef.current) {
      inputRef.current.focus()
    }
  }

  // Format for display using existing supabase service function
  const formattedValue = formatPhoneNumber(value)

  const hasError = error || (validationError && value)

  return (
    <Field className={`relative ${className}`}>
      {/* Modern Glass-morphism Container */}
      <div className="relative">
        {/* Background Glass Effect */}
        <div className={`
          absolute inset-0 rounded-xl transition-all duration-300 ease-out
          ${isFocused 
            ? 'bg-white/10 dark:bg-white/5 backdrop-blur-sm shadow-lg ring-2 ring-sky-500/50 dark:ring-sky-400/50' 
            : 'bg-white/5 dark:bg-white/2 backdrop-blur-xs shadow-md hover:shadow-lg hover:bg-white/8 dark:hover:bg-white/4'
          }
          ${hasError ? 'ring-2 ring-red-500/50 bg-red-50/5 dark:bg-red-900/5' : ''}
        `} />

        {/* Floating Label */}
        <div className="relative">
          <div className={`
            absolute left-16 z-10 pointer-events-none transition-all duration-300 ease-out
            ${isFloating 
              ? 'top-2 text-xs font-medium text-sky-600 dark:text-sky-400 transform scale-90 origin-left' 
              : 'top-1/2 -translate-y-1/2 text-base text-gray-500 dark:text-gray-400'
            }
            ${hasError && isFloating ? 'text-red-500 dark:text-red-400' : ''}
            ${isFocused ? 'animate-glow' : ''}
          `}>
            {label}
          </div>

          {/* Input Container */}
          <div className="relative flex items-center">
            {/* Country Selector */}
            <div className="relative">
              <CountrySelector
                value={countryCode}
                onChange={handleCountryChange}
                disabled={disabled}
                className="h-12 px-3 border-r border-gray-200/50 dark:border-gray-700/50"
              />
            </div>

            {/* Enhanced Input */}
            <div className="flex-1 relative">
              <Input
                ref={inputRef}
                type="tel"
                value={formattedValue}
                onChange={handleChange}
                onPaste={handlePaste}
                onFocus={handleFocus}
                onBlur={handleBlur}
                placeholder={isFloating ? placeholder : ''}
                disabled={disabled}
                autoFocus={autoFocus}
                autoComplete="tel"
                inputMode="tel"
                aria-label={label}
                aria-describedby={description ? 'phone-description' : undefined}
                aria-invalid={hasError ? 'true' : 'false'}
                aria-required={true}
                className={`
                  w-full h-12 bg-transparent border-0 pl-4 pr-12
                  ${isFloating ? 'pt-6 pb-2' : 'pt-3 pb-3'}
                  text-gray-900 dark:text-white placeholder:text-gray-400
                  focus:outline-none focus:ring-0 transition-all duration-300
                  ${hasError ? 'text-red-600 dark:text-red-400' : ''}
                  text-lg sm:text-base
                `}
              />

              {/* Focus Glow Effect */}
              {isFocused && (
                <div className="absolute inset-0 rounded-xl animate-glow pointer-events-none opacity-50" />
              )}

              {/* Typing Animation Effect */}
              {isFocused && value && (
                <div className="absolute inset-0 rounded-xl bg-gradient-to-r from-sky-500/5 to-purple-500/5 animate-gradient pointer-events-none" />
              )}
            </div>

            {/* Status Indicator */}
            <div className="absolute right-3 top-1/2 -translate-y-1/2">
              {validationError && value ? (
                <div className="flex items-center space-x-1">
                  <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                  <svg className="w-4 h-4 text-red-500 animate-bounce" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                  </svg>
                </div>
              ) : value && !validationError ? (
                <div className="flex items-center space-x-1">
                  <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                  <svg className="w-4 h-4 text-green-500 animate-bounce" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
              ) : null}
            </div>
          </div>
        </div>
      </div>

      {/* Description */}
      {description && !hasError && (
        <Description id="phone-description" className="text-sm text-gray-600 dark:text-gray-400 mt-3 ml-1 animate-fade-in">
          {description}
        </Description>
      )}
      
      {/* Error Message */}
      {validationError && value && (
        <ErrorMessage className="text-sm text-red-600 dark:text-red-400 mt-3 ml-1 animate-slide-in">
          {validationError}
        </ErrorMessage>
      )}
      
      {error && typeof error === 'string' && (
        <ErrorMessage className="text-sm text-red-600 dark:text-red-400 mt-3 ml-1 animate-slide-in">
          {error}
        </ErrorMessage>
      )}
    </Field>
  )
}