package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/api-gateway/internal/proxy"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
)

type AuthHandler struct {
	db           *database.DB
	config       *config.Config
	logger       *logrus.Logger
	serviceProxy *proxy.ServiceProxy
}

func NewAuthHandler(db *database.DB, cfg *config.Config, logger *logrus.Logger) *AuthHandler {
	// Create service proxy for user-service communication
	serviceProxy := proxy.NewServiceProxy(cfg, logger)
	
	return &AuthHandler{
		db:           db,
		config:       cfg,
		logger:       logger,
		serviceProxy: serviceProxy,
	}
}

// All authentication methods proxy to user-service for phone-based authentication

// Register handles user registration by proxying to user-service
func (h *AuthHandler) Register(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"method": "Register",
		"path":   c.Request.URL.Path,
	}).Info("Proxying auth registration request to user-service")
	
	// Proxy the request to user-service /api/v1/auth/register
	h.serviceProxy.ProxyUserRequest(c)
}

// Login handles user login by proxying to user-service
func (h *AuthHandler) Login(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"method": "Login", 
		"path":   c.Request.URL.Path,
	}).Info("Proxying auth login request to user-service")
	
	// Proxy the request to user-service /api/v1/auth/login
	h.serviceProxy.ProxyUserRequest(c)
}

// RefreshToken handles token refresh by proxying to user-service
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"method": "RefreshToken",
		"path":   c.Request.URL.Path,
	}).Info("Proxying auth refresh request to user-service")
	
	// Proxy the request to user-service /api/v1/auth/refresh
	h.serviceProxy.ProxyUserRequest(c)
}

// Logout handles user logout by proxying to user-service
func (h *AuthHandler) Logout(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"method": "Logout",
		"path":   c.Request.URL.Path,
	}).Info("Proxying auth logout request to user-service")
	
	// Proxy the request to user-service /api/v1/auth/logout
	h.serviceProxy.ProxyUserRequest(c)
}

// VerifyOTP handles OTP verification by proxying to user-service
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"method": "VerifyOTP",
		"path":   c.Request.URL.Path,
	}).Info("Proxying auth verify request to user-service")
	
	// Proxy the request to user-service /api/v1/auth/verify
	h.serviceProxy.ProxyUserRequest(c)
}

// ResendOTP handles OTP resend by proxying to user-service
func (h *AuthHandler) ResendOTP(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"method": "ResendOTP",
		"path":   c.Request.URL.Path,
	}).Info("Proxying auth resend request to user-service")
	
	// Proxy the request to user-service /api/v1/auth/resend
	h.serviceProxy.ProxyUserRequest(c)
}

// GetCurrentUser handles getting current user by proxying to user-service
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"method": "GetCurrentUser",
		"path":   c.Request.URL.Path,
	}).Info("Proxying auth me request to user-service")
	
	// Proxy the request to user-service /api/v1/auth/me
	h.serviceProxy.ProxyUserRequest(c)
}