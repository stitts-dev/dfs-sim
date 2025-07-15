package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SupabaseUserService handles user operations via Supabase API
type SupabaseUserService struct {
	supabaseURL    string
	serviceKey     string
	httpClient     *http.Client
}

// SupabaseUser represents user data from Supabase
type SupabaseUser struct {
	ID               uuid.UUID  `json:"id"`
	PhoneNumber      string     `json:"phone_number"`
	FirstName        *string    `json:"first_name,omitempty"`
	LastName         *string    `json:"last_name,omitempty"`
	SubscriptionTier string     `json:"subscription_tier"`
	SubscriptionStatus string   `json:"subscription_status"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	StripeCustomerID *string    `json:"stripe_customer_id,omitempty"`
	MonthlyOptimizationsUsed int `json:"monthly_optimizations_used"`
	MonthlySimulationsUsed   int `json:"monthly_simulations_used"`
	UsageResetDate   time.Time  `json:"usage_reset_date"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Preferences      *SupabaseUserPreferences `json:"preferences,omitempty"`
}

// SupabaseUserPreferences represents user preferences from Supabase
type SupabaseUserPreferences struct {
	ID                       uuid.UUID   `json:"id"`
	UserID                   uuid.UUID   `json:"user_id"`
	SportPreferences         []string    `json:"sport_preferences"`
	PlatformPreferences      []string    `json:"platform_preferences"`
	ContestTypePreferences   []string    `json:"contest_type_preferences"`
	Theme                    string      `json:"theme"`
	Language                 string      `json:"language"`
	NotificationsEnabled     bool        `json:"notifications_enabled"`
	TutorialCompleted        bool        `json:"tutorial_completed"`
	BeginnerMode             bool        `json:"beginner_mode"`
	TooltipsEnabled          bool        `json:"tooltips_enabled"`
	CreatedAt                time.Time   `json:"created_at"`
	UpdatedAt                time.Time   `json:"updated_at"`
}

// CreateUserRequest represents request to create a new user
type CreateUserRequest struct {
	ID          uuid.UUID `json:"id"`
	PhoneNumber string    `json:"phone_number"`
	FirstName   *string   `json:"first_name,omitempty"`
	LastName    *string   `json:"last_name,omitempty"`
}

// UpdateUserPreferencesRequest represents request to update user preferences
type UpdateUserPreferencesRequest struct {
	SportPreferences       []string `json:"sport_preferences,omitempty"`
	PlatformPreferences    []string `json:"platform_preferences,omitempty"`
	ContestTypePreferences []string `json:"contest_type_preferences,omitempty"`
	Theme                  string   `json:"theme,omitempty"`
	Language               string   `json:"language,omitempty"`
	NotificationsEnabled   *bool    `json:"notifications_enabled,omitempty"`
	TutorialCompleted      *bool    `json:"tutorial_completed,omitempty"`
	BeginnerMode           *bool    `json:"beginner_mode,omitempty"`
	TooltipsEnabled        *bool    `json:"tooltips_enabled,omitempty"`
}

// NewSupabaseUserService creates a new Supabase user service
func NewSupabaseUserService(supabaseURL, serviceKey string) *SupabaseUserService {
	return &SupabaseUserService{
		supabaseURL: supabaseURL,
		serviceKey:  serviceKey,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateUser creates user profile after Supabase Auth registration
func (s *SupabaseUserService) CreateUser(ctx context.Context, userID uuid.UUID, phoneNumber, firstName, lastName string) (*SupabaseUser, error) {
	user := CreateUserRequest{
		ID:          userID,
		PhoneNumber: phoneNumber,
	}
	
	if firstName != "" {
		user.FirstName = &firstName
	}
	if lastName != "" {
		user.LastName = &lastName
	}

	// Create user record in Supabase
	return s.insertUser(ctx, user)
}

// GetUser retrieves user by ID with preferences
func (s *SupabaseUserService) GetUser(ctx context.Context, userID uuid.UUID) (*SupabaseUser, error) {
	url := fmt.Sprintf("%s/rest/v1/users?id=eq.%s&select=*,preferences:user_preferences(*)", 
		s.supabaseURL, userID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var users []SupabaseUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return &users[0], nil
}

// UpdateUserPreferences updates user preferences with real-time sync
func (s *SupabaseUserService) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, preferences UpdateUserPreferencesRequest) error {
	// Build the UPSERT payload
	payload := map[string]interface{}{
		"user_id": userID,
	}
	
	// Add only non-nil fields to the payload
	if preferences.SportPreferences != nil {
		payload["sport_preferences"] = preferences.SportPreferences
	}
	if preferences.PlatformPreferences != nil {
		payload["platform_preferences"] = preferences.PlatformPreferences
	}
	if preferences.ContestTypePreferences != nil {
		payload["contest_type_preferences"] = preferences.ContestTypePreferences
	}
	if preferences.Theme != "" {
		payload["theme"] = preferences.Theme
	}
	if preferences.Language != "" {
		payload["language"] = preferences.Language
	}
	if preferences.NotificationsEnabled != nil {
		payload["notifications_enabled"] = *preferences.NotificationsEnabled
	}
	if preferences.TutorialCompleted != nil {
		payload["tutorial_completed"] = *preferences.TutorialCompleted
	}
	if preferences.BeginnerMode != nil {
		payload["beginner_mode"] = *preferences.BeginnerMode
	}
	if preferences.TooltipsEnabled != nil {
		payload["tooltips_enabled"] = *preferences.TooltipsEnabled
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/rest/v1/user_preferences", s.supabaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "resolution=merge-duplicates")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetUserByPhoneNumber retrieves user by phone number
func (s *SupabaseUserService) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*SupabaseUser, error) {
	url := fmt.Sprintf("%s/rest/v1/users?phone_number=eq.%s&select=*,preferences:user_preferences(*)", 
		s.supabaseURL, phoneNumber)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var users []SupabaseUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return &users[0], nil
}

// UpdateUsageCounters updates monthly usage for optimizations and simulations
func (s *SupabaseUserService) UpdateUsageCounters(ctx context.Context, userID uuid.UUID, optimizations, simulations int) error {
	payload := map[string]interface{}{
		"monthly_optimizations_used": optimizations,
		"monthly_simulations_used":   simulations,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/rest/v1/users?id=eq.%s", s.supabaseURL, userID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// insertUser creates a new user record in Supabase
func (s *SupabaseUserService) insertUser(ctx context.Context, user CreateUserRequest) (*SupabaseUser, error) {
	jsonData, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	url := fmt.Sprintf("%s/rest/v1/users", s.supabaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var users []SupabaseUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user creation failed")
	}

	return &users[0], nil
}

