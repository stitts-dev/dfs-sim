package database

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

type ConnectionConfig struct {
	DatabaseURL     string
	IsDevelopment   bool
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ServiceName     string
}

func NewConnection(databaseURL string, isDevelopment bool) (*DB, error) {
	config := ConnectionConfig{
		DatabaseURL:     databaseURL,
		IsDevelopment:   isDevelopment,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		ServiceName:     "unknown",
	}
	return NewConnectionWithConfig(config)
}

func NewConnectionWithConfig(config ConnectionConfig) (*DB, error) {
	logLevel := logger.Error
	if config.IsDevelopment {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(config.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Service-specific connection pool settings
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"service":         config.ServiceName,
		"max_idle_conns":  config.MaxIdleConns,
		"max_open_conns":  config.MaxOpenConns,
		"conn_max_lifetime": config.ConnMaxLifetime,
	}).Info("Database connection established successfully")

	return &DB{db}, nil
}

// Service-specific connection factory functions
func NewGolfServiceConnection(databaseURL string, isDevelopment bool) (*DB, error) {
	config := ConnectionConfig{
		DatabaseURL:     databaseURL,
		IsDevelopment:   isDevelopment,
		MaxIdleConns:    5,
		MaxOpenConns:    30,
		ConnMaxLifetime: time.Hour,
		ServiceName:     "golf-service",
	}
	return NewConnectionWithConfig(config)
}

func NewOptimizationServiceConnection(databaseURL string, isDevelopment bool) (*DB, error) {
	config := ConnectionConfig{
		DatabaseURL:     databaseURL,
		IsDevelopment:   isDevelopment,
		MaxIdleConns:    5,
		MaxOpenConns:    20,
		ConnMaxLifetime: time.Hour,
		ServiceName:     "optimization-service",
	}
	return NewConnectionWithConfig(config)
}

func NewGatewayServiceConnection(databaseURL string, isDevelopment bool) (*DB, error) {
	config := ConnectionConfig{
		DatabaseURL:     databaseURL,
		IsDevelopment:   isDevelopment,
		MaxIdleConns:    10,
		MaxOpenConns:    50,
		ConnMaxLifetime: time.Hour,
		ServiceName:     "api-gateway",
	}
	return NewConnectionWithConfig(config)
}

func NewUserServiceConnection(databaseURL string, isDevelopment bool) (*DB, error) {
	config := ConnectionConfig{
		DatabaseURL:     databaseURL,
		IsDevelopment:   isDevelopment,
		MaxIdleConns:    8,
		MaxOpenConns:    25,
		ConnMaxLifetime: time.Hour,
		ServiceName:     "user-service",
	}
	return NewConnectionWithConfig(config)
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (db *DB) HealthCheck() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}
