package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"gorm.io/datatypes"
)

// SupabaseUser represents a user in the Supabase-based system
type SupabaseUser struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	PhoneNumber      string     `gorm:"uniqueIndex;size:20;not null" json:"phone_number"`
	FirstName        *string    `gorm:"size:100" json:"first_name,omitempty"`
	LastName         *string    `gorm:"size:100" json:"last_name,omitempty"`
	
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
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Relationships
	Preferences *SupabaseUserPreferences `gorm:"foreignKey:UserID" json:"preferences,omitempty"`
}

// SupabaseUserPreferences stores user UI and optimization preferences
type SupabaseUserPreferences struct {
	ID                     uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID                 uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	SportPreferences       datatypes.JSON `gorm:"type:jsonb;default:'[\"nba\", \"nfl\", \"mlb\", \"golf\"]'" json:"sport_preferences"`
	PlatformPreferences    datatypes.JSON `gorm:"type:jsonb;default:'[\"draftkings\", \"fanduel\"]'" json:"platform_preferences"`
	ContestTypePreferences datatypes.JSON `gorm:"type:jsonb;default:'[\"gpp\", \"cash\"]'" json:"contest_type_preferences"`
	Theme                  string         `gorm:"size:20;default:light" json:"theme"`
	Language               string         `gorm:"size:10;default:en" json:"language"`
	NotificationsEnabled   bool           `gorm:"default:true" json:"notifications_enabled"`
	TutorialCompleted      bool           `gorm:"default:false" json:"tutorial_completed"`
	BeginnerMode           bool           `gorm:"default:true" json:"beginner_mode"`
	TooltipsEnabled        bool           `gorm:"default:true" json:"tooltips_enabled"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

// SupabaseSubscriptionTier represents subscription tier configuration
type SupabaseSubscriptionTier struct {
	ID                   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name                 string    `gorm:"uniqueIndex;size:20;not null" json:"name"`
	PriceCents           int       `gorm:"not null;default:0" json:"price_cents"`
	Currency             string    `gorm:"size:3;default:USD" json:"currency"`
	MonthlyOptimizations int       `gorm:"default:10" json:"monthly_optimizations"` // -1 = unlimited
	MonthlySimulations   int       `gorm:"default:5" json:"monthly_simulations"`   // -1 = unlimited
	AIRecommendations    bool      `gorm:"default:false" json:"ai_recommendations"`
	BankVerification     bool      `gorm:"default:false" json:"bank_verification"`
	PrioritySupport      bool      `gorm:"default:false" json:"priority_support"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// LegacyUserMapping maps legacy integer user IDs to Supabase UUIDs
type LegacyUserMapping struct {
	LegacyID     uint      `gorm:"primaryKey" json:"legacy_id"`
	SupabaseUUID uuid.UUID `gorm:"type:uuid;not null" json:"supabase_uuid"`
	CreatedAt    time.Time `json:"created_at"`
}

// SupabaseUser Methods

// TableName returns the table name for SupabaseUser
func (SupabaseUser) TableName() string {
	return "users"
}

// TableName returns the table name for SupabaseUserPreferences
func (SupabaseUserPreferences) TableName() string {
	return "user_preferences"
}

// TableName returns the table name for SupabaseSubscriptionTier
func (SupabaseSubscriptionTier) TableName() string {
	return "subscription_tiers"
}

// TableName returns the table name for LegacyUserMapping
func (LegacyUserMapping) TableName() string {
	return "legacy_user_mapping"
}

// IsSubscriptionActive checks if user has an active subscription
func (u *SupabaseUser) IsSubscriptionActive() bool {
	if u.SubscriptionStatus != "active" {
		return false
	}
	if u.SubscriptionExpiresAt != nil && u.SubscriptionExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

// GetTier returns the subscription tier configuration
func (u *SupabaseUser) GetTier(db *database.DB) (*SupabaseSubscriptionTier, error) {
	var tier SupabaseSubscriptionTier
	err := db.Where("name = ?", u.SubscriptionTier).First(&tier).Error
	return &tier, err
}

// CanOptimize checks if user can perform optimizations based on their tier and usage
func (u *SupabaseUser) CanOptimize(db *database.DB) (bool, error) {
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
func (u *SupabaseUser) CanSimulate(db *database.DB) (bool, error) {
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
func (u *SupabaseUser) IncrementOptimizationUsage(db *database.DB) error {
	return db.Model(u).UpdateColumn("monthly_optimizations_used", u.MonthlyOptimizationsUsed+1).Error
}

func (u *SupabaseUser) IncrementSimulationUsage(db *database.DB) error {
	return db.Model(u).UpdateColumn("monthly_simulations_used", u.MonthlySimulationsUsed+1).Error
}

// ResetUsageIfNeeded resets monthly usage if it's a new month
func (u *SupabaseUser) ResetUsageIfNeeded(db *database.DB) error {
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

// Database query methods for Supabase users

// GetSupabaseUserByID fetches user by UUID with preferences
func GetSupabaseUserByID(db *database.DB, userID uuid.UUID) (*SupabaseUser, error) {
	var user SupabaseUser
	err := db.Preload("Preferences").Where("id = ?", userID).First(&user).Error
	return &user, err
}

// GetSupabaseUserByPhoneNumber fetches user by phone number
func GetSupabaseUserByPhoneNumber(db *database.DB, phoneNumber string) (*SupabaseUser, error) {
	var user SupabaseUser
	err := db.Where("phone_number = ?", phoneNumber).First(&user).Error
	return &user, err
}

// CreateSupabaseUser creates a new Supabase user
func CreateSupabaseUser(db *database.DB, userID uuid.UUID, phoneNumber string, firstName, lastName *string) (*SupabaseUser, error) {
	user := &SupabaseUser{
		ID:               userID,
		PhoneNumber:      phoneNumber,
		FirstName:        firstName,
		LastName:         lastName,
		SubscriptionTier: "free",
		UsageResetDate:   time.Now(),
	}
	
	err := db.Create(user).Error
	return user, err
}

// CreateSupabaseUserPreferences creates default preferences for a user
func CreateSupabaseUserPreferences(db *database.DB, userID uuid.UUID) (*SupabaseUserPreferences, error) {
	preferences := &SupabaseUserPreferences{
		UserID:                 userID,
		SportPreferences:       datatypes.JSON(`["nba", "nfl", "mlb", "golf"]`),
		PlatformPreferences:    datatypes.JSON(`["draftkings", "fanduel"]`),
		ContestTypePreferences: datatypes.JSON(`["gpp", "cash"]`),
		Theme:                  "light",
		Language:               "en",
		NotificationsEnabled:   true,
		TutorialCompleted:      false,
		BeginnerMode:           true,
		TooltipsEnabled:        true,
	}
	
	err := db.Create(preferences).Error
	return preferences, err
}

// UpdateSupabaseUserPreferences updates user preferences
func UpdateSupabaseUserPreferences(db *database.DB, userID uuid.UUID, updates map[string]interface{}) error {
	return db.Model(&SupabaseUserPreferences{}).Where("user_id = ?", userID).Updates(updates).Error
}

// GetAllSupabaseSubscriptionTiers returns all available subscription tiers
func GetAllSupabaseSubscriptionTiers(db *database.DB) ([]SupabaseSubscriptionTier, error) {
	var tiers []SupabaseSubscriptionTier
	err := db.Order("price_cents ASC").Find(&tiers).Error
	return tiers, err
}

// GetSupabaseSubscriptionTierByName returns subscription tier by name
func GetSupabaseSubscriptionTierByName(db *database.DB, name string) (*SupabaseSubscriptionTier, error) {
	var tier SupabaseSubscriptionTier
	err := db.Where("name = ?", name).First(&tier).Error
	return &tier, err
}

// CreateLegacyUserMapping creates a mapping between legacy user ID and Supabase UUID
func CreateLegacyUserMapping(db *database.DB, legacyID uint, supabaseUUID uuid.UUID) error {
	mapping := &LegacyUserMapping{
		LegacyID:     legacyID,
		SupabaseUUID: supabaseUUID,
	}
	return db.Create(mapping).Error
}

// GetSupabaseUUIDByLegacyID retrieves Supabase UUID by legacy ID
func GetSupabaseUUIDByLegacyID(db *database.DB, legacyID uint) (uuid.UUID, error) {
	var mapping LegacyUserMapping
	err := db.Where("legacy_id = ?", legacyID).First(&mapping).Error
	if err != nil {
		return uuid.Nil, err
	}
	return mapping.SupabaseUUID, nil
}

// GetLegacyIDBySupabaseUUID retrieves legacy ID by Supabase UUID  
func GetLegacyIDBySupabaseUUID(db *database.DB, supabaseUUID uuid.UUID) (uint, error) {
	var mapping LegacyUserMapping
	err := db.Where("supabase_uuid = ?", supabaseUUID).First(&mapping).Error
	if err != nil {
		return 0, err
	}
	return mapping.LegacyID, nil
}