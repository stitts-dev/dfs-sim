package services

import (
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
)

// UserService handles user operations
type UserService struct {
	db     *database.DB
	config *config.Config
	logger *logrus.Logger
}

// NewUserService creates a new user service
func NewUserService(db *database.DB, cfg *config.Config, logger *logrus.Logger) *UserService {
	return &UserService{
		db:     db,
		config: cfg,
		logger: logger,
	}
}

// GetDB returns the database instance
func (s *UserService) GetDB() *database.DB {
	return s.db
}

// GetConfig returns the config instance
func (s *UserService) GetConfig() *config.Config {
	return s.config
}

// GetLogger returns the logger instance
func (s *UserService) GetLogger() *logrus.Logger {
	return s.logger
}