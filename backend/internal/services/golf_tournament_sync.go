package services

import (
	"fmt"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
)

// GolfTournamentSyncService handles syncing golf tournaments from RapidAPI to contests
type GolfTournamentSyncService struct {
	db           *database.DB
	golfProvider interface {
		GetCurrentTournament() (*providers.GolfTournamentData, error)
		GetTournamentSchedule() ([]providers.GolfTournamentData, error)
	}
	logger *logrus.Logger
}

// NewGolfTournamentSyncService creates a new golf tournament sync service
func NewGolfTournamentSyncService(db *database.DB, golfProvider interface {
	GetCurrentTournament() (*providers.GolfTournamentData, error)
	GetTournamentSchedule() ([]providers.GolfTournamentData, error)
}, logger *logrus.Logger) *GolfTournamentSyncService {
	return &GolfTournamentSyncService{
		db:           db,
		golfProvider: golfProvider,
		logger:       logger,
	}
}

// SyncCurrentTournament syncs the current golf tournament to contests
func (s *GolfTournamentSyncService) SyncCurrentTournament() error {
	s.logger.Info("Starting golf tournament sync")

	// Get current tournament from provider
	tournament, err := s.golfProvider.GetCurrentTournament()
	if err != nil {
		return fmt.Errorf("failed to get current tournament: %w", err)
	}

	// Check if contest already exists
	var existingContest models.Contest
	err = s.db.Where("sport = ? AND name = ?", "golf", tournament.Name).First(&existingContest).Error
	if err == nil {
		// Update existing contest
		return s.updateContest(&existingContest, tournament)
	}

	// Create new contest
	return s.createContestFromTournament(tournament)
}

// SyncTournamentSchedule syncs upcoming golf tournaments
func (s *GolfTournamentSyncService) SyncTournamentSchedule() error {
	s.logger.Info("Syncing golf tournament schedule")

	schedule, err := s.golfProvider.GetTournamentSchedule()
	if err != nil {
		return fmt.Errorf("failed to get tournament schedule: %w", err)
	}

	synced := 0
	for _, tournament := range schedule {
		// Only sync future tournaments
		if tournament.StartDate.Before(time.Now()) {
			continue
		}

		// Check if contest exists
		var existingContest models.Contest
		err = s.db.Where("sport = ? AND name = ?", "golf", tournament.Name).First(&existingContest).Error
		if err == nil {
			// Update existing
			if err := s.updateContest(&existingContest, &tournament); err != nil {
				s.logger.WithError(err).Errorf("Failed to update contest for tournament %s", tournament.Name)
			} else {
				synced++
			}
		} else {
			// Create new
			if err := s.createContestFromTournament(&tournament); err != nil {
				s.logger.WithError(err).Errorf("Failed to create contest for tournament %s", tournament.Name)
			} else {
				synced++
			}
		}
	}

	s.logger.Infof("Synced %d golf tournaments", synced)
	return nil
}

// createContestFromTournament creates a new contest from tournament data
func (s *GolfTournamentSyncService) createContestFromTournament(tournament *providers.GolfTournamentData) error {
	// First, ensure tournament exists in database
	var dbTournament models.GolfTournament
	err := s.db.Where("external_id = ?", tournament.ID).First(&dbTournament).Error
	if err != nil {
		// Create tournament if it doesn't exist
		dbTournament = models.GolfTournament{
			ExternalID: tournament.ID,
			Name:       tournament.Name,
			StartDate:  tournament.StartDate,
			EndDate:    tournament.EndDate,
			Status:     models.TournamentStatus(tournament.Status),
			CourseName: tournament.CourseName,
			CoursePar:  tournament.CoursePar,
			Purse:      tournament.Purse,
		}
		if err := s.db.Create(&dbTournament).Error; err != nil {
			return fmt.Errorf("failed to create tournament: %w", err)
		}
		s.logger.Infof("Created golf tournament: %s", tournament.Name)
	}

	// Create contests for both DraftKings and FanDuel
	platforms := []string{"draftkings", "fanduel"}
	contestTypes := []string{"gpp", "cash"}
	tournamentIDStr := dbTournament.ID.String()

	for _, platform := range platforms {
		for _, contestType := range contestTypes {
			contest := models.Contest{
				Platform:     platform,
				Sport:        "golf",
				ContestType:  contestType,
				Name:         s.generateContestName(tournament.Name, platform, contestType),
				EntryFee:     s.getDefaultEntryFee(contestType),
				PrizePool:    s.calculatePrizePool(tournament.Purse, contestType),
				MaxEntries:   s.getDefaultMaxEntries(contestType),
				SalaryCap:    50000, // Standard DFS salary cap
				StartTime:    tournament.StartDate,
				IsActive:     tournament.Status != "completed",
				IsMultiEntry: contestType == "gpp",
				MaxLineupsPerUser: func() int {
					if contestType == "gpp" {
						return 150
					}
					return 1
				}(),
				PositionRequirements: models.GetPositionRequirements("golf", platform),
				LastDataUpdate:       time.Now(),
				TournamentID:         &tournamentIDStr,
			}

			if err := s.db.Create(&contest).Error; err != nil {
				s.logger.WithError(err).Errorf("Failed to create %s %s contest for %s", platform, contestType, tournament.Name)
			} else {
				s.logger.Infof("Created %s %s contest for %s", platform, contestType, tournament.Name)
			}
		}
	}

	return nil
}

