import React, { useState, useRef, useEffect, useCallback } from 'react'
import { OTPInputProps } from '@/types/auth'
import { SparkleIcon } from '@/components/ui/SparkleIcon'
import { Button } from '@/components/ui/Button'
import clsx from 'clsx'

interface GlassOTPInputProps {
  index: number
  value: string
  onChange: (value: string) => void
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void
  onPaste: (e: React.ClipboardEvent<HTMLInputElement>) => void
  onFocus: () => void
  onClick: () => void
  disabled: boolean
  isActive: boolean
  hasError: boolean
  isSuccess: boolean
}

function GlassOTPInput({
  index,
  value,
  onChange,
  onKeyDown,
  onPaste,
  onFocus,
  onClick,
  disabled,
  isActive,
  hasError,
  isSuccess
}: GlassOTPInputProps) {
  return (
    <div className="relative">
      {/* Enhanced OTP input with better prominence and contrast */}
      <div className={`rounded-xl border-2 bg-white dark:bg-gray-800 transition-all duration-200 shadow-sm ${
        hasError 
          ? 'border-red-400 bg-red-50 dark:bg-red-900/20 shadow-red-100 dark:shadow-red-900/30' 
          : isSuccess
          ? 'border-green-400 bg-green-50 dark:bg-green-900/20 shadow-green-100 dark:shadow-green-900/30'
          : isActive
          ? 'border-sky-400 bg-sky-50 dark:bg-sky-900/20 shadow-sky-100 dark:shadow-sky-900/30'
          : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
      }`}>
        {/* Success indicator */}
        {isSuccess && value && (
          <div className="absolute -top-1 -right-1 z-10">
            <SparkleIcon className="w-4 h-4 text-green-500" animated />
          </div>
        )}
        
        <input
          type="text"
          inputMode="numeric"
          pattern="\d*"
          maxLength={1}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={onKeyDown}
          onPaste={onPaste}
          onFocus={onFocus}
          onClick={onClick}
          disabled={disabled}
          className={clsx(
            'w-12 h-12 sm:w-14 sm:h-14 md:w-14 md:h-14 text-center text-lg sm:text-xl md:text-xl font-bold',
            'bg-transparent text-gray-900 dark:text-white placeholder:text-gray-400',
            'focus:outline-none transition-all duration-200',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'rounded-xl'
          )}
          aria-label={`Digit ${index + 1}`}
        />
      </div>
    </div>
  )
}

export interface EnhancedOTPInputProps extends OTPInputProps {
  showSuccess?: boolean
}

