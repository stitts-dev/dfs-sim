name: "Modern Phone Input Component with 2025 Standards"
description: |
  Advanced autofill detection, international formatting, and accessibility compliance for phone number input with proper handling of browser autofill suggestions.

---

## Goal

Create a production-ready phone input component that handles modern browser autofill patterns, provides comprehensive international phone number support, and meets WCAG 2.2 AA accessibility standards while maintaining seamless integration with existing Supabase phone authentication flow.

## Why

- **Business Impact**: Phone authentication is the primary entry point for the DFS platform - poor UX directly impacts user conversion
- **User Experience**: Current implementation has inconsistent autofill handling, leading to failed registrations and user frustration
- **Accessibility Compliance**: WCAG 2.2 requirements mandate proper autocomplete attributes and mobile optimization
- **Modern Standards**: Browser autofill patterns have evolved significantly, requiring robust pattern detection and validation

## What

A comprehensive phone input component that:
- Handles all major browser autofill patterns (Chrome, Safari, Firefox, Edge)
- Provides international phone number formatting using libphonenumber-js
- Meets WCAG 2.2 AA accessibility requirements
- Maintains E.164 format compatibility with Supabase phone auth
- Preserves existing visual design (glass morphism styling)
- Optimizes mobile keyboard experience with proper input modes

### Success Criteria

- [ ] 99%+ success rate for browser autofill detection across major browsers
- [ ] Support for 50+ international country codes with proper formatting
- [ ] WCAG 2.2 AA accessibility compliance verification
- [ ] Mobile keyboard optimization with proper input modes
- [ ] Zero breaking changes to existing auth flow
- [ ] Comprehensive test coverage (>95%) for all input scenarios
- [ ] Performance impact <100ms for input handling

## All Needed Context

### Documentation & References

```yaml
# MUST READ - Include these in your context window
- url: https://github.com/catamphetamine/libphonenumber-js
  why: Primary library for international phone number parsing and formatting
  critical: Metadata sets (min/max/mobile) affect bundle size - use 'min' for production
  
- url: https://catamphetamine.gitlab.io/react-phone-number-input/
  why: Reference implementation patterns for React integration
  critical: Shows proper E.164 format handling and country code management

- url: https://supabase.com/docs/guides/auth/phone-login
  why: E.164 format requirements for phone authentication
  critical: Must maintain exact format compatibility for signInWithOtp

- url: https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Attributes/autocomplete
  why: Modern autocomplete attribute standards for phone inputs
  critical: Use autocomplete="tel" for full international numbers

- url: https://www.w3.org/TR/WCAG22/
  why: WCAG 2.2 requirements for form inputs and mobile accessibility
  critical: Target size minimum 24x24 CSS pixels, proper ARIA labels

- file: /Users/jstittsworth/fun/frontend/src/components/auth/EnhancedPhoneInput.tsx
  why: Current implementation patterns and glass morphism styling
  critical: Preserve existing visual design and GlassInput wrapper

- file: /Users/jstittsworth/fun/frontend/src/services/supabase.ts
  why: Existing phone validation utilities and E.164 normalization
  critical: validatePhoneNumber, normalizePhoneNumber, formatPhoneNumber functions

- file: /Users/jstittsworth/fun/frontend/src/components/auth/__tests__/PhoneInput.test.tsx
  why: Testing patterns and user interaction simulation
  critical: Mock service dependencies, use @testing-library/user-event

- file: /Users/jstittsworth/fun/frontend/src/catalyst/fieldset.tsx
  why: Catalyst UI Kit component patterns for form inputs
  critical: Field, Label, Input, Description, ErrorMessage components
```

### Current Codebase Structure

```bash
frontend/
├── src/
│   ├── components/
│   │   └── auth/
│   │       ├── EnhancedPhoneInput.tsx    # Current advanced implementation
│   │       ├── PhoneInput.tsx           # Basic Catalyst implementation
│   │       └── __tests__/
│   │           └── PhoneInput.test.tsx  # Test patterns
│   ├── services/
│   │   └── supabase.ts                  # Phone validation utilities
│   ├── hooks/
│   │   └── usePhoneAuth.ts              # React Query integration
│   ├── store/
│   │   └── auth.ts                      # Zustand auth store
│   ├── catalyst/
│   │   └── fieldset.tsx                 # UI Kit components
│   └── types/
│       └── auth.ts                      # Type definitions
```

