package services

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// WeatherService provides weather data and golf impact analysis
type WeatherService struct {
	openWeatherProvider *providers.OpenWeatherProvider
	redisClient        *redis.Client
	courseLocations    map[string]*providers.WeatherLocation
}

// WeatherImpactConfig represents configuration for weather impact calculations
type WeatherImpactConfig struct {
	HighWindThreshold    float64 // mph
	ModerateWindThreshold float64 // mph
	ColdTempThreshold    float64 // degrees F
	WetBulbColdThreshold float64 // wet-bulb temperature
}

// NewWeatherService creates a new weather service with OpenWeather integration
func NewWeatherService(redisClient *redis.Client) (*WeatherService, error) {
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENWEATHER_API_KEY environment variable not set")
	}

	openWeatherProvider := providers.NewOpenWeatherProvider(apiKey, redisClient)

	// Initialize known golf course locations
	courseLocations := initializeCourseLocations()

	return &WeatherService{
		openWeatherProvider: openWeatherProvider,
		redisClient:        redisClient,
		courseLocations:    courseLocations,
	}, nil
}

// GetWeatherConditions retrieves current weather conditions for a golf course
func (w *WeatherService) GetWeatherConditions(ctx context.Context, courseID string) (*types.WeatherConditions, error) {
	location, exists := w.courseLocations[courseID]
	if !exists {
		return nil, fmt.Errorf("course location not found for course ID: %s", courseID)
	}

	return w.openWeatherProvider.GetWeatherConditions(ctx, location)
}

// CalculateGolfImpact calculates weather impact on golf performance based on research
func (w *WeatherService) CalculateGolfImpact(conditions *types.WeatherConditions) *types.WeatherImpact {
	if conditions == nil {
		return &types.WeatherImpact{}
	}

	impact := &types.WeatherImpact{}
	config := w.getWeatherImpactConfig()

	// Wind is most significant factor (19-27% variance impact)
	windImpact := w.calculateWindImpact(float64(conditions.WindSpeed), config)
	impact.ScoreImpact += windImpact.scoreImpact
	impact.VarianceMultiplier = windImpact.varianceMultiplier
	impact.WindAdvantage = windImpact.advantage

	// Temperature impact (wet-bulb better predictor than air temp)
	tempImpact := w.calculateTemperatureImpact(float64(conditions.Temperature), conditions.Humidity, config)
	impact.ScoreImpact += tempImpact.scoreImpact
	impact.VarianceMultiplier *= tempImpact.varianceMultiplier

	// Precipitation impact (soft conditions)
	precipImpact := w.calculatePrecipitationImpact(conditions.Conditions)
	impact.SoftConditions = precipImpact.softConditions
	impact.DistanceReduction = precipImpact.distanceReduction
	impact.ScoreImpact += precipImpact.scoreImpact

	// Tee time advantage calculation
	impact.TeeTimeAdvantage = w.calculateTeeTimeAdvantage(conditions)

	return impact
}

// GetWeatherAdvantageScore calculates a player's advantage score based on weather conditions
func (w *WeatherService) GetWeatherAdvantageScore(ctx context.Context, courseID string, playerStats *types.PlayerStats) (float64, error) {
	conditions, err := w.GetWeatherConditions(ctx, courseID)
	if err != nil {
		return 0.0, err
	}

	impact := w.CalculateGolfImpact(conditions)
	
	// Calculate player-specific advantage based on historical performance in similar conditions
	// This would be enhanced with actual player wind/weather performance data
	baseAdvantage := impact.WindAdvantage + impact.TeeTimeAdvantage

	// Adjust based on player strengths
	if playerStats != nil {
		// Players with better ball striking typically handle wind better
		if playerStats.StrokesGainedTeeToGreen > 0.5 {
			baseAdvantage *= 1.2
		} else if playerStats.StrokesGainedTeeToGreen < -0.5 {
			baseAdvantage *= 0.8
		}
	}

	return baseAdvantage, nil
}

// calculateWindImpact determines wind impact on golf performance
func (w *WeatherService) calculateWindImpact(windSpeed float64, config *WeatherImpactConfig) windImpactResult {
	result := windImpactResult{}

	if windSpeed > 20 {
		// High wind conditions (>20 mph)
		result.scoreImpact = 2.5 // strokes added
		result.varianceMultiplier = 1.4
		result.advantage = -0.15 // Negative advantage (harder conditions)
	} else if windSpeed > config.ModerateWindThreshold {
		// Moderate wind (15-20 mph)
		result.scoreImpact = 1.5
		result.varianceMultiplier = 1.25
		result.advantage = -0.08
	} else if windSpeed > 10 {
		// Light wind (10-15 mph)
		result.scoreImpact = 0.75
		result.varianceMultiplier = 1.1
		result.advantage = -0.03
	} else {
		// Calm conditions (<10 mph)
		result.scoreImpact = 0.0
		result.varianceMultiplier = 1.0
		result.advantage = 0.05 // Slight advantage in calm conditions
	}

	return result
}

