package services

import (
	"fmt"
	"log"
)

// SMSService interface for sending SMS messages
type SMSService interface {
	SendOTP(phoneNumber, code string) error
	SendMessage(phoneNumber, message string) error
}

// MockSMSService for development - logs to console instead of sending real SMS
type MockSMSService struct{}

func NewMockSMSService() *MockSMSService {
	return &MockSMSService{}
}

func (s *MockSMSService) SendOTP(phoneNumber, code string) error {
	log.Printf("🔐 MOCK SMS: Send OTP '%s' to %s", code, phoneNumber)
	fmt.Printf("\n📱 SMS VERIFICATION CODE: %s\n   Phone: %s\n   Code expires in 10 minutes\n\n", code, phoneNumber)
	return nil
}

func (s *MockSMSService) SendMessage(phoneNumber, message string) error {
	log.Printf("📨 MOCK SMS: Send message to %s: %s", phoneNumber, message)
	return nil
}

// TwilioSMSService for production use with Twilio
type TwilioSMSService struct {
	accountSID string
	authToken  string
	fromNumber string
}

func NewTwilioSMSService(accountSID, authToken, fromNumber string) *TwilioSMSService {
	return &TwilioSMSService{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
	}
}

func (s *TwilioSMSService) SendOTP(phoneNumber, code string) error {
	message := fmt.Sprintf("Your DFS Optimizer verification code is: %s\n\nDon't share this code with anyone. Code expires in 10 minutes.", code)
	return s.SendMessage(phoneNumber, message)
}

func (s *TwilioSMSService) SendMessage(phoneNumber, message string) error {
	// TODO: Implement Twilio API call
	// For now, just log it
	log.Printf("📨 TWILIO SMS: Send to %s: %s", phoneNumber, message)
	return nil
}

// TelnyxSMSService for production use with Telnyx (low-cost alternative)
type TelnyxSMSService struct {
	apiKey     string
	fromNumber string
}

func NewTelnyxSMSService(apiKey, fromNumber string) *TelnyxSMSService {
	return &TelnyxSMSService{
		apiKey:     apiKey,
		fromNumber: fromNumber,
	}
}

func (s *TelnyxSMSService) SendOTP(phoneNumber, code string) error {
	message := fmt.Sprintf("Your DFS Optimizer verification code is: %s\n\nDon't share this code with anyone. Code expires in 10 minutes.", code)
	return s.SendMessage(phoneNumber, message)
}

func (s *TelnyxSMSService) SendMessage(phoneNumber, message string) error {
	// TODO: Implement Telnyx API call
	// For now, just log it
	log.Printf("📨 TELNYX SMS: Send to %s: %s", phoneNumber, message)
	return nil
}

// SupabaseSMSService for use with Supabase Auth (recommended for our stack)
type SupabaseSMSService struct {
	apiKey     string
	projectURL string
}

func NewSupabaseSMSService(apiKey, projectURL string) *SupabaseSMSService {
	return &SupabaseSMSService{
		apiKey:     apiKey,
		projectURL: projectURL,
	}
}

func (s *SupabaseSMSService) SendOTP(phoneNumber, code string) error {
	message := fmt.Sprintf("Your DFS Optimizer verification code is: %s\n\nDon't share this code with anyone. Code expires in 10 minutes.", code)
	return s.SendMessage(phoneNumber, message)
}

func (s *SupabaseSMSService) SendMessage(phoneNumber, message string) error {
	// TODO: Implement Supabase SMS API call
	// For now, just log it
	log.Printf("📨 SUPABASE SMS: Send to %s: %s", phoneNumber, message)
	return nil
}