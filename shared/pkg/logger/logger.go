package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// InitLogger initializes the structured logger with proper configuration
func InitLogger(logLevel string, isDevelopment bool) *logrus.Logger {
	log := logrus.New()

	// Override with environment if not provided
	if logLevel == "" {
		logLevel = os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			if isDevelopment {
				logLevel = "debug"
			} else {
				logLevel = "info"
			}
		}
	}

	if level, err := logrus.ParseLevel(strings.ToLower(logLevel)); err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(logrus.InfoLevel)
		log.WithField("invalid_level", logLevel).Warn("Invalid LOG_LEVEL, using INFO")
	}

	// Set formatter based on environment
	if !isDevelopment || strings.ToLower(os.Getenv("LOG_FORMAT")) == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
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
		return InitLogger("info", false)
	}
	return Logger
}

// WithService creates a logger with service context
func WithService(serviceName string) *logrus.Entry {
	return GetLogger().WithField("service", serviceName)
}

// WithCorrelationID creates a logger with correlation ID for distributed tracing
func WithCorrelationID(correlationID string) *logrus.Entry {
	return GetLogger().WithField("correlation_id", correlationID)
}

// WithServiceContext creates a logger with service and correlation context
func WithServiceContext(serviceName, correlationID string) *logrus.Entry {
	return GetLogger().WithFields(logrus.Fields{
		"service":        serviceName,
		"correlation_id": correlationID,
	})
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

// WithGolfContext creates a logger with golf-specific context
func WithGolfContext(tournamentID, playerID string) *logrus.Entry {
	fields := logrus.Fields{}
	if tournamentID != "" {
		fields["tournament_id"] = tournamentID
	}
	if playerID != "" {
		fields["player_id"] = playerID
	}
	return GetLogger().WithFields(fields)
}

// WithHTTPContext creates a logger with HTTP request context
func WithHTTPContext(method, path, userAgent string) *logrus.Entry {
	return GetLogger().WithFields(logrus.Fields{
		"http_method":    method,
		"http_path":      path,
		"http_user_agent": userAgent,
	})
}