// calculateTemperatureImpact determines temperature impact using wet-bulb calculation
func (w *WeatherService) calculateTemperatureImpact(temperature float64, humidity int, config *WeatherImpactConfig) tempImpactResult {
	wetBulb := w.calculateWetBulb(temperature, float64(humidity))
	result := tempImpactResult{}

	if wetBulb < config.WetBulbColdThreshold {
		// Cold conditions affect ball flight and player comfort
		result.scoreImpact = 0.5
		result.varianceMultiplier = 1.1
	} else if temperature > 85 {
		// Hot conditions affect player stamina
		result.scoreImpact = 0.3
		result.varianceMultiplier = 1.05
	} else {
		// Optimal conditions
		result.scoreImpact = 0.0
		result.varianceMultiplier = 1.0
	}

	return result
}

// calculatePrecipitationImpact determines rain/moisture impact
func (w *WeatherService) calculatePrecipitationImpact(conditions string) precipImpactResult {
	result := precipImpactResult{}

	switch conditions {
	case "Rain", "Drizzle", "Thunderstorm":
		result.softConditions = true
		result.distanceReduction = 0.05 // 5% distance loss
		result.scoreImpact = 1.0        // Scores generally higher
	case "Snow":
		result.softConditions = true
		result.distanceReduction = 0.10 // 10% distance loss
		result.scoreImpact = 2.0        // Significantly harder conditions
	default:
		result.softConditions = false
		result.distanceReduction = 0.0
		result.scoreImpact = 0.0
	}

	return result
}

// calculateTeeTimeAdvantage determines morning vs afternoon advantage
func (w *WeatherService) calculateTeeTimeAdvantage(conditions *types.WeatherConditions) float64 {
	// Morning rounds typically have calmer conditions
	// This is a simplified calculation - would be enhanced with hourly forecasts
	
	if conditions.WindSpeed < 10 {
		return 0.1 // Slight advantage in calm conditions
	} else if conditions.WindSpeed > 15 {
		return -0.1 // Disadvantage in windy conditions
	}
	
	return 0.0
}

// calculateWetBulb calculates wet-bulb temperature from air temp and humidity
func (w *WeatherService) calculateWetBulb(airTemp, humidity float64) float64 {
	// Simplified wet-bulb calculation
	// More accurate calculation would use iterative approach
	return airTemp*math.Atan(0.151977*math.Sqrt(humidity+8.313659)) +
		math.Atan(airTemp+humidity) -
		math.Atan(humidity-1.676331) +
		0.00391838*math.Pow(humidity, 1.5)*math.Atan(0.023101*humidity) - 4.686035
}

// getWeatherImpactConfig returns configuration for weather impact calculations
func (w *WeatherService) getWeatherImpactConfig() *WeatherImpactConfig {
	return &WeatherImpactConfig{
		HighWindThreshold:    20.0,
		ModerateWindThreshold: 15.0,
		ColdTempThreshold:    50.0,
		WetBulbColdThreshold: 50.0,
	}
}

// initializeCourseLocations initializes known golf course locations
func initializeCourseLocations() map[string]*providers.WeatherLocation {
	// This would be loaded from a database or configuration file
	// For now, returning a few example courses
	return map[string]*providers.WeatherLocation{
		"augusta_national": {
			Latitude:  33.5030,
			Longitude: -82.0200,
			City:      "Augusta",
			State:     "GA",
			Country:   "US",
		},
		"pebble_beach": {
			Latitude:  36.5684,
			Longitude: -121.9493,
			City:      "Pebble Beach",
			State:     "CA",
			Country:   "US",
		},
		"pinehurst_no2": {
			Latitude:  35.1827,
			Longitude: -79.4659,
			City:      "Pinehurst",
			State:     "NC",
			Country:   "US",
		},
		"st_andrews": {
			Latitude:  56.3398,
			Longitude: -2.7967,
			City:      "St Andrews",
			State:     "Scotland",
			Country:   "UK",
		},
	}
}

// Helper structs for impact calculations
type windImpactResult struct {
	scoreImpact       float64
	varianceMultiplier float64
	advantage         float64
}

type tempImpactResult struct {
	scoreImpact       float64
	varianceMultiplier float64
}

type precipImpactResult struct {
	softConditions    bool
	distanceReduction float64
	scoreImpact       float64
}