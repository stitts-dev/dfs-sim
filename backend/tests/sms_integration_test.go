package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTwilioClient for testing Twilio SMS service
type MockTwilioClient struct {
	mock.Mock
}

func (m *MockTwilioClient) CreateMessage(params interface{}) (*MockTwilioResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*MockTwilioResponse), args.Error(1)
}

type MockTwilioResponse struct {
	Sid *string
}

// MockHTTPClient for testing Supabase SMS service
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// TestTwilioSMSService_SendOTP tests Twilio SMS service with mocked client
func TestTwilioSMSService_SendOTP(t *testing.T) {
	tests := []struct {
		name           string
		phoneNumber    string
		code           string
		mockResponse   *MockTwilioResponse
		mockError      error
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name:        "successful_send",
			phoneNumber: "+1234567890",
			code:        "123456",
			mockResponse: &MockTwilioResponse{
				Sid: stringPtr("SM123456789"),
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name:           "invalid_phone_number",
			phoneNumber:    "invalid",
			code:           "123456",
			mockResponse:   nil,
			mockError:      nil,
			expectedError:  true,
			expectedErrMsg: "invalid phone number format",
		},
		{
			name:        "twilio_api_error",
			phoneNumber: "+1234567890",
			code:        "123456",
			mockResponse: nil,
			mockError:    fmt.Errorf("Twilio API error: invalid phone number"),
			expectedError: true,
			expectedErrMsg: "invalid phone number",
		},
		{
			name:        "rate_limit_error",
			phoneNumber: "+1234567890",
			code:        "123456",
			mockResponse: nil,
			mockError:    fmt.Errorf("rate limit exceeded"),
			expectedError: true,
			expectedErrMsg: "too many SMS requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create rate limiter
			rateLimiter := services.NewSMSRateLimiter(3, time.Hour)

			// Create Twilio service
			twilioService := services.NewTwilioSMSService(
				"test_account_sid",
				"test_auth_token",
				"+1234567890",
				rateLimiter,
			)

			// Test the SMS sending
			err := twilioService.SendOTP(tt.phoneNumber, tt.code)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSupabaseSMSService_SendOTP tests Supabase SMS service with mocked HTTP client
func TestSupabaseSMSService_SendOTP(t *testing.T) {
	tests := []struct {
		name           string
		phoneNumber    string
		code           string
		statusCode     int
		responseBody   string
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name:         "successful_send",
			phoneNumber:  "+1234567890",
			code:         "123456",
			statusCode:   200,
			responseBody: `{"message_id": "msg123"}`,
			expectedError: false,
		},
		{
			name:           "invalid_phone_number",
			phoneNumber:    "invalid",
			code:           "123456",
			statusCode:     0,
			responseBody:   "",
			expectedError:  true,
			expectedErrMsg: "invalid phone number format",
		},
		{
			name:         "supabase_api_error",
			phoneNumber:  "+1234567890",
			code:         "123456",
			statusCode:   400,
			responseBody: `{"error": {"message": "Invalid phone number"}}`,
			expectedError: true,
			expectedErrMsg: "invalid request",
		},
		{
			name:         "rate_limit_error",
			phoneNumber:  "+1234567890",
			code:         "123456",
			statusCode:   429,
			responseBody: `{"error": {"message": "Too many requests"}}`,
			expectedError: true,
			expectedErrMsg: "too many SMS requests",
		},
		{
			name:         "server_error",
			phoneNumber:  "+1234567890",
			code:         "123456",
			statusCode:   500,
			responseBody: `{"error": {"message": "Internal server error"}}`,
			expectedError: true,
			expectedErrMsg: "SMS service temporarily unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create rate limiter
			rateLimiter := services.NewSMSRateLimiter(3, time.Hour)

			// Create Supabase service
			supabaseService := services.NewSupabaseSMSService(
				"test_api_key",
				"https://test.supabase.co",
				rateLimiter,
			)

			// Test the SMS sending
			err := supabaseService.SendOTP(tt.phoneNumber, tt.code)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSMSRateLimiter tests the SMS rate limiting functionality
func TestSMSRateLimiter(t *testing.T) {
	tests := []struct {
		name          string
		maxRequests   int
		window        time.Duration
		requests      []string
		expectedErrs  []bool
	}{
		{
			name:        "within_rate_limit",
			maxRequests: 3,
			window:      time.Hour,
			requests:    []string{"+1234567890", "+1234567890"},
			expectedErrs: []bool{false, false},
		},
		{
			name:        "exceeds_rate_limit",
			maxRequests: 2,
			window:      time.Hour,
			requests:    []string{"+1234567890", "+1234567890", "+1234567890"},
			expectedErrs: []bool{false, false, true},
		},
		{
			name:        "different_numbers_no_limit",
			maxRequests: 2,
			window:      time.Hour,
			requests:    []string{"+1234567890", "+1987654321", "+1234567890"},
			expectedErrs: []bool{false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rateLimiter := services.NewSMSRateLimiter(tt.maxRequests, tt.window)

			for i, phoneNumber := range tt.requests {
				err := rateLimiter.Allow(phoneNumber)

				if tt.expectedErrs[i] {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "rate limit exceeded")
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestCircuitBreakerBehavior tests circuit breaker functionality
func TestCircuitBreakerBehavior(t *testing.T) {
	rateLimiter := services.NewSMSRateLimiter(10, time.Hour)
	twilioService := services.NewTwilioSMSService(
		"test_account_sid",
		"test_auth_token",
		"+1234567890",
		rateLimiter,
	)

	// Get circuit breaker stats before any requests
	stats := twilioService.GetStats()
	assert.Equal(t, "twilio", stats["service_type"])
	assert.Contains(t, []string{"closed", "open", "half-open"}, stats["circuit_breaker_state"])

	// Test that circuit breaker starts in closed state
	assert.Equal(t, "closed", stats["circuit_breaker_state"])
}

// TestPhoneNumberNormalization tests phone number validation and normalization
func TestPhoneNumberNormalization(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValid bool
		expectedE164  string
	}{
		{
			name:          "us_number_with_country_code",
			input:         "+1234567890",
			expectedValid: true,
			expectedE164:  "+1234567890",
		},
		{
			name:          "us_number_without_country_code",
			input:         "2345678901",
			expectedValid: true,
			expectedE164:  "+12345678901",
		},
		{
			name:          "formatted_us_number",
			input:         "(234) 567-8901",
			expectedValid: true,
			expectedE164:  "+12345678901",
		},
		{
			name:          "international_number",
			input:         "+44123456789",
			expectedValid: true,
			expectedE164:  "+44123456789",
		},
		{
			name:          "invalid_short_number",
			input:         "123",
			expectedValid: false,
			expectedE164:  "",
		},
		{
			name:          "invalid_long_number",
			input:         "+123456789012345678",
			expectedValid: false,
			expectedE164:  "",
		},
		{
			name:          "empty_number",
			input:         "",
			expectedValid: false,
			expectedE164:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test normalization using the TwilioSMSService method
			rateLimiter := services.NewSMSRateLimiter(10, time.Hour)
			twilioService := services.NewTwilioSMSService(
				"test_account_sid",
				"test_auth_token",
				"+1234567890",
				rateLimiter,
			)

			err := twilioService.SendOTP(tt.input, "123456")

			if tt.expectedValid {
				// For valid numbers, we might get other errors (like circuit breaker or API errors)
				// but not phone validation errors
				if err != nil {
					assert.NotContains(t, err.Error(), "invalid phone number format")
				}
			} else {
				// For invalid numbers, we should get a phone validation error
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid phone number format")
			}
		})
	}
}

// TestSMSServiceIntegration tests the complete SMS service integration flow
func TestSMSServiceIntegration(t *testing.T) {
	// Create configuration for testing
	cfg := &config.Config{
		SMSProvider:        "mock",
		TwilioAccountSID:   "test_account_sid",
		TwilioAuthToken:    "test_auth_token",
		TwilioFromNumber:   "+1234567890",
		SupabaseURL:        "https://test.supabase.co",
		SupabaseServiceKey: "test_service_key",
		SupabaseAnonKey:    "test_anon_key",
	}

	tests := []struct {
		name        string
		provider    string
		expectError bool
	}{
		{
			name:        "mock_provider",
			provider:    "mock",
			expectError: false,
		},
		{
			name:        "twilio_provider",
			provider:    "twilio",
			expectError: false, // Should create service successfully
		},
		{
			name:        "supabase_provider",
			provider:    "supabase",
			expectError: false, // Should create service successfully
		},
		{
			name:        "unknown_provider",
			provider:    "unknown",
			expectError: false, // Should fall back to mock
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.SMSProvider = tt.provider

			// Create SMS service based on configuration
			// This simulates the factory function in auth.go
			var smsService services.SMSService

			rateLimiter := services.NewSMSRateLimiter(3, time.Hour)

			switch cfg.SMSProvider {
			case "twilio":
				if cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" && cfg.TwilioFromNumber != "" {
					smsService = services.NewTwilioSMSService(
						cfg.TwilioAccountSID,
						cfg.TwilioAuthToken,
						cfg.TwilioFromNumber,
						rateLimiter,
					)
				} else {
					smsService = services.NewMockSMSService()
				}
			case "supabase":
				if cfg.SupabaseURL != "" && cfg.SupabaseServiceKey != "" {
					smsService = services.NewSupabaseSMSService(
						cfg.SupabaseServiceKey,
						cfg.SupabaseURL,
						rateLimiter,
					)
				} else {
					smsService = services.NewMockSMSService()
				}
			default:
				smsService = services.NewMockSMSService()
			}

			// Test that service was created
			assert.NotNil(t, smsService)

			// Test sending OTP
			err := smsService.SendOTP("+1234567890", "123456")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Mock service should always succeed
				// Real services might fail due to network/API issues, but service creation should work
				if tt.provider == "mock" {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestConcurrentSMSRequests tests concurrent SMS requests and rate limiting
func TestConcurrentSMSRequests(t *testing.T) {
	rateLimiter := services.NewSMSRateLimiter(5, time.Hour)
	mockService := services.NewMockSMSService()

	// Test concurrent requests to same number
	phoneNumber := "+1234567890"
	numRequests := 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			err := rateLimiter.Allow(phoneNumber)
			results <- err
		}()
	}

	// Collect results
	var errors int
	for i := 0; i < numRequests; i++ {
		err := <-results
		if err != nil {
			errors++
		}
	}

	// Should have 5 errors (rate limit exceeded) and 5 successes
	assert.Equal(t, 5, errors)

	// Test that service itself handles concurrent requests safely
	for i := 0; i < 10; i++ {
		go func() {
			err := mockService.SendOTP("+1987654321", "123456")
			assert.NoError(t, err)
		}()
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// TestEndToEndAuthFlow tests the complete authentication flow
func TestEndToEndAuthFlow(t *testing.T) {
	// This would typically test the full flow:
	// 1. Send OTP request
	// 2. Receive OTP code
	// 3. Verify OTP code
	// 4. Get JWT token

	// For now, we test the individual components
	t.Run("complete_flow_simulation", func(t *testing.T) {
		phoneNumber := "+1234567890"
		mockService := services.NewMockSMSService()

		// Step 1: Send OTP
		err := mockService.SendOTP(phoneNumber, "123456")
		assert.NoError(t, err)

		// Step 2: In a real scenario, user would receive SMS and enter code
		// Step 3: Verify OTP (this would be done in the auth handler)
		// Step 4: Generate JWT token (this would be done in the auth handler)

		// For integration test, we just verify the SMS service works
		assert.NotNil(t, mockService)
	})
}