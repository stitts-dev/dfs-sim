package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jstittsworth/dfs-optimizer/internal/api/middleware"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

// SupabaseAuthHandler handles Supabase-based authentication
type SupabaseAuthHandler struct {
	db                *database.DB
	cfg               *config.Config
	supabaseUser      *services.SupabaseUserService
	authMiddleware    *middleware.SupabaseAuthMiddleware
}

// SupabaseLoginRequest represents phone login request
type SupabaseLoginRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

// SupabaseVerifyRequest represents OTP verification request
type SupabaseVerifyRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Code        string `json:"code" binding:"required,len=6"`
}

// SupabaseAuthResponse represents authentication response
type SupabaseAuthResponse struct {
	AccessToken  string                `json:"access_token"`
	RefreshToken string                `json:"refresh_token"`
	ExpiresIn    int                   `json:"expires_in"`
	TokenType    string                `json:"token_type"`
	User         *models.SupabaseUser  `json:"user"`
	IsNewUser    bool                  `json:"is_new_user"`
}

// NewSupabaseAuthHandler creates a new Supabase auth handler
func NewSupabaseAuthHandler(db *database.DB, cfg *config.Config) *SupabaseAuthHandler {
	supabaseUserService := services.NewSupabaseUserService(cfg.SupabaseURL, cfg.SupabaseServiceKey)
	authMiddleware := middleware.NewSupabaseAuthMiddleware(cfg.SupabaseURL)

	return &SupabaseAuthHandler{
		db:             db,
		cfg:            cfg,
		supabaseUser:   supabaseUserService,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers all Supabase auth routes
func (h *SupabaseAuthHandler) RegisterRoutes(group *gin.RouterGroup) {
	supabaseAuth := group.Group("/auth/supabase")
	{
		supabaseAuth.POST("/login", h.LoginWithPhone)
		supabaseAuth.POST("/verify", h.VerifyOTP)
		supabaseAuth.POST("/logout", h.authMiddleware.SupabaseAuthRequired(), h.Logout)
		supabaseAuth.GET("/me", h.authMiddleware.SupabaseAuthRequired(), h.GetCurrentUser)
		supabaseAuth.POST("/refresh", h.RefreshToken)
	}
}

// LoginWithPhone initiates phone authentication via Supabase
// POST /api/v1/auth/supabase/login
func (h *SupabaseAuthHandler) LoginWithPhone(c *gin.Context) {
	var req SupabaseLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Validate and normalize phone number (reuse logic from existing auth handler)
	normalizedPhone, err := h.validateAndNormalizePhone(req.PhoneNumber)
	if err != nil {
		utils.SendValidationError(c, "Invalid phone number", err.Error())
		return
	}

	// Use existing Supabase SMS service to send OTP
	// This integrates with the circuit breaker and rate limiting already implemented
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Note: For full Supabase Auth integration, we would call:
	// supabase.auth.signInWithOtp({ phone: normalizedPhone })
	// For now, we'll use the existing SMS service pattern

	// Send OTP through Supabase Auth API
	otpURL := h.cfg.SupabaseURL + "/auth/v1/otp"
	payload := map[string]string{
		"phone": normalizedPhone,
	}

	// Use HTTP client to call Supabase Auth OTP endpoint
	if err := h.sendSupabaseOTP(ctx, otpURL, payload); err != nil {
		utils.SendInternalError(c, "Failed to send verification code")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Verification code sent via Supabase",
		"phone_number": normalizedPhone,
		"expires_in":   600, // 10 minutes
		"provider":     "supabase",
	})
}

// VerifyOTP verifies OTP and completes authentication via Supabase
// POST /api/v1/auth/supabase/verify
func (h *SupabaseAuthHandler) VerifyOTP(c *gin.Context) {
	var req SupabaseVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Validate and normalize phone number
	normalizedPhone, err := h.validateAndNormalizePhone(req.PhoneNumber)
	if err != nil {
		utils.SendValidationError(c, "Invalid phone number", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Verify OTP with Supabase Auth
	authResponse, err := h.verifySupabaseOTP(ctx, normalizedPhone, req.Code)
	if err != nil {
		utils.SendValidationError(c, "Invalid verification code", err.Error())
		return
	}

	// Extract user ID from Supabase response
	userID, err := uuid.Parse(authResponse.User.ID.String())
	if err != nil {
		utils.SendInternalError(c, "Invalid user ID from Supabase")
		return
	}

	// Check if user exists in our database, create if not
	user, err := models.GetSupabaseUserByID(h.db, userID)
	isNewUser := false

	if err != nil {
		// User doesn't exist, create them
		firstName := ""
		lastName := ""
		if authResponse.User.UserMetadata != nil {
			if fn, ok := authResponse.User.UserMetadata["first_name"].(string); ok {
				firstName = fn
			}
			if ln, ok := authResponse.User.UserMetadata["last_name"].(string); ok {
				lastName = ln
			}
		}

		var fnPtr, lnPtr *string
		if firstName != "" {
			fnPtr = &firstName
		}
		if lastName != "" {
			lnPtr = &lastName
		}

		user, err = models.CreateSupabaseUser(h.db, userID, normalizedPhone, fnPtr, lnPtr)
		if err != nil {
			utils.SendInternalError(c, "Failed to create user profile")
			return
		}

		// Create default preferences
		_, err = models.CreateSupabaseUserPreferences(h.db, userID)
		if err != nil {
			utils.SendInternalError(c, "Failed to create user preferences")
			return
		}

		isNewUser = true
	}

	// Load user with preferences
	user, err = models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to load user profile")
		return
	}

	utils.SendSuccess(c, SupabaseAuthResponse{
		AccessToken:  authResponse.AccessToken,
		RefreshToken: authResponse.RefreshToken,
		ExpiresIn:    authResponse.ExpiresIn,
		TokenType:    "bearer",
		User:         user,
		IsNewUser:    isNewUser,
	})
}

// GetCurrentUser returns the current authenticated user via Supabase
// GET /api/v1/auth/supabase/me
func (h *SupabaseAuthHandler) GetCurrentUser(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	user, err := models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to get user profile")
		return
	}

	utils.SendSuccess(c, user)
}

// RefreshToken refreshes the Supabase access token
// POST /api/v1/auth/supabase/refresh
func (h *SupabaseAuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Call Supabase refresh token endpoint
	refreshResponse, err := h.refreshSupabaseToken(ctx, req.RefreshToken)
	if err != nil {
		utils.SendUnauthorized(c, "Token refresh failed")
		return
	}

	utils.SendSuccess(c, gin.H{
		"access_token":  refreshResponse.AccessToken,
		"refresh_token": refreshResponse.RefreshToken,
		"expires_in":    refreshResponse.ExpiresIn,
		"token_type":    "bearer",
	})
}

// Logout signs out from Supabase
// POST /api/v1/auth/supabase/logout
func (h *SupabaseAuthHandler) Logout(c *gin.Context) {
	// Get token from Authorization header
	token := c.GetHeader("Authorization")
	if token == "" {
		utils.SendUnauthorized(c, "Authorization header required")
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Call Supabase logout endpoint
	if err := h.logoutFromSupabase(ctx, token); err != nil {
		utils.SendInternalError(c, "Failed to logout from Supabase")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Successfully logged out",
	})
}

// Helper methods

// validateAndNormalizePhone validates and normalizes phone number (same as existing auth handler)
func (h *SupabaseAuthHandler) validateAndNormalizePhone(phone string) (string, error) {
	// TODO: Import the same logic from the existing auth handler
	// For now, return as-is (implement proper validation)
	return phone, nil
}

// sendSupabaseOTP sends OTP via Supabase Auth API
func (h *SupabaseAuthHandler) sendSupabaseOTP(ctx context.Context, url string, payload map[string]string) error {
	// TODO: Implement HTTP call to Supabase OTP endpoint
	// This should use the service key for server-side calls
	return nil
}

// verifySupabaseOTP verifies OTP with Supabase Auth
func (h *SupabaseAuthHandler) verifySupabaseOTP(ctx context.Context, phone, code string) (*SupabaseAuthVerifyResponse, error) {
	// TODO: Implement HTTP call to Supabase verify endpoint
	// Return parsed auth response with tokens and user info
	return nil, nil
}

// refreshSupabaseToken refreshes access token via Supabase
func (h *SupabaseAuthHandler) refreshSupabaseToken(ctx context.Context, refreshToken string) (*SupabaseTokenResponse, error) {
	// TODO: Implement HTTP call to Supabase token refresh endpoint
	return nil, nil
}

// logoutFromSupabase logs out from Supabase
func (h *SupabaseAuthHandler) logoutFromSupabase(ctx context.Context, token string) error {
	// TODO: Implement HTTP call to Supabase logout endpoint
	return nil
}

// Response types for Supabase API calls

type SupabaseAuthVerifyResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID           uuid.UUID              `json:"id"`
		Email        string                 `json:"email,omitempty"`
		Phone        string                 `json:"phone,omitempty"`
		UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
	} `json:"user"`
}

type SupabaseTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}