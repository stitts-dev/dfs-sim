package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// DeliveryManager manages alert delivery across multiple channels
type DeliveryManager struct {
	redisClient    *redis.Client
	logger         *logrus.Logger
	
	// Channel handlers
	websocketHandler *WebSocketHandler
	emailHandler     *EmailHandler
	pushHandler      *PushHandler
	smsHandler       *SMSHandler
	
	// Configuration
	retryAttempts    int
	retryDelay       time.Duration
	deliveryTimeout  time.Duration
}

// DeliveryChannel represents different alert delivery methods
type ChannelHandler interface {
	DeliverAlert(alert models.Alert, userID int) error
	GetChannelType() models.DeliveryChannel
	IsAvailable() bool
	GetDeliveryStats() ChannelStats
}

// ChannelStats contains delivery statistics for a channel
type ChannelStats struct {
	ChannelType        models.DeliveryChannel `json:"channel_type"`
	MessagesDelivered  int64                  `json:"messages_delivered"`
	MessagesFailed     int64                  `json:"messages_failed"`
	AverageLatency     time.Duration          `json:"average_latency"`
	LastDeliveryTime   time.Time              `json:"last_delivery_time"`
	IsHealthy          bool                   `json:"is_healthy"`
	ErrorRate          float64                `json:"error_rate"`
}

// NewDeliveryManager creates a new delivery manager
func NewDeliveryManager(redisClient *redis.Client, logger *logrus.Logger) *DeliveryManager {
	dm := &DeliveryManager{
		redisClient:     redisClient,
		logger:          logger,
		retryAttempts:   3,
		retryDelay:      time.Second,
		deliveryTimeout: 10 * time.Second,
	}
	
	// Initialize channel handlers
	dm.websocketHandler = NewWebSocketHandler(redisClient, logger)
	dm.emailHandler = NewEmailHandler(logger)
	dm.pushHandler = NewPushHandler(logger)
	dm.smsHandler = NewSMSHandler(logger)
	
	return dm
}

// DeliverAlert delivers an alert through the specified channel
func (dm *DeliveryManager) DeliverAlert(alert models.Alert, channel models.DeliveryChannel, userID int) error {
	handler := dm.getChannelHandler(channel)
	if handler == nil {
		return fmt.Errorf("no handler available for channel: %s", channel)
	}
	
	if !handler.IsAvailable() {
		return fmt.Errorf("channel %s is not available", channel)
	}
	
	// Attempt delivery with retries
	var lastErr error
	for attempt := 0; attempt < dm.retryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(dm.retryDelay * time.Duration(attempt))
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), dm.deliveryTimeout)
		err := dm.deliverWithTimeout(ctx, handler, alert, userID)
		cancel()
		
		if err == nil {
			dm.logger.WithFields(logrus.Fields{
				"channel":   channel,
				"user_id":   userID,
				"alert_id":  alert.ID,
				"attempt":   attempt + 1,
			}).Debug("Alert delivered successfully")
			return nil
		}
		
		lastErr = err
		dm.logger.WithError(err).WithFields(logrus.Fields{
			"channel":  channel,
			"user_id":  userID,
			"alert_id": alert.ID,
			"attempt":  attempt + 1,
		}).Warn("Alert delivery attempt failed")
	}
	
	return fmt.Errorf("failed to deliver alert after %d attempts: %w", dm.retryAttempts, lastErr)
}

// deliverWithTimeout delivers an alert with timeout context
func (dm *DeliveryManager) deliverWithTimeout(ctx context.Context, handler ChannelHandler, alert models.Alert, userID int) error {
	done := make(chan error, 1)
	
	go func() {
		done <- handler.DeliverAlert(alert, userID)
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("delivery timeout exceeded")
	}
}

// getChannelHandler returns the appropriate handler for a channel
func (dm *DeliveryManager) getChannelHandler(channel models.DeliveryChannel) ChannelHandler {
	switch channel {
	case models.DeliveryChannelWebSocket:
		return dm.websocketHandler
	case models.DeliveryChannelEmail:
		return dm.emailHandler
	case models.DeliveryChannelPush:
		return dm.pushHandler
	case models.DeliveryChannelSMS:
		return dm.smsHandler
	default:
		return nil
	}
}

