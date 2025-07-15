package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLogger creates a structured logger middleware for requests
func RequestLogger(logger *logrus.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(startTime)
		
		entry := logger.WithFields(logrus.Fields{
			"service":    "api-gateway",
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency":    latency,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		// Add user context if available
		if userID, exists := c.Get("user_id"); exists {
			entry = entry.WithField("user_id", userID)
		}

		// Add query parameters if present
		if c.Request.URL.RawQuery != "" {
			entry = entry.WithField("query", c.Request.URL.RawQuery)
		}

		// Log based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			entry.Error("Internal Server Error")
		case status >= 400:
			entry.Warn("Client Error")
		case status >= 300:
			entry.Info("Redirect")
		default:
			entry.Info("Request completed")
		}
	})
}

// ErrorLogger logs detailed error information
func ErrorLogger(logger *logrus.Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Log any errors that occurred during request processing
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.WithFields(logrus.Fields{
					"service":   "api-gateway",
					"method":    c.Request.Method,
					"path":      c.Request.URL.Path,
					"error":     err.Error(),
					"client_ip": c.ClientIP(),
				}).Error("Request error")
			}
		}
	})
}