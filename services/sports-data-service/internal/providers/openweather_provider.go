package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// OpenWeatherProvider implements weather data retrieval from OpenWeatherMap API
type OpenWeatherProvider struct {
	client       *http.Client
	redisClient  *redis.Client
	apiKey       string
	baseURL      string
	cacheTTL     time.Duration
	rateLimit    *RateLimiter
}

// OpenWeatherResponse represents the API response from OpenWeatherMap
type OpenWeatherResponse struct {
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Name string `json:"name"`
}

// WeatherLocation represents a golf course location for weather lookup
type WeatherLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Country   string  `json:"country"`
}

// NewOpenWeatherProvider creates a new OpenWeather API client
func NewOpenWeatherProvider(apiKey string, redisClient *redis.Client) *OpenWeatherProvider {
	return &OpenWeatherProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		redisClient: redisClient,
		apiKey:      apiKey,
		baseURL:     "https://api.openweathermap.org/data/2.5",
		cacheTTL:    1 * time.Hour, // Cache weather data for 1 hour
		rateLimit:   NewRateLimiter(60, time.Minute), // 60 requests per minute
	}
}

// GetWeatherConditions retrieves current weather conditions for a location
func (p *OpenWeatherProvider) GetWeatherConditions(ctx context.Context, location *WeatherLocation) (*types.WeatherConditions, error) {
	if location == nil {
		return nil, fmt.Errorf("location is required")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("weather:%f,%f", location.Latitude, location.Longitude)
	if cached, err := p.getCachedWeather(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Check rate limit
	if !p.rateLimit.Allow() {
		return nil, fmt.Errorf("weather API rate limit exceeded")
	}

	// Build API URL
	apiURL, err := p.buildWeatherURL(location)
	if err != nil {
		return nil, fmt.Errorf("failed to build weather API URL: %w", err)
	}

	// Make API request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create weather request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch weather data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather response: %w", err)
	}

	var weatherResp OpenWeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		return nil, fmt.Errorf("failed to parse weather response: %w", err)
	}

	// Convert to WeatherConditions
	conditions := p.convertToWeatherConditions(&weatherResp)

	// Cache the result
	if err := p.cacheWeather(ctx, cacheKey, conditions); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to cache weather data: %v\n", err)
	}

	return conditions, nil
}

// GetWeatherForecast retrieves 5-day weather forecast for a location
func (p *OpenWeatherProvider) GetWeatherForecast(ctx context.Context, location *WeatherLocation) ([]types.WeatherConditions, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("forecast:%f,%f", location.Latitude, location.Longitude)
	if cached, err := p.getCachedForecast(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Check rate limit
	if !p.rateLimit.Allow() {
		return nil, fmt.Errorf("weather API rate limit exceeded")
	}

	// Build forecast API URL
	apiURL, err := p.buildForecastURL(location)
	if err != nil {
		return nil, fmt.Errorf("failed to build forecast API URL: %w", err)
	}

	// Make API request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create forecast request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast API returned status %d", resp.StatusCode)
	}

	// Parse forecast response (simplified - would need full implementation)
	var forecast []types.WeatherConditions
	// Implementation would parse the 5-day forecast response
	// For now, return empty slice
	
	// Cache the result
	if err := p.cacheForecast(ctx, cacheKey, forecast); err != nil {
		fmt.Printf("Failed to cache forecast data: %v\n", err)
	}

	return forecast, nil
}

// buildWeatherURL constructs the OpenWeatherMap current weather API URL
func (p *OpenWeatherProvider) buildWeatherURL(location *WeatherLocation) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("OpenWeather API key not configured")
	}

	params := url.Values{}
	params.Add("lat", strconv.FormatFloat(location.Latitude, 'f', 6, 64))
	params.Add("lon", strconv.FormatFloat(location.Longitude, 'f', 6, 64))
	params.Add("appid", p.apiKey)
	params.Add("units", "imperial") // Fahrenheit for US golf courses

	return fmt.Sprintf("%s/weather?%s", p.baseURL, params.Encode()), nil
}

// buildForecastURL constructs the OpenWeatherMap forecast API URL
func (p *OpenWeatherProvider) buildForecastURL(location *WeatherLocation) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("OpenWeather API key not configured")
	}

	params := url.Values{}
	params.Add("lat", strconv.FormatFloat(location.Latitude, 'f', 6, 64))
	params.Add("lon", strconv.FormatFloat(location.Longitude, 'f', 6, 64))
	params.Add("appid", p.apiKey)
	params.Add("units", "imperial")

	return fmt.Sprintf("%s/forecast?%s", p.baseURL, params.Encode()), nil
}

// convertToWeatherConditions converts OpenWeather response to our format
func (p *OpenWeatherProvider) convertToWeatherConditions(resp *OpenWeatherResponse) *types.WeatherConditions {
	var windDir string
	if resp.Wind.Deg >= 0 && resp.Wind.Deg <= 360 {
		// Convert degrees to cardinal direction
		dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
		index := int((float64(resp.Wind.Deg) + 11.25) / 22.5) % 16
		windDir = dirs[index]
	}

	var conditions string
	if len(resp.Weather) > 0 {
		conditions = resp.Weather[0].Main
	}

	return &types.WeatherConditions{
		Temperature: int(resp.Main.Temp),
		WindSpeed:   int(resp.Wind.Speed),
		WindDir:     windDir,
		Conditions:  conditions,
		Humidity:    resp.Main.Humidity,
	}
}

// getCachedWeather retrieves weather data from cache
func (p *OpenWeatherProvider) getCachedWeather(ctx context.Context, key string) (*types.WeatherConditions, error) {
	if p.redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	data, err := p.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var conditions types.WeatherConditions
	if err := json.Unmarshal([]byte(data), &conditions); err != nil {
		return nil, err
	}

	return &conditions, nil
}

// cacheWeather stores weather data in cache
func (p *OpenWeatherProvider) cacheWeather(ctx context.Context, key string, conditions *types.WeatherConditions) error {
	if p.redisClient == nil {
		return nil // No caching if Redis not available
	}

	data, err := json.Marshal(conditions)
	if err != nil {
		return err
	}

	return p.redisClient.Set(ctx, key, data, p.cacheTTL).Err()
}

// getCachedForecast retrieves forecast data from cache
func (p *OpenWeatherProvider) getCachedForecast(ctx context.Context, key string) ([]types.WeatherConditions, error) {
	if p.redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	data, err := p.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var forecast []types.WeatherConditions
	if err := json.Unmarshal([]byte(data), &forecast); err != nil {
		return nil, err
	}

	return forecast, nil
}

// cacheForecast stores forecast data in cache
func (p *OpenWeatherProvider) cacheForecast(ctx context.Context, key string, forecast []types.WeatherConditions) error {
	if p.redisClient == nil {
		return nil // No caching if Redis not available
	}

	data, err := json.Marshal(forecast)
	if err != nil {
		return err
	}

	return p.redisClient.Set(ctx, key, data, p.cacheTTL).Err()
}

// RateLimiter implements simple rate limiting
type RateLimiter struct {
	requests int
	window   time.Duration
	tokens   chan struct{}
	reset    time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: requests,
		window:   window,
		tokens:   make(chan struct{}, requests),
		reset:    time.Now().Add(window),
	}

	// Fill the bucket initially
	for i := 0; i < requests; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refill goroutine
	go rl.refill()

	return rl
}

// Allow checks if a request is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// refill replenishes the token bucket
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.window / time.Duration(rl.requests))
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}
}