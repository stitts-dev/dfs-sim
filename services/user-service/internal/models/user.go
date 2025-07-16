package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"gorm.io/datatypes"
)

// User represents a user that references Supabase auth.users
type User struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	FirstName        *string    `gorm:"size:100" json:"first_name,omitempty"`
	LastName         *string    `gorm:"size:100" json:"last_name,omitempty"`

	// Subscription and billing
	SubscriptionTier      string     `gorm:"size:50;default:free" json:"subscription_tier"`
	SubscriptionStatus    string     `gorm:"size:50;default:active" json:"subscription_status"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at,omitempty"`
	StripeCustomerID      *string    `gorm:"size:255" json:"stripe_customer_id,omitempty"`

	// Usage tracking
	MonthlyOptimizationsUsed int       `gorm:"default:0" json:"monthly_optimizations_used"`
	MonthlySimulationsUsed   int       `gorm:"default:0" json:"monthly_simulations_used"`
	UsageResetDate           time.Time `gorm:"type:date;default:CURRENT_DATE" json:"usage_reset_date"`

	// Account status
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relationships
	Preferences *UserPreferences `gorm:"foreignKey:UserID" json:"preferences,omitempty"`
}

// UserPreferences stores user UI and optimization preferences
type UserPreferences struct {
	ID                     uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID                 uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
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

	// Relationships
	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

// SubscriptionTier represents subscription tier configuration
type SubscriptionTier struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name                 string    `gorm:"uniqueIndex;size:50;not null" json:"name"`
	PriceCents           int       `gorm:"not null;default:0" json:"price_cents"`
	Currency             string    `gorm:"size:10;default:USD" json:"currency"`
	MonthlyOptimizations int       `gorm:"default:10" json:"monthly_optimizations"` // -1 = unlimited
	MonthlySimulations   int       `gorm:"default:5" json:"monthly_simulations"`   // -1 = unlimited
	AIRecommendations    bool      `gorm:"default:false" json:"ai_recommendations"`
	BankVerification     bool      `gorm:"default:false" json:"bank_verification"`
	PrioritySupport      bool      `gorm:"default:false" json:"priority_support"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}


// Model Methods

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}

// GetName returns the computed full name for API compatibility
func (u *User) GetName() string {
	var name string
	if u.FirstName != nil {
		name = *u.FirstName
	}
	if u.LastName != nil {
		if name != "" {
			name += " "
		}
		name += *u.LastName
	}
	return name
}

// TableName returns the table name for UserPreferences
func (UserPreferences) TableName() string {
	return "user_preferences"
}

// TableName returns the table name for SubscriptionTier
func (SubscriptionTier) TableName() string {
	return "subscription_tiers"
}


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

// UpdateLoginTime updates the last login timestamp
func (u *User) UpdateLoginTime(db *database.DB) error {
	now := time.Now()
	u.LastLoginAt = &now
	return db.Model(u).UpdateColumn("last_login_at", now).Error
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

// Database query methods for integer-based users

// GetUserByID fetches user by ID with preferences
func GetUserByID(db *database.DB, userID uuid.UUID) (*User, error) {
	var user User
	err := db.Preload("Preferences").Where("id = ?", userID).First(&user).Error
	return &user, err
}

// CreateUser creates a new user with given Supabase auth user ID
func CreateUser(db *database.DB, userID uuid.UUID) (*User, error) {
	user := &User{
		ID:               userID,
		SubscriptionTier: "free",
		UsageResetDate:   time.Now(),
		IsActive:         true,
	}

	err := db.Create(user).Error
	return user, err
}

// CreateUserPreferences creates default preferences for a user
func CreateUserPreferences(db *database.DB, userID uuid.UUID) (*UserPreferences, error) {
	preferences := &UserPreferences{
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

// UpdateUserPreferences updates user preferences
func UpdateUserPreferences(db *database.DB, userID uuid.UUID, updates map[string]interface{}) error {
	return db.Model(&UserPreferences{}).Where("user_id = ?", userID).Updates(updates).Error
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

