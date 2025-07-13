import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SignupFlow, LoginFlow, AuthFlow } from '../'
import '@testing-library/jest-dom'

// Mock the auth hook and store
const mockUsePhoneAuth = {
  sendOTP: jest.fn(),
  verifyCode: jest.fn(),
  resendCode: jest.fn(),
  isSendingOTP: false,
  isVerifyingOTP: false,
  isResendingOTP: false,
  error: null,
  clearError: jest.fn(),
  normalizePhoneNumber: (phone: string) => phone.startsWith('+') ? phone : '+1' + phone
}

const mockUseAuthStore = {
  user: null,
  currentPhoneNumber: null,
  otpSent: false
}

jest.mock('@/hooks/usePhoneAuth', () => ({
  usePhoneAuth: () => mockUsePhoneAuth
}))

jest.mock('@/store/auth', () => ({
  useAuthStore: () => mockUseAuthStore
}))

// Mock Catalyst components to avoid complex rendering
jest.mock('@/catalyst', () => ({
  Dialog: ({ children, open }: any) => open ? <div data-testid="dialog">{children}</div> : null,
  DialogTitle: ({ children }: any) => <h1>{children}</h1>,
  DialogBody: ({ children }: any) => <div>{children}</div>,
  DialogActions: ({ children }: any) => <div>{children}</div>,
  Button: ({ children, onClick, disabled, ...props }: any) => (
    <button onClick={onClick} disabled={disabled} {...props}>
      {children}
    </button>
  ),
  Field: ({ children }: any) => <div>{children}</div>,
  Label: ({ children }: any) => <label>{children}</label>,
  Input: (props: any) => <input {...props} />,
  Description: ({ children }: any) => <div>{children}</div>,
  ErrorMessage: ({ children }: any) => <div className="error">{children}</div>
}))

// Mock phone input and OTP components
jest.mock('../PhoneInput', () => ({
  PhoneInput: ({ value, onChange, onValidate, error }: any) => (
    <div>
      <label>Phone Number</label>
      <input
        data-testid="phone-input"
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        onBlur={() => onValidate?.(value.length >= 10)}
      />
      {error && <div className="error">{error}</div>}
    </div>
  )
}))

jest.mock('../OTPVerification', () => ({
  OTPVerification: ({ phoneNumber, value, onChange, onVerify, onResend, error }: any) => (
    <div>
      <div>Verify {phoneNumber}</div>
      <input
        data-testid="otp-input"
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
      />
      <button onClick={() => onVerify?.(value)}>Verify Code</button>
      <button onClick={onResend}>Resend Code</button>
      {error && <div className="error">{error}</div>}
    </div>
  )
}))

const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: { retry: false },
    mutations: { retry: false }
  }
})

const renderWithQueryClient = (component: React.ReactElement) => {
  const queryClient = createTestQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      {component}
    </QueryClientProvider>
  )
}