// updateContest updates an existing contest with new tournament data
func (s *GolfTournamentSyncService) updateContest(contest *models.Contest, tournament *providers.GolfTournamentData) error {
	// First, ensure tournament exists in database
	var dbTournament models.GolfTournament
	err := s.db.Where("external_id = ?", tournament.ID).First(&dbTournament).Error
	if err != nil {
		// Create tournament if it doesn't exist
		dbTournament = models.GolfTournament{
			ExternalID: tournament.ID,
			Name:       tournament.Name,
			StartDate:  tournament.StartDate,
			EndDate:    tournament.EndDate,
			Status:     models.TournamentStatus(tournament.Status),
			CourseName: tournament.CourseName,
			CoursePar:  tournament.CoursePar,
			Purse:      tournament.Purse,
		}
		if err := s.db.Create(&dbTournament).Error; err != nil {
			return fmt.Errorf("failed to create tournament: %w", err)
		}
		s.logger.Infof("Created golf tournament: %s", tournament.Name)
	}

	tournamentIDStr := dbTournament.ID.String()
	updates := map[string]interface{}{
		"start_time":       tournament.StartDate,
		"is_active":        tournament.Status != "completed",
		"last_data_update": time.Now(),
		"tournament_id":    &tournamentIDStr,
	}

	// Update prize pool if tournament purse changed
	if tournament.Purse > 0 {
		updates["prize_pool"] = s.calculatePrizePool(tournament.Purse, contest.ContestType)
	}

	return s.db.Model(contest).Updates(updates).Error
}

// generateContestName creates a contest name based on tournament and type
func (s *GolfTournamentSyncService) generateContestName(tournamentName, platform, contestType string) string {
	platformPrefix := ""
	if platform == "draftkings" {
		platformPrefix = "DK"
	} else {
		platformPrefix = "FD"
	}

	contestSuffix := ""
	if contestType == "gpp" {
		contestSuffix = "GPP"
	} else {
		contestSuffix = "Cash Game"
	}

	return fmt.Sprintf("%s %s - %s", platformPrefix, tournamentName, contestSuffix)
}

// getDefaultEntryFee returns default entry fee based on contest type
func (s *GolfTournamentSyncService) getDefaultEntryFee(contestType string) float64 {
	if contestType == "gpp" {
		return 20.0 // $20 GPP
	}
	return 5.0 // $5 cash game
}

// calculatePrizePool calculates DFS prize pool based on tournament purse
func (s *GolfTournamentSyncService) calculatePrizePool(tournamentPurse float64, contestType string) float64 {
	// For DFS, prize pools are much smaller than actual tournament purse
	// This is just an example calculation
	if contestType == "gpp" {
		return 100000 // $100K GPP
	}
	return 10000 // $10K cash game
}

// getDefaultMaxEntries returns default max entries based on contest type
func (s *GolfTournamentSyncService) getDefaultMaxEntries(contestType string) int {
	if contestType == "gpp" {
		return 10000 // Large field GPP
	}
	return 100 // Small cash game
}

// SyncAllActiveTournaments syncs all active golf tournaments
func (s *GolfTournamentSyncService) SyncAllActiveTournaments() error {
	// First sync current tournament
	if err := s.SyncCurrentTournament(); err != nil {
		s.logger.WithError(err).Error("Failed to sync current tournament")
	}

	// Then sync schedule
	if err := s.SyncTournamentSchedule(); err != nil {
		s.logger.WithError(err).Error("Failed to sync tournament schedule")
	}

	// Clean up old completed tournaments
	cutoffDate := time.Now().AddDate(0, -1, 0) // 1 month ago
	result := s.db.Model(&models.Contest{}).
		Where("sport = ? AND start_time < ? AND is_active = ?", "golf", cutoffDate, false).
		Update("is_active", false)

	if result.Error != nil {
		s.logger.WithError(result.Error).Error("Failed to deactivate old golf contests")
	} else if result.RowsAffected > 0 {
		s.logger.Infof("Deactivated %d old golf contests", result.RowsAffected)
	}

	return nil
}