export const EnhancedOTPInput: React.FC<EnhancedOTPInputProps> = ({
  value,
  onChange,
  length = 6,
  disabled = false,
  autoFocus = false,
  onComplete,
  error = false,
  showSuccess = true
}) => {
  const inputRefs = useRef<(HTMLInputElement | null)[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [completedIndexes, setCompletedIndexes] = useState<Set<number>>(new Set())

  // Initialize refs array
  useEffect(() => {
    inputRefs.current = inputRefs.current.slice(0, length)
  }, [length])

  // Auto-focus first input when component mounts
  useEffect(() => {
    if (autoFocus && inputRefs.current[0]) {
      inputRefs.current[0].focus()
    }
  }, [autoFocus])

  // Handle completion callback
  useEffect(() => {
    if (value.length === length && onComplete) {
      onComplete(value)
    }
  }, [value, length, onComplete])

  // Track completed inputs for success animation
  useEffect(() => {
    const completed = new Set<number>()
    for (let i = 0; i < value.length; i++) {
      if (value[i] && /^\d$/.test(value[i])) {
        completed.add(i)
      }
    }
    setCompletedIndexes(completed)
  }, [value])

  const handleChange = (index: number, digit: string) => {
    // Only allow single digits
    if (digit.length > 1) {
      digit = digit.slice(-1)
    }
    
    // Only allow numbers
    if (digit && !/^\d$/.test(digit)) {
      return
    }

    const newValue = value.split('')
    newValue[index] = digit
    const updatedValue = newValue.join('').slice(0, length)
    
    onChange(updatedValue)

    // Move to next input if digit was entered
    if (digit && index < length - 1) {
      const nextInput = inputRefs.current[index + 1]
      if (nextInput) {
        nextInput.focus()
        setActiveIndex(index + 1)
      }
    }
  }

  const handleKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    // Handle backspace
    if (e.key === 'Backspace') {
      e.preventDefault()
      
      if (value[index]) {
        // Clear current digit
        handleChange(index, '')
      } else if (index > 0) {
        // Move to previous input and clear it
        const prevInput = inputRefs.current[index - 1]
        if (prevInput) {
          prevInput.focus()
          setActiveIndex(index - 1)
          handleChange(index - 1, '')
        }
      }
    }
    
    // Handle arrow keys
    if (e.key === 'ArrowLeft' && index > 0) {
      const prevInput = inputRefs.current[index - 1]
      if (prevInput) {
        prevInput.focus()
        setActiveIndex(index - 1)
      }
    }
    
    if (e.key === 'ArrowRight' && index < length - 1) {
      const nextInput = inputRefs.current[index + 1]
      if (nextInput) {
        nextInput.focus()
        setActiveIndex(index + 1)
      }
    }
  }

  const handlePaste = useCallback((e: React.ClipboardEvent) => {
    e.preventDefault()
    const pastedData = e.clipboardData.getData('text')
    const digits = pastedData.replace(/\D/g, '').slice(0, length)
    
    if (digits) {
      onChange(digits)
      
      // Focus the last filled input or next empty one
      const targetIndex = Math.min(digits.length, length - 1)
      const targetInput = inputRefs.current[targetIndex]
      if (targetInput) {
        targetInput.focus()
        setActiveIndex(targetIndex)
      }
    }
  }, [length, onChange])

  const handleFocus = (index: number) => {
    setActiveIndex(index)
  }

  const handleClick = (index: number) => {
    // Find the first empty slot or click target
    const firstEmptyIndex = value.split('').findIndex(digit => !digit)
    const targetIndex = firstEmptyIndex !== -1 ? Math.min(firstEmptyIndex, index) : index
    
    const targetInput = inputRefs.current[targetIndex]
    if (targetInput) {
      targetInput.focus()
      setActiveIndex(targetIndex)
    }
  }

  return (
    <div className="flex justify-center gap-2 sm:gap-3 md:gap-4">
      {Array.from({ length }, (_, index) => (
        <GlassOTPInput
          key={index}
          index={index}
          value={value[index] || ''}
          onChange={(digit) => handleChange(index, digit)}
          onKeyDown={(e) => handleKeyDown(index, e)}
          onPaste={handlePaste}
          onFocus={() => handleFocus(index)}
          onClick={() => handleClick(index)}
          disabled={disabled}
          isActive={activeIndex === index}
          hasError={!!error}
          isSuccess={showSuccess && completedIndexes.has(index)}
        />
      ))}
    </div>
  )
}

export interface EnhancedOTPVerificationProps {
  phoneNumber: string
  value: string
  onChange: (value: string) => void
  onVerify: (code: string) => Promise<void>
  onResend: () => Promise<void>
  isVerifying?: boolean
  isResending?: boolean
  error?: string | null
  disabled?: boolean
  autoFocus?: boolean
  title?: string
  subtitle?: string
}

