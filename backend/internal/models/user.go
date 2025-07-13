package models

import (
	"time"

	"github.com/jstittsworth/dfs-optimizer/pkg/database"
)

// User represents a user in the system with phone-based authentication
type User struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	PhoneNumber     string     `gorm:"uniqueIndex;size:20;not null" json:"phone_number"`
	PhoneVerified   bool       `gorm:"default:false" json:"phone_verified"`
	Email           *string    `gorm:"size:255" json:"email,omitempty"`
	EmailVerified   bool       `gorm:"default:false" json:"email_verified"`
	FirstName       *string    `gorm:"size:100" json:"first_name,omitempty"`
	LastName        *string    `gorm:"size:100" json:"last_name,omitempty"`
	
	// Subscription and billing
	SubscriptionTier      string     `gorm:"size:20;default:free" json:"subscription_tier"`
	SubscriptionStatus    string     `gorm:"size:20;default:active" json:"subscription_status"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	StripeCustomerID      *string    `gorm:"size:100" json:"stripe_customer_id,omitempty"`
	
	// Usage tracking
	MonthlyOptimizationsUsed int       `gorm:"default:0" json:"monthly_optimizations_used"`
	MonthlySimulationsUsed   int       `gorm:"default:0" json:"monthly_simulations_used"`
	UsageResetDate           time.Time `gorm:"type:date;default:CURRENT_DATE" json:"usage_reset_date"`
	
	// Account status
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	
	// Relationships
	Preferences *UserPreferences `gorm:"foreignKey:UserID" json:"preferences,omitempty"`
}

// SubscriptionTier represents subscription tier configuration
type SubscriptionTier struct {
	ID                   uint   `gorm:"primaryKey" json:"id"`
	Name                 string `gorm:"uniqueIndex;size:20;not null" json:"name"`
	PriceCents           int    `gorm:"not null;default:0" json:"price_cents"`
	Currency             string `gorm:"size:3;default:USD" json:"currency"`
	MonthlyOptimizations int    `gorm:"default:10" json:"monthly_optimizations"` // -1 = unlimited
	MonthlySimulations   int    `gorm:"default:5" json:"monthly_simulations"`   // -1 = unlimited
	AIRecommendations    bool   `gorm:"default:false" json:"ai_recommendations"`
	BankVerification     bool   `gorm:"default:false" json:"bank_verification"`
	PrioritySupport      bool   `gorm:"default:false" json:"priority_support"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PhoneVerificationCode represents OTP codes for phone verification
