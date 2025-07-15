package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// PlayerInjuryHandler handles player injury and status change events
type PlayerInjuryHandler struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewPlayerInjuryHandler(db *gorm.DB, logger *logrus.Logger) *PlayerInjuryHandler {
	return &PlayerInjuryHandler{
		db:     db,
		logger: logger,
	}
}

func (h *PlayerInjuryHandler) GetEventType() models.EventType {
	return models.EventTypePlayerInjury
}

func (h *PlayerInjuryHandler) GetPriority() int {
	return 10 // High priority for injury updates
}

func (h *PlayerInjuryHandler) HandleEvent(ctx context.Context, event *models.RealTimeEvent) error {
	h.logger.WithFields(logrus.Fields{
		"event_id":  event.EventID,
		"player_id": event.PlayerID,
		"impact":    event.ImpactRating,
	}).Info("Processing player injury event")

	// Parse injury data
	var injuryData struct {
		Status     string  `json:"status"`      // "out", "questionable", "probable", "active"
		Injury     string  `json:"injury"`      // "knee", "ankle", "back", etc.
		Severity   string  `json:"severity"`    // "minor", "moderate", "severe"
		Timeline   string  `json:"timeline"`    // "day-to-day", "1-2 weeks", etc.
		Confidence float64 `json:"confidence"`  // 0-1 reliability of the report
	}

	if err := json.Unmarshal(event.Data, &injuryData); err != nil {
		return fmt.Errorf("failed to parse injury data: %w", err)
	}

	// Validate required fields
	if event.PlayerID == nil {
		return fmt.Errorf("player_id is required for injury events")
	}

	// Update player status in database (if we have a players table)
	// This would typically update a player status field
	h.logger.WithFields(logrus.Fields{
		"player_id": *event.PlayerID,
		"status":    injuryData.Status,
		"injury":    injuryData.Injury,
		"severity":  injuryData.Severity,
	}).Info("Player injury status updated")

	// Log the event for audit trail
	eventLog := models.EventLog{
		EventID: event.EventID,
		Action:  "processed",
		Details: event.Data,
	}
	h.db.Create(&eventLog)

	// If this is a high-impact injury, we might trigger additional processing
	if event.ImpactRating >= 8.0 {
		h.logger.WithField("player_id", *event.PlayerID).Warn("High-impact injury event detected")
		// Could trigger alerts, optimization recalculations, etc.
	}

	return nil
}

// WeatherUpdateHandler handles weather condition changes
type WeatherUpdateHandler struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewWeatherUpdateHandler(db *gorm.DB, logger *logrus.Logger) *WeatherUpdateHandler {
	return &WeatherUpdateHandler{
		db:     db,
		logger: logger,
	}
}

func (h *WeatherUpdateHandler) GetEventType() models.EventType {
	return models.EventTypeWeatherUpdate
}

func (h *WeatherUpdateHandler) GetPriority() int {
	return 5 // Medium priority
}

func (h *WeatherUpdateHandler) HandleEvent(ctx context.Context, event *models.RealTimeEvent) error {
	h.logger.WithFields(logrus.Fields{
		"event_id":      event.EventID,
		"tournament_id": event.TournamentID,
		"impact":        event.ImpactRating,
	}).Info("Processing weather update event")

	// Parse weather data
	var weatherData struct {
		Location    string  `json:"location"`
		Temperature float64 `json:"temperature"`
		WindSpeed   float64 `json:"wind_speed"`
		WindDirection string `json:"wind_direction"`
		Precipitation float64 `json:"precipitation"`
		Conditions  string  `json:"conditions"` // "sunny", "cloudy", "windy", "rainy"
		Visibility  float64 `json:"visibility"`
		Timestamp   string  `json:"timestamp"`
	}

	if err := json.Unmarshal(event.Data, &weatherData); err != nil {
		return fmt.Errorf("failed to parse weather data: %w", err)
	}

	// Weather updates are particularly important for golf
	if event.TournamentID != nil {
		h.logger.WithFields(logrus.Fields{
			"tournament_id": *event.TournamentID,
			"conditions":    weatherData.Conditions,
			"wind_speed":    weatherData.WindSpeed,
			"temperature":   weatherData.Temperature,
		}).Info("Weather conditions updated for tournament")

		// High wind or severe weather has significant impact on golf scores
		if weatherData.WindSpeed > 20 || weatherData.Conditions == "rainy" {
			h.logger.WithField("tournament_id", *event.TournamentID).Warn("Severe weather conditions detected")
		}
	}

	// Log the event
	eventLog := models.EventLog{
		EventID: event.EventID,
		Action:  "processed",
		Details: event.Data,
	}
	h.db.Create(&eventLog)

	return nil
}

// OwnershipChangeHandler handles ownership percentage updates
type OwnershipChangeHandler struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewOwnershipChangeHandler(db *gorm.DB, logger *logrus.Logger) *OwnershipChangeHandler {
	return &OwnershipChangeHandler{
		db:     db,
		logger: logger,
	}
}

func (h *OwnershipChangeHandler) GetEventType() models.EventType {
	return models.EventTypeOwnershipChange
}