export const EnhancedOTPVerification: React.FC<EnhancedOTPVerificationProps> = ({
  phoneNumber,
  value,
  onChange,
  onVerify,
  onResend,
  isVerifying = false,
  isResending = false,
  error = null,
  disabled = false,
  autoFocus = true,
  title = "Enter Verification Code",
  subtitle
}) => {
  const [timeLeft, setTimeLeft] = useState(60) // 60 second countdown
  const [canResend, setCanResend] = useState(false)
  const [isComplete, setIsComplete] = useState(false)

  // Countdown timer for resend
  useEffect(() => {
    if (timeLeft > 0) {
      const timer = setTimeout(() => setTimeLeft(timeLeft - 1), 1000)
      return () => clearTimeout(timer)
    } else {
      setCanResend(true)
    }
  }, [timeLeft])

  // Reset timer when resending
  useEffect(() => {
    if (isResending) {
      setTimeLeft(60)
      setCanResend(false)
    }
  }, [isResending])

  // Track completion state
  useEffect(() => {
    setIsComplete(value.length === 6)
  }, [value])

  const handleComplete = async (code: string) => {
    if (code.length === 6 && !isVerifying) {
      await onVerify(code)
    }
  }

  const handleResend = async () => {
    if (canResend && !isResending) {
      await onResend()
    }
  }

  const formatPhoneForDisplay = (phone: string) => {
    // Format phone number for display (e.g., +1234567890 -> +1 (234) 567-*****)
    if (phone.startsWith('+1') && phone.length === 12) {
      return `+1 (${phone.slice(2, 5)}) ${phone.slice(5, 8)}-****`
    }
    return phone.replace(/(.{6}).*/, '$1****')
  }

  const defaultSubtitle = `We sent a 6-digit code to ${formatPhoneForDisplay(phoneNumber)}`

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center space-y-3">
        <h3 className="text-xl font-semibold text-gray-900 dark:text-white">
          {title}
        </h3>
        <p className="text-sm text-gray-600 dark:text-gray-400 max-w-md mx-auto leading-relaxed">
          {subtitle || defaultSubtitle}
        </p>
      </div>

      {/* OTP Input */}
      <div className="space-y-7">
        <div className="bg-white/50 dark:bg-gray-800/30 rounded-2xl p-6 backdrop-blur-sm">
          <EnhancedOTPInput
            value={value}
            onChange={onChange}
            length={6}
            disabled={disabled || isVerifying}
            autoFocus={autoFocus}
            onComplete={handleComplete}
            error={!!error}
            showSuccess={!error}
          />
        </div>

        {/* Error Message */}
        {error && (
          <div className="text-center">
            <div className="inline-flex items-center px-4 py-2 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <svg className="w-4 h-4 text-red-500 mr-2" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <span className="text-sm text-red-700 dark:text-red-300">{error}</span>
            </div>
          </div>
        )}

        {/* Success Indicator */}
        {isComplete && !error && !isVerifying && (
          <div className="text-center">
            <div className="inline-flex items-center px-4 py-2 rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
              <SparkleIcon className="w-4 h-4 text-green-500 mr-2" animated />
              <span className="text-sm text-green-700 dark:text-green-300">Code complete</span>
            </div>
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="space-y-4">
        <Button
          onClick={() => handleComplete(value)}
          disabled={value.length !== 6 || isVerifying || disabled}
          loading={isVerifying}
          variant="primary"
          size="lg"
          className="w-full mt-2"
        >
          {isVerifying ? 'Verifying Code...' : 'Verify Code'}
        </Button>

        {/* Resend Section */}
        <div className="text-center">
          {canResend ? (
            <button
              onClick={handleResend}
              disabled={isResending}
              className={clsx(
                'inline-flex items-center text-sm font-medium transition-all duration-200',
                'text-sky-600 hover:text-sky-500 dark:text-sky-400 dark:hover:text-sky-300',
                'hover:underline focus:outline-none focus:underline',
                isResending && 'opacity-50 cursor-not-allowed'
              )}
            >
              {isResending ? (
                <>
                  <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                  Sending new code...
                </>
              ) : (
                <>
                  <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  Resend code
                </>
              )}
            </button>
          ) : (
            <div className="flex items-center justify-center text-sm text-gray-500 dark:text-gray-400">
              <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Resend code in {timeLeft}s
            </div>
          )}
        </div>
      </div>
    </div>
  )
}