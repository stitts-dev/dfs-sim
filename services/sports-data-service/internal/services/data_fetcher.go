package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/google/uuid"
)

// DataFetcherService handles scheduled data fetching from external providers
type DataFetcherService struct {
	db                *database.DB
	logger            *logrus.Logger
	cron              *cron.Cron
	golfSyncService   *GolfTournamentSyncService
	dataGolfProvider  *providers.DataGolfClient
	circuitBreaker    *CircuitBreakerService
	cache             *CacheService
	ctx               context.Context
	cancel            context.CancelFunc
	mu                sync.RWMutex
	jobs              map[string]JobInfo
	isRunning         bool
	golfSportID       uuid.UUID
}

// JobInfo represents information about a scheduled job
type JobInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Schedule    string    `json:"schedule"`
	LastRun     time.Time `json:"last_run"`
	NextRun     time.Time `json:"next_run"`
	Status      string    `json:"status"`
	RunCount    int       `json:"run_count"`
	ErrorCount  int       `json:"error_count"`
	LastError   string    `json:"last_error,omitempty"`
	Duration    time.Duration `json:"duration"`
	IsEnabled   bool      `json:"is_enabled"`
}

// NewDataFetcherService creates a new data fetcher service with scheduling capabilities
func NewDataFetcherService(
	db *database.DB,
	logger *logrus.Logger,
	golfSyncService *GolfTournamentSyncService,
	dataGolfProvider *providers.DataGolfClient,
	circuitBreaker *CircuitBreakerService,
	cache *CacheService,
) *DataFetcherService {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create cron scheduler with logger
	cronLogger := cron.VerbosePrintfLogger(logger)
	c := cron.New(cron.WithLogger(cronLogger))

	// Get golf sport ID lazily to avoid prepared statement conflicts
	var golfSportID uuid.UUID

	return &DataFetcherService{
		db:                db,
		logger:            logger,
		cron:              c,
		golfSyncService:   golfSyncService,
		dataGolfProvider:  dataGolfProvider,
		circuitBreaker:    circuitBreaker,
		cache:             cache,
		ctx:               ctx,
		cancel:            cancel,
		jobs:              make(map[string]JobInfo),
		isRunning:         false,
		golfSportID:       golfSportID,
	}
}

// Start starts the data fetcher service with scheduled jobs
func (dfs *DataFetcherService) Start() error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	if dfs.isRunning {
		return fmt.Errorf("data fetcher service is already running")
	}

	dfs.logger.WithField("component", "data_fetcher").Info("Starting DataFetcherService with scheduled jobs")

	// Schedule tournament sync jobs
	if err := dfs.scheduleJobs(); err != nil {
		return fmt.Errorf("failed to schedule jobs: %w", err)
	}

	// Start the cron scheduler
	dfs.cron.Start()
	dfs.isRunning = true

	dfs.logger.WithField("component", "data_fetcher").Info("DataFetcherService started successfully")

	return nil
}

// scheduleJobs sets up all scheduled jobs
func (dfs *DataFetcherService) scheduleJobs() error {
	// Tournament sync - Every 30 minutes during tournament weeks
	if err := dfs.addJob("tournament_sync", "*/30 * * * *", "Tournament data sync", dfs.syncTournamentData); err != nil {
		return fmt.Errorf("failed to schedule tournament sync job: %w", err)
	}

	// Player data sync - Every 15 minutes during active tournaments
	if err := dfs.addJob("player_sync", "*/15 * * * *", "Player data sync", dfs.syncPlayerData); err != nil {
		return fmt.Errorf("failed to schedule player sync job: %w", err)
	}

	// Contest creation - Every hour to create new contests
	if err := dfs.addJob("contest_creation", "0 * * * *", "Contest creation", dfs.createContests); err != nil {
		return fmt.Errorf("failed to schedule contest creation job: %w", err)
	}

	// Cache warming - Every 6 hours to warm frequently accessed data
	if err := dfs.addJob("cache_warming", "0 */6 * * *", "Cache warming", dfs.warmCache); err != nil {
		return fmt.Errorf("failed to schedule cache warming job: %w", err)
	}

	// Daily cleanup - Every day at 2 AM to clean up old data
	if err := dfs.addJob("daily_cleanup", "0 2 * * *", "Daily cleanup", dfs.dailyCleanup); err != nil {
		return fmt.Errorf("failed to schedule daily cleanup job: %w", err)
	}

	// Weekly tournament discovery - Every Monday at 8 AM
	if err := dfs.addJob("weekly_tournament_discovery", "0 8 * * 1", "Weekly tournament discovery", dfs.discoverNewTournaments); err != nil {
		return fmt.Errorf("failed to schedule weekly tournament discovery job: %w", err)
	}

	return nil
}

