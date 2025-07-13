import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PhoneInput, PhoneInputWithCountryCode } from '../PhoneInput'
import '@testing-library/jest-dom'

// Mock the services to avoid external dependencies
jest.mock('@/services/supabase', () => ({
  formatPhoneNumber: (phone: string) => {
    const cleaned = phone.replace(/\D/g, '')
    if (cleaned.length === 0) return ''
    if (cleaned.length <= 3) return cleaned
    if (cleaned.length <= 6) return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3)}`
    return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6, 10)}`
  },
  validatePhoneNumber: (phone: string) => {
    const cleaned = phone.replace(/\D/g, '')
    return cleaned.length >= 10 && cleaned.length <= 11
  },
  normalizePhoneNumber: (phone: string) => {
    let cleaned = phone.replace(/\D/g, '')
    if (cleaned.length === 10) cleaned = '1' + cleaned
    return '+' + cleaned
  }
}))

describe('PhoneInput', () => {
  const defaultProps = {
    value: '',
    onChange: jest.fn(),
    onValidate: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders phone input field with label and placeholder', () => {
    render(<PhoneInput {...defaultProps} />)
    
    expect(screen.getByText('Phone Number')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('+1 (555) 123-4567')).toBeInTheDocument()
    expect(screen.getByText('Enter your phone number to receive a verification code')).toBeInTheDocument()
  })

  it('formats phone number as user types', async () => {
    const user = userEvent.setup()
    const mockOnChange = jest.fn()
    
    render(<PhoneInput {...defaultProps} onChange={mockOnChange} />)
    
    const input = screen.getByRole('textbox')
    
    // Type a phone number
    await user.type(input, '1234567890')
    
    // Should have called onChange with raw digits
    expect(mockOnChange).toHaveBeenCalledWith('1234567890')
    
    // Input should show formatted value
    await waitFor(() => {
      expect(input).toHaveValue('(123) 456-7890')
    })
  })

  it('validates phone number and calls onValidate', async () => {
    const user = userEvent.setup()
    const mockOnValidate = jest.fn()
    
    render(<PhoneInput {...defaultProps} value="1234567890" onValidate={mockOnValidate} />)
    
    await waitFor(() => {
      expect(mockOnValidate).toHaveBeenCalledWith(true)
    })
  })

  it('shows validation error for invalid phone number', async () => {
    render(<PhoneInput {...defaultProps} value="123" />)
    
    await waitFor(() => {
      expect(screen.getByText('Please enter a valid phone number')).toBeInTheDocument()
    })
  })

  it('shows custom error message', () => {
    render(<PhoneInput {...defaultProps} error="Custom error message" />)
    
    expect(screen.getByText('Custom error message')).toBeInTheDocument()
  })

  it('handles paste events correctly', async () => {
    const user = userEvent.setup()
    const mockOnChange = jest.fn()
    
    render(<PhoneInput {...defaultProps} onChange={mockOnChange} />)
    
    const input = screen.getByRole('textbox')
    
    // Focus input and paste phone number
    await user.click(input)
    await user.paste('(123) 456-7890')
    
    // Should extract only digits
    expect(mockOnChange).toHaveBeenCalledWith('1234567890')
  })

  it('limits input to 11 digits', async () => {
    const user = userEvent.setup()
    const mockOnChange = jest.fn()
    
    render(<PhoneInput {...defaultProps} onChange={mockOnChange} />)
    
    const input = screen.getByRole('textbox')
    
    // Type more than 11 digits
    await user.type(input, '123456789012345')
    
    // Should only keep first 11 digits
    expect(mockOnChange).toHaveBeenLastCalledWith('12345678901')
  })

  it('handles disabled state', () => {
    render(<PhoneInput {...defaultProps} disabled />)
    
    const input = screen.getByRole('textbox')
    expect(input).toBeDisabled()
  })

  it('auto-focuses when autoFocus is true', () => {
    render(<PhoneInput {...defaultProps} autoFocus />)
    
    const input = screen.getByRole('textbox')
    expect(input).toHaveFocus()
  })

  it('removes non-digit characters except during formatting', async () => {
    const user = userEvent.setup()
    const mockOnChange = jest.fn()
    
    render(<PhoneInput {...defaultProps} onChange={mockOnChange} />)
    
    const input = screen.getByRole('textbox')
    
    // Type phone number with extra characters
    await user.type(input, 'abc123def456ghi7890')
    
    // Should only keep digits
    expect(mockOnChange).toHaveBeenLastCalledWith('1234567890')
  })
})

