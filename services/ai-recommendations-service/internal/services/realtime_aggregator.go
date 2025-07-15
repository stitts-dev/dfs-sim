package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
)

// RealtimeAggregator collects and processes real-time data affecting DFS recommendations
type RealtimeAggregator struct {
	weatherService   *WeatherAPI
	injuryService    *InjuryAPI
	oddsService      *OddsAPI
	ownershipService *OwnershipTracker
	newsService      *NewsAggregator
	cache            *CacheService
	logger           *logrus.Logger
	subscribers      map[uint][]chan models.RealtimeDataPoint
	mu               sync.RWMutex
	stopChan         chan struct{}
	isRunning        bool
}

// WeatherAPI handles weather data integration
type WeatherAPI struct {
	apiKey    string
	baseURL   string
	logger    *logrus.Logger
	lastFetch time.Time
}

// InjuryAPI handles injury report monitoring
type InjuryAPI struct {
	sources []InjurySource
	logger  *logrus.Logger
}

// InjurySource represents different injury reporting sources
type InjurySource struct {
	Name        string
	URL         string
	Reliability float64 // 0-1 reliability score
	UpdateFreq  time.Duration
}

// OddsAPI handles betting odds integration
type OddsAPI struct {
	apiKey    string
	baseURL   string
	logger    *logrus.Logger
	providers []string
}

// OwnershipTracker monitors ownership percentage changes
type OwnershipTracker struct {
	cache          *CacheService
	logger         *logrus.Logger
	thresholds     map[string]float64 // Alert thresholds
	lastSnapshots  map[uint]map[uint]float64 // contestID -> playerID -> ownership
	mu             sync.RWMutex
}

// NewsAggregator collects breaking news affecting players
type NewsAggregator struct {
	sources []NewsSource
	logger  *logrus.Logger
	filters []NewsFilter
}

// NewsSource represents news providers
type NewsSource struct {
	Name        string
	URL         string
	RSS         string
	Reliability float64
	SportTypes  []string
}

// NewsFilter defines what news is relevant
type NewsFilter struct {
	Keywords    []string
	PlayerNames []string
	Importance  string // "critical", "high", "medium", "low"
}

// WeatherData represents weather information
type WeatherData struct {
	Location    string    `json:"location"`
	Temperature float64   `json:"temperature"`
	WindSpeed   float64   `json:"wind_speed"`
	WindDirection string  `json:"wind_direction"`
	Humidity    float64   `json:"humidity"`
	Pressure    float64   `json:"pressure"`
	Conditions  string    `json:"conditions"`
	Visibility  float64   `json:"visibility"`
	Forecast    []HourlyForecast `json:"forecast"`
	Timestamp   time.Time `json:"timestamp"`
}

// HourlyForecast represents hourly weather predictions
type HourlyForecast struct {
	Time         time.Time `json:"time"`
	Temperature  float64   `json:"temperature"`
	WindSpeed    float64   `json:"wind_speed"`
	Precipitation float64  `json:"precipitation"`
	Conditions   string    `json:"conditions"`
}

// InjuryReport represents injury information
type InjuryReport struct {
	PlayerID    uint      `json:"player_id"`
	PlayerName  string    `json:"player_name"`
	InjuryType  string    `json:"injury_type"`
	Severity    string    `json:"severity"` // "questionable", "doubtful", "out"
	Status      string    `json:"status"`
	LastUpdate  time.Time `json:"last_update"`
	Source      string    `json:"source"`
	Reliability float64   `json:"reliability"`
	ImpactRating float64  `json:"impact_rating"` // -5 to +5 DFS impact
}

