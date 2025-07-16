package ownership

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// OwnershipTracker manages real-time ownership calculations and tracking
type OwnershipTracker struct {
	db             *gorm.DB
	redisClient    *redis.Client
	logger         *logrus.Logger
	trendAnalyzer  *TrendAnalyzer
	leverageCalc   *LeverageCalculator
	
	// Configuration
	updateInterval time.Duration
	cacheTTL       time.Duration
	
	// Active tracking
	activeContests map[string]*ContestTracker
	contestMutex   sync.RWMutex
	
	// Control channels
	stopChan       chan struct{}
	trackChan      chan string // Contest IDs to track
	
	// Metrics
	trackingStats  *TrackingStats
	statsMutex     sync.Mutex
}

// ContestTracker tracks ownership for a specific contest
type ContestTracker struct {
	ContestID      string
	LastUpdate     time.Time
	UpdateInterval time.Duration
	IsActive       bool
	PlayerCount    int
	EntryCount     int
	LockTime       time.Time
	
	// Ownership data
	CurrentOwnership map[uint]float64    // player_id -> ownership %
	StackOwnership   map[string]float64  // stack_key -> ownership %
	
	// Historical tracking
	OwnershipHistory []models.OwnershipSnapshot
	
	mu sync.RWMutex
}

// TrackingStats contains ownership tracking metrics
type TrackingStats struct {
	ContestsTracked     int       `json:"contests_tracked"`
	SnapshotsCreated    int64     `json:"snapshots_created"`
	TrendsCalculated    int64     `json:"trends_calculated"`
	CacheHits          int64     `json:"cache_hits"`
	CacheMisses        int64     `json:"cache_misses"`
	AverageUpdateTime  time.Duration `json:"average_update_time"`
	LastUpdateTime     time.Time `json:"last_update_time"`
	ErrorCount         int64     `json:"error_count"`
}

// OwnershipSnapshot represents a point-in-time ownership state
type OwnershipSnapshot struct {
	ContestID        string             `json:"contest_id"`
	Timestamp        time.Time          `json:"timestamp"`
	PlayerOwnership  map[uint]float64   `json:"player_ownership"`
	StackOwnership   map[string]float64 `json:"stack_ownership"`
	TotalEntries     int                `json:"total_entries"`
	TimeToLock       time.Duration      `json:"time_to_lock"`
	ChangeVelocity   map[uint]float64   `json:"change_velocity"`   // rate of change per hour
	LeverageScores   map[uint]float64   `json:"leverage_scores"`   // contrarian opportunity scores
}

// NewOwnershipTracker creates a new ownership tracker
func NewOwnershipTracker(db *gorm.DB, redisClient *redis.Client, logger *logrus.Logger) *OwnershipTracker {
	tracker := &OwnershipTracker{
		db:             db,
		redisClient:    redisClient,
		logger:         logger,
		updateInterval: 2 * time.Minute, // Default update interval
		cacheTTL:       10 * time.Minute,
		activeContests: make(map[string]*ContestTracker),
		stopChan:       make(chan struct{}),
		trackChan:      make(chan string, 100),
		trackingStats:  &TrackingStats{},
	}

	// Initialize sub-components
	tracker.trendAnalyzer = NewTrendAnalyzer(redisClient, logger)
	tracker.leverageCalc = NewLeverageCalculator(logger)

	return tracker
}

// Start begins ownership tracking
func (ot *OwnershipTracker) Start(ctx context.Context) error {
	ot.logger.Info("Starting ownership tracker")

	// Start tracking worker
	go ot.trackingWorker(ctx)

	// Start periodic update worker
	go ot.periodicUpdateWorker(ctx)

	// Wait for stop signal
	<-ot.stopChan
	ot.logger.Info("Ownership tracker stopped")

	return nil
}

// Stop stops ownership tracking
func (ot *OwnershipTracker) Stop() {
	close(ot.stopChan)
}