// GetChannelStats returns statistics for all delivery channels
func (dm *DeliveryManager) GetChannelStats() map[models.DeliveryChannel]ChannelStats {
	stats := make(map[models.DeliveryChannel]ChannelStats)
	
	handlers := []ChannelHandler{
		dm.websocketHandler,
		dm.emailHandler,
		dm.pushHandler,
		dm.smsHandler,
	}
	
	for _, handler := range handlers {
		if handler != nil {
			stats[handler.GetChannelType()] = handler.GetDeliveryStats()
		}
	}
	
	return stats
}

// WebSocketHandler handles WebSocket alert delivery
type WebSocketHandler struct {
	redisClient *redis.Client
	logger      *logrus.Logger
	stats       *ChannelStats
}

func NewWebSocketHandler(redisClient *redis.Client, logger *logrus.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		redisClient: redisClient,
		logger:      logger,
		stats: &ChannelStats{
			ChannelType: models.DeliveryChannelWebSocket,
			IsHealthy:   true,
		},
	}
}

func (wh *WebSocketHandler) DeliverAlert(alert models.Alert, userID int) error {
	startTime := time.Now()
	
	// Publish alert to Redis channel for WebSocket delivery
	alertMessage := map[string]interface{}{
		"type":      "alert",
		"alert":     alert,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
	}
	
	messageBytes, err := json.Marshal(alertMessage)
	if err != nil {
		wh.updateStats(false, time.Since(startTime))
		return fmt.Errorf("failed to marshal alert message: %w", err)
	}
	
	// Publish to user-specific channel
	channel := fmt.Sprintf("alerts:user:%d", userID)
	if err := wh.redisClient.Publish(context.Background(), channel, messageBytes).Err(); err != nil {
		wh.updateStats(false, time.Since(startTime))
		return fmt.Errorf("failed to publish WebSocket alert: %w", err)
	}
	
	wh.updateStats(true, time.Since(startTime))
	return nil
}

func (wh *WebSocketHandler) GetChannelType() models.DeliveryChannel {
	return models.DeliveryChannelWebSocket
}

func (wh *WebSocketHandler) IsAvailable() bool {
	return wh.redisClient != nil
}

func (wh *WebSocketHandler) GetDeliveryStats() ChannelStats {
	return *wh.stats
}

func (wh *WebSocketHandler) updateStats(success bool, latency time.Duration) {
	if success {
		wh.stats.MessagesDelivered++
		wh.stats.LastDeliveryTime = time.Now()
	} else {
		wh.stats.MessagesFailed++
	}
	
	// Update average latency
	if wh.stats.AverageLatency == 0 {
		wh.stats.AverageLatency = latency
	} else {
		wh.stats.AverageLatency = (wh.stats.AverageLatency + latency) / 2
	}
	
	// Update error rate
	total := wh.stats.MessagesDelivered + wh.stats.MessagesFailed
	if total > 0 {
		wh.stats.ErrorRate = float64(wh.stats.MessagesFailed) / float64(total) * 100
	}
	
	// Update health status
	wh.stats.IsHealthy = wh.stats.ErrorRate < 10.0 // Healthy if error rate < 10%
}

// EmailHandler handles email alert delivery
type EmailHandler struct {
	logger      *logrus.Logger
	stats       *ChannelStats
	smtpConfig  *SMTPConfig
	isConfigured bool
}

type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	FromName   string
	UseTLS     bool
}

func NewEmailHandler(logger *logrus.Logger) *EmailHandler {
	return &EmailHandler{
		logger: logger,
		stats: &ChannelStats{
			ChannelType: models.DeliveryChannelEmail,
			IsHealthy:   false, // Requires configuration
		},
		isConfigured: false, // Would be set based on environment variables
	}
}

