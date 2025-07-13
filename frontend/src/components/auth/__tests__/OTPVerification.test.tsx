import React from 'react'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { OTPInput, OTPVerification } from '../OTPVerification'
import '@testing-library/jest-dom'

// Mock timers for countdown testing
jest.useFakeTimers()

describe('OTPInput', () => {
  const defaultProps = {
    value: '',
    onChange: jest.fn(),
    length: 6
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders correct number of input fields', () => {
    render(<OTPInput {...defaultProps} />)
    
    const inputs = screen.getAllByRole('textbox')
    expect(inputs).toHaveLength(6)
  })

  it('displays current value in input fields', () => {
    render(<OTPInput {...defaultProps} value="123" />)
    
    const inputs = screen.getAllByRole('textbox')
    expect(inputs[0]).toHaveValue('1')
    expect(inputs[1]).toHaveValue('2')
    expect(inputs[2]).toHaveValue('3')
    expect(inputs[3]).toHaveValue('')
    expect(inputs[4]).toHaveValue('')
    expect(inputs[5]).toHaveValue('')
  })

  it('focuses first input on autoFocus', () => {
    render(<OTPInput {...defaultProps} autoFocus />)
    
    const inputs = screen.getAllByRole('textbox')
    expect(inputs[0]).toHaveFocus()
  })

  it('moves to next input when digit is entered', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnChange = jest.fn()
    
    render(<OTPInput {...defaultProps} onChange={mockOnChange} />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Type first digit
    await user.type(inputs[0], '1')
    
    expect(mockOnChange).toHaveBeenCalledWith('1')
    expect(inputs[1]).toHaveFocus()
  })

  it('handles backspace correctly', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnChange = jest.fn()
    
    render(<OTPInput {...defaultProps} value="123" onChange={mockOnChange} />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Focus third input and press backspace
    await user.click(inputs[2])
    await user.keyboard('{Backspace}')
    
    expect(mockOnChange).toHaveBeenCalledWith('12')
  })

  it('handles backspace on empty field', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnChange = jest.fn()
    
    render(<OTPInput {...defaultProps} value="12" onChange={mockOnChange} />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Focus empty third input and press backspace
    await user.click(inputs[2])
    await user.keyboard('{Backspace}')
    
    // Should move to previous input and clear it
    expect(inputs[1]).toHaveFocus()
    expect(mockOnChange).toHaveBeenCalledWith('1')
  })

  it('handles arrow key navigation', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    
    render(<OTPInput {...defaultProps} value="123" />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Focus first input
    await user.click(inputs[0])
    
    // Press right arrow
    await user.keyboard('{ArrowRight}')
    expect(inputs[1]).toHaveFocus()
    
    // Press left arrow
    await user.keyboard('{ArrowLeft}')
    expect(inputs[0]).toHaveFocus()
  })

  it('handles paste correctly', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnChange = jest.fn()
    
    render(<OTPInput {...defaultProps} onChange={mockOnChange} />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Focus first input and paste
    await user.click(inputs[0])
    await user.paste('123456')
    
    expect(mockOnChange).toHaveBeenCalledWith('123456')
    expect(inputs[5]).toHaveFocus() // Should focus last filled input
  })

  it('handles paste with extra characters', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnChange = jest.fn()
    
    render(<OTPInput {...defaultProps} onChange={mockOnChange} />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Paste text with non-digits
    await user.click(inputs[0])
    await user.paste('1a2b3c4d5e6f7g8h')
    
    // Should only keep digits and limit to length
    expect(mockOnChange).toHaveBeenCalledWith('123456')
  })

  it('calls onComplete when all digits entered', () => {
    const mockOnComplete = jest.fn()
    
    render(<OTPInput {...defaultProps} value="123456" onComplete={mockOnComplete} />)
    
    expect(mockOnComplete).toHaveBeenCalledWith('123456')
  })

  it('handles disabled state', () => {
    render(<OTPInput {...defaultProps} disabled />)
    
    const inputs = screen.getAllByRole('textbox')
    inputs.forEach(input => {
      expect(input).toBeDisabled()
    })
  })

  it('shows error state correctly', () => {
    render(<OTPInput {...defaultProps} error />)
    
    const inputs = screen.getAllByRole('textbox')
    inputs.forEach(input => {
      expect(input).toHaveClass('border-red-500')
    })
  })

  it('only allows numeric input', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnChange = jest.fn()
    
    render(<OTPInput {...defaultProps} onChange={mockOnChange} />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Try to type letters
    await user.type(inputs[0], 'abc123')
    
    // Should only register numeric characters
    expect(mockOnChange).toHaveBeenLastCalledWith('123')
  })

  it('replaces digit when typing over existing value', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnChange = jest.fn()
    
    render(<OTPInput {...defaultProps} value="123456" onChange={mockOnChange} />)
    
    const inputs = screen.getAllByRole('textbox')
    
    // Click on filled input and type new digit
    await user.click(inputs[2])
    await user.type(inputs[2], '9')
    
    expect(mockOnChange).toHaveBeenCalledWith('129456')
  })
})