// addJob adds a new scheduled job
func (dfs *DataFetcherService) addJob(id, schedule, name string, jobFunc func()) error {
	entryID, err := dfs.cron.AddFunc(schedule, func() {
		dfs.runJob(id, name, jobFunc)
	})
	if err != nil {
		return fmt.Errorf("failed to add job %s: %w", id, err)
	}

	// Get next run time
	entries := dfs.cron.Entries()
	var nextRun time.Time
	for _, entry := range entries {
		if entry.ID == entryID {
			nextRun = entry.Next
			break
		}
	}

	// Store job info
	dfs.jobs[id] = JobInfo{
		ID:        id,
		Name:      name,
		Schedule:  schedule,
		NextRun:   nextRun,
		Status:    "scheduled",
		IsEnabled: true,
	}

	dfs.logger.WithFields(logrus.Fields{
		"component": "data_fetcher",
		"job_id":    id,
		"job_name":  name,
		"schedule":  schedule,
		"next_run":  nextRun,
	}).Info("Scheduled job added")

	return nil
}

// runJob executes a job with error handling and metrics
func (dfs *DataFetcherService) runJob(id, name string, jobFunc func()) {
	dfs.mu.Lock()
	job, exists := dfs.jobs[id]
	if !exists {
		dfs.mu.Unlock()
		return
	}

	if !job.IsEnabled {
		dfs.mu.Unlock()
		return
	}

	job.Status = "running"
	job.LastRun = time.Now()
	job.RunCount++
	dfs.jobs[id] = job
	dfs.mu.Unlock()

	logger := dfs.logger.WithFields(logrus.Fields{
		"component": "data_fetcher",
		"job_id":    id,
		"job_name":  name,
		"run_count": job.RunCount,
	})

	logger.Info("Starting scheduled job")
	startTime := time.Now()

	// Execute job with panic recovery
	defer func() {
		if r := recover(); r != nil {
			logger.WithField("panic", r).Error("Job panicked")
			dfs.updateJobStatus(id, "failed", fmt.Sprintf("panic: %v", r), time.Since(startTime))
		}
	}()

	// Run the job
	jobFunc()

	duration := time.Since(startTime)
	logger.WithField("duration", duration).Info("Job completed successfully")
	dfs.updateJobStatus(id, "completed", "", duration)
}

// updateJobStatus updates the status of a job
func (dfs *DataFetcherService) updateJobStatus(id, status, errorMsg string, duration time.Duration) {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	job, exists := dfs.jobs[id]
	if !exists {
		return
	}

	job.Status = status
	job.Duration = duration
	
	if errorMsg != "" {
		job.ErrorCount++
		job.LastError = errorMsg
	}

	// Update next run time
	entries := dfs.cron.Entries()
	for _, entry := range entries {
		if entry.Job != nil {
			job.NextRun = entry.Next
			break
		}
	}

	dfs.jobs[id] = job
}

// syncTournamentData synchronizes tournament data from external providers
func (dfs *DataFetcherService) syncTournamentData() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "tournament_sync")
	
	// Check if DataGolf provider is available
	if dfs.dataGolfProvider == nil {
		logger.Error("DataGolf provider is not available, skipping tournament sync")
		return
	}

	// Check if we should skip sync based on circuit breaker
	if dfs.circuitBreaker.GetState("datagolf") == gobreaker.StateOpen {
		logger.Warn("DataGolf provider is unavailable, skipping tournament sync")
		return
	}

	// Sync all active tournaments
	if err := dfs.golfSyncService.SyncAllActiveTournaments(); err != nil {
		logger.WithError(err).Error("Failed to sync tournament data")
		return
	}

	logger.Info("Tournament data synchronized successfully")
}