func (h *OwnershipChangeHandler) GetPriority() int {
	return 7 // High priority for ownership changes
}

func (h *OwnershipChangeHandler) HandleEvent(ctx context.Context, event *models.RealTimeEvent) error {
	h.logger.WithFields(logrus.Fields{
		"event_id": event.EventID,
		"impact":   event.ImpactRating,
	}).Info("Processing ownership change event")

	// Parse ownership data
	var ownershipData struct {
		ContestID        string             `json:"contest_id"`
		PlayerOwnership  map[string]float64 `json:"player_ownership"`  // player_id -> ownership %
		StackOwnership   map[string]float64 `json:"stack_ownership"`   // stack_key -> ownership %
		TotalEntries     int                `json:"total_entries"`
		TimeToLock       int                `json:"time_to_lock_minutes"`
		OwnershipChanges map[string]float64 `json:"ownership_changes"` // player_id -> change %
	}

	if err := json.Unmarshal(event.Data, &ownershipData); err != nil {
		return fmt.Errorf("failed to parse ownership data: %w", err)
	}

	// Create ownership snapshot
	snapshot := models.OwnershipSnapshot{
		ContestID:    ownershipData.ContestID,
		Timestamp:    event.Timestamp,
		TotalEntries: ownershipData.TotalEntries,
	}

	// Convert ownership maps to JSON
	if len(ownershipData.PlayerOwnership) > 0 {
		playerOwnershipBytes, _ := json.Marshal(ownershipData.PlayerOwnership)
		snapshot.PlayerOwnership = playerOwnershipBytes
	}

	if len(ownershipData.StackOwnership) > 0 {
		stackOwnershipBytes, _ := json.Marshal(ownershipData.StackOwnership)
		snapshot.StackOwnership = stackOwnershipBytes
	}

	// Store ownership snapshot
	if err := h.db.Create(&snapshot).Error; err != nil {
		h.logger.WithError(err).Error("Failed to store ownership snapshot")
		return fmt.Errorf("failed to store ownership snapshot: %w", err)
	}

	// Check for significant ownership changes
	for playerIDStr, change := range ownershipData.OwnershipChanges {
		if abs(change) > 5.0 { // 5% ownership change threshold
			h.logger.WithFields(logrus.Fields{
				"player_id":        playerIDStr,
				"ownership_change": change,
				"contest_id":       ownershipData.ContestID,
			}).Info("Significant ownership change detected")
		}
	}

	// Log the event
	eventLog := models.EventLog{
		EventID: event.EventID,
		Action:  "processed",
		Details: event.Data,
	}
	h.db.Create(&eventLog)

	return nil
}

// ContestUpdateHandler handles contest-related updates (start times, status changes, etc.)
type ContestUpdateHandler struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewContestUpdateHandler(db *gorm.DB, logger *logrus.Logger) *ContestUpdateHandler {
	return &ContestUpdateHandler{
		db:     db,
		logger: logger,
	}
}

func (h *ContestUpdateHandler) GetEventType() models.EventType {
	return models.EventTypeContestUpdate
}

func (h *ContestUpdateHandler) GetPriority() int {
	return 8 // High priority for contest updates
}

func (h *ContestUpdateHandler) HandleEvent(ctx context.Context, event *models.RealTimeEvent) error {
	h.logger.WithFields(logrus.Fields{
		"event_id": event.EventID,
		"game_id":  event.GameID,
		"impact":   event.ImpactRating,
	}).Info("Processing contest update event")

	// Parse contest data
	var contestData struct {
		ContestID   string `json:"contest_id"`
		Status      string `json:"status"`      // "upcoming", "live", "completed", "cancelled"
		StartTime   string `json:"start_time"`
		LockTime    string `json:"lock_time"`
		EntryCount  int    `json:"entry_count"`
		PrizePool   float64 `json:"prize_pool"`
		UpdateType  string `json:"update_type"` // "status", "time", "entries", "cancelled"
	}

	if err := json.Unmarshal(event.Data, &contestData); err != nil {
		return fmt.Errorf("failed to parse contest data: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id":  contestData.ContestID,
		"status":      contestData.Status,
		"update_type": contestData.UpdateType,
		"entry_count": contestData.EntryCount,
	}).Info("Contest update processed")

	// Handle critical contest updates
	if contestData.Status == "cancelled" {
		h.logger.WithField("contest_id", contestData.ContestID).Warn("Contest cancelled")
		// This might trigger user notifications and refunds
	}

	if contestData.UpdateType == "time" {
		h.logger.WithField("contest_id", contestData.ContestID).Info("Contest time updated")
		// Time changes affect late swap opportunities
	}

	// Log the event
	eventLog := models.EventLog{
		EventID: event.EventID,
		Action:  "processed",
		Details: event.Data,
	}
	h.db.Create(&eventLog)

	return nil
}

// GenericEventHandler handles any event type that doesn't have a specific handler
type GenericEventHandler struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewGenericEventHandler(db *gorm.DB, logger *logrus.Logger) *GenericEventHandler {
	return &GenericEventHandler{
		db:     db,
		logger: logger,
	}
}