describe('PhoneInputWithCountryCode', () => {
  const defaultProps = {
    value: '',
    onChange: jest.fn(),
    showCountrySelector: true
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('shows country code selector when enabled', () => {
    render(<PhoneInputWithCountryCode {...defaultProps} />)
    
    expect(screen.getByText('Select your country code and enter your phone number')).toBeInTheDocument()
    // Country selector should be present (test for select element)
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('falls back to regular PhoneInput when country selector disabled', () => {
    render(<PhoneInputWithCountryCode {...defaultProps} showCountrySelector={false} />)
    
    expect(screen.getByText('Enter your phone number to receive a verification code')).toBeInTheDocument()
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
  })

  it('handles country code changes', async () => {
    const user = userEvent.setup()
    const mockOnChange = jest.fn()
    
    render(<PhoneInputWithCountryCode {...defaultProps} onChange={mockOnChange} />)
    
    const countrySelect = screen.getByRole('combobox')
    
    // Change country code
    await user.selectOptions(countrySelect, '+44')
    
    // Should update with new country code
    await waitFor(() => {
      expect(mockOnChange).toHaveBeenCalledWith('44')
    })
  })

  it('combines country code and local number', async () => {
    const user = userEvent.setup()
    const mockOnChange = jest.fn()
    
    render(<PhoneInputWithCountryCode {...defaultProps} onChange={mockOnChange} defaultCountryCode="+44" />)
    
    const input = screen.getByRole('textbox')
    
    // Type local number
    await user.type(input, '1234567890')
    
    // Should combine country code with local number
    expect(mockOnChange).toHaveBeenLastCalledWith('441234567890')
  })
})

describe('PhoneInput Edge Cases', () => {
  const defaultProps = {
    value: '',
    onChange: jest.fn(),
    onValidate: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('handles empty value gracefully', () => {
    render(<PhoneInput {...defaultProps} value="" />)
    
    const input = screen.getByRole('textbox')
    expect(input).toHaveValue('')
  })

  it('handles undefined value gracefully', () => {
    // @ts-ignore - Testing runtime behavior
    render(<PhoneInput {...defaultProps} value={undefined} />)
    
    const input = screen.getByRole('textbox')
    expect(input).toHaveValue('')
  })

  it('clears validation error when user starts typing valid number', async () => {
    const user = userEvent.setup()
    
    render(<PhoneInput {...defaultProps} value="123" />)
    
    // Should show validation error initially
    await waitFor(() => {
      expect(screen.getByText('Please enter a valid phone number')).toBeInTheDocument()
    })
    
    const input = screen.getByRole('textbox')
    
    // Type more digits to make it valid
    await user.clear(input)
    await user.type(input, '1234567890')
    
    // Validation error should be gone
    await waitFor(() => {
      expect(screen.queryByText('Please enter a valid phone number')).not.toBeInTheDocument()
    })
  })

  it('shows both validation error and custom error', () => {
    render(<PhoneInput {...defaultProps} value="123" error="API error occurred" />)
    
    expect(screen.getByText('Please enter a valid phone number')).toBeInTheDocument()
    expect(screen.getByText('API error occurred')).toBeInTheDocument()
  })

  it('handles rapid typing without issues', async () => {
    const user = userEvent.setup()
    const mockOnChange = jest.fn()
    
    render(<PhoneInput {...defaultProps} onChange={mockOnChange} />)
    
    const input = screen.getByRole('textbox')
    
    // Rapidly type numbers
    await user.type(input, '1234567890', { delay: 1 })
    
    // Should handle all changes
    expect(mockOnChange).toHaveBeenCalledTimes(10)
    expect(mockOnChange).toHaveBeenLastCalledWith('1234567890')
  })

  it('maintains cursor position during formatting', async () => {
    const user = userEvent.setup()
    
    render(<PhoneInput {...defaultProps} value="123456" />)
    
    const input = screen.getByRole('textbox') as HTMLInputElement
    
    // Input should be formatted
    await waitFor(() => {
      expect(input.value).toBe('(123) 456')
    })
    
    // Cursor should be at the end after formatting
    expect(input.selectionStart).toBe(8)
  })
})