// syncPlayerData populates players for contests that don't have them yet
func (dfs *DataFetcherService) syncPlayerData() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "player_sync")
	
	// Check if DataGolf provider is available
	if dfs.dataGolfProvider == nil {
		logger.Error("DataGolf provider is not available, skipping player sync")
		return
	}

	// Check if we should skip sync based on circuit breaker
	if dfs.circuitBreaker.GetState("datagolf") == gobreaker.StateOpen {
		logger.Warn("DataGolf provider is unavailable, skipping player sync")
		return
	}

	// Get active golf contests that might need player population
	var activeContests []types.Contest
	if err := dfs.db.
		Joins("JOIN sports ON contests.sport_id = sports.id").
		Where("sports.name = ? AND contests.is_active = ? AND contests.start_time >= ?", 
			"Golf", true, time.Now().Add(-24*time.Hour)).
		Find(&activeContests).Error; err != nil {
		logger.WithError(err).Error("Failed to get active contests")
		return
	}

	if len(activeContests) == 0 {
		logger.Debug("No active contests found, skipping player sync")
		return
	}

	logger.WithField("active_contests", len(activeContests)).Info("Found active contests for player sync")

	// Check each contest for missing players and populate them
	populated := 0
	failed := 0
	
	for _, contest := range activeContests {
		contestLogger := logger.WithField("contest_id", contest.ID)
		
		// Check if contest already has players
		var playerCount int64
		if err := dfs.db.Model(&types.Player{}).Where("contest_id = ?", contest.ID).Count(&playerCount).Error; err != nil {
			contestLogger.WithError(err).Error("Failed to count players for contest")
			continue
		}

		if playerCount > 0 {
			contestLogger.WithField("player_count", playerCount).Debug("Contest already has players, skipping")
			continue
		}

		// Contest has no players, populate them
		contestLogger.Info("Contest has no players, populating from tournament data")
		
		// Get players from DataGolf provider
		players, err := dfs.dataGolfProvider.GetPlayers(types.SportGolf, time.Now().Format("2006-01-02"))
		if err != nil {
			contestLogger.WithError(err).Error("Failed to get players from provider")
			failed++
			continue
		}

		if len(players) == 0 {
			contestLogger.Warn("No players found from provider")
			continue
		}

		// Get golf sport ID
		golfSportID, err := dfs.getGolfSportID()
		if err != nil {
			contestLogger.WithError(err).Error("Failed to get golf sport ID")
			failed++
			continue
		}

		// Create player records for this contest
		playersCreated := 0
		for _, player := range players {
			// Convert string fields to pointers
			var position, team, imageURL *string
			if player.Position != "" {
				position = &player.Position
			}
			if player.Team != "" {
				team = &player.Team
			}
			if player.ImageURL != "" {
				imageURL = &player.ImageURL
			}

			// Extract actual data from DataGolf API response
			projectedPoints := dfs.extractProjectedPoints(player)
			salaryDK := dfs.extractSalaryDK(player, contest.Platform)
			salaryFD := dfs.extractSalaryFD(player, contest.Platform)
			isActive := true

			dbPlayer := types.Player{
				ID:              uuid.New(),
				SportID:         golfSportID,
				ExternalID:      player.ExternalID,
				Name:            player.Name,
				Position:        position,
				Team:            team,
				ContestID:       &contest.ID,
				ProjectedPoints: &projectedPoints,
				SalaryDK:        &salaryDK,
				SalaryFD:        &salaryFD,
				IsActive:        &isActive,
				GameTime:        &contest.StartTime,
				ImageURL:        imageURL,
			}

			if err := dfs.db.Create(&dbPlayer).Error; err != nil {
				contestLogger.WithError(err).WithField("player_name", player.Name).Error("Failed to create player")
				continue
			}

			playersCreated++
		}

		if playersCreated > 0 {
			contestLogger.WithField("players_created", playersCreated).Info("Successfully populated contest players")
			populated++
		} else {
			contestLogger.Error("Failed to create any players for contest")
			failed++
		}
	}

	logger.WithFields(logrus.Fields{
		"populated": populated,
		"failed":    failed,
	}).Info("Player population sync completed")
}

// createContests creates new contests based on available tournaments
func (dfs *DataFetcherService) createContests() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "contest_creation")
	
	// This will be handled by the updated GolfTournamentSyncService
	if err := dfs.golfSyncService.SyncAllActiveTournaments(); err != nil {
		logger.WithError(err).Error("Failed to create contests")
		return
	}

	logger.Info("Contest creation completed successfully")
}

// getGolfSportID gets the golf sport ID with caching
func (dfs *DataFetcherService) getGolfSportID() (uuid.UUID, error) {
	if dfs.golfSportID != uuid.Nil {
		return dfs.golfSportID, nil
	}
	
	var sport struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	err := dfs.db.Raw("SELECT id FROM sports WHERE name = ? LIMIT 1", "Golf").Scan(&sport).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get golf sport ID - ensure 'Golf' sport exists in database: %w", err)
	}
	
	if sport.ID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("golf sport not found in database - ensure 'Golf' sport exists")
	}
	
	dfs.golfSportID = sport.ID
	return dfs.golfSportID, nil
}

