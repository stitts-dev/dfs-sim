package services

import (
	"fmt"
	"time"

	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
	"github.com/google/uuid"
)

// GolfTournamentSyncService handles syncing golf tournaments from RapidAPI to contests
type GolfTournamentSyncService struct {
	db           *database.DB
	golfProvider interface {
		GetCurrentTournament() (*providers.GolfTournamentData, error)
		GetTournamentSchedule() ([]providers.GolfTournamentData, error)
	}
	logger   *logrus.Logger
	golfSportID uuid.UUID
}

// NewGolfTournamentSyncService creates a new golf tournament sync service
func NewGolfTournamentSyncService(db *database.DB, golfProvider interface {
	GetCurrentTournament() (*providers.GolfTournamentData, error)
	GetTournamentSchedule() ([]providers.GolfTournamentData, error)
}, logger *logrus.Logger) *GolfTournamentSyncService {
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

	return &GolfTournamentSyncService{
		db:           db,
		golfProvider: golfProvider,
		logger:       logger,
		golfSportID:  golfSportID,
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

	// Create tournament and contests (this method handles duplicates)
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
		// Only sync future tournaments (include current and upcoming)
		if tournament.StartDate.Before(time.Now().AddDate(0, 0, -1)) {
			continue
		}

		// Create tournament and contests (this method handles duplicates)
		if err := s.createContestFromTournament(&tournament); err != nil {
			s.logger.WithError(err).Errorf("Failed to create contest for tournament %s", tournament.Name)
		} else {
			synced++
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
	} else {
		// Update existing tournament
		updates := map[string]interface{}{
			"name":        tournament.Name,
			"start_date":  tournament.StartDate,
			"end_date":    tournament.EndDate,
			"status":      models.TournamentStatus(tournament.Status),
			"course_name": tournament.CourseName,
			"course_par":  tournament.CoursePar,
			"purse":       tournament.Purse,
		}
		if err := s.db.Model(&dbTournament).Updates(updates).Error; err != nil {
			s.logger.WithError(err).Warnf("Failed to update tournament: %s", tournament.Name)
		}
	}

	// Create contests for both DraftKings and FanDuel
	platforms := []string{"draftkings", "fanduel"}
	contestTypes := []string{"gpp", "cash"}
	tournamentID := dbTournament.ID
	tournamentIDStr := dbTournament.ID.String()
	contestsCreated := 0

	for _, platform := range platforms {
		for _, contestType := range contestTypes {
			// Check if contest already exists for this tournament/platform/type combination
			var existingContest types.Contest
			err = s.db.Where("sport_id = ? AND tournament_id = ? AND platform = ? AND contest_type = ?", 
				s.golfSportID, tournamentIDStr, platform, contestType).First(&existingContest).Error
			
			if err == nil {
				// Contest exists, update it
				updates := map[string]interface{}{
					"name":              s.generateContestName(tournament.Name, platform, contestType),
					"start_time":        tournament.StartDate,
					"is_active":         tournament.Status != "completed",
					"last_data_update":  time.Now(),
					"prize_pool":        s.calculatePrizePool(tournament.Purse, contestType),
				}
				if err := s.db.Model(&existingContest).Updates(updates).Error; err != nil {
					s.logger.WithError(err).Errorf("Failed to update %s %s contest for %s", platform, contestType, tournament.Name)
				} else {
					s.logger.Infof("Updated %s %s contest for %s", platform, contestType, tournament.Name)
				}
				continue
			}

			// Create new contest
			contest := types.Contest{
				SportID:      s.golfSportID,
				Platform:     platform,
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
				PositionRequirements: types.GetPositionRequirements("golf", platform),
				LastDataUpdate:       func() *time.Time { t := time.Now(); return &t }(),
				TournamentID:         &tournamentID,
				ExternalID:           fmt.Sprintf("%s_%s_%s", tournament.ID, platform, contestType),
			}

			if err := s.db.Create(&contest).Error; err != nil {
				s.logger.WithError(err).Errorf("Failed to create %s %s contest for %s", platform, contestType, tournament.Name)
			} else {
				s.logger.Infof("Created %s %s contest for %s", platform, contestType, tournament.Name)
				contestsCreated++
			}
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"tournament":        tournament.Name,
		"contests_created":  contestsCreated,
		"tournament_id":     tournamentIDStr,
	}).Info("Tournament and contests sync completed")

	return nil
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
	result := s.db.Model(&types.Contest{}).
		Where("sport_id = ? AND start_time < ? AND is_active = ?", s.golfSportID, cutoffDate, false).
		Update("is_active", false)

	if result.Error != nil {
		s.logger.WithError(result.Error).Error("Failed to deactivate old golf contests")
	} else if result.RowsAffected > 0 {
		s.logger.Infof("Deactivated %d old golf contests", result.RowsAffected)
	}

	return nil
}
