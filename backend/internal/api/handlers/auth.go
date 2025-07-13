package handlers

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jstittsworth/dfs-optimizer/internal/api/middleware"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db      *database.DB
	cache   *services.CacheService
	cfg     *config.Config
	smsService services.SMSService
}

type RegisterRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
}

type VerifyRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Code        string `json:"code" binding:"required,len=6"`
}

type LoginRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type ResendRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type AuthResponse struct {
	Token       string      `json:"token"`
	ExpiresAt   time.Time   `json:"expires_at"`
	User        *models.User `json:"user"`
	IsNewUser   bool        `json:"is_new_user"`
}

func NewAuthHandler(db *database.DB, cache *services.CacheService, cfg *config.Config) *AuthHandler {
	// Initialize SMS service based on configuration
	smsService := createSMSService(cfg)
	
	return &AuthHandler{
		db:         db,
		cache:      cache,
		cfg:        cfg,
		smsService: smsService,
	}
}

// createSMSService creates the appropriate SMS service based on configuration
func createSMSService(cfg *config.Config) services.SMSService {
	// Create rate limiter: max 3 SMS per hour per phone number
	rateLimiter := services.NewSMSRateLimiter(3, time.Hour)
	
	switch cfg.SMSProvider {
	case "twilio":
		if cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" && cfg.TwilioFromNumber != "" {
			return services.NewTwilioSMSService(
				cfg.TwilioAccountSID,
				cfg.TwilioAuthToken,
				cfg.TwilioFromNumber,
				rateLimiter,
			)
		}
		// Fall back to mock if Twilio credentials are missing
		fmt.Printf("⚠️ Twilio credentials missing, falling back to mock SMS service\n")
		return services.NewMockSMSService()
		
	case "supabase":
		if cfg.SupabaseURL != "" && cfg.SupabaseServiceKey != "" {
			return services.NewSupabaseSMSService(
				cfg.SupabaseServiceKey,
				cfg.SupabaseURL,
				rateLimiter,
			)
		}
		// Fall back to mock if Supabase credentials are missing
		fmt.Printf("⚠️ Supabase credentials missing, falling back to mock SMS service\n")
		return services.NewMockSMSService()
		
	case "mock":
		return services.NewMockSMSService()
		
	default:
		fmt.Printf("⚠️ Unknown SMS provider '%s', using mock SMS service\n", cfg.SMSProvider)
		return services.NewMockSMSService()
	}
}

// RegisterRoutes registers all auth routes
func (h *AuthHandler) RegisterRoutes(group *gin.RouterGroup) {
	auth := group.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/verify", h.Verify)
		auth.POST("/login", h.Login)
		auth.POST("/resend", h.ResendCode)
		auth.GET("/me", middleware.AuthRequired(h.cfg.JWTSecret), h.GetCurrentUser)
		auth.POST("/refresh", middleware.AuthRequired(h.cfg.JWTSecret), h.RefreshToken)
	}
}

// Register initiates phone number registration by sending OTP
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
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

	// Check if user already exists
	existingUser, err := models.GetUserByPhoneNumber(h.db, normalizedPhone)
	if err != nil && err != gorm.ErrRecordNotFound {
		utils.SendInternalError(c, "Failed to check existing user")
		return
	}

	// If user exists and is verified, they should use login instead
	if existingUser != nil && existingUser.PhoneVerified {
		utils.SendValidationError(c, "Phone number already registered", "Please use login instead")
		return
	}

	// Generate and send OTP
	if err := h.generateAndSendOTP(normalizedPhone); err != nil {
		utils.SendInternalError(c, "Failed to send verification code")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Verification code sent",
		"phone_number": normalizedPhone,
		"expires_in":   600, // 10 minutes
	})
}

// Verify validates OTP and completes registration/login
// POST /api/v1/auth/verify
func (h *AuthHandler) Verify(c *gin.Context) {
	var req VerifyRequest
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

	// Validate verification code
	verCode, err := models.ValidateVerificationCode(h.db, normalizedPhone, req.Code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SendValidationError(c, "Invalid or expired verification code", "")
			return
		}
		utils.SendInternalError(c, "Failed to validate verification code")
		return
	}

	// Check if code was verified successfully
	if !verCode.Verified {
		utils.SendValidationError(c, "Too many verification attempts", "Please request a new code")
		return
	}

	// Get or create user
	user, err := models.GetUserByPhoneNumber(h.db, normalizedPhone)
	isNewUser := false
	
	if err == gorm.ErrRecordNotFound {
		// Create new user
		user, err = models.CreateUser(h.db, normalizedPhone)
		if err != nil {
			utils.SendInternalError(c, "Failed to create user")
			return
		}
		isNewUser = true
	} else if err != nil {
		utils.SendInternalError(c, "Failed to get user")
		return
	}

	// Mark phone as verified and update login time
	user.PhoneVerified = true
	if err := user.UpdateLoginTime(h.db); err != nil {
		utils.SendInternalError(c, "Failed to update user")
		return
	}
	
	if err := h.db.Model(user).UpdateColumn("phone_verified", true).Error; err != nil {
		utils.SendInternalError(c, "Failed to verify phone")
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.generateJWTToken(user)
	if err != nil {
		utils.SendInternalError(c, "Failed to generate token")
		return
	}

	// Clean up used verification code
	h.db.Delete(verCode)

	utils.SendSuccess(c, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
		IsNewUser: isNewUser,
	})
}