func (eh *EmailHandler) DeliverAlert(alert models.Alert, userID int) error {
	if !eh.isConfigured {
		return fmt.Errorf("email delivery not configured")
	}
	
	startTime := time.Now()
	
	// TODO: Implement actual email delivery
	// This would typically involve:
	// 1. Get user email address from database
	// 2. Format alert as HTML/text email
	// 3. Send via SMTP
	
	// For now, just log the alert
	eh.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"alert_id":  alert.ID,
		"title":     alert.Title,
		"message":   alert.Message,
		"priority":  alert.Priority,
	}).Info("Email alert would be sent")
	
	eh.updateStats(true, time.Since(startTime))
	return nil
}

func (eh *EmailHandler) GetChannelType() models.DeliveryChannel {
	return models.DeliveryChannelEmail
}

func (eh *EmailHandler) IsAvailable() bool {
	return eh.isConfigured
}

func (eh *EmailHandler) GetDeliveryStats() ChannelStats {
	return *eh.stats
}

func (eh *EmailHandler) updateStats(success bool, latency time.Duration) {
	if success {
		eh.stats.MessagesDelivered++
		eh.stats.LastDeliveryTime = time.Now()
	} else {
		eh.stats.MessagesFailed++
	}
	
	// Update average latency
	if eh.stats.AverageLatency == 0 {
		eh.stats.AverageLatency = latency
	} else {
		eh.stats.AverageLatency = (eh.stats.AverageLatency + latency) / 2
	}
	
	// Update error rate
	total := eh.stats.MessagesDelivered + eh.stats.MessagesFailed
	if total > 0 {
		eh.stats.ErrorRate = float64(eh.stats.MessagesFailed) / float64(total) * 100
	}
	
	eh.stats.IsHealthy = eh.stats.ErrorRate < 5.0 && eh.isConfigured
}

// PushHandler handles push notification delivery
type PushHandler struct {
	logger      *logrus.Logger
	stats       *ChannelStats
	isConfigured bool
	fcmKey      string
	apnsConfig  *APNSConfig
}

type APNSConfig struct {
	TeamID     string
	KeyID      string
	BundleID   string
	KeyFile    string
	Production bool
}

func NewPushHandler(logger *logrus.Logger) *PushHandler {
	return &PushHandler{
		logger: logger,
		stats: &ChannelStats{
			ChannelType: models.DeliveryChannelPush,
			IsHealthy:   false, // Requires configuration
		},
		isConfigured: false, // Would be set based on environment variables
	}
}

func (ph *PushHandler) DeliverAlert(alert models.Alert, userID int) error {
	if !ph.isConfigured {
		return fmt.Errorf("push notifications not configured")
	}
	
	startTime := time.Now()
	
	// TODO: Implement actual push notification delivery
	// This would typically involve:
	// 1. Get user device tokens from database
	// 2. Format push notification payload
	// 3. Send via FCM (Android) and/or APNS (iOS)
	
	// For now, just log the alert
	ph.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"alert_id":  alert.ID,
		"title":     alert.Title,
		"message":   alert.Message,
		"priority":  alert.Priority,
	}).Info("Push notification would be sent")
	
	ph.updateStats(true, time.Since(startTime))
	return nil
}

func (ph *PushHandler) GetChannelType() models.DeliveryChannel {
	return models.DeliveryChannelPush
}

func (ph *PushHandler) IsAvailable() bool {
	return ph.isConfigured
}

func (ph *PushHandler) GetDeliveryStats() ChannelStats {
	return *ph.stats
}

func (ph *PushHandler) updateStats(success bool, latency time.Duration) {
	if success {
		ph.stats.MessagesDelivered++
		ph.stats.LastDeliveryTime = time.Now()
	} else {
		ph.stats.MessagesFailed++
	}
	
	// Update average latency
	if ph.stats.AverageLatency == 0 {
		ph.stats.AverageLatency = latency
	} else {
		ph.stats.AverageLatency = (ph.stats.AverageLatency + latency) / 2
	}
	
	// Update error rate
	total := ph.stats.MessagesDelivered + ph.stats.MessagesFailed
	if total > 0 {
		ph.stats.ErrorRate = float64(ph.stats.MessagesFailed) / float64(total) * 100
	}
	
	ph.stats.IsHealthy = ph.stats.ErrorRate < 5.0 && ph.isConfigured
}

