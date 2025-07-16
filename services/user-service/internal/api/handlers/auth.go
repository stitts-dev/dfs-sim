package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/supabase-community/gotrue-go/types"
	"github.com/supabase-community/supabase-go"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/user-service/internal/models"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	db         *database.DB
	config      *config.Config
	logger      *logrus.Logger
	supabase   *supabase.Client
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *database.DB, cfg *config.Config, logger *logrus.Logger) *AuthHandler {
	fmt.Printf("🔑 AuthHandler: Creating new auth handler\n")

	// Initialize Supabase client
	supabaseClient, err := supabase.NewClient(
		cfg.SupabaseURL,
		cfg.SupabaseServiceKey,
		nil,
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Supabase client")
	}

	fmt.Printf("🔑 AuthHandler: Auth handler created successfully\n")
	return &AuthHandler{
		db:         db,
		config:      cfg,
		logger:      logger,
		supabase:   supabaseClient,
	}
}

// Types for request/response
type OTPRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type VerifyRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Code        string `json:"code" binding:"required,len=6"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         *models.User `json:"user"`
	IsNewUser    bool         `json:"is_new_user"`
}

type SupabaseUser struct {
	ID       string `json:"id"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Metadata struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"user_metadata"`
}

// Register initiates phone number registration by sending OTP via Supabase
func (h *AuthHandler) Register(c *gin.Context) {
	var req OTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate and normalize phone number
	normalizedPhone, err := h.validateAndNormalizePhone(req.PhoneNumber)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	// Use Supabase to send OTP
	err = h.supabase.Auth.OTP(types.OTPRequest{
		Phone:      normalizedPhone,
		CreateUser: true,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to send OTP via Supabase")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Verification code sent",
		"phone_number": normalizedPhone,
		"expires_in":   3600, // 1 hour (Supabase default)
	})
}

// Login handles user login with phone number (same as register in Supabase)
func (h *AuthHandler) Login(c *gin.Context) {
	// In Supabase phone auth, login and register are the same operation
	// Supabase will create the user if they don't exist, or send OTP if they do
	h.Register(c)
}

// VerifyOTP handles OTP verification using Supabase
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate and normalize phone number
	normalizedPhone, err := h.validateAndNormalizePhone(req.PhoneNumber)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	// Verify OTP with Supabase
	response, err := h.supabase.Auth.VerifyForUser(types.VerifyForUserRequest{
		Type:   "sms",
		Token:  req.Code,
		Phone:  normalizedPhone,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to verify OTP with Supabase")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification code"})
		return
	}

	// Check if we got a valid session
	if response.AccessToken == "" || response.User.ID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification code"})
		return
	}

	// Use Supabase user ID
	supabaseUserID := response.User.ID

	// Get or create user in our database
	user, err := models.GetUserByID(h.db, supabaseUserID)
	isNewUser := false

	if err == gorm.ErrRecordNotFound {
		// Create new user
		user, err = models.CreateUser(h.db, supabaseUserID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to create user")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		isNewUser = true

		// Create default preferences for new user
		if _, err := models.CreateUserPreferences(h.db, supabaseUserID); err != nil {
			h.logger.WithError(err).Error("Failed to create user preferences")
			// Don't fail the request, just log the error
		}
	} else if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Update login time
	if err := user.UpdateLoginTime(h.db); err != nil {
		h.logger.WithError(err).Error("Failed to update login time")
		// Don't fail the request, just log the error
	}

	// Return Supabase session tokens
	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    time.Unix(response.ExpiresAt, 0),
		User:         user,
		IsNewUser:    isNewUser,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// ResendCode resends OTP for registration or login
func (h *AuthHandler) ResendCode(c *gin.Context) {
	// In Supabase, resending is the same as initial OTP request
	h.Register(c)
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Parse UUID from context
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		h.logger.WithError(err).Error("Invalid user ID format")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	user, err := models.GetUserByID(h.db, userUUID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// RefreshToken handles token refresh using Supabase
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get refresh token from request
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Refresh session with Supabase
	response, err := h.supabase.Auth.RefreshToken(req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Error("Failed to refresh session")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	if response.AccessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  response.AccessToken,
		"refresh_token": response.RefreshToken,
		"expires_at":    time.Unix(response.ExpiresAt, 0),
	})
}

// Helper functions

// validateAndNormalizePhone validates and normalizes phone number to E.164 format
func (h *AuthHandler) validateAndNormalizePhone(phone string) (string, error) {
	// Remove all non-digit characters except +
	re := regexp.MustCompile(`[^\d+]`)
	cleaned := re.ReplaceAllString(phone, "")

	// Add + if not present
	if !regexp.MustCompile(`^\+`).MatchString(cleaned) {
		// Assume US number if no country code
		if regexp.MustCompile(`^\d{10}$`).MatchString(cleaned) {
			cleaned = "+1" + cleaned
		} else {
			return "", fmt.Errorf("invalid phone number format")
		}
	}

	// Validate E.164 format
	if !regexp.MustCompile(`^\+[1-9]\d{1,14}$`).MatchString(cleaned) {
		return "", fmt.Errorf("invalid phone number format")
	}

	return cleaned, nil
}