### Desired Codebase Structure

```bash
frontend/
├── src/
│   ├── components/
│   │   └── auth/
│   │       ├── ModernPhoneInput.tsx           # NEW: Main component
│   │       ├── CountryCodeSelector.tsx        # NEW: Enhanced country selector
│   │       ├── PhoneInputCore.tsx            # NEW: Core input logic
│   │       ├── __tests__/
│   │       │   ├── ModernPhoneInput.test.tsx  # NEW: Comprehensive tests
│   │       │   └── autofill-patterns.test.tsx # NEW: Autofill scenarios
│   │       └── utils/
│   │           ├── autofillDetection.ts       # NEW: Pattern detection
│   │           ├── phoneFormatting.ts         # NEW: International formatting
│   │           └── accessibilityUtils.ts      # NEW: A11y helpers
│   ├── services/
│   │   └── supabase.ts                        # Enhanced with new patterns
│   └── types/
│       └── phoneInput.ts                      # NEW: Component types
```

### Known Gotchas & Library Quirks

```typescript
// CRITICAL: libphonenumber-js bundle size impact
// Use 'min' metadata for production: ~80KB vs 'max' at 145KB
import { parsePhoneNumber } from 'libphonenumber-js/min'

// CRITICAL: Supabase requires exact E.164 format
// Example: "+1234567890" (plus sign + country code + number)
// NOT: "1234567890" or "(123) 456-7890"

// CRITICAL: Browser autofill patterns vary significantly
// Chrome: "+1 (412) 527-4078"
// Safari: "+14125274078" 
// Firefox: "412-527-4078"
// Edge: "+1 412 527 4078"

// CRITICAL: React state sync issues
// Display value (formatted) vs actual value (E.164) must be separate
// Use controlled component pattern with proper state management

// CRITICAL: Mobile keyboard optimization
// inputMode="tel" triggers numeric keyboard
// autoComplete="tel" enables proper autofill
// type="tel" provides semantic meaning

// CRITICAL: Accessibility requirements
// ARIA labels required for screen readers
// Target size minimum 24x24 CSS pixels (WCAG 2.2)
// Focus management for keyboard navigation
```

## Implementation Blueprint

### Data Models and Structure

```typescript
// Core types for phone input component
interface PhoneInputProps {
  value: string;                    // E.164 format
  onChange: (value: string) => void;
  onValidate?: (isValid: boolean) => void;
  placeholder?: string;
  disabled?: boolean;
  autoFocus?: boolean;
  className?: string;
  error?: string;
  label?: string;
  required?: boolean;
}

interface CountryData {
  code: string;        // ISO 3166-1 alpha-2 code
  name: string;        // Country name
  dialCode: string;    // International dial code
  format: string;      // National format pattern
  priority: number;    // Display priority
}

interface AutofillPattern {
  regex: RegExp;
  parser: (match: string) => { countryCode: string; number: string };
  confidence: number;
}

interface PhoneValidationResult {
  isValid: boolean;
  formattedValue: string;
  e164Value: string;
  countryCode: string;
  nationalNumber: string;
  errorMessage?: string;
}
```

### Task List (Implementation Order)