describe('SignupFlow', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
    onComplete: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
    mockUseAuthStore.user = null
    mockUseAuthStore.otpSent = false
    mockUseAuthStore.currentPhoneNumber = null
    mockUsePhoneAuth.error = null
  })

  it('renders phone step initially', () => {
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    expect(screen.getByText('Create Your Account')).toBeInTheDocument()
    expect(screen.getByTestId('phone-input')).toBeInTheDocument()
    expect(screen.getByText('Send Verification Code')).toBeInTheDocument()
  })

  it('advances to verification step when OTP sent', () => {
    mockUseAuthStore.otpSent = true
    mockUseAuthStore.currentPhoneNumber = '+1234567890'
    
    renderWithQueryClient(<SignupFlow {...defaultProps} initialStep="verification" />)
    
    expect(screen.getByText('Verify Your Phone')).toBeInTheDocument()
    expect(screen.getByTestId('otp-input')).toBeInTheDocument()
  })

  it('shows completion step when user is created', () => {
    mockUseAuthStore.user = {
      id: 1,
      phone_number: '+1234567890',
      phone_verified: true,
      subscription_tier: 'free'
    } as any
    
    renderWithQueryClient(<SignupFlow {...defaultProps} initialStep="complete" />)
    
    expect(screen.getByText('Welcome to DFS Optimizer!')).toBeInTheDocument()
    expect(screen.getByText('Your account has been created successfully')).toBeInTheDocument()
  })

  it('sends OTP when valid phone number submitted', async () => {
    const user = userEvent.setup()
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    const phoneInput = screen.getByTestId('phone-input')
    const sendButton = screen.getByText('Send Verification Code')
    
    await user.type(phoneInput, '1234567890')
    await user.click(sendButton)
    
    expect(mockUsePhoneAuth.sendOTP).toHaveBeenCalledWith('+11234567890')
  })

  it('disables send button for invalid phone number', async () => {
    const user = userEvent.setup()
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    const phoneInput = screen.getByTestId('phone-input')
    const sendButton = screen.getByText('Send Verification Code')
    
    await user.type(phoneInput, '123')
    
    expect(sendButton).toBeDisabled()
  })

  it('verifies OTP when code submitted', async () => {
    const user = userEvent.setup()
    
    mockUseAuthStore.currentPhoneNumber = '+1234567890'
    
    renderWithQueryClient(<SignupFlow {...defaultProps} initialStep="verification" />)
    
    const otpInput = screen.getByTestId('otp-input')
    const verifyButton = screen.getByText('Verify Code')
    
    await user.type(otpInput, '123456')
    await user.click(verifyButton)
    
    expect(mockUsePhoneAuth.verifyCode).toHaveBeenCalledWith('+1234567890', '123456')
  })

  it('allows going back from verification step', async () => {
    const user = userEvent.setup()
    
    renderWithQueryClient(<SignupFlow {...defaultProps} initialStep="verification" />)
    
    const backButton = screen.getByText('← Change phone number')
    await user.click(backButton)
    
    await waitFor(() => {
      expect(screen.getByText('Create Your Account')).toBeInTheDocument()
    })
  })

  it('shows error messages', () => {
    mockUsePhoneAuth.error = 'Phone number already registered'
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    expect(screen.getByText('Phone number already registered')).toBeInTheDocument()
  })

  it('calls onComplete when signup successful', () => {
    const mockOnComplete = jest.fn()
    const testUser = { id: 1, phone_number: '+1234567890' }
    
    mockUseAuthStore.user = testUser as any
    
    renderWithQueryClient(<SignupFlow {...defaultProps} onComplete={mockOnComplete} />)
    
    // Should call onComplete with user
    expect(mockOnComplete).toHaveBeenCalledWith(testUser)
  })
})

describe('LoginFlow', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
    onComplete: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
    mockUseAuthStore.user = null
    mockUseAuthStore.otpSent = false
    mockUseAuthStore.currentPhoneNumber = null
    mockUsePhoneAuth.error = null
  })

  it('renders login form initially', () => {
    renderWithQueryClient(<LoginFlow {...defaultProps} />)
    
    expect(screen.getByText('Welcome Back')).toBeInTheDocument()
    expect(screen.getByText('Enter your phone number to sign in to your account')).toBeInTheDocument()
    expect(screen.getByTestId('phone-input')).toBeInTheDocument()
  })

  it('shows signup link when onSwitchToSignup provided', () => {
    const mockSwitchToSignup = jest.fn()
    
    renderWithQueryClient(
      <LoginFlow {...defaultProps} onSwitchToSignup={mockSwitchToSignup} />
    )
    
    expect(screen.getByText("Don't have an account?")).toBeInTheDocument()
    expect(screen.getByText('Sign up')).toBeInTheDocument()
  })

  it('calls onSwitchToSignup when signup link clicked', async () => {
    const user = userEvent.setup()
    const mockSwitchToSignup = jest.fn()
    
    renderWithQueryClient(
      <LoginFlow {...defaultProps} onSwitchToSignup={mockSwitchToSignup} />
    )
    
    const signupLink = screen.getByText('Sign up')
    await user.click(signupLink)
    
    expect(mockSwitchToSignup).toHaveBeenCalled()
  })

  it('advances to verification after sending OTP', () => {
    mockUseAuthStore.otpSent = true
    mockUseAuthStore.currentPhoneNumber = '+1234567890'
    
    renderWithQueryClient(<LoginFlow {...defaultProps} initialStep="verification" />)
    
    expect(screen.getByText('Verify Your Identity')).toBeInTheDocument()
    expect(screen.getByText('Enter the verification code to sign in')).toBeInTheDocument()
  })

  it('shows completion message when login successful', () => {
    mockUseAuthStore.user = {
      id: 1,
      phone_number: '+1234567890',
      phone_verified: true
    } as any
    
    renderWithQueryClient(<LoginFlow {...defaultProps} initialStep="complete" />)
    
    expect(screen.getByText('Welcome Back!')).toBeInTheDocument()
    expect(screen.getByText("You've been signed in successfully")).toBeInTheDocument()
  })
})

