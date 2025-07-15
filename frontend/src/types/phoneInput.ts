/**
 * TypeScript type definitions for modern phone input components
 */

export interface PhoneInputProps {
  value: string                           // E.164 format
  onChange: (value: string) => void
  onValidate?: (isValid: boolean) => void
  placeholder?: string
  disabled?: boolean
  autoFocus?: boolean
  className?: string
  error?: string | boolean
  label?: string
  description?: string
  required?: boolean
  showCountrySelector?: boolean
  defaultCountryCode?: string
}

export interface CountryData {
  code: string        // ISO 3166-1 alpha-2 code
  name: string        // Country name
  dialCode: string    // International dial code
  format: string      // National format pattern
  priority: number    // Display priority
  flag: string        // Emoji flag
}

export interface AutofillPattern {
  regex: RegExp
  parser: (match: string) => { countryCode: string; number: string } | null
  confidence: number
  description: string
}

export interface ParsedPhoneNumber {
  countryCode: string
  number: string
  confidence: number
  source: string
}

export interface PhoneValidationResult {
  isValid: boolean
  formattedValue: string
  e164Value: string
  countryCode: string
  nationalNumber: string
  errorMessage?: string
}

export interface PhoneFormatResult {
  formatted: string
  e164: string
  isValid: boolean
  country: string
  nationalNumber: string
  errorMessage?: string
}

export interface AutofillProcessResult {
  isAutofill: boolean
  isValid: boolean
  e164: string
  formatted: string
  confidence: number
  source: string
  errorMessage?: string
}

export interface AccessibilityConfig {
  label?: string
  description?: string
  error?: string
  required?: boolean
  invalid?: boolean
  describedBy?: string
  expandable?: boolean
  hasPopup?: boolean
}

export interface AriaAttributes {
  'aria-label'?: string
  'aria-describedby'?: string
  'aria-required'?: boolean
  'aria-invalid'?: boolean
  'aria-errormessage'?: string
  'aria-expanded'?: boolean
  'aria-haspopup'?: boolean
  'role'?: string
}

export interface PhoneInputState {
  value: string
  displayValue: string
  countryCode: string
  isValid: boolean
  error: string | null
  isFocused: boolean
  isAutofilled: boolean
}

export interface CountrySelectorProps {
  value: string
  onChange: (code: string) => void
  disabled?: boolean
  countries?: CountryData[]
  searchable?: boolean
  placeholder?: string
  className?: string
}

export interface PhoneInputCoreProps extends PhoneInputProps {
  countryCode: string
  onCountryCodeChange: (code: string) => void
  onAutofillDetected?: (result: AutofillProcessResult) => void
}

export interface ModernPhoneInputProps extends PhoneInputProps {
  // Additional props specific to modern implementation
  enableAutofillDetection?: boolean
  autofillConfidenceThreshold?: number
  performanceMode?: boolean
  debugMode?: boolean
}

// Event types
export interface PhoneInputChangeEvent {
  value: string
  e164: string
  formatted: string
  isValid: boolean
  countryCode: string
  source: 'manual' | 'autofill' | 'paste'
}

export interface PhoneInputFocusEvent {
  focused: boolean
  element: HTMLInputElement
}

export interface PhoneInputValidationEvent {
  isValid: boolean
  errorMessage?: string
  value: string
  e164: string
}

// Utility types
export type PhoneInputRef = HTMLInputElement

export type CountryCode = string // ISO 3166-1 alpha-2

export type DialCode = string // International dial code with +

export type E164Format = string // E.164 international format

export type NationalFormat = string // National format with formatting

// Configuration types
export interface PhoneInputConfig {
  defaultCountry: CountryCode
  enableAutofill: boolean
  autofillThreshold: number
  maxDigits: number
  allowedCountries?: CountryCode[]
  priorityCountries?: CountryCode[]
  enableFormatting: boolean
  enableValidation: boolean
  debounceMs: number
}

// Error types
export interface PhoneInputError {
  code: string
  message: string
  field?: string
  value?: string
}

export type PhoneInputErrorCode = 
  | 'INVALID_FORMAT'
  | 'INVALID_COUNTRY'
  | 'TOO_SHORT'
  | 'TOO_LONG'
  | 'INVALID_CHARACTERS'
  | 'AUTOFILL_FAILED'
  | 'VALIDATION_FAILED'

// Component state types
export interface PhoneInputComponentState {
  // Input state
  inputValue: string
  displayValue: string
  
  // Validation state
  isValid: boolean
  validationError: string | null
  
  // Country state
  selectedCountry: CountryCode
  detectedCountry: CountryCode | null
  
  // UI state
  isFocused: boolean
  isDropdownOpen: boolean
  
  // Autofill state
  isAutofilled: boolean
  autofillConfidence: number
  autofillSource: string
  
  // Error state
  error: PhoneInputError | null
}

// Hook types
export interface UsePhoneInputReturn {
  state: PhoneInputComponentState
  actions: {
    setValue: (value: string) => void
    setCountry: (country: CountryCode) => void
    setFocus: (focused: boolean) => void
    clearError: () => void
    reset: () => void
    validate: () => boolean
  }
  refs: {
    inputRef: React.RefObject<HTMLInputElement>
    countryRef: React.RefObject<HTMLSelectElement>
  }
  helpers: {
    formatValue: (value: string) => string
    validateValue: (value: string) => PhoneValidationResult
    normalizeValue: (value: string) => string
    getCountryData: (code: CountryCode) => CountryData | null
  }
}

// Testing types
export interface PhoneInputTestUtils {
  simulateInput: (value: string) => void
  simulateAutofill: (pattern: string) => void
  simulateCountryChange: (country: CountryCode) => void
  simulateFocus: () => void
  simulateBlur: () => void
  getDisplayValue: () => string
  getE164Value: () => string
  getValidationState: () => boolean
  getErrorMessage: () => string | null
}