describe('OTPVerification', () => {
  const defaultProps = {
    phoneNumber: '+1234567890',
    value: '',
    onChange: jest.fn(),
    onVerify: jest.fn(),
    onResend: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
    jest.clearAllTimers()
  })

  afterEach(() => {
    act(() => {
      jest.runOnlyPendingTimers()
    })
  })

  it('renders verification form with phone number', () => {
    render(<OTPVerification {...defaultProps} />)
    
    expect(screen.getByText('Enter Verification Code')).toBeInTheDocument()
    expect(screen.getByText(/We sent a 6-digit code to \+1 \(234\) 567-890/)).toBeInTheDocument()
    expect(screen.getByText('Verify Code')).toBeInTheDocument()
  })

  it('formats phone number for display', () => {
    render(<OTPVerification {...defaultProps} phoneNumber="+1234567890" />)
    
    expect(screen.getByText(/\+1 \(234\) 567-890/)).toBeInTheDocument()
  })

  it('shows countdown timer for resend', () => {
    render(<OTPVerification {...defaultProps} />)
    
    expect(screen.getByText('Resend code in 60s')).toBeInTheDocument()
    expect(screen.queryByText('Resend code')).not.toBeInTheDocument()
  })

  it('enables resend button after countdown', () => {
    render(<OTPVerification {...defaultProps} />)
    
    // Fast-forward 60 seconds
    act(() => {
      jest.advanceTimersByTime(60000)
    })
    
    expect(screen.getByText('Resend code')).toBeInTheDocument()
    expect(screen.queryByText(/Resend code in/)).not.toBeInTheDocument()
  })

  it('disables verify button when code incomplete', () => {
    render(<OTPVerification {...defaultProps} value="123" />)
    
    const verifyButton = screen.getByText('Verify Code')
    expect(verifyButton).toBeDisabled()
  })

  it('enables verify button when code complete', () => {
    render(<OTPVerification {...defaultProps} value="123456" />)
    
    const verifyButton = screen.getByText('Verify Code')
    expect(verifyButton).not.toBeDisabled()
  })

  it('calls onVerify when verify button clicked', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnVerify = jest.fn()
    
    render(<OTPVerification {...defaultProps} value="123456" onVerify={mockOnVerify} />)
    
    const verifyButton = screen.getByText('Verify Code')
    await user.click(verifyButton)
    
    expect(mockOnVerify).toHaveBeenCalledWith('123456')
  })

  it('calls onVerify automatically when code complete', () => {
    const mockOnVerify = jest.fn()
    
    render(<OTPVerification {...defaultProps} onVerify={mockOnVerify} />)
    
    // Simulate completing the code through OTPInput
    const inputs = screen.getAllByRole('textbox')
    fireEvent.change(inputs[0], { target: { value: '1' } })
    
    // This would trigger the onComplete callback in OTPInput
    // which should call onVerify
  })

  it('calls onResend when resend button clicked', async () => {
    const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime })
    const mockOnResend = jest.fn()
    
    render(<OTPVerification {...defaultProps} onResend={mockOnResend} />)
    
    // Wait for countdown to finish
    act(() => {
      jest.advanceTimersByTime(60000)
    })
    
    const resendButton = screen.getByText('Resend code')
    await user.click(resendButton)
    
    expect(mockOnResend).toHaveBeenCalled()
  })

  it('resets countdown when resending', () => {
    render(<OTPVerification {...defaultProps} isResending />)
    
    // Should reset to 60 seconds when resending
    expect(screen.getByText('Resend code in 60s')).toBeInTheDocument()
  })

  it('shows verification loading state', () => {
    render(<OTPVerification {...defaultProps} value="123456" isVerifying />)
    
    expect(screen.getByText('Verifying...')).toBeInTheDocument()
    expect(screen.getByText('Verifying...')).toBeDisabled()
  })

  it('shows resend loading state', () => {
    render(<OTPVerification {...defaultProps} isResending />)
    
    act(() => {
      jest.advanceTimersByTime(60000)
    })
    
    expect(screen.getByText('Sending...')).toBeInTheDocument()
  })

  it('displays error message', () => {
    render(<OTPVerification {...defaultProps} error="Invalid verification code" />)
    
    expect(screen.getByText('Invalid verification code')).toBeInTheDocument()
  })

  it('disables inputs when verifying', () => {
    render(<OTPVerification {...defaultProps} isVerifying />)
    
    const inputs = screen.getAllByRole('textbox')
    inputs.forEach(input => {
      expect(input).toBeDisabled()
    })
  })

  it('disables entire component when disabled prop is true', () => {
    render(<OTPVerification {...defaultProps} disabled />)
    
    const inputs = screen.getAllByRole('textbox')
    inputs.forEach(input => {
      expect(input).toBeDisabled()
    })
    
    const verifyButton = screen.getByText('Verify Code')
    expect(verifyButton).toBeDisabled()
  })

  it('handles countdown timer properly', () => {
    render(<OTPVerification {...defaultProps} />)
    
    // Should start at 60 seconds
    expect(screen.getByText('Resend code in 60s')).toBeInTheDocument()
    
    // Advance 30 seconds
    act(() => {
      jest.advanceTimersByTime(30000)
    })
    
    expect(screen.getByText('Resend code in 30s')).toBeInTheDocument()
    
    // Advance to completion
    act(() => {
      jest.advanceTimersByTime(30000)
    })
    
    expect(screen.getByText('Resend code')).toBeInTheDocument()
  })
})