```yaml
Task 1: Install and Configure Dependencies
MODIFY frontend/package.json:
  - ADD: "libphonenumber-js": "^1.10.51"
  - ENSURE: React 18+ for concurrent features

Task 2: Create Core Utilities
CREATE src/components/auth/utils/autofillDetection.ts:
  - IMPLEMENT: Comprehensive regex patterns for all major browsers
  - PATTERN: Follow existing validation pattern from services/supabase.ts
  - FUNCTION: detectAutofillPattern(), parseAutofillValue()

CREATE src/components/auth/utils/phoneFormatting.ts:
  - IMPLEMENT: International phone formatting using libphonenumber-js/min
  - PATTERN: Mirror existing formatPhoneNumber in services/supabase.ts
  - FUNCTION: formatPhoneNumber(), validatePhoneNumber(), normalizeToE164()

CREATE src/components/auth/utils/accessibilityUtils.ts:
  - IMPLEMENT: ARIA attribute helpers and focus management
  - PATTERN: Follow Catalyst UI Kit accessibility patterns
  - FUNCTION: generateAriaAttributes(), manageFocusState()

Task 3: Enhanced Country Code Selector
CREATE src/components/auth/CountryCodeSelector.tsx:
  - MIRROR: Existing CountrySelector from EnhancedPhoneInput.tsx
  - ENHANCE: Add search functionality and flag display
  - PRESERVE: Glass morphism styling from existing implementation
  - INTEGRATE: libphonenumber-js country metadata

Task 4: Core Phone Input Component
CREATE src/components/auth/PhoneInputCore.tsx:
  - IMPLEMENT: Core input logic with autofill detection
  - PATTERN: Follow controlled component pattern from existing inputs
  - INTEGRATE: All utility functions from Task 2
  - ACCESSIBILITY: Proper ARIA labels and mobile optimization

Task 5: Main Component Assembly
CREATE src/components/auth/ModernPhoneInput.tsx:
  - ASSEMBLE: CountryCodeSelector + PhoneInputCore + validation
  - PRESERVE: Glass morphism styling from EnhancedPhoneInput.tsx
  - INTEGRATE: Existing GlassInput wrapper component
  - MAINTAIN: Same prop interface as existing PhoneInput

Task 6: Service Integration
MODIFY src/services/supabase.ts:
  - ENHANCE: validatePhoneNumber to handle new autofill patterns
  - PRESERVE: Existing E.164 normalization logic
  - ADD: Support for new international number formats
  - MAINTAIN: Backward compatibility with existing code

Task 7: Type Definitions
CREATE src/types/phoneInput.ts:
  - DEFINE: All TypeScript interfaces from data models
  - MIRROR: Existing auth types structure
  - EXPORT: Common types for component reuse

Task 8: Comprehensive Testing
CREATE src/components/auth/__tests__/ModernPhoneInput.test.tsx:
  - MIRROR: Testing patterns from existing PhoneInput.test.tsx
  - TEST: All autofill scenarios with user-event simulation
  - VALIDATE: Accessibility compliance with screen reader testing
  - MOCK: Service dependencies following existing patterns

CREATE src/components/auth/__tests__/autofill-patterns.test.tsx:
  - TEST: All browser autofill patterns with realistic data
  - VALIDATE: E.164 format conversion accuracy
  - SIMULATE: Paste events and keyboard input
  - EDGE_CASES: International numbers and malformed inputs

Task 9: Integration and Replacement
MODIFY existing components to use ModernPhoneInput:
  - UPDATE: Auth forms to import new component
  - PRESERVE: Existing prop interfaces for compatibility
  - VALIDATE: All existing functionality works unchanged
  - MONITOR: Performance impact and bundle size

Task 10: Performance Optimization
OPTIMIZE component for production:
  - LAZY_LOAD: Country metadata to reduce initial bundle
  - DEBOUNCE: Validation calls to prevent excessive API calls
  - MEMOIZE: Expensive formatting operations
  - MEASURE: Performance impact and optimization opportunities
```

### Implementation Pseudocode

