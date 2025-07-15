package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SMSService interface for sending SMS messages
type SMSService interface {
	SendOTP(phoneNumber, code string) error
}

// MockSMSService is a mock implementation for development/testing
type MockSMSService struct{}

// SendOTP implements SMSService for mock service
func (m *MockSMSService) SendOTP(phoneNumber, code string) error {
	// Log the OTP code for development/testing
	fmt.Printf("📱 SMS Mock: Sending OTP %s to %s\n", code, phoneNumber)
	
	// In development, we can output the code to logs
	// In production, this would actually send via Supabase/Twilio
	return nil
}

// SupabaseSMSService implements SMS sending via Supabase
type SupabaseSMSService struct {
	serviceKey string
	projectURL string
	rateLimiter *SMSRateLimiter
}

// NewSupabaseSMSService creates a new Supabase SMS service
func NewSupabaseSMSService(serviceKey, projectURL string, rateLimiter *SMSRateLimiter) *SupabaseSMSService {
	return &SupabaseSMSService{
		serviceKey:  serviceKey,
		projectURL:  projectURL,
		rateLimiter: rateLimiter,
	}
}

// SendOTP implements SMSService for Supabase
func (s *SupabaseSMSService) SendOTP(phoneNumber, code string) error {
	// Check rate limiting
	if s.rateLimiter != nil {
		if !s.rateLimiter.Allow(phoneNumber) {
			return fmt.Errorf("rate limit exceeded for phone number")
		}
	}

	// Supabase Auth OTP API payload (simpler approach)
	payload := map[string]interface{}{
		"phone": phoneNumber,
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SMS payload: %w", err)
	}
	
	// Make API request to Supabase
	url := fmt.Sprintf("%s/auth/v1/otp", s.projectURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create SMS request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.serviceKey))
	req.Header.Set("apikey", s.serviceKey)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned status %d", resp.StatusCode)
	}
	
	fmt.Printf("📱 Supabase SMS: Successfully sent OTP to %s\n", phoneNumber)
	return nil
}

// TwilioSMSService implements SMS sending via Twilio
type TwilioSMSService struct {
	accountSID  string
	authToken   string
	fromNumber  string
	rateLimiter *SMSRateLimiter
}

// NewTwilioSMSService creates a new Twilio SMS service
func NewTwilioSMSService(accountSID, authToken, fromNumber string, rateLimiter *SMSRateLimiter) *TwilioSMSService {
	return &TwilioSMSService{
		accountSID:  accountSID,
		authToken:   authToken,
		fromNumber:  fromNumber,
		rateLimiter: rateLimiter,
	}
}

// SendOTP implements SMSService for Twilio
func (t *TwilioSMSService) SendOTP(phoneNumber, code string) error {
	// Check rate limiting
	if t.rateLimiter != nil {
		if !t.rateLimiter.Allow(phoneNumber) {
			return fmt.Errorf("rate limit exceeded for phone number")
		}
	}

	// TODO: Implement actual Twilio API call
	// For now, just log like mock service
	fmt.Printf("📱 Twilio SMS: Would send OTP %s to %s\n", code, phoneNumber)
	
	return nil
}

// SMSRateLimiter provides rate limiting for SMS sending
type SMSRateLimiter struct {
	maxRequests int
	window      time.Duration
	requests    map[string][]time.Time
}

// NewSMSRateLimiter creates a new SMS rate limiter
func NewSMSRateLimiter(maxRequests int, window time.Duration) *SMSRateLimiter {
	return &SMSRateLimiter{
		maxRequests: maxRequests,
		window:      window,
		requests:    make(map[string][]time.Time),
	}
}

// Allow checks if a request is allowed for the given phone number
func (r *SMSRateLimiter) Allow(phoneNumber string) bool {
	now := time.Now()
	
	// Clean up old requests
	if requests, exists := r.requests[phoneNumber]; exists {
		var validRequests []time.Time
		for _, req := range requests {
			if now.Sub(req) < r.window {
				validRequests = append(validRequests, req)
			}
		}
		r.requests[phoneNumber] = validRequests
	}
	
	// Check if we can add another request
	currentRequests := len(r.requests[phoneNumber])
	if currentRequests >= r.maxRequests {
		return false
	}
	
	// Add the new request
	r.requests[phoneNumber] = append(r.requests[phoneNumber], now)
	return true
}