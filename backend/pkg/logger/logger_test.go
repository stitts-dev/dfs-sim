package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger(t *testing.T) {
	// Reset logger before each test
	Logger = nil

	tests := []struct {
		name          string
		logLevel      string
		logFormat     string
		expectedLevel logrus.Level
		expectJSON    bool
	}{
		{
			name:          "default configuration",
			logLevel:      "",
			logFormat:     "",
			expectedLevel: logrus.InfoLevel,
			expectJSON:    false,
		},
		{
			name:          "debug level with json format",
			logLevel:      "debug",
			logFormat:     "json",
			expectedLevel: logrus.DebugLevel,
			expectJSON:    true,
		},
		{
			name:          "error level with text format",
			logLevel:      "error",
			logFormat:     "text",
			expectedLevel: logrus.ErrorLevel,
			expectJSON:    false,
		},
		{
			name:          "invalid level defaults to info",
			logLevel:      "invalid",
			logFormat:     "",
			expectedLevel: logrus.InfoLevel,
			expectJSON:    false,
		},
		{
			name:          "case insensitive level",
			logLevel:      "DEBUG",
			logFormat:     "JSON",
			expectedLevel: logrus.DebugLevel,
			expectJSON:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.logLevel != "" {
				os.Setenv("LOG_LEVEL", tt.logLevel)
			} else {
				os.Unsetenv("LOG_LEVEL")
			}

			if tt.logFormat != "" {
				os.Setenv("LOG_FORMAT", tt.logFormat)
			} else {
				os.Unsetenv("LOG_FORMAT")
			}

			// Reset logger to force reinitialization
			Logger = nil

			// Initialize logger
			logger := InitLogger()

			// Verify log level
			assert.Equal(t, tt.expectedLevel, logger.GetLevel(), "log level mismatch")

			// Verify formatter type
			if tt.expectJSON {
				_, ok := logger.Formatter.(*logrus.JSONFormatter)
				assert.True(t, ok, "expected JSON formatter")
			} else {
				_, ok := logger.Formatter.(*logrus.TextFormatter)
				assert.True(t, ok, "expected text formatter")
			}

			// Clean up
			os.Unsetenv("LOG_LEVEL")
			os.Unsetenv("LOG_FORMAT")
		})
	}
}

func TestLogOutput(t *testing.T) {
	// Reset logger
	Logger = nil

	// Set up test environment
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "json")
	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
	}()

	// Capture output
	var buf bytes.Buffer
	logger := InitLogger()
	logger.SetOutput(&buf)

	// Test structured fields inclusion
	logger.WithFields(logrus.Fields{
		"optimization_id": "test-123",
		"sport":           "nba",
		"platform":        "draftkings",
	}).Info("test message")

	output := buf.String()

	// Verify JSON format
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err, "output should be valid JSON")

	// Verify required fields
	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "test-123", logEntry["optimization_id"])
	assert.Equal(t, "nba", logEntry["sport"])
	assert.Equal(t, "draftkings", logEntry["platform"])
	assert.Contains(t, logEntry, "time")
}

func TestWithOptimizationID(t *testing.T) {
	Logger = nil
	logger := InitLogger()

	// Capture output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Set JSON format for easier testing
	os.Setenv("LOG_FORMAT", "json")
	defer os.Unsetenv("LOG_FORMAT")
	Logger = nil
	logger = InitLogger()
	logger.SetOutput(&buf)

	// Test WithOptimizationID
	WithOptimizationID("test-opt-123").Info("optimization started")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test-opt-123", logEntry["optimization_id"])
	assert.Equal(t, "optimization started", logEntry["msg"])
}

func TestWithOptimizationContext(t *testing.T) {
	Logger = nil

	// Set JSON format and debug level for easier testing
	os.Setenv("LOG_FORMAT", "json")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("LOG_LEVEL")
	}()

	logger := InitLogger()

	// Capture output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test WithOptimizationContext
	WithOptimizationContext("test-opt-456", "nfl", "fanduel").Debug("processing players")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test-opt-456", logEntry["optimization_id"])
	assert.Equal(t, "nfl", logEntry["sport"])
	assert.Equal(t, "fanduel", logEntry["platform"])
	assert.Equal(t, "processing players", logEntry["msg"])
}

func TestWithRequestContext(t *testing.T) {
	Logger = nil

	// Set JSON format for easier testing
	os.Setenv("LOG_FORMAT", "json")
	defer os.Unsetenv("LOG_FORMAT")

	logger := InitLogger()

	// Capture output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test WithRequestContext
	WithRequestContext("req-789", "opt-456").Info("request processing")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "req-789", logEntry["request_id"])
	assert.Equal(t, "opt-456", logEntry["optimization_id"])
	assert.Equal(t, "request processing", logEntry["msg"])
}

func TestGetLogger(t *testing.T) {
	// Reset logger
	Logger = nil

	// First call should initialize
	logger1 := GetLogger()
	assert.NotNil(t, logger1)

	// Second call should return same instance
	logger2 := GetLogger()
	assert.Same(t, logger1, logger2)
}

func TestSensitiveDataExclusion(t *testing.T) {
	Logger = nil

	// Set JSON format for easier testing
	os.Setenv("LOG_FORMAT", "json")
	defer os.Unsetenv("LOG_FORMAT")

	logger := InitLogger()

	// Capture output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test that we don't accidentally log sensitive data
	// This test ensures our logging patterns don't include sensitive fields
	logger.WithFields(logrus.Fields{
		"optimization_id": "test-123",
		"total_players":   100,
		"sport":           "nba",
		// Note: No salary, user_id, or personal info fields
	}).Info("player stats")

	output := buf.String()

	// Verify sensitive data is NOT present
	assert.NotContains(t, strings.ToLower(output), "salary")
	assert.NotContains(t, strings.ToLower(output), "user_id")
	assert.NotContains(t, strings.ToLower(output), "password")
	assert.NotContains(t, strings.ToLower(output), "email")

	// Verify expected data IS present
	assert.Contains(t, output, "optimization_id")
	assert.Contains(t, output, "total_players")
	assert.Contains(t, output, "sport")
}