```typescript
// Task 2: Core autofill detection utility
function detectAutofillPattern(value: string): AutofillPattern | null {
  // PATTERN: Match existing validation approach in services/supabase.ts
  const patterns: AutofillPattern[] = [
    // Chrome: "+1 (412) 527-4078"
    { 
      regex: /^\+(\d{1,3})\s?\((\d{3})\)\s?(\d{3})-(\d{4})$/,
      parser: (match) => parseChromeBracketFormat(match),
      confidence: 0.9
    },
    // Safari: "+14125274078"
    {
      regex: /^\+(\d{1,3})(\d{10})$/,
      parser: (match) => parseSafariCompactFormat(match),
      confidence: 0.8
    },
    // Firefox: "412-527-4078"
    {
      regex: /^(\d{3})-(\d{3})-(\d{4})$/,
      parser: (match) => parseFirefoxDashFormat(match),
      confidence: 0.7
    }
  ];
  
  // CRITICAL: Return highest confidence match
  return patterns
    .filter(p => p.regex.test(value))
    .sort((a, b) => b.confidence - a.confidence)[0] || null;
}

// Task 4: Core phone input with autofill handling
function PhoneInputCore({ value, onChange, onValidate }: PhoneInputProps) {
  // PATTERN: Follow controlled component pattern from existing inputs
  const [displayValue, setDisplayValue] = useState('');
  const [countryCode, setCountryCode] = useState('US');
  
  // CRITICAL: Separate display value from actual E.164 value
  const handleInputChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    const inputValue = e.target.value;
    
    // PATTERN: Detect autofill like existing paste handling
    const autofillPattern = detectAutofillPattern(inputValue);
    
    if (autofillPattern) {
      // CRITICAL: Parse autofill and convert to E.164
      const parsed = autofillPattern.parser(inputValue);
      const e164Value = normalizeToE164(parsed.number, parsed.countryCode);
      
      // PATTERN: Follow existing validation approach
      const validation = validatePhoneNumber(e164Value);
      
      if (validation.isValid) {
        setDisplayValue(validation.formattedValue);
        onChange(validation.e164Value);
        onValidate?.(true);
      }
    } else {
      // PATTERN: Regular typing follows existing formatting logic
      const formatted = formatPhoneNumber(inputValue, countryCode);
      setDisplayValue(formatted);
      
      // CRITICAL: Always validate E.164 format for Supabase
      const e164 = normalizeToE164(inputValue, countryCode);
      const isValid = validatePhoneNumber(e164).isValid;
      
      onChange(e164);
      onValidate?.(isValid);
    }
  }, [countryCode, onChange, onValidate]);
  
  // ACCESSIBILITY: Proper ARIA attributes for WCAG 2.2
  const ariaAttributes = generateAriaAttributes({
    label: 'Phone number',
    required: true,
    invalid: !isValid,
    describedBy: error ? 'phone-error' : undefined
  });
  
  return (
    <input
      type="tel"
      inputMode="tel"
      autoComplete="tel"
      value={displayValue}
      onChange={handleInputChange}
      {...ariaAttributes}
      className="modern-phone-input"
    />
  );
}

// Task 5: Main component assembly
function ModernPhoneInput(props: PhoneInputProps) {
  // PRESERVE: Glass morphism styling from EnhancedPhoneInput.tsx
  return (
    <GlassInput className="modern-phone-input-container">
      <Field>
        <Label>{props.label}</Label>
        <div className="phone-input-wrapper">
          <CountryCodeSelector 
            value={countryCode}
            onChange={setCountryCode}
          />
          <PhoneInputCore 
            {...props}
            countryCode={countryCode}
          />
        </div>
        {props.error && <ErrorMessage>{props.error}</ErrorMessage>}
      </Field>
    </GlassInput>
  );
}
```

### Integration Points

```yaml
DEPENDENCIES:
  - add: "libphonenumber-js": "^1.10.51"
  - ensure: React 18+ for concurrent features
  - bundle: Use 'min' metadata set to minimize impact

SERVICES:
  - modify: services/supabase.ts
  - enhance: validatePhoneNumber, normalizePhoneNumber, formatPhoneNumber
  - preserve: E.164 format compatibility

COMPONENTS:
  - replace: EnhancedPhoneInput.tsx usage with ModernPhoneInput.tsx
  - preserve: All existing prop interfaces
  - maintain: Glass morphism styling

TESTING:
  - add: Comprehensive test suite following existing patterns
  - mock: Service dependencies with realistic implementations
  - validate: Accessibility compliance with screen reader testing

TYPES:
  - create: src/types/phoneInput.ts
  - export: Common interfaces for component reuse
  - maintain: Backward compatibility with existing auth types
```

## Validation Loop

### Level 1: Syntax & Style

