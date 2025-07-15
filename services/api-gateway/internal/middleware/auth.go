package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// AuthRequired middleware validates Supabase JWT tokens
func AuthRequired(supabaseJWTSecret string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
			c.Abort()
			return
		}

		// Parse and validate Supabase JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("invalid signing method", jwt.ValidationErrorSignatureInvalid)
			}
			return []byte(supabaseJWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Set user information in context from Supabase JWT claims
		// In Supabase, the user ID is in the "sub" claim
		if userID, exists := claims["sub"]; exists {
			c.Set("user_id", userID)
		}
		if email, exists := claims["email"]; exists {
			c.Set("email", email)
		}
		// Phone number is also available in Supabase JWT
		if phone, exists := claims["phone"]; exists {
			c.Set("phone", phone)
		}

		c.Next()
	})
}

// OptionalAuth middleware validates Supabase JWT tokens but doesn't require them
func OptionalAuth(supabaseJWTSecret string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.Next()
			return
		}

		// Parse and validate Supabase JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("invalid signing method", jwt.ValidationErrorSignatureInvalid)
			}
			return []byte(supabaseJWTSecret), nil
		})

		if err == nil && token.Valid {
			// Extract claims if token is valid
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				// In Supabase, the user ID is in the "sub" claim
				if userID, exists := claims["sub"]; exists {
					c.Set("user_id", userID)
				}
				if email, exists := claims["email"]; exists {
					c.Set("email", email)
				}
				if phone, exists := claims["phone"]; exists {
					c.Set("phone", phone)
				}
			}
		}

		c.Next()
	})
}