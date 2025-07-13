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

// TwilioSMSService is now implemented in twilio_sms.go
// This removes the stub implementation

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

// SupabaseSMSService is now implemented in supabase_sms.go
// This removes the stub implementation