// SMSHandler handles SMS alert delivery
type SMSHandler struct {
	logger       *logrus.Logger
	stats        *ChannelStats
	isConfigured bool
	twilioSID    string
	twilioToken  string
	twilioFrom   string
}

func NewSMSHandler(logger *logrus.Logger) *SMSHandler {
	return &SMSHandler{
		logger: logger,
		stats: &ChannelStats{
			ChannelType: models.DeliveryChannelSMS,
			IsHealthy:   false, // Requires configuration
		},
		isConfigured: false, // Would be set based on environment variables
	}
}

func (sh *SMSHandler) DeliverAlert(alert models.Alert, userID int) error {
	if !sh.isConfigured {
		return fmt.Errorf("SMS delivery not configured")
	}
	
	startTime := time.Now()
	
	// TODO: Implement actual SMS delivery
	// This would typically involve:
	// 1. Get user phone number from database
	// 2. Format alert as SMS message (with length limits)
	// 3. Send via Twilio or other SMS provider
	
	// For now, just log the alert
	sh.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"alert_id":  alert.ID,
		"title":     alert.Title,
		"message":   alert.Message,
		"priority":  alert.Priority,
	}).Info("SMS alert would be sent")
	
	sh.updateStats(true, time.Since(startTime))
	return nil
}

func (sh *SMSHandler) GetChannelType() models.DeliveryChannel {
	return models.DeliveryChannelSMS
}

func (sh *SMSHandler) IsAvailable() bool {
	return sh.isConfigured
}

func (sh *SMSHandler) GetDeliveryStats() ChannelStats {
	return *sh.stats
}

func (sh *SMSHandler) updateStats(success bool, latency time.Duration) {
	if success {
		sh.stats.MessagesDelivered++
		sh.stats.LastDeliveryTime = time.Now()
	} else {
		sh.stats.MessagesFailed++
	}
	
	// Update average latency
	if sh.stats.AverageLatency == 0 {
		sh.stats.AverageLatency = latency
	} else {
		sh.stats.AverageLatency = (sh.stats.AverageLatency + latency) / 2
	}
	
	// Update error rate
	total := sh.stats.MessagesDelivered + sh.stats.MessagesFailed
	if total > 0 {
		sh.stats.ErrorRate = float64(sh.stats.MessagesFailed) / float64(total) * 100
	}
	
	sh.stats.IsHealthy = sh.stats.ErrorRate < 5.0 && sh.isConfigured
}

// Utility functions for actual implementations

// sendHTTPRequest sends an HTTP request for push notifications or webhooks
func sendHTTPRequest(method, url string, headers map[string]string, body []byte, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	
	req, err := http.NewRequest(method, url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// formatAlertForSMS formats an alert message for SMS (with length limits)
func formatAlertForSMS(alert models.Alert) string {
	message := fmt.Sprintf("DFS Alert: %s", alert.Title)
	
	// SMS messages are typically limited to 160 characters
	maxLength := 140 // Leave room for "DFS Alert: " prefix
	if len(alert.Message) <= maxLength {
		message = fmt.Sprintf("DFS Alert: %s", alert.Message)
	} else {
		// Truncate message with ellipsis
		truncated := alert.Message[:maxLength-3] + "..."
		message = fmt.Sprintf("DFS Alert: %s", truncated)
	}
	
	return message
}

// formatAlertForEmail formats an alert as HTML email
func formatAlertForEmail(alert models.Alert) (string, string) {
	subject := fmt.Sprintf("DFS Alert: %s", alert.Title)
	
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
</head>
<body>
    <h1>%s</h1>
    <p><strong>Priority:</strong> %s</p>
    <p>%s</p>
    <hr>
    <p><small>Alert ID: %s | Time: %s</small></p>
</body>
</html>
`, alert.Title, alert.Title, alert.Priority, alert.Message, alert.ID, alert.CreatedAt.Format("2006-01-02 15:04:05"))

	return subject, htmlBody
}