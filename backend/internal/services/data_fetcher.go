package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// DataFetcherService handles scheduled data updates from external providers
type DataFetcherService struct {
	db               *database.DB
	cache            *CacheService
	aggregator       *DataAggregator
	logger           *logrus.Logger
	cron             *cron.Cron
	mu               sync.Mutex
	isRunning        bool
	fetchInterval    time.Duration
	golfSyncService  *GolfTournamentSyncService
	contestDiscovery *ContestDiscoveryService
	config           *config.Config
}

// NewDataFetcherService creates a new data fetcher service
func NewDataFetcherService(
	db *database.DB,
	cache *CacheService,
	aggregator *DataAggregator,
	logger *logrus.Logger,
	fetchInterval time.Duration,
	cfg *config.Config,
) *DataFetcherService {
	return &DataFetcherService{
		db:               db,
		cache:            cache,
		aggregator:       aggregator,
		logger:           logger,
		cron:             cron.New(),
		fetchInterval:    fetchInterval,
		contestDiscovery: NewContestDiscoveryService(db),
		config:           cfg,
	}
}

// Start begins the scheduled data fetching
func (s *DataFetcherService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("data fetcher is already running")
	}

	// Only schedule background jobs if enabled
	if s.config.EnableBackgroundJobs {
		// Schedule regular updates
		schedule := fmt.Sprintf("@every %s", s.fetchInterval.String())
		_, err := s.cron.AddFunc(schedule, s.fetchAllContests)
		if err != nil {
			return fmt.Errorf("failed to schedule data fetcher: %w", err)
		}

		// Schedule more frequent updates during contest hours
		// NBA games typically start between 7-10 PM ET
		_, err = s.cron.AddFunc("0 17-22 * * *", s.fetchActiveContests) // Every hour from 5-10 PM
		if err != nil {
			return fmt.Errorf("failed to schedule active contest fetcher: %w", err)
		}

		// Schedule daily cleanup
		_, err = s.cron.AddFunc("0 3 * * *", s.cleanupOldData) // 3 AM daily
		if err != nil {
			return fmt.Errorf("failed to schedule cleanup: %w", err)
		}

		// Schedule contest discovery every 30 minutes
		_, err = s.cron.AddFunc("*/30 * * * *", s.discoverContests) // Every 30 minutes
		if err != nil {
			return fmt.Errorf("failed to schedule contest discovery: %w", err)
		}

		s.logger.Info("Background data fetching jobs scheduled")
	} else {
		s.logger.Info("Background data fetching jobs disabled by configuration")
	}

	// Always schedule golf tournament sync if not in golf-only mode or if golf is explicitly supported
	if !s.config.GolfOnlyMode || s.isGolfSupported() {
		_, err := s.cron.AddFunc("0 6,18 * * *", s.syncGolfTournaments) // 6 AM and 6 PM
		if err != nil {
			return fmt.Errorf("failed to schedule golf tournament sync: %w", err)
		}
		s.logger.Info("Golf tournament sync scheduled")
	}

	s.cron.Start()
	s.isRunning = true

	// Run initial fetch only if background jobs are enabled
	if s.config.EnableBackgroundJobs {
		go s.fetchAllContests()
	}

	s.logger.Info("Data fetcher service started")
	return nil
}

// Stop halts the scheduled data fetching
func (s *DataFetcherService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()

	s.isRunning = false
	s.logger.Info("Data fetcher service stopped")
}

// fetchAllContests fetches data for all upcoming contests
func (s *DataFetcherService) fetchAllContests() {
	s.logger.Info("Starting scheduled data fetch for all contests")

	// First, discover any new contests
	s.discoverContests()

	// Get all upcoming contests
	var contests []models.Contest
	err := s.db.DB.Where("start_time > ?", time.Now()).Find(&contests).Error
	if err != nil {
		s.logger.Errorf("Failed to fetch contests: %v", err)
		return
	}

	s.logger.Infof("Found %d upcoming contests to update", len(contests))

	// Process each contest
	for _, contest := range contests {
		s.fetchContestData(contest)
	}

	s.logger.Info("Completed scheduled data fetch")
}

// fetchActiveContests fetches data for contests starting soon
func (s *DataFetcherService) fetchActiveContests() {
	s.logger.Info("Starting active contest data fetch")

	// Get contests starting in the next 3 hours
	var contests []models.Contest
	err := s.db.DB.Where("start_time BETWEEN ? AND ?",
		time.Now(),
		time.Now().Add(3*time.Hour),
	).Find(&contests).Error
	if err != nil {
		s.logger.Errorf("Failed to fetch active contests: %v", err)
		return
	}

	s.logger.Infof("Found %d active contests to update", len(contests))

	// Process with higher priority
	for _, contest := range contests {
		s.fetchContestData(contest)
	}
}

