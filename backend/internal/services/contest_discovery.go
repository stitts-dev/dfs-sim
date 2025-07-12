package services

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ContestDiscoveryService struct {
	db         *database.DB
	dkProvider *providers.DraftKingsProvider
}

func NewContestDiscoveryService(db *database.DB) *ContestDiscoveryService {
	return &ContestDiscoveryService{
		db:         db,
		dkProvider: providers.NewDraftKingsProvider(),
	}
}

// parseDraftKingsDate parses DraftKings date format /Date(timestamp)/
func parseDraftKingsDate(dateStr string) (time.Time, error) {
	// DraftKings uses /Date(timestamp)/ format
	re := regexp.MustCompile(`/Date\((\d+)\)/`)
	matches := re.FindStringSubmatch(dateStr)
	if len(matches) != 2 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}

	timestamp, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	// Convert milliseconds to seconds
	return time.Unix(timestamp/1000, (timestamp%1000)*1000000), nil
}

// DiscoverContests fetches contests from DraftKings and creates/updates them in the database
func (cds *ContestDiscoveryService) DiscoverContests(sport dfs.Sport) error {
	logrus.Infof("Starting contest discovery for sport: %s", sport)

	// Fetch contests from DraftKings provider
	contests, err := cds.dkProvider.GetContests(sport)
	if err != nil {
		return fmt.Errorf("failed to fetch contests from DraftKings: %w", err)
	}

	logrus.Infof("Found %d contests from DraftKings for sport %s", len(contests), sport)

	// Process each contest
	for _, dkContest := range contests {
		if err := cds.processContest(dkContest, string(sport)); err != nil {
			logrus.Errorf("Failed to process contest %d: %v", dkContest.ID, err)
			continue
		}
	}

	return nil
}

// processContest creates or updates a contest in the database
func (cds *ContestDiscoveryService) processContest(dkContest providers.DraftKingsContestInfo, sport string) error {
	// Skip contests that are not active or have no entries allowed
	if !dkContest.IsActive || dkContest.MaxEntries == 0 {
		return nil
	}

	// Parse start time - DraftKings uses /Date(timestamp)/ format
	startTime, err := parseDraftKingsDate(dkContest.StartTime)
	if err != nil {
		logrus.Warnf("Failed to parse start time for contest %d: %v", dkContest.ID, err)
		// Set start time to 2 hours from now as fallback
		startTime = time.Now().Add(2 * time.Hour)
	}

	// Skip contests that have already started
	if startTime.Before(time.Now()) {
		return nil
	}

	// Determine contest type
	contestType := "gpp"
	if dkContest.ContestType == "50/50" || dkContest.ContestType == "Double Up" || dkContest.ContestType == "Head to Head" {
		contestType = "cash"
	}

	// Set salary cap based on sport if not provided
	salaryCap := dkContest.SalaryCap
	if salaryCap == 0 {
		switch sport {
		case "nba":
			salaryCap = 50000
		case "nfl":
			salaryCap = 50000
		case "mlb":
			salaryCap = 50000
		case "nhl":
			salaryCap = 50000
		case "golf":
			salaryCap = 50000
		}
	}

	// Check if contest already exists
	var existingContest models.Contest
	err = cds.db.Where("external_id = ? AND platform = ?", strconv.Itoa(dkContest.ID), "draftkings").First(&existingContest).Error

	if err == nil {
		// Contest exists, update it
		existingContest.Name = dkContest.Name
		existingContest.EntryFee = dkContest.EntryFee
		existingContest.PrizePool = dkContest.PrizePool
		existingContest.MaxEntries = dkContest.MaxEntries
		existingContest.TotalEntries = dkContest.TotalEntries
		existingContest.StartTime = startTime
		existingContest.IsMultiEntry = dkContest.IsMultiEntry
		existingContest.MaxLineupsPerUser = dkContest.MaxLineupsPerUser
		existingContest.SalaryCap = salaryCap
		existingContest.IsActive = dkContest.IsActive
		existingContest.DraftGroupID = strconv.Itoa(dkContest.DraftGroupID)
		existingContest.LastSyncTime = time.Now()

		if err := cds.db.Save(&existingContest).Error; err != nil {
			return fmt.Errorf("failed to update contest %d: %w", dkContest.ID, err)
		}

		logrus.Debugf("Updated contest %d: %s", dkContest.ID, dkContest.Name)
	} else if err == gorm.ErrRecordNotFound {
		// Contest doesn't exist, create it
		newContest := models.Contest{
			Platform:             "draftkings",
			Sport:                sport,
			ContestType:          contestType,
			Name:                 dkContest.Name,
			EntryFee:             dkContest.EntryFee,
			PrizePool:            dkContest.PrizePool,
			MaxEntries:           dkContest.MaxEntries,
			TotalEntries:         dkContest.TotalEntries,
			SalaryCap:            salaryCap,
			StartTime:            startTime,
			IsActive:             dkContest.IsActive,
			IsMultiEntry:         dkContest.IsMultiEntry,
			MaxLineupsPerUser:    dkContest.MaxLineupsPerUser,
			ExternalID:           strconv.Itoa(dkContest.ID),
			DraftGroupID:         strconv.Itoa(dkContest.DraftGroupID),
			LastSyncTime:         time.Now(),
			PositionRequirements: models.GetPositionRequirements(sport, "draftkings"),
		}

		if err := cds.db.Create(&newContest).Error; err != nil {
			return fmt.Errorf("failed to create contest %d: %w", dkContest.ID, err)
		}

		logrus.Infof("Created new contest %d: %s", dkContest.ID, dkContest.Name)
	} else {
		return fmt.Errorf("failed to query contest %d: %w", dkContest.ID, err)
	}

	return nil
}