describe('AuthFlow', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
    onComplete: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders login flow by default', () => {
    renderWithQueryClient(<AuthFlow {...defaultProps} />)
    
    expect(screen.getByText('Welcome Back')).toBeInTheDocument()
  })

  it('renders signup flow when initialMode is signup', () => {
    renderWithQueryClient(<AuthFlow {...defaultProps} initialMode="signup" />)
    
    expect(screen.getByText('Already have an account?')).toBeInTheDocument()
    expect(screen.getByText('Sign in')).toBeInTheDocument()
  })

  it('switches between login and signup modes', async () => {
    const user = userEvent.setup()
    
    renderWithQueryClient(<AuthFlow {...defaultProps} initialMode="signup" />)
    
    // Should show signup mode
    expect(screen.getByText('Sign in')).toBeInTheDocument()
    
    // Click sign in link
    const signinLink = screen.getByText('Sign in')
    await user.click(signinLink)
    
    // Should switch to login mode
    await waitFor(() => {
      expect(screen.getByText('Welcome Back')).toBeInTheDocument()
    })
  })
})

describe('Authentication Error Handling', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
    onComplete: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
    mockUsePhoneAuth.error = null
  })

  it('displays network errors', () => {
    mockUsePhoneAuth.error = 'Network error occurred'
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    expect(screen.getByText('Network error occurred')).toBeInTheDocument()
  })

  it('displays validation errors', () => {
    mockUsePhoneAuth.error = 'Invalid phone number'
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    expect(screen.getByText('Invalid phone number')).toBeInTheDocument()
  })

  it('displays rate limiting errors', () => {
    mockUsePhoneAuth.error = 'Too many SMS requests, please try again later'
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    expect(screen.getByText('Too many SMS requests, please try again later')).toBeInTheDocument()
  })

  it('clears errors when starting new flow', async () => {
    const user = userEvent.setup()
    
    mockUsePhoneAuth.error = 'Previous error'
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    expect(screen.getByText('Previous error')).toBeInTheDocument()
    
    // Clear error should be called when component mounts or user takes action
    expect(mockUsePhoneAuth.clearError).toHaveBeenCalled()
  })
})

describe('Loading States', () => {
  const defaultProps = {
    isOpen: true,
    onClose: jest.fn(),
    onComplete: jest.fn()
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('shows loading state when sending OTP', () => {
    mockUsePhoneAuth.isSendingOTP = true
    
    renderWithQueryClient(<SignupFlow {...defaultProps} />)
    
    expect(screen.getByText('Sending Code...')).toBeInTheDocument()
    expect(screen.getByText('Sending Code...')).toBeDisabled()
  })

  it('shows loading state when verifying OTP', () => {
    mockUsePhoneAuth.isVerifyingOTP = true
    
    renderWithQueryClient(<SignupFlow {...defaultProps} initialStep="verification" />)
    
    expect(screen.getByText('Verifying...')).toBeInTheDocument()
  })

  it('shows loading state when resending OTP', () => {
    mockUsePhoneAuth.isResendingOTP = true
    
    renderWithQueryClient(<SignupFlow {...defaultProps} initialStep="verification" />)
    
    expect(screen.getByText('Sending...')).toBeInTheDocument()
  })
})