// warmCache warms frequently accessed data
func (dfs *DataFetcherService) warmCache() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "cache_warming")
	
	// Warm DataGolf cache if available
	if dfs.dataGolfProvider != nil {
		if err := dfs.dataGolfProvider.WarmCache(); err != nil {
			logger.WithError(err).Warn("Failed to warm DataGolf cache")
		}
	}

	// Warm commonly accessed data
	cacheKeys := []string{
		"golf:tournaments:active",
		"golf:contests:active",
		"golf:players:current",
	}

	for _, key := range cacheKeys {
		if dfs.cache.GetSimple(key, nil) != nil {
			logger.WithField("cache_key", key).Debug("Cache miss, could be warmed")
		}
	}

	logger.Info("Cache warming completed")
}

// dailyCleanup performs daily maintenance tasks
func (dfs *DataFetcherService) dailyCleanup() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "daily_cleanup")
	
	// Clean up old completed tournaments
	golfSportID, err := dfs.getGolfSportID()
	if err != nil {
		logger.WithError(err).Error("Failed to get golf sport ID for cleanup")
		return
	}
	
	cutoffDate := time.Now().AddDate(0, -1, 0) // 1 month ago
	result := dfs.db.Model(&types.Contest{}).
		Where("sport_id = ? AND start_time < ? AND is_active = ?", golfSportID, cutoffDate, false).
		Update("is_active", false)

	if result.Error != nil {
		logger.WithError(result.Error).Error("Failed to deactivate old contests")
	} else if result.RowsAffected > 0 {
		logger.WithField("contests_deactivated", result.RowsAffected).Info("Deactivated old contests")
	}

	logger.Info("Daily cleanup completed")
}

// discoverNewTournaments discovers new tournaments for the upcoming week
func (dfs *DataFetcherService) discoverNewTournaments() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "tournament_discovery")
	
	// Check if DataGolf provider is available
	if dfs.dataGolfProvider == nil {
		logger.Error("DataGolf provider is not available, skipping tournament discovery")
		return
	}

	// Get tournament schedule from DataGolf provider
	schedule, err := dfs.dataGolfProvider.GetTournamentSchedule()
	if err != nil {
		logger.WithError(err).Error("Failed to get tournament schedule")
		return
	}

	// Sync upcoming tournaments
	if err := dfs.golfSyncService.SyncTournamentSchedule(); err != nil {
		logger.WithError(err).Error("Failed to sync tournament schedule")
		return
	}

	logger.WithField("tournaments_found", len(schedule)).Info("Tournament discovery completed")
}

// getAvailableProvider returns the DataGolf provider if available
func (dfs *DataFetcherService) getAvailableProvider() interface {
	GetCurrentTournament() (*providers.GolfTournamentData, error)
	GetTournamentSchedule() ([]providers.GolfTournamentData, error)
	GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error)
} {
	if dfs.dataGolfProvider != nil && dfs.circuitBreaker.GetState("datagolf") != gobreaker.StateOpen {
		return dfs.dataGolfProvider
	}
	return nil
}

// GetStatus returns the current status of the data fetcher service
func (dfs *DataFetcherService) GetStatus() map[string]interface{} {
	dfs.mu.RLock()
	defer dfs.mu.RUnlock()

	return map[string]interface{}{
		"is_running":   dfs.isRunning,
		"jobs":         dfs.jobs,
		"job_count":    len(dfs.jobs),
		"cron_entries": len(dfs.cron.Entries()),
		"status":       "healthy",
	}
}

// GetJobs returns information about all scheduled jobs
func (dfs *DataFetcherService) GetJobs() map[string]JobInfo {
	dfs.mu.RLock()
	defer dfs.mu.RUnlock()

	// Create a copy to avoid race conditions
	jobs := make(map[string]JobInfo)
	for k, v := range dfs.jobs {
		jobs[k] = v
	}

	return jobs
}

// EnableJob enables a scheduled job
func (dfs *DataFetcherService) EnableJob(id string) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	job, exists := dfs.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	job.IsEnabled = true
	dfs.jobs[id] = job

	dfs.logger.WithField("job_id", id).Info("Job enabled")
	return nil
}

// DisableJob disables a scheduled job
func (dfs *DataFetcherService) DisableJob(id string) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	job, exists := dfs.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	job.IsEnabled = false
	dfs.jobs[id] = job

	dfs.logger.WithField("job_id", id).Info("Job disabled")
	return nil
}