// fetchContestData fetches and updates data for a specific contest
func (s *DataFetcherService) fetchContestData(contest models.Contest) {
	s.logger.Infof("Fetching data for contest %d: %s", contest.ID, contest.Name)

	// Determine sport from contest
	sport := s.getSportFromContest(contest)
	if sport == "" {
		s.logger.Warnf("Unknown sport for contest %d", contest.ID)
		return
	}

	// Use aggregator to fetch and merge data
	players, err := s.aggregator.AggregatePlayersForContest(contest.ID, sport)
	if err != nil {
		s.logger.Errorf("Failed to aggregate players for contest %d: %v", contest.ID, err)
		return
	}

	// If NBA or LOL, note that DraftKings data is rate-limited to hourly refresh
	if sport == "nba" || sport == "lol" {
		s.logger.Infof("DraftKings data for contest %d is refreshed hourly via provider cache/rate limit", contest.ID)
	}

	s.logger.Infof("Aggregated %d players for contest %d", len(players), contest.ID)

	// Update contest last updated time
	s.db.DB.Model(&contest).Update("last_data_update", time.Now())

	// Cache contest data
	cacheKey := fmt.Sprintf("contest:players:%d", contest.ID)
	s.cache.SetSimple(cacheKey, players, 30*time.Minute)

	// Trigger WebSocket update if contest is starting soon
	if contest.StartTime.Before(time.Now().Add(1 * time.Hour)) {
		s.notifyDataUpdate(contest.ID)
	}
}

// getSportFromContest determines the sport type from contest
func (s *DataFetcherService) getSportFromContest(contest models.Contest) string {
	// Return the sport string directly - the aggregator will handle conversion
	return contest.Sport
}

// notifyDataUpdate sends WebSocket notification about data updates
func (s *DataFetcherService) notifyDataUpdate(contestID uint) {
	// This would integrate with the WebSocket hub
	// For now, just log it
	s.logger.Infof("Data updated for contest %d - WebSocket notification would be sent", contestID)
}

// cleanupOldData removes data for past contests
func (s *DataFetcherService) cleanupOldData() {
	s.logger.Info("Starting daily cleanup of old data")

	// Delete players for contests that ended more than 7 days ago
	cutoffDate := time.Now().AddDate(0, 0, -7)

	result := s.db.DB.Where("contest_id IN (?)",
		s.db.DB.Table("contests").
			Select("id").
			Where("start_time < ?", cutoffDate),
	).Delete(&models.Player{})

	if result.Error != nil {
		s.logger.Errorf("Failed to cleanup old players: %v", result.Error)
	} else {
		s.logger.Infof("Cleaned up %d old player records", result.RowsAffected)
	}

	// Clear old cache entries
	s.cache.Flush()
}

// FetchOnDemand allows manual triggering of data fetch for a contest
func (s *DataFetcherService) FetchOnDemand(contestID uint) error {
	var contest models.Contest
	err := s.db.DB.First(&contest, contestID).Error
	if err != nil {
		return fmt.Errorf("contest not found: %w", err)
	}

	// Run fetch in background
	go s.fetchContestData(contest)

	return nil
}

// GetFetchStatus returns the current status of the fetcher
func (s *DataFetcherService) GetFetchStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries := s.cron.Entries()
	nextRuns := make([]time.Time, 0, len(entries))
	for _, entry := range entries {
		nextRuns = append(nextRuns, entry.Next)
	}

	return map[string]interface{}{
		"is_running":     s.isRunning,
		"fetch_interval": s.fetchInterval.String(),
		"next_runs":      nextRuns,
		"cron_jobs":      len(entries),
	}
}

// syncGolfTournaments syncs golf tournaments from API to contests
func (s *DataFetcherService) syncGolfTournaments() {
	if s.golfSyncService == nil {
		s.logger.Warn("Golf sync service not initialized, skipping tournament sync")
		return
	}

	s.logger.Info("Starting scheduled golf tournament sync")

	if err := s.golfSyncService.SyncAllActiveTournaments(); err != nil {
		s.logger.Errorf("Failed to sync golf tournaments: %v", err)
	} else {
		s.logger.Info("Golf tournament sync completed successfully")
	}
}

// SetGolfSyncService sets the golf sync service
func (s *DataFetcherService) SetGolfSyncService(service *GolfTournamentSyncService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.golfSyncService = service
}

// discoverContests runs contest discovery for all supported sports
func (s *DataFetcherService) discoverContests() {
	s.logger.Info("Starting scheduled contest discovery")

	// Discover contests for each supported sport
	supportedSports := []dfs.Sport{
		dfs.SportNBA,
		dfs.SportNFL,
		dfs.SportMLB,
		dfs.SportNHL,
		dfs.SportGolf,
		"lol",
	}

	for _, sport := range supportedSports {
		if err := s.contestDiscovery.DiscoverContests(sport); err != nil {
			s.logger.Errorf("Failed to discover contests for sport %s: %v", sport, err)
		}
	}

	// Cleanup expired contests
	if err := s.contestDiscovery.CleanupExpiredContests(); err != nil {
		s.logger.Errorf("Failed to cleanup expired contests: %v", err)
	}

	s.logger.Info("Completed scheduled contest discovery")
}

// isGolfSupported checks if golf is in the supported sports list
func (s *DataFetcherService) isGolfSupported() bool {
	for _, sport := range s.config.SupportedSports {
		if sport == "golf" {
			return true
		}
	}
	return false
}

// DiscoverContestsOnDemand allows manual triggering of contest discovery
func (s *DataFetcherService) DiscoverContestsOnDemand(sport string) error {
	s.logger.Infof("Running on-demand contest discovery for sport: %s", sport)

	if err := s.contestDiscovery.DiscoverContests(dfs.Sport(sport)); err != nil {
		return fmt.Errorf("failed to discover contests for sport %s: %w", sport, err)
	}

	return nil
}
