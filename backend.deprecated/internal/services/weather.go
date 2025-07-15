package services

import (
	"context"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
)

// WeatherService provides weather data for sporting events
type WeatherService struct {
	db     *database.DB
	cache  *CacheService
	logger *logrus.Logger
}

// NewWeatherService creates a new weather service
func NewWeatherService(db *database.DB, cache *CacheService, logger *logrus.Logger) *WeatherService {
	return &WeatherService{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// GetImpactFactor returns a multiplier for weather impact on performance
func (ws *WeatherService) GetImpactFactor(date time.Time) float64 {
	// Simple stub implementation
	// In a real implementation, this would fetch actual weather data
	// and calculate impact based on conditions
	return 1.0 // No impact
}

// GetWeatherConditions returns weather conditions for a location and date
func (ws *WeatherService) GetWeatherConditions(ctx context.Context, location string, date time.Time) (*models.WeatherConditions, error) {
	// Stub implementation
	// In production, this would call a weather API
	return &models.WeatherConditions{
		Temperature: 72,
		WindSpeed:   10,
		WindDir:     "SW",
		Conditions:  "partly_cloudy",
		Humidity:    65,
	}, nil
}

// GetGolfWeatherImpact calculates weather impact specifically for golf
func (ws *WeatherService) GetGolfWeatherImpact(conditions models.WeatherConditions) float64 {
	impact := 1.0

	// Wind is the biggest factor in golf
	if conditions.WindSpeed > 25 {
		impact *= 1.08 // Very difficult conditions
	} else if conditions.WindSpeed > 20 {
		impact *= 1.06
	} else if conditions.WindSpeed > 15 {
		impact *= 1.04
	} else if conditions.WindSpeed > 10 {
		impact *= 1.02
	}

	// Rain impact
	if conditions.Conditions == "rain" || conditions.Conditions == "heavy_rain" {
		impact *= 1.05
	} else if conditions.Conditions == "light_rain" || conditions.Conditions == "drizzle" {
		impact *= 1.02
	}

	// Temperature extremes
	if conditions.Temperature < 45 || conditions.Temperature > 95 {
		impact *= 1.03
	}

	return impact
}
