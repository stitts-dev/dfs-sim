package middleware

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// SupabaseAuthMiddleware handles Supabase JWT token validation
type SupabaseAuthMiddleware struct {
	supabaseURL string
	publicKey   *rsa.PublicKey
	httpClient  *http.Client
}

// SupabaseClaims represents Supabase JWT claims
type SupabaseClaims struct {
	Role                 string `json:"role"`
	Email                string `json:"email,omitempty"`
	Phone                string `json:"phone,omitempty"`
	EmailConfirmedAt     string `json:"email_confirmed_at,omitempty"`
	PhoneConfirmedAt     string `json:"phone_confirmed_at,omitempty"`
	AppMetadata          map[string]interface{} `json:"app_metadata,omitempty"`
	UserMetadata         map[string]interface{} `json:"user_metadata,omitempty"`
	jwt.RegisteredClaims
}

// JWKSResponse represents the JWKS endpoint response
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewSupabaseAuthMiddleware creates a new Supabase authentication middleware
func NewSupabaseAuthMiddleware(supabaseURL string) *SupabaseAuthMiddleware {
	return &SupabaseAuthMiddleware{
		supabaseURL: supabaseURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// SupabaseAuthRequired validates Supabase JWT tokens
func (m *SupabaseAuthMiddleware) SupabaseAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}
		
		// Validate Supabase JWT token
		claims, err := m.validateSupabaseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			c.Abort()
			return
		}

		// Extract user ID from claims
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", userID)
		c.Set("user_claims", claims)
		c.Set("authenticated", true)
		
		// Set additional user info if available
		if claims.Email != "" {
			c.Set("user_email", claims.Email)
		}
		if claims.Phone != "" {
			c.Set("user_phone", claims.Phone)
		}
		
		c.Next()
	}
}

// SupabaseAuthOptional validates Supabase JWT tokens but doesn't require them
func (m *SupabaseAuthMiddleware) SupabaseAuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue as anonymous user
			c.Set("authenticated", false)
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			// Invalid format, continue as anonymous user
			c.Set("authenticated", false)
			c.Next()
			return
		}
		
		// Try to validate Supabase JWT token
		claims, err := m.validateSupabaseToken(tokenString)
		if err != nil {
			// Invalid token, continue as anonymous user
			c.Set("authenticated", false)
			c.Next()
			return
		}

		// Extract user ID from claims
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			// Invalid user ID, continue as anonymous user
			c.Set("authenticated", false)
			c.Next()
			return
		}

		// Set user context for authenticated user
		c.Set("user_id", userID)
		c.Set("user_claims", claims)
		c.Set("authenticated", true)
		
		// Set additional user info if available
		if claims.Email != "" {
			c.Set("user_email", claims.Email)
		}
		if claims.Phone != "" {
			c.Set("user_phone", claims.Phone)
		}
		
		c.Next()
	}
}

// validateSupabaseToken validates a Supabase JWT token
func (m *SupabaseAuthMiddleware) validateSupabaseToken(tokenString string) (*SupabaseClaims, error) {
	// Parse token without verification first to get the kid
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &SupabaseClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Get key ID from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid kid in token header")
	}

	// Get public key for verification
	publicKey, err := m.getPublicKey(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and verify token with public key
	parsedToken, err := jwt.ParseWithClaims(tokenString, &SupabaseClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token is using RSA
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(*SupabaseClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract claims")
	}

	// Additional validation
	if claims.Role != "authenticated" {
		return nil, fmt.Errorf("invalid user role: %s", claims.Role)
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token is expired")
	}

	return claims, nil
}

// getPublicKey fetches the public key from Supabase JWKS endpoint
func (m *SupabaseAuthMiddleware) getPublicKey(kid string) (*rsa.PublicKey, error) {
	// TODO: Implement caching for public keys to avoid repeated requests
	
	jwksURL := fmt.Sprintf("%s/auth/v1/jwks", m.supabaseURL)
	resp, err := m.httpClient.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS request failed with status %d", resp.StatusCode)
	}

	var jwksResp JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwksResp); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS response: %w", err)
	}

	// Find the key with matching kid
	for _, key := range jwksResp.Keys {
		if key.Kid == kid && key.Kty == "RSA" {
			return m.parseRSAPublicKey(key)
		}
	}

	return nil, fmt.Errorf("public key not found for kid: %s", kid)
}

// parseRSAPublicKey converts JWK to RSA public key
func (m *SupabaseAuthMiddleware) parseRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// This is a simplified implementation
	// In production, you should use a proper JWK library like github.com/lestrrat-go/jwx
	return nil, fmt.Errorf("RSA public key parsing not implemented - use proper JWK library")
}

// GetUserIDFromContext extracts user ID from gin context
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user ID type in context")
	}

	return uid, nil
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	authenticated, exists := c.Get("authenticated")
	if !exists {
		return false
	}

	auth, ok := authenticated.(bool)
	return ok && auth
}