// CleanupExpiredContests removes contests that have started and are no longer active
func (cds *ContestDiscoveryService) CleanupExpiredContests() error {
	logrus.Info("Cleaning up expired contests")

	// Mark contests as inactive if they started more than 6 hours ago
	sixHoursAgo := time.Now().Add(-6 * time.Hour)

	result := cds.db.Model(&models.Contest{}).
		Where("start_time < ? AND is_active = true", sixHoursAgo).
		Update("is_active", false)

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired contests: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		logrus.Infof("Marked %d contests as inactive", result.RowsAffected)
	}

	return nil
}

// SyncSpecificContest syncs a specific contest by its external ID
func (cds *ContestDiscoveryService) SyncSpecificContest(externalID string, sport dfs.Sport) error {
	logrus.Infof("Syncing specific contest %s for sport %s", externalID, sport)

	// For now, we'll re-run the full discovery for the sport
	// In a production system, you might want to fetch a specific contest
	return cds.DiscoverContests(sport)
}

// GetDiscoveryStatus returns status information about contest discovery
func (cds *ContestDiscoveryService) GetDiscoveryStatus() (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Get contest counts by sport
	var sportCounts []struct {
		Sport string
		Count int64
	}

	if err := cds.db.Model(&models.Contest{}).
		Select("sport, count(*) as count").
		Where("is_active = true").
		Group("sport").
		Find(&sportCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get sport counts: %w", err)
	}

	status["active_contests_by_sport"] = sportCounts

	// Get latest sync time
	var latestSync time.Time
	if err := cds.db.Model(&models.Contest{}).
		Select("MAX(last_sync_time) as latest_sync").
		Where("last_sync_time IS NOT NULL").
		Scan(&latestSync).Error; err != nil {
		logrus.Warnf("Failed to get latest sync time: %v", err)
	} else {
		status["latest_sync"] = latestSync
	}

	// Get total active contests
	var totalActive int64
	if err := cds.db.Model(&models.Contest{}).
		Where("is_active = true").
		Count(&totalActive).Error; err != nil {
		return nil, fmt.Errorf("failed to get total active contests: %w", err)
	}

	status["total_active_contests"] = totalActive

	return status, nil
}
