package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"
)

// SupabaseSMSService implements SMSService using Supabase Auth
type SupabaseSMSService struct {
	apiKey     string
	projectURL string
	httpClient *http.Client
	logger     *log.Logger
	circuitBreaker CircuitBreaker
	rateLimiter    RateLimiter
}

// SupabaseOTPRequest represents the request payload for Supabase OTP
type SupabaseOTPRequest struct {
	Phone string `json:"phone"`
}

// SupabaseOTPResponse represents the response from Supabase OTP endpoint
type SupabaseOTPResponse struct {
	MessageID string `json:"message_id,omitempty"`
	Error     *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// NewSupabaseSMSService creates a new Supabase SMS service
func NewSupabaseSMSService(apiKey, projectURL string, rateLimiter RateLimiter) *SupabaseSMSService {
	return &SupabaseSMSService{
		apiKey:     apiKey,
		projectURL: projectURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:         log.Default(),
		circuitBreaker: newSimpleCircuitBreaker(5, 30*time.Second), // 5 failures, 30s timeout
		rateLimiter:    rateLimiter,
	}
}

// SendOTP sends an OTP code via Supabase Auth
func (s *SupabaseSMSService) SendOTP(phoneNumber, code string) error {
	// Note: With Supabase Auth, we don't pass the code directly
	// Supabase generates and sends the OTP code automatically
	// The 'code' parameter is ignored in this implementation
	
	// Check circuit breaker
	if !s.circuitBreaker.Allow() {
		s.logger.Printf("❌ Supabase SMS: Circuit breaker is open, rejecting request")
		return fmt.Errorf("SMS service temporarily unavailable")
	}

	// Validate phone number format (E.164)
	normalizedNumber, err := s.normalizePhoneNumber(phoneNumber)
	if err != nil {
		return fmt.Errorf("invalid phone number format: %w", err)
	}

	// Check rate limiting
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Allow(normalizedNumber); err != nil {
			s.logger.Printf("⚠️ Supabase SMS: Rate limited for %s", normalizedNumber)
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Prepare request payload
	payload := SupabaseOTPRequest{
		Phone: normalizedNumber,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/auth/v1/otp", s.projectURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.apiKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	s.logger.Printf("📨 Supabase SMS: Sending OTP to %s", normalizedNumber)

	// Make HTTP request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.circuitBreaker.RecordFailure()
		s.logger.Printf("❌ Supabase SMS: HTTP request failed - %v", err)
		return fmt.Errorf("failed to send OTP: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		s.circuitBreaker.RecordFailure()
		s.logger.Printf("❌ Supabase SMS: API error (status %d) - %s", resp.StatusCode, string(body))
		return s.mapSupabaseError(resp.StatusCode, body)
	}

	// Parse response
	var otpResp SupabaseOTPResponse
	if err := json.Unmarshal(body, &otpResp); err != nil {
		s.logger.Printf("⚠️ Supabase SMS: Failed to parse response, but status was OK")
	}

	// Check for error in response
	if otpResp.Error != nil {
		s.circuitBreaker.RecordFailure()
		s.logger.Printf("❌ Supabase SMS: API error - %s", otpResp.Error.Message)
		return fmt.Errorf("SMS service error: %s", otpResp.Error.Message)
	}

	// Record success
	s.circuitBreaker.RecordSuccess()
	s.logger.Printf("✅ Supabase SMS: OTP sent successfully to %s", normalizedNumber)

	return nil
}

// SendMessage sends a custom message (not supported by Supabase Auth directly)
func (s *SupabaseSMSService) SendMessage(phoneNumber, message string) error {
	// Supabase Auth only supports OTP messages, not custom messages
	// This would require using a different Supabase service or custom implementation
	s.logger.Printf("⚠️ Supabase SMS: Custom messages not supported, use SendOTP instead")
	return fmt.Errorf("custom SMS messages not supported by Supabase Auth")
}

// normalizePhoneNumber ensures phone number is in E.164 format
func (s *SupabaseSMSService) normalizePhoneNumber(phone string) (string, error) {
	// Remove all non-digit characters except +
	re := regexp.MustCompile(`[^\d+]`)
	cleaned := re.ReplaceAllString(phone, "")
	
	// Add + if not present
	if !regexp.MustCompile(`^\+`).MatchString(cleaned) {
		// Assume US number if no country code
		if regexp.MustCompile(`^\d{10}$`).MatchString(cleaned) {
			cleaned = "+1" + cleaned
		} else {
			return "", fmt.Errorf("invalid phone number format")
		}
	}
	
	// Validate E.164 format
	if !regexp.MustCompile(`^\+[1-9]\d{1,14}$`).MatchString(cleaned) {
		return "", fmt.Errorf("invalid phone number format")
	}
	
	return cleaned, nil
}

// mapSupabaseError maps Supabase-specific errors to user-friendly messages
func (s *SupabaseSMSService) mapSupabaseError(statusCode int, body []byte) error {
	bodyStr := string(body)
	
	switch statusCode {
	case http.StatusBadRequest:
		if regexp.MustCompile(`(?i)invalid.*phone`).MatchString(bodyStr) {
			return fmt.Errorf("invalid phone number")
		}
		if regexp.MustCompile(`(?i)rate.*limit`).MatchString(bodyStr) {
			return fmt.Errorf("too many SMS requests, please try again later")
		}
		return fmt.Errorf("invalid request: %s", bodyStr)
	case http.StatusUnauthorized:
		return fmt.Errorf("SMS service authentication failed")
	case http.StatusForbidden:
		return fmt.Errorf("SMS service access denied")
	case http.StatusTooManyRequests:
		return fmt.Errorf("too many SMS requests, please try again later")
	case http.StatusInternalServerError:
		return fmt.Errorf("SMS service temporarily unavailable")
	default:
		return fmt.Errorf("SMS service error (status %d): %s", statusCode, bodyStr)
	}
}

// GetStats returns circuit breaker and service statistics
func (s *SupabaseSMSService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"circuit_breaker_state": s.circuitBreaker.State(),
		"service_type":          "supabase",
	}
}