package main

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// jwtSecretString will be set at build time via -ldflags -X
var jwtSecretString = "default-secret-change-me"

// Convert string to byte slice for use with JWT library
var jwtSecret = []byte(jwtSecretString)

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Generate JWT access token
func generateAccessToken(user *User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // 24 hours
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Generate refresh token
func generateRefreshToken() string {
	// Generate a simple refresh token using JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return ""
	}
	return tokenString
}

// Validate JWT token
func validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenMalformed
	}

	return claims, nil
}

// Auth middleware to validate JWT tokens
func authMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authorization header format",
		})
	}

	token := parts[1]
	claims, err := validateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Store user info in context
	c.Locals("userID", claims.UserID)
	c.Locals("username", claims.Username)
	c.Locals("role", claims.Role)

	return c.Next()
}

// Admin middleware to ensure user has admin role
func adminMiddleware(c *fiber.Ctx) error {
	role := c.Locals("role")
	if role == nil || role.(string) != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	return c.Next()
}
