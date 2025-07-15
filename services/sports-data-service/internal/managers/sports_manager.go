package managers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/models"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SportsManager handles sports discovery and management
type SportsManager struct {
	db     *gorm.DB
	cache  types.CacheProvider
	logger *logrus.Logger
}

// NewSportsManager creates a new sports manager instance
func NewSportsManager(db *gorm.DB, cache types.CacheProvider, logger *logrus.Logger) *SportsManager {
	return &SportsManager{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// GetOrCreateSport finds existing sport or creates new one
func (sm *SportsManager) GetOrCreateSport(ctx context.Context, name, abbreviation string) (*models.Sport, error) {
	// Normalize inputs
	name = strings.TrimSpace(name)
	abbreviation = strings.ToUpper(strings.TrimSpace(abbreviation))
	
	if name == "" || abbreviation == "" {
		return nil, errors.New("sport name and abbreviation are required")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("sport:%s", abbreviation)
	var sport models.Sport
	if err := sm.cache.Get(ctx, cacheKey, &sport); err == nil {
		sm.logger.Debugf("Found sport %s in cache", abbreviation)
		return &sport, nil
	}

	// Try to find existing sport by abbreviation
	err := sm.db.WithContext(ctx).Where("abbreviation = ?", abbreviation).First(&sport).Error
	if err == nil {
		// Cache the found sport
		sm.cache.Set(ctx, cacheKey, sport, 3600) // Cache for 1 hour
		sm.logger.Infof("Found existing sport: %s (%s)", name, abbreviation)
		return &sport, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		sm.logger.WithError(err).Errorf("Error querying sport %s", abbreviation)
		return nil, fmt.Errorf("error querying sport: %w", err)
	}

	// Sport doesn't exist, create it
	sport = models.Sport{
		ID:           uuid.New(),
		Name:         name,
		Abbreviation: abbreviation,
		IsActive:     true,
	}

	if err := sm.db.WithContext(ctx).Create(&sport).Error; err != nil {
		sm.logger.WithError(err).Errorf("Error creating sport %s (%s)", name, abbreviation)
		return nil, fmt.Errorf("error creating sport: %w", err)
	}

	// Cache the new sport
	sm.cache.Set(ctx, cacheKey, sport, 3600)
	
	sm.logger.Infof("Created new sport: %s (%s) with ID %s", name, abbreviation, sport.ID)
	return &sport, nil
}

// GetSportByAbbreviation retrieves sport by abbreviation
func (sm *SportsManager) GetSportByAbbreviation(ctx context.Context, abbreviation string) (*models.Sport, error) {
	abbreviation = strings.ToUpper(strings.TrimSpace(abbreviation))
	
	// Check cache first
	cacheKey := fmt.Sprintf("sport:%s", abbreviation)
	var sport models.Sport
	if err := sm.cache.Get(ctx, cacheKey, &sport); err == nil {
		return &sport, nil
	}

	// Query database
	err := sm.db.WithContext(ctx).Where("abbreviation = ? AND is_active = ?", abbreviation, true).First(&sport).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sport %s not found", abbreviation)
		}
		return nil, fmt.Errorf("error querying sport: %w", err)
	}

	// Cache the result
	sm.cache.Set(ctx, cacheKey, sport, 3600)
	return &sport, nil
}

// ListActiveSports returns all active sports
func (sm *SportsManager) ListActiveSports(ctx context.Context) ([]models.Sport, error) {
	var sports []models.Sport
	
	err := sm.db.WithContext(ctx).Where("is_active = ?", true).Order("name").Find(&sports).Error
	if err != nil {
		sm.logger.WithError(err).Error("Error listing active sports")
		return nil, fmt.Errorf("error listing sports: %w", err)
	}

	return sports, nil
}

// MapSportName maps common sport name variations to standard abbreviations
func (sm *SportsManager) MapSportName(sportName string) (name, abbreviation string) {
	sportName = strings.ToLower(strings.TrimSpace(sportName))
	
	switch {
	case strings.Contains(sportName, "golf"), strings.Contains(sportName, "pga"):
		return "Golf", "GOLF"
	case strings.Contains(sportName, "basketball"), strings.Contains(sportName, "nba"):
		return "Basketball", "NBA"
	case strings.Contains(sportName, "football"), strings.Contains(sportName, "nfl"):
		return "Football", "NFL"
	case strings.Contains(sportName, "baseball"), strings.Contains(sportName, "mlb"):
		return "Baseball", "MLB"
	case strings.Contains(sportName, "hockey"), strings.Contains(sportName, "nhl"):
		return "Hockey", "NHL"
	case strings.Contains(sportName, "soccer"), strings.Contains(sportName, "mls"):
		return "Soccer", "MLS"
	case strings.Contains(sportName, "tennis"):
		return "Tennis", "TENNIS"
	default:
		// Default to title case name and uppercase abbreviation
		return strings.Title(sportName), strings.ToUpper(sportName)
	}
}

// EnsureGolfSport ensures golf sport exists (helper for golf providers)
func (sm *SportsManager) EnsureGolfSport(ctx context.Context) (*models.Sport, error) {
	return sm.GetOrCreateSport(ctx, "Golf", "GOLF")
}

// EnsureNBASport ensures NBA sport exists (helper for NBA providers)
func (sm *SportsManager) EnsureNBASport(ctx context.Context) (*models.Sport, error) {
	return sm.GetOrCreateSport(ctx, "Basketball", "NBA")
}

// EnsureNFLSport ensures NFL sport exists (helper for NFL providers)
func (sm *SportsManager) EnsureNFLSport(ctx context.Context) (*models.Sport, error) {
	return sm.GetOrCreateSport(ctx, "Football", "NFL")
}