// TrackContest adds a contest to active tracking
func (ot *OwnershipTracker) TrackContest(contestID string, lockTime time.Time) error {
	ot.contestMutex.Lock()
	defer ot.contestMutex.Unlock()

	if _, exists := ot.activeContests[contestID]; exists {
		return fmt.Errorf("contest %s is already being tracked", contestID)
	}

	contestTracker := &ContestTracker{
		ContestID:        contestID,
		LastUpdate:       time.Time{},
		UpdateInterval:   ot.updateInterval,
		IsActive:         true,
		LockTime:         lockTime,
		CurrentOwnership: make(map[uint]float64),
		StackOwnership:   make(map[string]float64),
		OwnershipHistory: make([]models.OwnershipSnapshot, 0),
	}

	ot.activeContests[contestID] = contestTracker

	// Signal tracking worker
	select {
	case ot.trackChan <- contestID:
	default:
		ot.logger.Warn("Track channel full, contest tracking may be delayed")
	}

	ot.statsMutex.Lock()
	ot.trackingStats.ContestsTracked++
	ot.statsMutex.Unlock()

	ot.logger.WithFields(logrus.Fields{
		"contest_id": contestID,
		"lock_time":  lockTime,
	}).Info("Started tracking contest ownership")

	return nil
}

// StopTrackingContest removes a contest from active tracking
func (ot *OwnershipTracker) StopTrackingContest(contestID string) {
	ot.contestMutex.Lock()
	defer ot.contestMutex.Unlock()

	if tracker, exists := ot.activeContests[contestID]; exists {
		tracker.IsActive = false
		delete(ot.activeContests, contestID)

		ot.statsMutex.Lock()
		ot.trackingStats.ContestsTracked--
		ot.statsMutex.Unlock()

		ot.logger.WithField("contest_id", contestID).Info("Stopped tracking contest ownership")
	}
}

// GetCurrentOwnership returns current ownership data for a contest
func (ot *OwnershipTracker) GetCurrentOwnership(contestID string) (*OwnershipSnapshot, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("ownership:current:%s", contestID)
	cached, err := ot.redisClient.Get(context.Background(), cacheKey).Result()
	if err == nil {
		var snapshot OwnershipSnapshot
		if err := json.Unmarshal([]byte(cached), &snapshot); err == nil {
			ot.incrementCacheHit()
			return &snapshot, nil
		}
	}

	ot.incrementCacheMiss()

	// Get from active tracker
	ot.contestMutex.RLock()
	tracker, exists := ot.activeContests[contestID]
	ot.contestMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("contest %s is not being tracked", contestID)
	}

	return ot.calculateCurrentSnapshot(tracker)
}

// GetOwnershipTrends returns ownership trends for a contest
func (ot *OwnershipTracker) GetOwnershipTrends(contestID string, timeRange time.Duration) ([]models.OwnershipTrend, error) {
	return ot.trendAnalyzer.CalculateTrends(contestID, timeRange)
}

// GetLeverageScores returns contrarian opportunity scores for players
func (ot *OwnershipTracker) GetLeverageScores(contestID string) (map[uint]float64, error) {
	ownership, err := ot.GetCurrentOwnership(contestID)
	if err != nil {
		return nil, err
	}

	return ot.leverageCalc.CalculateLeverageScores(ownership.PlayerOwnership, ownership.TotalEntries), nil
}

// trackingWorker processes tracking requests
func (ot *OwnershipTracker) trackingWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ot.stopChan:
			return
		case contestID := <-ot.trackChan:
			if err := ot.updateContestOwnership(contestID); err != nil {
				ot.logger.WithError(err).WithField("contest_id", contestID).Error("Failed to update contest ownership")
				ot.incrementErrorCount()
			}
		}
	}
}

// periodicUpdateWorker updates all active contests periodically
func (ot *OwnershipTracker) periodicUpdateWorker(ctx context.Context) {
	ticker := time.NewTicker(ot.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ot.stopChan:
			return
		case <-ticker.C:
			ot.updateAllActiveContests()
		}
	}
}

// updateAllActiveContests updates ownership for all active contests
func (ot *OwnershipTracker) updateAllActiveContests() {
	ot.contestMutex.RLock()
	contestIDs := make([]string, 0, len(ot.activeContests))
	for contestID, tracker := range ot.activeContests {
		// Only update if enough time has passed and contest is still active
		if tracker.IsActive && time.Since(tracker.LastUpdate) >= tracker.UpdateInterval {
			// Skip if contest has already locked
			if !tracker.LockTime.IsZero() && time.Now().After(tracker.LockTime) {
				continue
			}
			contestIDs = append(contestIDs, contestID)
		}
	}
	ot.contestMutex.RUnlock()

	// Update contests in parallel
	var wg sync.WaitGroup
	for _, contestID := range contestIDs {
		wg.Add(1)
		go func(cid string) {
			defer wg.Done()
			if err := ot.updateContestOwnership(cid); err != nil {
				ot.logger.WithError(err).WithField("contest_id", cid).Error("Failed to update contest ownership")
				ot.incrementErrorCount()
			}
		}(contestID)
	}
	wg.Wait()
}

