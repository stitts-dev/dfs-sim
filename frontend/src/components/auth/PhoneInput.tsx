import React, { useState, useEffect } from 'react'
import { Field, Label, Input, Description, ErrorMessage } from '@/catalyst'
import { PhoneInputProps } from '@/types/auth'
import { formatPhoneNumber, validatePhoneNumber } from '@/services/supabase'

export const PhoneInput: React.FC<PhoneInputProps> = ({
  value,
  onChange,
  onValidate,
  disabled = false,
  autoFocus = false,
  error = false,
  className
}) => {
  const [validationError, setValidationError] = useState<string | null>(null)

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

  // Format for display using existing supabase service function
  const formattedValue = formatPhoneNumber(value)

  return (
    <Field className={className}>
      <Label className="text-sm font-medium text-gray-900 dark:text-gray-100">
        Phone Number
      </Label>
      <Input
        type="tel"
        value={formattedValue}
        onChange={handleChange}
        onPaste={handlePaste}
        placeholder="+1 (555) 123-4567"
        disabled={disabled}
        autoFocus={autoFocus}
        autoComplete="tel"
        className={`
          bg-white dark:bg-gray-800 
          border-2 border-gray-300 dark:border-gray-600 
          rounded-lg px-4 py-2.5 text-base
          text-gray-900 dark:text-white
          placeholder:text-gray-500
          focus:border-sky-400 focus:ring-0
          disabled:opacity-50 disabled:cursor-not-allowed
          ${error || (validationError && value) ? 'border-red-400 bg-red-50 dark:bg-red-900/20' : ''}
        `}
      />
      
      <Description className="text-sm text-gray-600 dark:text-gray-400 mt-1">
        Enter your phone number to receive a verification code
      </Description>
      
      {validationError && value && (
        <ErrorMessage className="text-sm text-red-600 dark:text-red-400 mt-1">
          {validationError}
        </ErrorMessage>
      )}
      
      {error && typeof error === 'string' && (
        <ErrorMessage className="text-sm text-red-600 dark:text-red-400 mt-1">
          {error}
        </ErrorMessage>
      )}
    </Field>
  )
}