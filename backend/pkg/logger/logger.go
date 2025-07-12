package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// InitLogger initializes the structured logger with proper configuration
func InitLogger() *logrus.Logger {
	log := logrus.New()

	// Set log level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // Default to info level
	}

	if level, err := logrus.ParseLevel(strings.ToLower(logLevel)); err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(logrus.InfoLevel)
		log.WithField("invalid_level", logLevel).Warn("Invalid LOG_LEVEL, using INFO")
	}

	// Set formatter based on environment
	if strings.ToLower(os.Getenv("LOG_FORMAT")) == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Set output to stdout (can be configured later for file output)
	log.SetOutput(os.Stdout)

	// Store global logger reference
	Logger = log

	return log
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	if Logger == nil {
		return InitLogger()
	}
	return Logger
}

// WithOptimizationID creates a logger with optimization context
func WithOptimizationID(optimizationID string) *logrus.Entry {
	return GetLogger().WithField("optimization_id", optimizationID)
}

// WithOptimizationContext creates a logger with full optimization context
func WithOptimizationContext(optimizationID, sport, platform string) *logrus.Entry {
	return GetLogger().WithFields(logrus.Fields{
		"optimization_id": optimizationID,
		"sport":           sport,
		"platform":        platform,
	})
}

// WithRequestContext creates a logger with request context
func WithRequestContext(requestID, optimizationID string) *logrus.Entry {
	return GetLogger().WithFields(logrus.Fields{
		"request_id":      requestID,
		"optimization_id": optimizationID,
	})
}
