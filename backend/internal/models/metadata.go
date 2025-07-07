package models

import (
	"time"

	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"gorm.io/datatypes"
)

// PlayerMetadata stores additional information about player positions
type PlayerMetadata struct {
	PlayerID            int       `gorm:"primaryKey" json:"player_id"`
	Player              Player    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	FullPositionName    string    `json:"full_position_name"`
	PositionDescription string    `json:"position_description"`
	TypicalScoring      string    `json:"typical_scoring"`
	CreatedAt           time.Time `json:"created_at"`
}

// TeamInfo stores full team information
type TeamInfo struct {
	Abbreviation string    `gorm:"primaryKey;size:10" json:"abbreviation"`
	FullName     string    `gorm:"size:100;not null" json:"full_name"`
	Stadium      string    `gorm:"size:100" json:"stadium"`
	Outdoor      bool      `json:"outdoor"`
	Timezone     string    `gorm:"size:50" json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GlossaryTerm stores DFS terminology and explanations
type GlossaryTerm struct {
	ID           uint                   `gorm:"primaryKey" json:"id"`
	Term         string                 `gorm:"uniqueIndex;size:100;not null" json:"term"`
	Category     string                 `gorm:"size:50;not null" json:"category"` // general, sport-specific, strategy, platform
	Definition   string                 `gorm:"type:text;not null" json:"definition"`
	Examples     datatypes.JSON         `json:"examples"`
	RelatedTerms []string `gorm:"type:text[]" json:"related_terms"`
	Difficulty   string                 `gorm:"size:20" json:"difficulty"` // beginner, intermediate, advanced
	Sport        string                 `gorm:"size:20" json:"sport"`      // optional sport association
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// AIRecommendation stores AI recommendation history for analytics
type AIRecommendation struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       int            `json:"user_id"`
	ContestID    int            `json:"contest_id"`
	Contest      Contest        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Request      datatypes.JSON `json:"request"`
	Response     datatypes.JSON `json:"response"`
	Confidence   float64        `json:"confidence"`
	WasUsed      bool           `json:"was_used"`
	LineupResult *float64       `json:"lineup_result"` // actual points if tracked
	CreatedAt    time.Time      `json:"created_at"`
}

// UserPreferences stores user UI preferences
type UserPreferences struct {
	UserID              int                   `gorm:"primaryKey" json:"user_id"`
	BeginnerMode        bool                  `gorm:"default:false" json:"beginner_mode"`
	ShowTooltips        bool                  `gorm:"default:true" json:"show_tooltips"`
	TooltipDelay        int                   `gorm:"default:500" json:"tooltip_delay"`
	PreferredSports     []string `gorm:"type:text[]" json:"preferred_sports"`
	AISuggestionsEnabled bool                  `gorm:"default:true" json:"ai_suggestions_enabled"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
}

// GetTeamByAbbreviation fetches team info by abbreviation
func GetTeamByAbbreviation(db *database.DB, abbreviation string) (*TeamInfo, error) {
	var team TeamInfo
	err := db.Where("abbreviation = ?", abbreviation).First(&team).Error
	return &team, err
}

// GetGlossaryTerms fetches all glossary terms with optional filters
func GetGlossaryTerms(db *database.DB, category, difficulty, sport string) ([]GlossaryTerm, error) {
	var terms []GlossaryTerm
	query := db.Model(&GlossaryTerm{})

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}
	if sport != "" {
		query = query.Where("sport = ? OR sport IS NULL", sport)
	}

	err := query.Order("term ASC").Find(&terms).Error
	return terms, err
}

// SearchGlossaryTerms performs fuzzy search on glossary terms
func SearchGlossaryTerms(db *database.DB, search string) ([]GlossaryTerm, error) {
	var terms []GlossaryTerm
	err := db.Where("term ILIKE ? OR definition ILIKE ?", "%"+search+"%", "%"+search+"%").
		Order("term ASC").
		Find(&terms).Error
	return terms, err
}

// GetUserPreferences fetches or creates user preferences
func GetUserPreferences(db *database.DB, userID int) (*UserPreferences, error) {
	var prefs UserPreferences
	err := db.FirstOrCreate(&prefs, UserPreferences{UserID: userID}).Error
	return &prefs, err
}

// UpdateUserPreferences updates user preferences
func UpdateUserPreferences(db *database.DB, userID int, updates map[string]interface{}) error {
	return db.Model(&UserPreferences{}).Where("user_id = ?", userID).Updates(updates).Error
}