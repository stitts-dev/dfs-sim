import React, { useState, useRef, useEffect, useCallback } from 'react'
import { Field, Label, Description, ErrorMessage, Button } from '@/catalyst'
import { OTPInputProps } from '@/types/auth'
import clsx from 'clsx'

export const OTPInput: React.FC<OTPInputProps> = ({
  value,
  onChange,
  length = 6,
  disabled = false,
  autoFocus = false,
  onComplete,
  error = false
}) => {
  const inputRefs = useRef<(HTMLInputElement | null)[]>([])
  const [activeIndex, setActiveIndex] = useState(0)

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
    <div className="flex justify-center gap-2 sm:gap-3">
      {Array.from({ length }, (_, index) => (
        <input
          key={index}
          ref={(el) => (inputRefs.current[index] = el)}
          type="text"
          inputMode="numeric"
          pattern="\d*"
          maxLength={1}
          value={value[index] || ''}
          onChange={(e) => handleChange(index, e.target.value)}
          onKeyDown={(e) => handleKeyDown(index, e)}
          onPaste={handlePaste}
          onFocus={() => handleFocus(index)}
          onClick={() => handleClick(index)}
          disabled={disabled}
          className={clsx(
            // Base styles
            'w-12 h-12 sm:w-14 sm:h-14 text-center text-lg sm:text-xl font-semibold',
            'border-2 rounded-lg transition-all duration-200',
            'focus:outline-none focus:ring-2 focus:ring-offset-2',
            
            // Light mode colors
            'bg-white border-zinc-300 text-zinc-900',
            'focus:border-blue-500 focus:ring-blue-500',
            
            // Dark mode colors
            'dark:bg-zinc-800 dark:border-zinc-600 dark:text-zinc-100',
            'dark:focus:border-blue-400 dark:focus:ring-blue-400',
            
            // Error state
            error && 'border-red-500 focus:border-red-500 focus:ring-red-500',
            error && 'dark:border-red-400 dark:focus:border-red-400 dark:focus:ring-red-400',
            
            // Disabled state
            disabled && 'opacity-50 cursor-not-allowed bg-zinc-100 dark:bg-zinc-700',
            
            // Active state
            activeIndex === index && !disabled && 'ring-2 ring-blue-500 dark:ring-blue-400',
            
            // Filled state
            value[index] && 'border-green-500 dark:border-green-400'
          )}
        />
      ))}
    </div>
  )
}

export const OTPVerification: React.FC<{
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
}> = ({
  phoneNumber,
  value,
  onChange,
  onVerify,
  onResend,
  isVerifying = false,
  isResending = false,
  error = null,
  disabled = false,
  autoFocus = true
}) => {
  const [timeLeft, setTimeLeft] = useState(60) // 60 second countdown
  const [canResend, setCanResend] = useState(false)

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
    // Format phone number for display (e.g., +1234567890 -> +1 (234) 567-890)
    if (phone.startsWith('+1') && phone.length === 12) {
      return `+1 (${phone.slice(2, 5)}) ${phone.slice(5, 8)}-${phone.slice(8)}`
    }
    return phone
  }

  return (
    <Field className="space-y-6">
      <div className="text-center">
        <Label className="text-lg font-semibold">Enter Verification Code</Label>
        <Description className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          We sent a 6-digit code to {formatPhoneForDisplay(phoneNumber)}
        </Description>
      </div>

      <div className="space-y-4">
        <OTPInput
          value={value}
          onChange={onChange}
          length={6}
          disabled={disabled || isVerifying}
          autoFocus={autoFocus}
          onComplete={handleComplete}
          error={!!error}
        />

        {error && (
          <ErrorMessage className="text-center text-sm text-red-600 dark:text-red-400">
            {error}
          </ErrorMessage>
        )}
      </div>

      <div className="flex flex-col items-center space-y-4">
        <Button
          onClick={() => handleComplete(value)}
          disabled={value.length !== 6 || isVerifying || disabled}
          className="w-full sm:w-auto px-8"
        >
          {isVerifying ? 'Verifying...' : 'Verify Code'}
        </Button>

        <div className="text-center">
          {canResend ? (
            <button
              onClick={handleResend}
              disabled={isResending}
              className="text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300 font-medium text-sm transition-colors"
            >
              {isResending ? 'Sending...' : 'Resend code'}
            </button>
          ) : (
            <p className="text-sm text-zinc-500 dark:text-zinc-400">
              Resend code in {timeLeft}s
            </p>
          )}
        </div>
      </div>
    </Field>
  )
}

// Standalone OTP verification modal/dialog
export const OTPDialog: React.FC<{
  isOpen: boolean
  onClose: () => void
  phoneNumber: string
  onVerify: (code: string) => Promise<void>
  onResend: () => Promise<void>
  isVerifying?: boolean
  isResending?: boolean
  error?: string | null
}> = ({
  isOpen,
  onClose,
  phoneNumber,
  onVerify,
  onResend,
  isVerifying = false,
  isResending = false,
  error = null
}) => {
  const [code, setCode] = useState('')

  // Reset code when dialog opens
  useEffect(() => {
    if (isOpen) {
      setCode('')
    }
  }, [isOpen])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50">
      <div className="bg-white dark:bg-zinc-800 rounded-lg p-6 w-full max-w-md">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
            Verify Phone Number
          </h2>
          <button
            onClick={onClose}
            className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
          >
            ✕
          </button>
        </div>

        <OTPVerification
          phoneNumber={phoneNumber}
          value={code}
          onChange={setCode}
          onVerify={onVerify}
          onResend={onResend}
          isVerifying={isVerifying}
          isResending={isResending}
          error={error}
          autoFocus={true}
        />
      </div>
    </div>
  )
}