// OwnershipSnapshot represents ownership data at a point in time
type OwnershipSnapshot struct {
	ContestID   uint                   `json:"contest_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Ownership   map[uint]float64       `json:"ownership"` // playerID -> ownership %
	Trends      map[uint]string        `json:"trends"`    // playerID -> "rising"/"falling"/"stable"
	Changes     map[uint]float64       `json:"changes"`   // playerID -> change since last snapshot
}

// NewsItem represents a news article or update
type NewsItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Source      string    `json:"source"`
	PlayerIDs   []uint    `json:"player_ids"`
	Importance  string    `json:"importance"`
	ImpactRating float64  `json:"impact_rating"`
	Timestamp   time.Time `json:"timestamp"`
	Keywords    []string  `json:"keywords"`
}

// NewRealtimeAggregator creates a new real-time data aggregator
func NewRealtimeAggregator(cache *CacheService, logger *logrus.Logger) *RealtimeAggregator {
	ra := &RealtimeAggregator{
		cache:       cache,
		logger:      logger,
		subscribers: make(map[uint][]chan models.RealtimeDataPoint),
		stopChan:    make(chan struct{}),
		isRunning:   false,
	}

	// Initialize sub-services
	ra.weatherService = NewWeatherAPI(logger)
	ra.injuryService = NewInjuryAPI(logger)
	ra.oddsService = NewOddsAPI(logger)
	ra.ownershipService = NewOwnershipTracker(cache, logger)
	ra.newsService = NewNewsAggregator(logger)

	return ra
}

// Start begins the real-time data collection
func (ra *RealtimeAggregator) Start(ctx context.Context) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if ra.isRunning {
		return fmt.Errorf("aggregator is already running")
	}

	ra.isRunning = true
	
	// Start data collection goroutines
	go ra.runWeatherMonitoring(ctx)
	go ra.runInjuryMonitoring(ctx)
	go ra.runOwnershipMonitoring(ctx)
	go ra.runNewsMonitoring(ctx)
	go ra.runOddsMonitoring(ctx)

	ra.logger.Info("Real-time data aggregator started")
	return nil
}

// Stop stops the real-time data collection
func (ra *RealtimeAggregator) Stop() {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if !ra.isRunning {
		return
	}

	close(ra.stopChan)
	ra.isRunning = false
	
	ra.logger.Info("Real-time data aggregator stopped")
}

// Subscribe allows services to receive real-time updates for specific contests
func (ra *RealtimeAggregator) Subscribe(contestID uint) <-chan models.RealtimeDataPoint {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	updateChan := make(chan models.RealtimeDataPoint, 100)
	
	if ra.subscribers[contestID] == nil {
		ra.subscribers[contestID] = make([]chan models.RealtimeDataPoint, 0)
	}
	
	ra.subscribers[contestID] = append(ra.subscribers[contestID], updateChan)
	
	ra.logger.WithField("contest_id", contestID).Debug("New subscriber added")
	
	return updateChan
}

// Unsubscribe removes a subscription
func (ra *RealtimeAggregator) Unsubscribe(contestID uint, updateChan <-chan models.RealtimeDataPoint) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if subscribers, exists := ra.subscribers[contestID]; exists {
		for i, ch := range subscribers {
			if ch == updateChan {
				// Remove channel from slice
				ra.subscribers[contestID] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				break
			}
		}
		
		// Clean up empty subscription lists
		if len(ra.subscribers[contestID]) == 0 {
			delete(ra.subscribers, contestID)
		}
	}
}

// StreamUpdates returns a channel of real-time updates for a specific contest
func (ra *RealtimeAggregator) StreamUpdates(ctx context.Context, contestID uint) <-chan models.RealtimeDataPoint {
	return ra.Subscribe(contestID)
}

// broadcastUpdate sends updates to all subscribers of a contest
func (ra *RealtimeAggregator) broadcastUpdate(contestID uint, update models.RealtimeDataPoint) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	if subscribers, exists := ra.subscribers[contestID]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- update:
			default:
				// Channel is full, skip this update
				ra.logger.Warn("Subscriber channel full, dropping update")
			}
		}
	}

	// Also cache the update
	ra.cacheRealtimeData(update)
}

// Weather monitoring
func (ra *RealtimeAggregator) runWeatherMonitoring(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute) // Check weather every 15 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ra.stopChan:
			return
		case <-ticker.C:
			ra.updateWeatherData(ctx)
		}
	}
}

func (ra *RealtimeAggregator) updateWeatherData(ctx context.Context) {
	// Get active contest locations
	locations := ra.getActiveContestLocations()
	
	for _, location := range locations {
		weather, err := ra.weatherService.GetCurrentWeather(location)
		if err != nil {
			ra.logger.WithError(err).WithField("location", location).Warn("Failed to fetch weather data")
			continue
		}

		// Create real-time data point
		update := models.RealtimeDataPoint{
			PlayerID:     0, // Weather affects all players at location
			ContestID:    0, // Will be set when broadcasting to specific contests
			DataType:     "weather",
			Value:        ra.marshalData(weather),
			Confidence:   0.9, // Weather data is generally reliable
			ImpactRating: func(v float64) *float64 { return &v }(ra.calculateWeatherImpact(weather)),
			Source:       "weather_api",
			Timestamp:    time.Now(),
			ExpiresAt:    &[]time.Time{time.Now().Add(30 * time.Minute)}[0],
		}

		// Broadcast to relevant contests
		ra.broadcastWeatherUpdate(update, location)
	}
}

// Injury monitoring
func (ra *RealtimeAggregator) runInjuryMonitoring(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Check injuries every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ra.stopChan:
			return
		case <-ticker.C:
			ra.updateInjuryReports(ctx)
		}
	}
}

func (ra *RealtimeAggregator) updateInjuryReports(ctx context.Context) {
	reports, err := ra.injuryService.GetLatestReports()
	if err != nil {
		ra.logger.WithError(err).Warn("Failed to fetch injury reports")
		return
	}

	for _, report := range reports {
		update := models.RealtimeDataPoint{
			PlayerID:     report.PlayerID,
			ContestID:    0, // Will be determined when broadcasting
			DataType:     "injury",
			Value:        ra.marshalData(report),
			Confidence:   report.Reliability,
			ImpactRating: func(v float64) *float64 { return &v }(report.ImpactRating),
			Source:       report.Source,
			Timestamp:    time.Now(),
			ExpiresAt:    &[]time.Time{time.Now().Add(2 * time.Hour)}[0],
		}

		// Broadcast to contests containing this player
		ra.broadcastPlayerUpdate(update, report.PlayerID)
	}
}

// Ownership monitoring
func (ra *RealtimeAggregator) runOwnershipMonitoring(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute) // Check ownership every 2 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ra.stopChan:
			return
		case <-ticker.C:
			ra.updateOwnershipData(ctx)
		}
	}
}

func (ra *RealtimeAggregator) updateOwnershipData(ctx context.Context) {
	activeContests := ra.getActiveContests()
	
	for _, contestID := range activeContests {
		snapshot, err := ra.ownershipService.GetCurrentSnapshot(contestID)
		if err != nil {
			ra.logger.WithError(err).WithField("contest_id", contestID).Warn("Failed to fetch ownership data")
			continue
		}

		// Check for significant ownership changes
		changes := ra.ownershipService.DetectSignificantChanges(contestID, snapshot)
		
		for playerID, change := range changes {
			update := models.RealtimeDataPoint{
				PlayerID:     playerID,
				ContestID:    contestID,
				DataType:     "ownership",
				Value:        ra.marshalData(map[string]interface{}{
					"current_ownership": snapshot.Ownership[playerID],
					"change": change,
					"trend": snapshot.Trends[playerID],
				}),
				Confidence:   0.8, // Ownership data has some uncertainty
				ImpactRating: func(v float64) *float64 { return &v }(ra.calculateOwnershipImpact(change)),
				Source:       "ownership_tracker",
				Timestamp:    time.Now(),
				ExpiresAt:    &[]time.Time{time.Now().Add(5 * time.Minute)}[0],
			}

			ra.broadcastUpdate(contestID, update)
		}
	}
}

// News monitoring
func (ra *RealtimeAggregator) runNewsMonitoring(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Minute) // Check news every 3 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ra.stopChan:
			return
		case <-ticker.C:
			ra.updateNewsData(ctx)
		}
	}
}

func (ra *RealtimeAggregator) updateNewsData(ctx context.Context) {
	newsItems, err := ra.newsService.GetLatestNews()
	if err != nil {
		ra.logger.WithError(err).Warn("Failed to fetch news data")
		return
	}

	for _, item := range newsItems {
		for _, playerID := range item.PlayerIDs {
			update := models.RealtimeDataPoint{
				PlayerID:     playerID,
				ContestID:    0, // Will be determined when broadcasting
				DataType:     "news",
				Value:        ra.marshalData(item),
				Confidence:   0.7, // News reliability varies
				ImpactRating: func(v float64) *float64 { return &v }(item.ImpactRating),
				Source:       item.Source,
				Timestamp:    time.Now(),
				ExpiresAt:    &[]time.Time{time.Now().Add(30 * time.Minute)}[0],
			}

			ra.broadcastPlayerUpdate(update, playerID)
		}
	}
}

// Odds monitoring
func (ra *RealtimeAggregator) runOddsMonitoring(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute) // Check odds every 10 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ra.stopChan:
			return
		case <-ticker.C:
			ra.updateOddsData(ctx)
		}
	}
}

func (ra *RealtimeAggregator) updateOddsData(ctx context.Context) {
	// Implementation would fetch odds data and broadcast changes
	ra.logger.Debug("Updating odds data (implementation needed)")
}

// Helper methods
func (ra *RealtimeAggregator) marshalData(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		ra.logger.WithError(err).Error("Failed to marshal real-time data")
		return json.RawMessage("{}")
	}
	return json.RawMessage(bytes)
}

func (ra *RealtimeAggregator) cacheRealtimeData(update models.RealtimeDataPoint) {
	key := fmt.Sprintf("realtime:%s:%d:%d", update.DataType, update.PlayerID, update.ContestID)
	ttl := 15 * time.Minute
	if update.ExpiresAt != nil {
		ttl = time.Until(*update.ExpiresAt)
	}
	
	if err := ra.cache.Set(key, update, ttl); err != nil {
		ra.logger.WithError(err).Warn("Failed to cache real-time data")
	}
}

func (ra *RealtimeAggregator) broadcastPlayerUpdate(update models.RealtimeDataPoint, playerID uint) {
	// Find contests containing this player and broadcast
	contests := ra.getContestsForPlayer(playerID)
	for _, contestID := range contests {
		update.ContestID = contestID
		ra.broadcastUpdate(contestID, update)
	}
}

func (ra *RealtimeAggregator) broadcastWeatherUpdate(update models.RealtimeDataPoint, location string) {
	// Find contests at this location and broadcast
	contests := ra.getContestsAtLocation(location)
	for _, contestID := range contests {
		update.ContestID = contestID
		ra.broadcastUpdate(contestID, update)
	}
}

// Placeholder methods that would integrate with the main database
func (ra *RealtimeAggregator) getActiveContestLocations() []string {
	// Would query database for active contest locations
	return []string{"TPC Sawgrass", "Augusta National", "Riviera"}
}

func (ra *RealtimeAggregator) getActiveContests() []uint {
	// Would query database for active contest IDs
	return []uint{1, 2, 3}
}

func (ra *RealtimeAggregator) getContestsForPlayer(playerID uint) []uint {
	// Would query database for contests containing this player
	return []uint{1, 2, 3}
}

func (ra *RealtimeAggregator) getContestsAtLocation(location string) []uint {
	// Would query database for contests at this location
	return []uint{1, 2, 3}
}

// Impact calculation methods
func (ra *RealtimeAggregator) calculateWeatherImpact(weather *WeatherData) float64 {
	impact := 0.0
	
	// High wind increases difficulty
	if weather.WindSpeed > 15 {
		impact -= 2.0
	} else if weather.WindSpeed > 10 {
		impact -= 1.0
	}
	
	// Rain makes conditions difficult
	if strings.Contains(strings.ToLower(weather.Conditions), "rain") {
		impact -= 1.5
	}
	
	// Perfect conditions help scoring
	if weather.WindSpeed < 5 && !strings.Contains(strings.ToLower(weather.Conditions), "rain") {
		impact += 1.0
	}
	
	// Clamp between -5 and +5
	if impact < -5 {
		impact = -5
	} else if impact > 5 {
		impact = 5
	}
	
	return impact
}

func (ra *RealtimeAggregator) calculateOwnershipImpact(change float64) float64 {
	// Significant ownership changes create leverage opportunities
	absChange := change
	if absChange < 0 {
		absChange = -absChange
	}
	
	if absChange > 10 {
		return 3.0 // High leverage opportunity
	} else if absChange > 5 {
		return 2.0 // Medium leverage opportunity
	} else if absChange > 2 {
		return 1.0 // Low leverage opportunity
	}
	
	return 0.0 // No significant impact
}

// GetLatestData returns the most recent real-time data for a player
func (ra *RealtimeAggregator) GetLatestData(playerID uint, dataType string) (*models.RealtimeDataPoint, error) {
	key := fmt.Sprintf("realtime:%s:%d:*", dataType, playerID)
	
	var latestData models.RealtimeDataPoint
	err := ra.cache.Get(key, &latestData)
	if err != nil {
		return nil, fmt.Errorf("no recent data found: %w", err)
	}
	
	return &latestData, nil
}

// IsHealthy checks if the aggregator is functioning properly
func (ra *RealtimeAggregator) IsHealthy() bool {
	return ra.isRunning && ra.cache.IsHealthy()
}

// Stub implementations for sub-services (would be fully implemented)
func NewWeatherAPI(logger *logrus.Logger) *WeatherAPI {
	return &WeatherAPI{logger: logger}
}

func (w *WeatherAPI) GetCurrentWeather(location string) (*WeatherData, error) {
	// Stub implementation
	return &WeatherData{
		Location: location,
		Temperature: 72.0,
		WindSpeed: 8.0,
		Conditions: "Partly Cloudy",
		Timestamp: time.Now(),
	}, nil
}

func NewInjuryAPI(logger *logrus.Logger) *InjuryAPI {
	return &InjuryAPI{logger: logger}
}

func (i *InjuryAPI) GetLatestReports() ([]InjuryReport, error) {
	// Stub implementation
	return []InjuryReport{}, nil
}

func NewOddsAPI(logger *logrus.Logger) *OddsAPI {
	return &OddsAPI{logger: logger}
}

func NewOwnershipTracker(cache *CacheService, logger *logrus.Logger) *OwnershipTracker {
	return &OwnershipTracker{
		cache: cache,
		logger: logger,
		thresholds: map[string]float64{
			"significant_change": 5.0,
			"major_change": 10.0,
		},
		lastSnapshots: make(map[uint]map[uint]float64),
	}
}

func (o *OwnershipTracker) GetCurrentSnapshot(contestID uint) (*OwnershipSnapshot, error) {
	// Stub implementation
	return &OwnershipSnapshot{
		ContestID: contestID,
		Timestamp: time.Now(),
		Ownership: make(map[uint]float64),
		Trends: make(map[uint]string),
		Changes: make(map[uint]float64),
	}, nil
}

func (o *OwnershipTracker) DetectSignificantChanges(contestID uint, snapshot *OwnershipSnapshot) map[uint]float64 {
	// Stub implementation
	return make(map[uint]float64)
}

func NewNewsAggregator(logger *logrus.Logger) *NewsAggregator {
	return &NewsAggregator{logger: logger}
}

func (n *NewsAggregator) GetLatestNews() ([]NewsItem, error) {
	// Stub implementation
	return []NewsItem{}, nil
}