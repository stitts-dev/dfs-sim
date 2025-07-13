package services

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// TwilioSMSService implements SMSService using Twilio API
type TwilioSMSService struct {
	client         *twilio.RestClient
	fromNumber     string
	logger         *log.Logger
	circuitBreaker CircuitBreaker
	rateLimiter    RateLimiter
}

// CircuitBreaker interface for handling external service failures
type CircuitBreaker interface {
	State() string
	RecordSuccess()
	RecordFailure()
	Allow() bool
}

// RateLimiter interface for SMS rate limiting
type RateLimiter interface {
	Allow(phoneNumber string) error
}

// Simple in-memory circuit breaker implementation
type simpleCircuitBreaker struct {
	failures    int
	lastFailure time.Time
	threshold   int
	timeout     time.Duration
	state       string // "closed", "open", "half-open"
}

func newSimpleCircuitBreaker(threshold int, timeout time.Duration) *simpleCircuitBreaker {
	return &simpleCircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     "closed",
	}
}

func (cb *simpleCircuitBreaker) State() string {
	// Check if we should transition from open to half-open
	if cb.state == "open" && time.Since(cb.lastFailure) > cb.timeout {
		cb.state = "half-open"
	}
	return cb.state
}

func (cb *simpleCircuitBreaker) Allow() bool {
	return cb.State() != "open"
}

func (cb *simpleCircuitBreaker) RecordSuccess() {
	cb.failures = 0
	cb.state = "closed"
}

func (cb *simpleCircuitBreaker) RecordFailure() {
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

// NewTwilioSMSService creates a new Twilio SMS service
func NewTwilioSMSService(accountSID, authToken, fromNumber string, rateLimiter RateLimiter) *TwilioSMSService {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	return &TwilioSMSService{
		client:         client,
		fromNumber:     fromNumber,
		logger:         log.Default(),
		circuitBreaker: newSimpleCircuitBreaker(5, 30*time.Second), // 5 failures, 30s timeout
		rateLimiter:    rateLimiter,
	}
}

// SendOTP sends an OTP code via Twilio SMS
func (s *TwilioSMSService) SendOTP(phoneNumber, code string) error {
	message := fmt.Sprintf("Your DFS Optimizer verification code is: %s\n\nDon't share this code with anyone. Code expires in 10 minutes.", code)
	return s.SendMessage(phoneNumber, message)
}

// SendMessage sends an SMS message via Twilio
func (s *TwilioSMSService) SendMessage(phoneNumber, message string) error {
	// Check circuit breaker
	if !s.circuitBreaker.Allow() {
		s.logger.Printf("❌ Twilio SMS: Circuit breaker is open, rejecting request")
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
			s.logger.Printf("⚠️ Twilio SMS: Rate limited for %s", normalizedNumber)
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Prepare Twilio API request
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(normalizedNumber)
	params.SetFrom(s.fromNumber)
	params.SetBody(message)

	s.logger.Printf("📨 Twilio SMS: Sending to %s", normalizedNumber)

	// Make API call with error handling
	resp, err := s.client.Api.CreateMessage(params)
	if err != nil {
		s.circuitBreaker.RecordFailure()
		s.logger.Printf("❌ Twilio SMS: API error - %v", err)
		return s.mapTwilioError(err)
	}

	// Record success
	s.circuitBreaker.RecordSuccess()
	
	if resp.Sid != nil {
		s.logger.Printf("✅ Twilio SMS: Message sent successfully (SID: %s)", *resp.Sid)
	} else {
		s.logger.Printf("✅ Twilio SMS: Message sent successfully")
	}

	return nil
}

// normalizePhoneNumber ensures phone number is in E.164 format
func (s *TwilioSMSService) normalizePhoneNumber(phone string) (string, error) {
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

// mapTwilioError maps Twilio-specific errors to user-friendly messages
func (s *TwilioSMSService) mapTwilioError(err error) error {
	errStr := err.Error()
	
	// Common Twilio error patterns
	switch {
	case regexp.MustCompile(`(?i)invalid.*phone.*number`).MatchString(errStr):
		return fmt.Errorf("invalid phone number")
	case regexp.MustCompile(`(?i)unverified.*number`).MatchString(errStr):
		return fmt.Errorf("phone number not verified for trial account")
	case regexp.MustCompile(`(?i)insufficient.*funds`).MatchString(errStr):
		return fmt.Errorf("SMS service temporarily unavailable")
	case regexp.MustCompile(`(?i)rate.*limit`).MatchString(errStr):
		return fmt.Errorf("too many SMS requests, please try again later")
	case regexp.MustCompile(`(?i)blocked.*number`).MatchString(errStr):
		return fmt.Errorf("unable to send SMS to this number")
	default:
		return fmt.Errorf("failed to send SMS: %w", err)
	}
}

// GetStats returns circuit breaker and service statistics
func (s *TwilioSMSService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"circuit_breaker_state": s.circuitBreaker.State(),
		"service_type":          "twilio",
	}
}