type PhoneVerificationCode struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"size:20;not null" json:"phone_number"`
	Code        string    `gorm:"size:6;not null" json:"code"`
	ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`
	Attempts    int       `gorm:"default:0" json:"attempts"`
	Verified    bool      `gorm:"default:false" json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserMethods for business logic

// IsSubscriptionActive checks if user has an active subscription
func (u *User) IsSubscriptionActive() bool {
	if u.SubscriptionStatus != "active" {
		return false
	}
	if u.SubscriptionExpiresAt != nil && u.SubscriptionExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

// GetTier returns the subscription tier configuration
func (u *User) GetTier(db *database.DB) (*SubscriptionTier, error) {
	var tier SubscriptionTier
	err := db.Where("name = ?", u.SubscriptionTier).First(&tier).Error
	return &tier, err
}

// CanOptimize checks if user can perform optimizations based on their tier and usage
func (u *User) CanOptimize(db *database.DB) (bool, error) {
	// Reset usage if it's a new month
	if err := u.ResetUsageIfNeeded(db); err != nil {
		return false, err
	}
	
	tier, err := u.GetTier(db)
	if err != nil {
		return false, err
	}
	
	// Unlimited optimizations
	if tier.MonthlyOptimizations == -1 {
		return true, nil
	}
	
	return u.MonthlyOptimizationsUsed < tier.MonthlyOptimizations, nil
}

// CanSimulate checks if user can perform simulations based on their tier and usage
func (u *User) CanSimulate(db *database.DB) (bool, error) {
	// Reset usage if it's a new month
	if err := u.ResetUsageIfNeeded(db); err != nil {
		return false, err
	}
	
	tier, err := u.GetTier(db)
	if err != nil {
		return false, err
	}
	
	// Unlimited simulations
	if tier.MonthlySimulations == -1 {
		return true, nil
	}
	
	return u.MonthlySimulationsUsed < tier.MonthlySimulations, nil
}

// IncrementUsage increments usage counters
func (u *User) IncrementOptimizationUsage(db *database.DB) error {
	return db.Model(u).UpdateColumn("monthly_optimizations_used", u.MonthlyOptimizationsUsed+1).Error
}

func (u *User) IncrementSimulationUsage(db *database.DB) error {
	return db.Model(u).UpdateColumn("monthly_simulations_used", u.MonthlySimulationsUsed+1).Error
}

// ResetUsageIfNeeded resets monthly usage if it's a new month
func (u *User) ResetUsageIfNeeded(db *database.DB) error {
	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	resetMonth := time.Date(u.UsageResetDate.Year(), u.UsageResetDate.Month(), 1, 0, 0, 0, 0, u.UsageResetDate.Location())
	
	if currentMonth.After(resetMonth) {
		updates := map[string]interface{}{
			"monthly_optimizations_used": 0,
			"monthly_simulations_used":   0,
			"usage_reset_date":           now,
		}
		if err := db.Model(u).Updates(updates).Error; err != nil {
			return err
		}
		u.MonthlyOptimizationsUsed = 0
		u.MonthlySimulationsUsed = 0
		u.UsageResetDate = now
	}
	
	return nil
}

// Database query methods

// GetUserByPhoneNumber fetches user by phone number
func GetUserByPhoneNumber(db *database.DB, phoneNumber string) (*User, error) {
	var user User
	err := db.Where("phone_number = ?", phoneNumber).First(&user).Error
	return &user, err
}

// GetUserByID fetches user by ID with preferences
func GetUserByID(db *database.DB, userID uint) (*User, error) {
	var user User
	err := db.Preload("Preferences").Where("id = ?", userID).First(&user).Error
	return &user, err
}

// CreateUser creates a new user
func CreateUser(db *database.DB, phoneNumber string) (*User, error) {
	user := &User{
		PhoneNumber:      phoneNumber,
		PhoneVerified:    false,
		SubscriptionTier: "free",
		UsageResetDate:   time.Now(),
	}
	
	err := db.Create(user).Error
	return user, err
}

// UpdateUserLoginTime updates the last login timestamp
func (u *User) UpdateLoginTime(db *database.DB) error {
	now := time.Now()
	u.LastLoginAt = &now
	return db.Model(u).UpdateColumn("last_login_at", now).Error
}

// VerificationCode methods

// CreateVerificationCode creates a new verification code for phone number
func CreateVerificationCode(db *database.DB, phoneNumber, code string) (*PhoneVerificationCode, error) {
	// Clean up any existing codes for this phone number
	db.Where("phone_number = ?", phoneNumber).Delete(&PhoneVerificationCode{})
	
	verCode := &PhoneVerificationCode{
		PhoneNumber: phoneNumber,
		Code:        code,
		ExpiresAt:   time.Now().Add(10 * time.Minute), // 10 minute expiry
	}
	
	err := db.Create(verCode).Error
	return verCode, err
}

// ValidateVerificationCode validates and marks code as used
func ValidateVerificationCode(db *database.DB, phoneNumber, code string) (*PhoneVerificationCode, error) {
	var verCode PhoneVerificationCode
	
	// Find unexpired, unverified code
	err := db.Where("phone_number = ? AND code = ? AND expires_at > ? AND verified = false", 
		phoneNumber, code, time.Now()).First(&verCode).Error
	if err != nil {
		return nil, err
	}
	
	// Increment attempts
	verCode.Attempts++
	
	// Check attempt limit (max 3 attempts)
	if verCode.Attempts > 3 {
		db.Model(&verCode).Updates(map[string]interface{}{
			"attempts": verCode.Attempts,
			"verified": false,
		})
		return nil, err
	}
	
	// Mark as verified
	verCode.Verified = true
	err = db.Model(&verCode).Updates(map[string]interface{}{
		"attempts": verCode.Attempts,
		"verified": true,
	}).Error
	
	return &verCode, err
}

// GetAllSubscriptionTiers returns all available subscription tiers
func GetAllSubscriptionTiers(db *database.DB) ([]SubscriptionTier, error) {
	var tiers []SubscriptionTier
	err := db.Order("price_cents ASC").Find(&tiers).Error
	return tiers, err
}

// GetSubscriptionTierByName returns subscription tier by name
func GetSubscriptionTierByName(db *database.DB, name string) (*SubscriptionTier, error) {
	var tier SubscriptionTier
	err := db.Where("name = ?", name).First(&tier).Error
	return &tier, err
}