```bash
# Run these FIRST - fix any errors before proceeding
npm run lint                                    # ESLint with --fix
npm run type-check                             # TypeScript compilation
npm run test:unit -- --testNamePattern="ModernPhoneInput"  # Basic functionality

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests

```typescript
// CREATE ModernPhoneInput.test.tsx with comprehensive test cases:
describe('ModernPhoneInput', () => {
  test('handles Chrome autofill pattern', async () => {
    const mockOnChange = jest.fn();
    render(<ModernPhoneInput onChange={mockOnChange} />);
    
    const input = screen.getByRole('textbox');
    await userEvent.type(input, '+1 (412) 527-4078');
    
    expect(mockOnChange).toHaveBeenCalledWith('+14125274078');
  });
  
  test('handles Safari autofill pattern', async () => {
    const mockOnChange = jest.fn();
    render(<ModernPhoneInput onChange={mockOnChange} />);
    
    const input = screen.getByRole('textbox');
    await userEvent.type(input, '+14125274078');
    
    expect(mockOnChange).toHaveBeenCalledWith('+14125274078');
  });
  
  test('validates international numbers', async () => {
    const mockOnValidate = jest.fn();
    render(<ModernPhoneInput onValidate={mockOnValidate} />);
    
    const input = screen.getByRole('textbox');
    await userEvent.type(input, '+44 20 7946 0958');
    
    expect(mockOnValidate).toHaveBeenCalledWith(true);
  });
  
  test('meets accessibility requirements', () => {
    render(<ModernPhoneInput label="Phone number" required />);
    
    const input = screen.getByLabelText('Phone number');
    expect(input).toHaveAttribute('type', 'tel');
    expect(input).toHaveAttribute('inputMode', 'tel');
    expect(input).toHaveAttribute('autoComplete', 'tel');
    expect(input).toHaveAttribute('aria-required', 'true');
  });
});
```

```bash
# Run and iterate until passing:
npm run test:unit -- --testNamePattern="ModernPhoneInput" --verbose
# If failing: Read error, understand root cause, fix code, re-run
```

### Level 3: Integration Test

```bash
# Start the development server
npm run dev

# Test the component in browser
# 1. Open http://localhost:5173/auth/register
# 2. Try different autofill patterns:
#    - Chrome: "+1 (412) 527-4078"
#    - Safari: "+14125274078"
#    - Firefox: "412-527-4078"
#    - Edge: "+1 412 527 4078"
# 3. Verify E.164 format in network tab
# 4. Test screen reader with VoiceOver/NVDA
# 5. Test mobile keyboard on actual device

# Expected: All patterns convert to proper E.164 format
# Expected: Accessibility tools announce field purpose correctly
```

## Final Validation Checklist

- [ ] All unit tests pass: `npm run test:unit`
- [ ] No linting errors: `npm run lint`
- [ ] No type errors: `npm run type-check`
- [ ] Manual browser testing with all autofill patterns successful
- [ ] Accessibility validation with screen reader passes
- [ ] Mobile keyboard optimization verified on actual devices
- [ ] Performance impact measured and under 100ms
- [ ] Bundle size impact acceptable (under 100KB added)
- [ ] E.164 format compatibility with Supabase verified
- [ ] Zero breaking changes to existing auth flow confirmed

---

## Anti-Patterns to Avoid

- ❌ Don't assume all browsers implement autofill identically
- ❌ Don't mix display formatting with E.164 value storage
- ❌ Don't skip accessibility testing with actual screen readers
- ❌ Don't ignore mobile keyboard optimization
- ❌ Don't hardcode country codes or phone patterns
- ❌ Don't break existing Supabase phone auth compatibility
- ❌ Don't increase bundle size unnecessarily (use libphonenumber-js/min)
- ❌ Don't implement custom validation when libraries handle it better

## Implementation Confidence Score: 9/10

**High Confidence Factors:**
- Comprehensive codebase analysis revealing clear patterns to follow
- Detailed external research on all required libraries and standards
- Existing test patterns provide clear validation approach
- Clear integration points with current auth flow
- Specific pseudocode addressing all critical implementation details

**Risk Mitigation:**
- Autofill patterns may vary with browser updates (2% risk)
- International phone number edge cases (5% risk)
- Mobile keyboard behavior differences (3% risk)
- Performance impact of libphonenumber-js (1% risk)

**Success Likelihood:** 95% - This PRP provides sufficient context and implementation guidance for one-pass success with Claude Code.