import React, { useState, useEffect } from 'react'
import { Field, Label, Input, Description, ErrorMessage } from '@/catalyst'
import { PhoneInputProps, PhoneValidationResult } from '@/types/auth'
import { formatPhoneNumber, validatePhoneNumber, normalizePhoneNumber } from '@/services/supabase'
import clsx from 'clsx'

export const PhoneInput: React.FC<PhoneInputProps> = ({
  value,
  onChange,
  onValidate,
  disabled = false,
  autoFocus = false,
  error = false,
  className
}) => {
  const [formatted, setFormatted] = useState('')
  const [isValid, setIsValid] = useState(false)
  const [validationError, setValidationError] = useState<string | null>(null)

  // Format phone number as user types
  useEffect(() => {
    if (value) {
      const formattedValue = formatPhoneNumber(value)
      setFormatted(formattedValue)
      
      // Validate phone number
      const validation = validatePhoneInput(value)
      setIsValid(validation.isValid)
      setValidationError(validation.error || null)
      
      // Notify parent of validation result
      onValidate?.(validation.isValid)
    } else {
      setFormatted('')
      setIsValid(false)
      setValidationError(null)
      onValidate?.(false)
    }
  }, [value, onValidate])

  const validatePhoneInput = (phone: string): PhoneValidationResult => {
    if (!phone) {
      return { isValid: false, formatted: '', error: 'Phone number is required' }
    }

    try {
      const normalized = normalizePhoneNumber(phone)
      const isValidNumber = validatePhoneNumber(phone)
      
      if (!isValidNumber) {
        return { 
          isValid: false, 
          formatted: normalized, 
          error: 'Please enter a valid phone number' 
        }
      }

      return { 
        isValid: true, 
        formatted: normalized 
      }
    } catch (error) {
      return { 
        isValid: false, 
        formatted: phone, 
        error: 'Invalid phone number format' 
      }
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const input = e.target.value
    const digitsOnly = input.replace(/\D/g, '')
    
    // Limit to 11 digits (1 + 10 for US numbers)
    const limitedDigits = digitsOnly.slice(0, 11)
    
    // Update parent with raw digits
    onChange?.(limitedDigits)
  }

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault()
    const pastedText = e.clipboardData.getData('text')
    const digitsOnly = pastedText.replace(/\D/g, '')
    const limitedDigits = digitsOnly.slice(0, 11)
    onChange?.(limitedDigits)
  }

  return (
    <Field className={className}>
      <Label>Phone Number</Label>
      <Input
        type="tel"
        value={formatted}
        onChange={handleChange}
        onPaste={handlePaste}
        placeholder="+1 (555) 123-4567"
        disabled={disabled}
        autoFocus={autoFocus}
        autoComplete="tel"
        className={clsx(
          'w-full',
          error || (!isValid && value) ? 'border-red-500 data-invalid:border-red-500' : ''
        )}
        data-invalid={error || (!isValid && value)}
      />
      
      <Description className="text-sm text-zinc-600 dark:text-zinc-400">
        Enter your phone number to receive a verification code
      </Description>
      
      {(validationError && value) && (
        <ErrorMessage className="text-sm text-red-600 dark:text-red-400">
          {validationError}
        </ErrorMessage>
      )}
      
      {error && typeof error === 'string' && (
        <ErrorMessage className="text-sm text-red-600 dark:text-red-400">
          {error}
        </ErrorMessage>
      )}
    </Field>
  )
}

// Country code selector component (optional enhancement)
export const CountryCodeSelector: React.FC<{
  value: string
  onChange: (code: string) => void
  disabled?: boolean
}> = ({ value, onChange, disabled = false }) => {
  const commonCountries = [
    { code: '+1', country: 'US', flag: '🇺🇸' },
    { code: '+1', country: 'CA', flag: '🇨🇦' },
    { code: '+44', country: 'UK', flag: '🇬🇧' },
    { code: '+49', country: 'DE', flag: '🇩🇪' },
    { code: '+33', country: 'FR', flag: '🇫🇷' },
    { code: '+61', country: 'AU', flag: '🇦🇺' },
  ]

  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      className={clsx(
        'absolute left-3 top-1/2 transform -translate-y-1/2',
        'text-sm font-medium text-zinc-900 dark:text-zinc-100',
        'bg-transparent border-none outline-none',
        'disabled:opacity-50'
      )}
    >
      {commonCountries.map((country) => (
        <option key={`${country.code}-${country.country}`} value={country.code}>
          {country.flag} {country.code}
        </option>
      ))}
    </select>
  )
}

// Advanced phone input with country code selector
export const PhoneInputWithCountryCode: React.FC<PhoneInputProps & {
  showCountrySelector?: boolean
  defaultCountryCode?: string
}> = ({
  value,
  onChange,
  onValidate,
  disabled = false,
  autoFocus = false,
  error = false,
  className,
  showCountrySelector = false,
  defaultCountryCode = '+1'
}) => {
  const [countryCode, setCountryCode] = useState(defaultCountryCode)
  const [localNumber, setLocalNumber] = useState('')

  useEffect(() => {
    if (value) {
      // Parse existing value to extract country code and local number
      if (value.startsWith('+')) {
        const normalized = normalizePhoneNumber(value)
        if (normalized.startsWith('+1')) {
          setCountryCode('+1')
          setLocalNumber(normalized.slice(2))
        } else {
          // For other country codes, extract first 1-3 digits after +
          const match = normalized.match(/^\+(\d{1,3})(\d+)$/)
          if (match) {
            setCountryCode(`+${match[1]}`)
            setLocalNumber(match[2])
          }
        }
      } else {
        setLocalNumber(value)
      }
    }
  }, [value])

  const handleLocalNumberChange = (newLocalNumber: string) => {
    setLocalNumber(newLocalNumber)
    const fullNumber = countryCode + newLocalNumber
    onChange?.(fullNumber.replace(/\D/g, ''))
  }

  const handleCountryCodeChange = (newCountryCode: string) => {
    setCountryCode(newCountryCode)
    const fullNumber = newCountryCode + localNumber
    onChange?.(fullNumber.replace(/\D/g, ''))
  }

  if (!showCountrySelector) {
    return (
      <PhoneInput
        value={value}
        onChange={onChange}
        onValidate={onValidate}
        disabled={disabled}
        autoFocus={autoFocus}
        error={error}
        className={className}
      />
    )
  }

  return (
    <Field className={className}>
      <Label>Phone Number</Label>
      <div className="relative">
        <CountryCodeSelector
          value={countryCode}
          onChange={handleCountryCodeChange}
          disabled={disabled}
        />
        <Input
          type="tel"
          value={formatPhoneNumber(localNumber)}
          onChange={(e) => {
            const input = e.target.value.replace(/\D/g, '')
            handleLocalNumberChange(input)
          }}
          placeholder="(555) 123-4567"
          disabled={disabled}
          autoFocus={autoFocus}
          autoComplete="tel"
          className={clsx(
            'pl-16', // Make room for country code selector
            error ? 'border-red-500' : ''
          )}
          data-invalid={error}
        />
      </div>
      
      <Description className="text-sm text-zinc-600 dark:text-zinc-400">
        Select your country code and enter your phone number
      </Description>
      
      {error && typeof error === 'string' && (
        <ErrorMessage className="text-sm text-red-600 dark:text-red-400">
          {error}
        </ErrorMessage>
      )}
    </Field>
  )
}