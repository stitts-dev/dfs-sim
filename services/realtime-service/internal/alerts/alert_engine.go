package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// AlertEngine manages real-time alert generation and delivery
type AlertEngine struct {
	db              *gorm.DB
	redisClient     *redis.Client
	logger          *logrus.Logger
	deliveryManager *DeliveryManager
	rateLimiter     *AlertRateLimiter
	
	// Alert rules cache
	rulesCache      map[int][]*models.AlertRule // user_id -> rules
	rulesMutex      sync.RWMutex
	cacheExpiry     time.Time
	
	// Alert queue
	alertQueue      chan AlertRequest
	deliveryQueue   chan DeliveryRequest
	
	// Statistics
	alertStats      *AlertStats
	statsMutex      sync.Mutex
	
	// Configuration
	config          *AlertConfig
	
	// Control channels
	stopChan        chan struct{}
	isRunning       bool
}

// AlertRequest represents a request to generate alerts
type AlertRequest struct {
	Event     models.RealTimeEvent
	Timestamp time.Time
}

// DeliveryRequest represents a request to deliver an alert
type DeliveryRequest struct {
	Alert    models.Alert
	UserID   int
	Channels []models.DeliveryChannel
	Priority string
}

// AlertStats contains alert system metrics
type AlertStats struct {
	AlertsGenerated   int64 `json:"alerts_generated"`
	AlertsDelivered   int64 `json:"alerts_delivered"`
	AlertsFailed      int64 `json:"alerts_failed"`
	AlertsRateLimited int64 `json:"alerts_rate_limited"`
	RulesEvaluated    int64 `json:"rules_evaluated"`
	RulesMatched      int64 `json:"rules_matched"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	LastProcessTime   time.Time `json:"last_process_time"`
}

// AlertConfig contains configuration for the alert engine
type AlertConfig struct {
	MaxQueueSize         int           `json:"max_queue_size"`
	ProcessingWorkers    int           `json:"processing_workers"`
	DeliveryWorkers      int           `json:"delivery_workers"`
	RuleCacheInterval    time.Duration `json:"rule_cache_interval"`
	DefaultRateLimit     int           `json:"default_rate_limit"`     // alerts per hour
	MaxDeliveryRetries   int           `json:"max_delivery_retries"`
	DeliveryTimeout      time.Duration `json:"delivery_timeout"`
	EnableHighPriority   bool          `json:"enable_high_priority"`
}

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// NewAlertEngine creates a new alert engine
func NewAlertEngine(db *gorm.DB, redisClient *redis.Client, logger *logrus.Logger) *AlertEngine {
	config := &AlertConfig{
		MaxQueueSize:         10000,
		ProcessingWorkers:    5,
		DeliveryWorkers:      10,
		RuleCacheInterval:    5 * time.Minute,
		DefaultRateLimit:     20, // 20 alerts per hour per user
		MaxDeliveryRetries:   3,
		DeliveryTimeout:      30 * time.Second,
		EnableHighPriority:   true,
	}
	
	engine := &AlertEngine{
		db:              db,
		redisClient:     redisClient,
		logger:          logger,
		rulesCache:      make(map[int][]*models.AlertRule),
		alertQueue:      make(chan AlertRequest, config.MaxQueueSize),
		deliveryQueue:   make(chan DeliveryRequest, config.MaxQueueSize*2),
		alertStats:      &AlertStats{},
		config:          config,
		stopChan:        make(chan struct{}),
	}
	
	// Initialize sub-components
	engine.deliveryManager = NewDeliveryManager(redisClient, logger)
	engine.rateLimiter = NewAlertRateLimiter(redisClient, config.DefaultRateLimit, logger)
	
	return engine
}

// Start begins alert processing
func (ae *AlertEngine) Start(ctx context.Context) error {
	ae.logger.Info("Starting alert engine")
	ae.isRunning = true
	
	// Start worker pools
	for i := 0; i < ae.config.ProcessingWorkers; i++ {
		go ae.alertProcessingWorker(ctx, i)
	}
	
	for i := 0; i < ae.config.DeliveryWorkers; i++ {
		go ae.alertDeliveryWorker(ctx, i)
	}
	
	// Start rule cache refresher
	go ae.ruleCacheRefresher(ctx)
	
	// Wait for stop signal
	<-ae.stopChan
	ae.logger.Info("Alert engine stopped")
	ae.isRunning = false
	
	return nil
}

// Stop stops the alert engine
func (ae *AlertEngine) Stop() {
	if ae.isRunning {
		close(ae.stopChan)
	}
}

// ProcessEvent processes a real-time event for alert generation
func (ae *AlertEngine) ProcessEvent(event models.RealTimeEvent) error {
	request := AlertRequest{
		Event:     event,
		Timestamp: time.Now(),
	}
	
	select {
	case ae.alertQueue <- request:
		return nil
	default:
		ae.incrementFailedStats()
		return fmt.Errorf("alert queue is full")
	}
}

// alertProcessingWorker processes alert requests
func (ae *AlertEngine) alertProcessingWorker(ctx context.Context, workerID int) {
	ae.logger.WithField("worker_id", workerID).Info("Starting alert processing worker")
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ae.stopChan:
			return
		case request := <-ae.alertQueue:
			ae.processAlertRequest(request)
		}
	}
}

// processAlertRequest processes a single alert request
func (ae *AlertEngine) processAlertRequest(request AlertRequest) {
	startTime := time.Now()
	
	// Get all active alert rules
	allRules := ae.getAllActiveRules()
	
	for userID, userRules := range allRules {
		ae.incrementRulesEvaluatedStats(int64(len(userRules)))
		
		for _, rule := range userRules {
			if ae.shouldTriggerAlert(request.Event, rule) {
				ae.incrementRulesMatchedStats()
				
				// Check rate limit
				if !ae.rateLimiter.CanSendAlert(userID, rule.RuleID) {
					ae.incrementRateLimitedStats()
					ae.logger.WithFields(logrus.Fields{
						"user_id": userID,
						"rule_id": rule.RuleID,
					}).Debug("Alert rate limited")
					continue
				}
				
				// Generate alert
				alert := ae.generateAlert(request.Event, rule, userID)
				
				// Queue for delivery
				deliveryRequest := DeliveryRequest{
					Alert:    alert,
					UserID:   userID,
					Channels: convertToDeliveryChannels(rule.DeliveryChannels),
					Priority: ae.calculatePriority(request.Event, rule),
				}
				
				select {
				case ae.deliveryQueue <- deliveryRequest:
					ae.incrementGeneratedStats()
				default:
					ae.incrementFailedStats()
					ae.logger.Warn("Delivery queue is full, dropping alert")
				}
			}
		}
	}
	
	// Update processing time stats
	ae.updateProcessingTime(time.Since(startTime))
}

// shouldTriggerAlert determines if an event should trigger an alert rule
func (ae *AlertEngine) shouldTriggerAlert(event models.RealTimeEvent, rule *models.AlertRule) bool {
	// Check if rule is active
	if !rule.IsActive {
		return false
	}
	
	// Check impact threshold
	if event.ImpactRating < rule.ImpactThreshold {
		return false
	}
	
	// Check event type filter
	if len(rule.EventTypes) > 0 {
		eventTypeMatch := false
		eventTypeStr := string(event.EventType)
		for _, allowedType := range rule.EventTypes {
			if allowedType == eventTypeStr {
				eventTypeMatch = true
				break
			}
		}
		if !eventTypeMatch {
			return false
		}
	}
	
	// Check sports filter
	if len(rule.Sports) > 0 {
		// This would need sport information from the event
		// For now, assume golf events match golf sport filter
		sportsMatch := false
		for _, sport := range rule.Sports {
			if sport == "golf" { // Simplified check
				sportsMatch = true
				break
			}
		}
		if !sportsMatch {
			return false
		}
	}
	
	return true
}

// generateAlert creates an alert from an event and rule
func (ae *AlertEngine) generateAlert(event models.RealTimeEvent, rule *models.AlertRule, userID int) models.Alert {
	alert := models.Alert{
		UserID:    userID,
		RuleID:    rule.RuleID,
		EventID:   event.EventID,
		Title:     ae.generateAlertTitle(event),
		Message:   ae.generateAlertMessage(event, rule),
		Priority:  ae.calculatePriority(event, rule),
		Channels:  convertToDeliveryChannels(rule.DeliveryChannels),
		CreatedAt: time.Now(),
	}
	
	return alert
}

// generateAlertTitle generates a title for an alert
func (ae *AlertEngine) generateAlertTitle(event models.RealTimeEvent) string {
	switch event.EventType {
	case models.EventTypePlayerInjury:
		return "Player Injury Update"
	case models.EventTypeWeatherUpdate:
		return "Weather Alert"
	case models.EventTypeOwnershipChange:
		return "Ownership Change Alert"
	case models.EventTypeContestUpdate:
		return "Contest Update"
	case models.EventTypePriceChange:
		return "Price Change Alert"
	case models.EventTypeNewsUpdate:
		return "Breaking News"
	default:
		return "DFS Alert"
	}
}

// generateAlertMessage generates a message for an alert
func (ae *AlertEngine) generateAlertMessage(event models.RealTimeEvent, rule *models.AlertRule) string {
	baseMessage := ""
	
	switch event.EventType {
	case models.EventTypePlayerInjury:
		baseMessage = ae.generateInjuryMessage(event)
	case models.EventTypeWeatherUpdate:
		baseMessage = ae.generateWeatherMessage(event)
	case models.EventTypeOwnershipChange:
		baseMessage = ae.generateOwnershipMessage(event)
	case models.EventTypeContestUpdate:
		baseMessage = ae.generateContestMessage(event)
	case models.EventTypePriceChange:
		baseMessage = ae.generatePriceMessage(event)
	case models.EventTypeNewsUpdate:
		baseMessage = ae.generateNewsMessage(event)
	default:
		baseMessage = fmt.Sprintf("Event: %s", event.EventType)
	}
	
	// Add impact information
	if event.ImpactRating >= 8.0 {
		baseMessage += " (HIGH IMPACT)"
	} else if event.ImpactRating >= 5.0 {
		baseMessage += " (Medium Impact)"
	}
	
	return baseMessage
}

// Message generation helpers
func (ae *AlertEngine) generateInjuryMessage(event models.RealTimeEvent) string {
	// Parse injury data from event
	var injuryData map[string]interface{}
	if err := json.Unmarshal(event.Data, &injuryData); err == nil {
		status := getString(injuryData, "status")
		injury := getString(injuryData, "injury")
		
		if event.PlayerID != nil {
			return fmt.Sprintf("Player %d injury update: %s (%s)", *event.PlayerID, status, injury)
		}
		
		return fmt.Sprintf("Player injury update: %s (%s)", status, injury)
	}
	
	return "Player injury status change"
}

func (ae *AlertEngine) generateWeatherMessage(event models.RealTimeEvent) string {
	var weatherData map[string]interface{}
	if err := json.Unmarshal(event.Data, &weatherData); err == nil {
		conditions := getString(weatherData, "conditions")
		windSpeed := getFloat64(weatherData, "wind_speed")
		
		if windSpeed > 20 {
			return fmt.Sprintf("Severe weather alert: %s with winds %.1f mph", conditions, windSpeed)
		}
		
		return fmt.Sprintf("Weather update: %s", conditions)
	}
	
	return "Weather conditions changed"
}

func (ae *AlertEngine) generateOwnershipMessage(event models.RealTimeEvent) string {
	var ownershipData map[string]interface{}
	if err := json.Unmarshal(event.Data, &ownershipData); err == nil {
		contestID := getString(ownershipData, "contest_id")
		return fmt.Sprintf("Ownership changes detected in contest %s", contestID)
	}
	
	return "Ownership percentages updated"
}

func (ae *AlertEngine) generateContestMessage(event models.RealTimeEvent) string {
	var contestData map[string]interface{}
	if err := json.Unmarshal(event.Data, &contestData); err == nil {
		status := getString(contestData, "status")
		updateType := getString(contestData, "update_type")
		
		return fmt.Sprintf("Contest %s: %s", updateType, status)
	}
	
	return "Contest information updated"
}

func (ae *AlertEngine) generatePriceMessage(event models.RealTimeEvent) string {
	var priceData map[string]interface{}
	if err := json.Unmarshal(event.Data, &priceData); err == nil {
		priceChange := getFloat64(priceData, "price_change")
		platform := getString(priceData, "platform")
		
		direction := "increased"
		if priceChange < 0 {
			direction = "decreased"
		}
		
		if event.PlayerID != nil {
			return fmt.Sprintf("Player %d price %s by $%.0f on %s", *event.PlayerID, direction, abs(priceChange), platform)
		}
		
		return fmt.Sprintf("Player price %s by $%.0f on %s", direction, abs(priceChange), platform)
	}
	
	return "Player price changed"
}

func (ae *AlertEngine) generateNewsMessage(event models.RealTimeEvent) string {
	var newsData map[string]interface{}
	if err := json.Unmarshal(event.Data, &newsData); err == nil {
		title := getString(newsData, "title")
		source := getString(newsData, "source")
		
		return fmt.Sprintf("Breaking: %s (via %s)", title, source)
	}
	
	return "Breaking news update"
}

// calculatePriority calculates alert priority based on event and rule
func (ae *AlertEngine) calculatePriority(event models.RealTimeEvent, rule *models.AlertRule) string {
	// High impact events get high priority
	if event.ImpactRating >= 9.0 {
		return "critical"
	} else if event.ImpactRating >= 7.0 {
		return "high"
	} else if event.ImpactRating >= 4.0 {
		return "medium"
	} else {
		return "low"
	}
}

// alertDeliveryWorker handles alert delivery
func (ae *AlertEngine) alertDeliveryWorker(ctx context.Context, workerID int) {
	ae.logger.WithField("worker_id", workerID).Info("Starting alert delivery worker")
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ae.stopChan:
			return
		case request := <-ae.deliveryQueue:
			ae.deliverAlert(request)
		}
	}
}

// deliverAlert delivers an alert through specified channels
func (ae *AlertEngine) deliverAlert(request DeliveryRequest) {
	success := false
	
	for _, channel := range request.Channels {
		err := ae.deliveryManager.DeliverAlert(request.Alert, channel, request.UserID)
		if err != nil {
			ae.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":  request.UserID,
				"alert_id": request.Alert.ID,
				"channel":  channel,
			}).Error("Failed to deliver alert")
		} else {
			success = true
		}
	}
	
	if success {
		ae.incrementDeliveredStats()
		// Mark alert as delivered
		now := time.Now()
		request.Alert.DeliveredAt = &now
		ae.db.Save(&request.Alert)
	} else {
		ae.incrementFailedStats()
	}
}

// getAllActiveRules retrieves all active alert rules with caching
func (ae *AlertEngine) getAllActiveRules() map[int][]*models.AlertRule {
	ae.rulesMutex.RLock()
	if time.Now().Before(ae.cacheExpiry) && len(ae.rulesCache) > 0 {
		// Return cached rules
		result := make(map[int][]*models.AlertRule)
		for userID, rules := range ae.rulesCache {
			result[userID] = rules
		}
		ae.rulesMutex.RUnlock()
		return result
	}
	ae.rulesMutex.RUnlock()
	
	// Refresh cache
	return ae.refreshRulesCache()
}

// refreshRulesCache refreshes the alert rules cache
func (ae *AlertEngine) refreshRulesCache() map[int][]*models.AlertRule {
	ae.rulesMutex.Lock()
	defer ae.rulesMutex.Unlock()
	
	var allRules []models.AlertRule
	if err := ae.db.Where("is_active = ?", true).Find(&allRules).Error; err != nil {
		ae.logger.WithError(err).Error("Failed to load alert rules")
		return ae.rulesCache // Return existing cache on error
	}
	
	// Group rules by user ID
	newCache := make(map[int][]*models.AlertRule)
	for i := range allRules {
		rule := &allRules[i]
		newCache[rule.UserID] = append(newCache[rule.UserID], rule)
	}
	
	ae.rulesCache = newCache
	ae.cacheExpiry = time.Now().Add(ae.config.RuleCacheInterval)
	
	ae.logger.WithField("rules_count", len(allRules)).Info("Refreshed alert rules cache")
	
	return newCache
}

// ruleCacheRefresher periodically refreshes the rules cache
func (ae *AlertEngine) ruleCacheRefresher(ctx context.Context) {
	ticker := time.NewTicker(ae.config.RuleCacheInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ae.stopChan:
			return
		case <-ticker.C:
			ae.refreshRulesCache()
		}
	}
}

// GetAlertStats returns alert system statistics
func (ae *AlertEngine) GetAlertStats() AlertStats {
	ae.statsMutex.Lock()
	defer ae.statsMutex.Unlock()
	return *ae.alertStats
}

// Metrics helpers
func (ae *AlertEngine) incrementGeneratedStats() {
	ae.statsMutex.Lock()
	ae.alertStats.AlertsGenerated++
	ae.statsMutex.Unlock()
}

func (ae *AlertEngine) incrementDeliveredStats() {
	ae.statsMutex.Lock()
	ae.alertStats.AlertsDelivered++
	ae.statsMutex.Unlock()
}

func (ae *AlertEngine) incrementFailedStats() {
	ae.statsMutex.Lock()
	ae.alertStats.AlertsFailed++
	ae.statsMutex.Unlock()
}

func (ae *AlertEngine) incrementRateLimitedStats() {
	ae.statsMutex.Lock()
	ae.alertStats.AlertsRateLimited++
	ae.statsMutex.Unlock()
}

func (ae *AlertEngine) incrementRulesEvaluatedStats(count int64) {
	ae.statsMutex.Lock()
	ae.alertStats.RulesEvaluated += count
	ae.statsMutex.Unlock()
}

func (ae *AlertEngine) incrementRulesMatchedStats() {
	ae.statsMutex.Lock()
	ae.alertStats.RulesMatched++
	ae.statsMutex.Unlock()
}

func (ae *AlertEngine) updateProcessingTime(duration time.Duration) {
	ae.statsMutex.Lock()
	if ae.alertStats.AverageProcessTime == 0 {
		ae.alertStats.AverageProcessTime = duration
	} else {
		ae.alertStats.AverageProcessTime = (ae.alertStats.AverageProcessTime + duration) / 2
	}
	ae.alertStats.LastProcessTime = time.Now()
	ae.statsMutex.Unlock()
}

// Utility functions
func convertToDeliveryChannels(channels []string) []models.DeliveryChannel {
	result := make([]models.DeliveryChannel, len(channels))
	for i, channel := range channels {
		result[i] = models.DeliveryChannel(channel)
	}
	return result
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	return 0.0
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}