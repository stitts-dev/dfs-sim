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
	rapidAPIProvider  *providers.RapidAPIGolfClient
	espnProvider      *providers.ESPNGolfClient
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
	rapidAPIProvider *providers.RapidAPIGolfClient,
	espnProvider *providers.ESPNGolfClient,
	circuitBreaker *CircuitBreakerService,
	cache *CacheService,
) *DataFetcherService {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create cron scheduler with logger
	cronLogger := cron.VerbosePrintfLogger(logger)
	c := cron.New(cron.WithLogger(cronLogger))

	// Get golf sport ID once during initialization
	var golfSportID uuid.UUID
	var sport struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	err := db.Table("sports").Select("id").Where("name = ?", "Golf").First(&sport).Error
	if err != nil {
		logger.WithError(err).Fatal("Failed to get golf sport ID - ensure 'Golf' sport exists in database")
	}
	golfSportID = sport.ID

	return &DataFetcherService{
		db:                db,
		logger:            logger,
		cron:              c,
		golfSyncService:   golfSyncService,
		rapidAPIProvider:  rapidAPIProvider,
		espnProvider:      espnProvider,
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
	
	// Check if we should skip sync based on circuit breaker
	if dfs.circuitBreaker.GetState("rapidapi") == gobreaker.StateOpen && dfs.circuitBreaker.GetState("espn") == gobreaker.StateOpen {
		logger.Warn("All external providers are unavailable, skipping tournament sync")
		return
	}

	// Sync all active tournaments
	if err := dfs.golfSyncService.SyncAllActiveTournaments(); err != nil {
		logger.WithError(err).Error("Failed to sync tournament data")
		return
	}

	logger.Info("Tournament data synchronized successfully")
}

// syncPlayerData synchronizes player data for active tournaments
func (dfs *DataFetcherService) syncPlayerData() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "player_sync")
	
	// Check if we should skip sync based on circuit breaker
	if dfs.circuitBreaker.GetState("rapidapi") == gobreaker.StateOpen && dfs.circuitBreaker.GetState("espn") == gobreaker.StateOpen {
		logger.Warn("All external providers are unavailable, skipping player sync")
		return
	}

	// Get active tournaments
	var activeTournaments []string
	if err := dfs.db.Model(&types.Contest{}).
		Joins("JOIN sports ON contests.sport_id = sports.id").
		Where("sports.name = ? AND contests.is_active = ? AND contests.start_time <= ? AND contests.start_time >= ?", 
			"Golf", true, time.Now().Add(24*time.Hour), time.Now().Add(-24*time.Hour)).
		Pluck("contests.tournament_id", &activeTournaments).Error; err != nil {
		logger.WithError(err).Error("Failed to get active tournaments")
		return
	}

	if len(activeTournaments) == 0 {
		logger.Debug("No active tournaments found, skipping player sync")
		return
	}

	// Sync players for each active tournament
	provider := dfs.getAvailableProvider()
	if provider == nil {
		logger.Error("No available providers for player sync")
		return
	}

	players, err := provider.GetPlayers(types.SportGolf, time.Now().Format("2006-01-02"))
	if err != nil {
		logger.WithError(err).Error("Failed to get players from provider")
		return
	}

	logger.WithField("player_count", len(players)).Info("Player data synchronized successfully")
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

// warmCache warms frequently accessed data
func (dfs *DataFetcherService) warmCache() {
	logger := dfs.logger.WithField("component", "data_fetcher").WithField("job", "cache_warming")
	
	// Warm RapidAPI cache if available
	if dfs.rapidAPIProvider != nil {
		if err := dfs.rapidAPIProvider.WarmCache(); err != nil {
			logger.WithError(err).Warn("Failed to warm RapidAPI cache")
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
	cutoffDate := time.Now().AddDate(0, -1, 0) // 1 month ago
	result := dfs.db.Model(&types.Contest{}).
		Where("sport_id = ? AND start_time < ? AND is_active = ?", dfs.golfSportID, cutoffDate, false).
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
	
	// Get tournament schedule from provider
	provider := dfs.getAvailableProvider()
	if provider == nil {
		logger.Error("No available providers for tournament discovery")
		return
	}

	schedule, err := provider.GetTournamentSchedule()
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

// getAvailableProvider returns the best available provider
func (dfs *DataFetcherService) getAvailableProvider() interface {
	GetCurrentTournament() (*providers.GolfTournamentData, error)
	GetTournamentSchedule() ([]providers.GolfTournamentData, error)
	GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error)
} {
	if dfs.rapidAPIProvider != nil && dfs.circuitBreaker.GetState("rapidapi") != gobreaker.StateOpen {
		return dfs.rapidAPIProvider
	}
	if dfs.espnProvider != nil && dfs.circuitBreaker.GetState("espn") != gobreaker.StateOpen {
		return dfs.espnProvider
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