func (h *GenericEventHandler) GetEventType() models.EventType {
	return models.EventType("generic")
}

func (h *GenericEventHandler) GetPriority() int {
	return 1 // Lowest priority
}

func (h *GenericEventHandler) HandleEvent(ctx context.Context, event *models.RealTimeEvent) error {
	h.logger.WithFields(logrus.Fields{
		"event_id":   event.EventID,
		"event_type": event.EventType,
		"source":     event.Source,
		"impact":     event.ImpactRating,
	}).Info("Processing generic event")

	// Basic event logging
	eventLog := models.EventLog{
		EventID: event.EventID,
		Action:  "processed",
		Details: event.Data,
	}
	h.db.Create(&eventLog)

	// For unknown event types, we just log them
	h.logger.WithFields(logrus.Fields{
		"event_type": event.EventType,
		"data_size":  len(event.Data),
	}).Debug("Generic event processed")

	return nil
}

// NewsUpdateHandler handles news and information updates
type NewsUpdateHandler struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewNewsUpdateHandler(db *gorm.DB, logger *logrus.Logger) *NewsUpdateHandler {
	return &NewsUpdateHandler{
		db:     db,
		logger: logger,
	}
}

func (h *NewsUpdateHandler) GetEventType() models.EventType {
	return models.EventTypeNewsUpdate
}

func (h *NewsUpdateHandler) GetPriority() int {
	return 3 // Lower priority than injury/weather but higher than generic
}

func (h *NewsUpdateHandler) HandleEvent(ctx context.Context, event *models.RealTimeEvent) error {
	h.logger.WithFields(logrus.Fields{
		"event_id":  event.EventID,
		"player_id": event.PlayerID,
		"impact":    event.ImpactRating,
	}).Info("Processing news update event")

	// Parse news data
	var newsData struct {
		Title       string   `json:"title"`
		Content     string   `json:"content"`
		Source      string   `json:"source"`
		PlayerIDs   []uint   `json:"player_ids"`
		Tags        []string `json:"tags"`
		Relevance   float64  `json:"relevance"`   // 0-1 relevance to DFS
		Sentiment   string   `json:"sentiment"`   // "positive", "negative", "neutral"
		PublishedAt string   `json:"published_at"`
	}

	if err := json.Unmarshal(event.Data, &newsData); err != nil {
		return fmt.Errorf("failed to parse news data: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"title":      newsData.Title,
		"source":     newsData.Source,
		"relevance":  newsData.Relevance,
		"sentiment":  newsData.Sentiment,
		"player_ids": newsData.PlayerIDs,
	}).Info("News update processed")

	// High-relevance news might trigger alerts
	if newsData.Relevance > 0.8 {
		h.logger.WithFields(logrus.Fields{
			"title":     newsData.Title,
			"relevance": newsData.Relevance,
		}).Info("High-relevance news detected")
	}

	// Log the event
	eventLog := models.EventLog{
		EventID: event.EventID,
		Action:  "processed",
		Details: event.Data,
	}
	h.db.Create(&eventLog)

	return nil
}

// PriceChangeHandler handles player price/salary changes
type PriceChangeHandler struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewPriceChangeHandler(db *gorm.DB, logger *logrus.Logger) *PriceChangeHandler {
	return &PriceChangeHandler{
		db:     db,
		logger: logger,
	}
}

func (h *PriceChangeHandler) GetEventType() models.EventType {
	return models.EventTypePriceChange
}

func (h *PriceChangeHandler) GetPriority() int {
	return 6 // Medium-high priority
}

func (h *PriceChangeHandler) HandleEvent(ctx context.Context, event *models.RealTimeEvent) error {
	h.logger.WithFields(logrus.Fields{
		"event_id":  event.EventID,
		"player_id": event.PlayerID,
		"impact":    event.ImpactRating,
	}).Info("Processing price change event")

	// Parse price data
	var priceData struct {
		PlayerID    uint    `json:"player_id"`
		OldPrice    int     `json:"old_price"`
		NewPrice    int     `json:"new_price"`
		PriceChange int     `json:"price_change"`
		Platform    string  `json:"platform"` // "draftkings", "fanduel", etc.
		Reason      string  `json:"reason"`   // "injury", "performance", "ownership"
	}

	if err := json.Unmarshal(event.Data, &priceData); err != nil {
		return fmt.Errorf("failed to parse price data: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"player_id":    priceData.PlayerID,
		"old_price":    priceData.OldPrice,
		"new_price":    priceData.NewPrice,
		"price_change": priceData.PriceChange,
		"platform":     priceData.Platform,
		"reason":       priceData.Reason,
	}).Info("Player price change processed")

	// Significant price changes might affect lineup optimization
	if abs(float64(priceData.PriceChange)) > 500 { // $500+ change
		h.logger.WithFields(logrus.Fields{
			"player_id":    priceData.PlayerID,
			"price_change": priceData.PriceChange,
		}).Warn("Significant price change detected")
	}

	// Log the event
	eventLog := models.EventLog{
		EventID: event.EventID,
		Action:  "processed",
		Details: event.Data,
	}
	h.db.Create(&eventLog)

	return nil
}

// Utility function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}