// updateContestOwnership updates ownership data for a specific contest
func (ot *OwnershipTracker) updateContestOwnership(contestID string) error {
	startTime := time.Now()

	ot.contestMutex.RLock()
	tracker, exists := ot.activeContests[contestID]
	ot.contestMutex.RUnlock()

	if !exists || !tracker.IsActive {
		return fmt.Errorf("contest %s is not actively tracked", contestID)
	}

	// Calculate new ownership snapshot
	snapshot, err := ot.calculateCurrentSnapshot(tracker)
	if err != nil {
		return fmt.Errorf("failed to calculate ownership snapshot: %w", err)
	}

	// Update tracker
	tracker.mu.Lock()
	tracker.CurrentOwnership = snapshot.PlayerOwnership
	tracker.StackOwnership = snapshot.StackOwnership
	tracker.LastUpdate = time.Now()
	tracker.EntryCount = snapshot.TotalEntries
	
	// Add to history (keep last 100 snapshots)
	// Convert maps to JSON for database storage
	var playerOwnershipJSON, stackOwnershipJSON datatypes.JSON
	if len(snapshot.PlayerOwnership) > 0 {
		playerBytes, _ := json.Marshal(snapshot.PlayerOwnership)
		playerOwnershipJSON = playerBytes
	}
	if len(snapshot.StackOwnership) > 0 {
		stackBytes, _ := json.Marshal(snapshot.StackOwnership)
		stackOwnershipJSON = stackBytes
	}
	
	tracker.OwnershipHistory = append(tracker.OwnershipHistory, models.OwnershipSnapshot{
		ContestID:       contestID,
		Timestamp:       snapshot.Timestamp,
		PlayerOwnership: playerOwnershipJSON,
		StackOwnership:  stackOwnershipJSON,
		TotalEntries:    snapshot.TotalEntries,
	})
	
	if len(tracker.OwnershipHistory) > 100 {
		tracker.OwnershipHistory = tracker.OwnershipHistory[1:]
	}
	tracker.mu.Unlock()

	// Store snapshot in database
	dbSnapshot := models.OwnershipSnapshot{
		ContestID:    contestID,
		Timestamp:    snapshot.Timestamp,
		TotalEntries: snapshot.TotalEntries,
	}

	// Convert maps to JSON
	if len(snapshot.PlayerOwnership) > 0 {
		playerBytes, _ := json.Marshal(snapshot.PlayerOwnership)
		dbSnapshot.PlayerOwnership = playerBytes
	}

	if len(snapshot.StackOwnership) > 0 {
		stackBytes, _ := json.Marshal(snapshot.StackOwnership)
		dbSnapshot.StackOwnership = stackBytes
	}

	if err := ot.db.Create(&dbSnapshot).Error; err != nil {
		ot.logger.WithError(err).Error("Failed to save ownership snapshot to database")
		// Continue even if DB save fails
	}

	// Cache the snapshot
	cacheKey := fmt.Sprintf("ownership:current:%s", contestID)
	snapshotJSON, _ := json.Marshal(snapshot)
	ot.redisClient.Set(context.Background(), cacheKey, snapshotJSON, ot.cacheTTL)

	// Update trends
	ot.trendAnalyzer.UpdateTrends(contestID, snapshot.PlayerOwnership, snapshot.Timestamp)

	// Update statistics
	ot.statsMutex.Lock()
	ot.trackingStats.SnapshotsCreated++
	ot.trackingStats.LastUpdateTime = time.Now()
	
	// Update average processing time
	processingTime := time.Since(startTime)
	if ot.trackingStats.AverageUpdateTime == 0 {
		ot.trackingStats.AverageUpdateTime = processingTime
	} else {
		ot.trackingStats.AverageUpdateTime = (ot.trackingStats.AverageUpdateTime + processingTime) / 2
	}
	ot.statsMutex.Unlock()

	ot.logger.WithFields(logrus.Fields{
		"contest_id":     contestID,
		"total_entries":  snapshot.TotalEntries,
		"processing_time": processingTime,
		"player_count":   len(snapshot.PlayerOwnership),
	}).Debug("Updated contest ownership")

	return nil
}