// Login initiates login by sending OTP to existing user
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
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

	// Check if user exists and is verified
	user, err := models.GetUserByPhoneNumber(h.db, normalizedPhone)
	if err == gorm.ErrRecordNotFound {
		utils.SendValidationError(c, "Phone number not registered", "Please register first")
		return
	} else if err != nil {
		utils.SendInternalError(c, "Failed to check user")
		return
	}

	if !user.PhoneVerified {
		utils.SendValidationError(c, "Phone number not verified", "Please complete registration first")
		return
	}

	if !user.IsActive {
		utils.SendValidationError(c, "Account deactivated", "Please contact support")
		return
	}

	// Generate and send OTP
	if err := h.generateAndSendOTP(normalizedPhone); err != nil {
		utils.SendInternalError(c, "Failed to send verification code")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Verification code sent",
		"phone_number": normalizedPhone,
		"expires_in":   600, // 10 minutes
	})
}

// ResendCode resends OTP for registration or login
// POST /api/v1/auth/resend
func (h *AuthHandler) ResendCode(c *gin.Context) {
	var req ResendRequest
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

	// Rate limiting: Check if we sent a code recently
	if h.cache != nil {
		cacheKey := fmt.Sprintf("sms_rate_limit:%s", normalizedPhone)
		exists, err := h.cache.Exists(c.Request.Context(), cacheKey)
		if err == nil && exists {
			utils.SendValidationError(c, "Code sent recently", "Please wait before requesting another code")
			return
		}
	}

	// Generate and send OTP
	if err := h.generateAndSendOTP(normalizedPhone); err != nil {
		utils.SendInternalError(c, "Failed to send verification code")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Verification code sent",
		"phone_number": normalizedPhone,
		"expires_in":   600, // 10 minutes
	})
}

// GetCurrentUser returns the current authenticated user
// GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	var userIDUint uint
	switch v := userID.(type) {
	case uint:
		userIDUint = v
	case int:
		userIDUint = uint(v)
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	user, err := models.GetUserByID(h.db, userIDUint)
	if err != nil {
		utils.SendInternalError(c, "Failed to get user")
		return
	}

	utils.SendSuccess(c, user)
}

// RefreshToken generates a new JWT token for the authenticated user
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	var userIDUint uint
	switch v := userID.(type) {
	case uint:
		userIDUint = v
	case int:
		userIDUint = uint(v)
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	user, err := models.GetUserByID(h.db, userIDUint)
	if err != nil {
		utils.SendInternalError(c, "Failed to get user")
		return
	}

	if !user.IsActive {
		utils.SendUnauthorized(c, "Account deactivated")
		return
	}

	// Generate new JWT token
	token, expiresAt, err := h.generateJWTToken(user)
	if err != nil {
		utils.SendInternalError(c, "Failed to generate token")
		return
	}

	utils.SendSuccess(c, gin.H{
		"token":      token,
		"expires_at": expiresAt,
	})
}

// Helper methods

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

func (h *AuthHandler) generateAndSendOTP(phoneNumber string) error {
	// Generate 6-digit code
	code, err := h.generateOTPCode()
	if err != nil {
		return err
	}

	// Store verification code in database
	_, err = models.CreateVerificationCode(h.db, phoneNumber, code)
	if err != nil {
		return err
	}

	// Send SMS (mock for now)
	if err := h.smsService.SendOTP(phoneNumber, code); err != nil {
		return err
	}

	// Set rate limiting cache if available
	if h.cache != nil {
		cacheKey := fmt.Sprintf("sms_rate_limit:%s", phoneNumber)
		h.cache.Set(context.Background(), cacheKey, "1", 60*time.Second) // 1 minute rate limit
	}

	return nil
}

func (h *AuthHandler) generateOTPCode() (string, error) {
	// Generate secure 6-digit code
	const digits = "0123456789"
	code := make([]byte, 6)
	
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}
	
	return string(code), nil
}

func (h *AuthHandler) generateJWTToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour token
	
	claims := &middleware.Claims{
		UserID: user.ID,
		Email:  "",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "dfs-optimizer",
		},
	}
	
	// Add email to claims if available
	if user.Email != nil {
		claims.Email = *user.Email
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}
	
	return tokenString, expiresAt, nil
}