// TriggerJob manually triggers a job
func (dfs *DataFetcherService) TriggerJob(id string) error {
	dfs.mu.RLock()
	job, exists := dfs.jobs[id]
	dfs.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	dfs.logger.WithField("job_id", id).Info("Manually triggering job")

	// Map job IDs to their functions
	jobFunctions := map[string]func(){
		"tournament_sync":             dfs.syncTournamentData,
		"player_sync":                 dfs.syncPlayerData,
		"contest_creation":            dfs.createContests,
		"cache_warming":               dfs.warmCache,
		"daily_cleanup":               dfs.dailyCleanup,
		"weekly_tournament_discovery": dfs.discoverNewTournaments,
	}

	jobFunc, exists := jobFunctions[id]
	if !exists {
		return fmt.Errorf("job function not found for %s", id)
	}

	// Run the job asynchronously
	go dfs.runJob(id, job.Name, jobFunc)

	return nil
}

// Stop stops the data fetcher service
func (dfs *DataFetcherService) Stop() error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	if !dfs.isRunning {
		return nil
	}

	dfs.logger.WithField("component", "data_fetcher").Info("Stopping DataFetcherService")

	// Stop the cron scheduler
	ctx := dfs.cron.Stop()
	select {
	case <-ctx.Done():
		dfs.logger.WithField("component", "data_fetcher").Info("Cron scheduler stopped gracefully")
	case <-time.After(5 * time.Second):
		dfs.logger.WithField("component", "data_fetcher").Warn("Cron scheduler stop timed out")
	}

	// Cancel context
	dfs.cancel()
	dfs.isRunning = false

	dfs.logger.WithField("component", "data_fetcher").Info("DataFetcherService stopped")
	return nil
}

// extractProjectedPoints extracts projected points from player stats
func (dfs *DataFetcherService) extractProjectedPoints(player types.PlayerData) float64 {
	if player.Stats == nil {
		return 0.0
	}

	// Convert stats to map for easier access
	statsMap, ok := player.Stats.(map[string]interface{})
	if !ok {
		return 0.0
	}

	// Try to get fantasy_points from DataGolf
	if fantasyPoints, exists := statsMap["fantasy_points"]; exists {
		if fp, ok := fantasyPoints.(float64); ok {
			return fp
		}
	}

	// Fallback: calculate projected points from probabilities
	winProb := dfs.getFloatFromStats(statsMap, "win_probability")
	top5Prob := dfs.getFloatFromStats(statsMap, "top5_probability")
	top10Prob := dfs.getFloatFromStats(statsMap, "top10_probability")
	makeCutProb := dfs.getFloatFromStats(statsMap, "make_cut_probability")

	// Basic projection calculation based on probabilities
	projectedPoints := (winProb * 30.0) + (top5Prob * 20.0) + (top10Prob * 15.0) + (makeCutProb * 10.0)
	
	if projectedPoints == 0.0 {
		return 45.0 // Default reasonable projection for golfers
	}

	return projectedPoints
}

// extractSalaryDK extracts DraftKings salary from player stats
func (dfs *DataFetcherService) extractSalaryDK(player types.PlayerData, platform string) int {
	if player.Stats == nil {
		return 7000 // Default DK salary
	}

	statsMap, ok := player.Stats.(map[string]interface{})
	if !ok {
		return 7000
	}

	if dkSalary, exists := statsMap["dk_salary"]; exists {
		if salary, ok := dkSalary.(float64); ok {
			return int(salary)
		}
	}

	// Default salary based on platform
	if platform == "draftkings" {
		return 7000
	}
	return 7000
}

// extractSalaryFD extracts FanDuel salary from player stats
func (dfs *DataFetcherService) extractSalaryFD(player types.PlayerData, platform string) int {
	if player.Stats == nil {
		return 9000 // Default FD salary
	}

	statsMap, ok := player.Stats.(map[string]interface{})
	if !ok {
		return 9000
	}

	if fdSalary, exists := statsMap["fd_salary"]; exists {
		if salary, ok := fdSalary.(float64); ok {
			return int(salary)
		}
	}

	// Default salary based on platform
	if platform == "fanduel" {
		return 9000
	}
	return 9000
}

// getFloatFromStats helper to safely extract float values from stats
func (dfs *DataFetcherService) getFloatFromStats(stats map[string]interface{}, key string) float64 {
	if val, exists := stats[key]; exists {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return 0.0
}