// calculateCurrentSnapshot calculates the current ownership snapshot for a contest
func (ot *OwnershipTracker) calculateCurrentSnapshot(tracker *ContestTracker) (*OwnershipSnapshot, error) {
	// In a real implementation, this would query the actual contest data source
	// For now, we'll simulate ownership calculations
	
	snapshot := &OwnershipSnapshot{
		ContestID:       tracker.ContestID,
		Timestamp:       time.Now(),
		PlayerOwnership: make(map[uint]float64),
		StackOwnership:  make(map[string]float64),
		TotalEntries:    tracker.EntryCount,
	}

	// Calculate time to lock
	if !tracker.LockTime.IsZero() {
		timeToLock := time.Until(tracker.LockTime)
		if timeToLock > 0 {
			snapshot.TimeToLock = timeToLock
		}
	}

	// TODO: Implement actual ownership calculation
	// This would typically involve:
	// 1. Querying contest lineups from DraftKings/FanDuel APIs
	// 2. Aggregating player usage across all lineups
	// 3. Calculating ownership percentages
	// 4. Computing stack ownership (QB+WR, RB+DST, etc.)

	// For now, simulate with some mock data
	snapshot.PlayerOwnership = ot.simulateOwnershipData(tracker.ContestID)
	snapshot.StackOwnership = ot.simulateStackOwnership(snapshot.PlayerOwnership)

	// Calculate trends and leverage scores
	if len(tracker.OwnershipHistory) > 0 {
		// Convert previous ownership from JSON to map
		prevOwnership := make(map[uint]float64)
		lastSnapshot := tracker.OwnershipHistory[len(tracker.OwnershipHistory)-1]
		if len(lastSnapshot.PlayerOwnership) > 0 {
			json.Unmarshal(lastSnapshot.PlayerOwnership, &prevOwnership)
		}
		
		snapshot.ChangeVelocity = ot.trendAnalyzer.CalculateVelocity(
			prevOwnership,
			snapshot.PlayerOwnership,
			time.Since(lastSnapshot.Timestamp),
		)
	}

	snapshot.LeverageScores = ot.leverageCalc.CalculateLeverageScores(
		snapshot.PlayerOwnership,
		snapshot.TotalEntries,
	)

	return snapshot, nil
}

// simulateOwnershipData generates mock ownership data for testing
func (ot *OwnershipTracker) simulateOwnershipData(contestID string) map[uint]float64 {
	// Mock data - in production this would come from actual contest APIs
	ownership := map[uint]float64{
		1001: 25.5, // High owned player
		1002: 18.3,
		1003: 12.7,
		1004: 8.9,
		1005: 15.2,
		1006: 6.4,  // Low owned player
		1007: 9.8,
		1008: 22.1,
		1009: 14.6,
		1010: 7.2,
	}

	return ownership
}

// simulateStackOwnership generates mock stack ownership data
func (ot *OwnershipTracker) simulateStackOwnership(playerOwnership map[uint]float64) map[string]float64 {
	// Mock stack data - typically QB+WR, RB+DST combinations
	stackOwnership := map[string]float64{
		"QB1001+WR1002": 8.5,
		"QB1001+WR1003": 6.2,
		"RB1005+DST1010": 4.8,
		"QB1008+WR1009": 7.3,
	}

	return stackOwnership
}

// GetTrackingStats returns current tracking statistics
func (ot *OwnershipTracker) GetTrackingStats() TrackingStats {
	ot.statsMutex.Lock()
	defer ot.statsMutex.Unlock()
	return *ot.trackingStats
}

// Metrics helpers
func (ot *OwnershipTracker) incrementCacheHit() {
	ot.statsMutex.Lock()
	ot.trackingStats.CacheHits++
	ot.statsMutex.Unlock()
}

func (ot *OwnershipTracker) incrementCacheMiss() {
	ot.statsMutex.Lock()
	ot.trackingStats.CacheMisses++
	ot.statsMutex.Unlock()
}

func (ot *OwnershipTracker) incrementErrorCount() {
	ot.statsMutex.Lock()
	ot.trackingStats.ErrorCount++
	ot.statsMutex.Unlock()
}

// GetActiveContests returns information about all actively tracked contests
func (ot *OwnershipTracker) GetActiveContests() map[string]*ContestTracker {
	ot.contestMutex.RLock()
	defer ot.contestMutex.RUnlock()

	// Return a copy to avoid race conditions
	contests := make(map[string]*ContestTracker)
	for contestID, tracker := range ot.activeContests {
		// Create a copy of the tracker
		trackerCopy := *tracker
		contests[contestID] = &trackerCopy
	}

	return contests
}