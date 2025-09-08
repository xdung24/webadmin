package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"golang.org/x/crypto/bcrypt"
)

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

// POST /auth/login
// Login godoc
//
//	@Summary		User login
//	@Description	Authenticate user and return tokens
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			loginRequest	body		LoginRequest	true	"Login credentials"
//	@Success		200				{object}	Token
//	@Failure		400				{object}	ErrorResponse	"Invalid request body or missing required fields"
//	@Failure		401				{object}	ErrorResponse	"Invalid credentials"
//	@Failure		500				{object}	ErrorResponse	"Internal server error"
//	@Router			/auth/login [POST]
func loginHandler(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username and password are required",
		})
	}

	// Get user from database
	user, err := getUserByUsername(req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Generate tokens
	accessToken, err := generateAccessToken(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	response := Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   86400, // 24 hours in seconds
	}

	// Generate refresh token if remember me is true
	if req.RememberMe {
		refreshToken := generateRefreshToken()
		expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
		if err := saveRefreshToken(user.ID, refreshToken, expiresAt); err == nil {
			response.RefreshToken = refreshToken
		}
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// POST /auth/refresh
// Refresh access token godoc
//
//	@Summary		Refresh access token
//	@Description	Generate a new access token using a refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			refreshRequest	body		RefreshRequest	true	"Refresh token"
//	@Success		200				{object}	Token
//	@Failure		400				{object}	ErrorResponse	"Invalid request body or missing refresh token"
//	@Failure		401				{object}	ErrorResponse	"Invalid or expired refresh token"
//	@Failure		500				{object}	ErrorResponse	"Failed to generate access token"
//	@Router			/auth/refresh [POST]
func refreshHandler(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Refresh token is required",
		})
	}

	// Validate refresh token
	userID, err := validateRefreshToken(req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	// Get user
	user, err := getUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Generate new access token
	accessToken, err := generateAccessToken(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   86400, // 24 hours in seconds
	})
}

// POST /auth/logout
// Logout godoc
//
//	@Summary		User logout
//	@Description	Log out user and invalidate refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			logoutRequest	body		LogoutRequest	false	"Refresh token to invalidate (optional)"
//	@Success		200				{object}	SuccessResponse	"Logout successful message"
//	@Failure		500				{object}	ErrorResponse	"Failed to process logout"
//	@Router			/auth/logout [POST]
func logoutHandler(c *fiber.Ctx) error {
	// Get refresh token from request body if provided
	var req LogoutRequest
	c.BodyParser(&req)

	// If refresh token is provided, delete it from database
	if req.RefreshToken != "" {
		deleteRefreshToken(req.RefreshToken)
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Logged out successfully",
	})
}

// GET /user
// Get user information by id
// @Summary		Get user information
// @Description	Get user information by ID
// @Tags			user
// @Accept			json
// @Produce		json
// @Success		200				{object}	UserResponse
// @Failure		404				{object}	ErrorResponse	"User not found"
// @Failure		500				{object}	ErrorResponse	"Failed to fetch user"
// @Router			/user [GET]
func userHandler(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int)

	user, err := getUserByID(userID)
	if err != nil {
		log.Warnf("Failed to get user by ID %d: %v", userID, err)
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to fetch user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Username:  user.Username,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// GET /admin/users
// Get all users (admin only)
// @Summary		Get all users
// @Description	Retrieve a paginated list of all users (admin only)
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			limit	query		int	false	"Number of users to return"	minimum(1)	default(10)
// @Param			offset	query		int	false	"Number of users to skip"		minimum(0)	default(0)
// @Success		200		{object}	UsersListResponse
// @Failure		400		{object}	ErrorResponse	"Invalid query parameters"
// @Failure		500		{object}	ErrorResponse	"Failed to fetch users"
// @Router			/admin/users [GET]
func getUsersHandler(c *fiber.Ctx) error {
	// Parse pagination parameters
	limit := 10
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	users, total, err := getAllUsers(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to fetch users",
		})
	}

	var userResponses []UserResponse
	for _, user := range users {
		userResponses = append(userResponses, UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Username:  user.Username,
			Role:      user.Role,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(UsersListResponse{
		Users: userResponses,
		Total: total,
	})
}

// POST /admin/users
// Create new user (admin only)
// @Summary		Create new user
// @Description	Create a new user (admin only)
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			createUserRequest	body		CreateUserRequest	true	"User details"
// @Success		201					{object}	UserResponse
// @Failure		400					{object}	ErrorResponse	"Invalid request body or missing required fields"
// @Failure		409					{object}	ErrorResponse	"Username or email already exists"
// @Failure		500					{object}	ErrorResponse	"Failed to create user"
// @Router			/admin/users [POST]
func createUserHandler(c *fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid request body",
		})
	}

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Name == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Username, email, name, and password are required",
		})
	}

	// Validate role
	if req.Role != "" && req.Role != "admin" && req.Role != "user" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Role must be 'admin' or 'user'",
		})
	}

	user, err := createUser(req)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error: "Username or email already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Username:  user.Username,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// GET /admin/users/:id
// Get user by ID (admin only)
// @Summary		Get user by ID
// @Description	Get user details by ID (admin only)
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id	path		int	true	"User ID"
// @Success		200	{object}	UserResponse
// @Failure		400	{object}	ErrorResponse	"Invalid user ID"
// @Failure		404	{object}	ErrorResponse	"User not found"
// @Failure		500	{object}	ErrorResponse	"Failed to fetch user"
// @Router			/admin/users/{id} [GET]
func getUserByIDHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid user ID",
		})
	}

	user, err := getUserByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to fetch user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Username:  user.Username,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// PUT /admin/users/:id
// Update user (admin only)
// @Summary		Update user
// @Description	Update user details by ID (admin only)
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id					path		int					true	"User ID"
// @Param			updateUserRequest	body		UpdateUserRequest	true	"Updated user details"
// @Success		200					{object}	UserResponse
// @Failure		400					{object}	ErrorResponse	"Invalid user ID or request body"
// @Failure		404					{object}	ErrorResponse	"User not found"
// @Failure		409					{object}	ErrorResponse	"Email already exists"
// @Failure		500					{object}	ErrorResponse	"Failed to update user"
// @Router			/admin/users/{id} [PUT]
func updateUserHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid user ID",
		})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid request body",
		})
	}

	// Validate role if provided
	if req.Role != "" && req.Role != "admin" && req.Role != "user" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Role must be 'admin' or 'user'",
		})
	}

	// Validate status if provided
	if req.Status != "" && req.Status != "active" && req.Status != "disabled" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Status must be 'active' or 'disabled'",
		})
	}

	user, err := updateUser(id, req)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "User not found",
			})
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error: "Email already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to update user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Username:  user.Username,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// DELETE /admin/users/:id
// Delete user (admin only)
// @Summary		Delete user
// @Description	Delete user by ID (admin only)
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id	path		int	true	"User ID"
// @Success		200	{object}	SuccessResponse	"User deleted successfully message"
// @Failure		400	{object}	ErrorResponse	"Invalid user ID or cannot delete own account"
// @Failure		500	{object}	ErrorResponse	"Failed to delete user"
// @Router			/admin/users/{id} [DELETE]
func deleteUserHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid user ID",
		})
	}

	// Prevent deleting own account
	currentUserID := c.Locals("userID").(int)
	if id == currentUserID {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Cannot delete your own account",
		})
	}

	err = deleteUser(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to delete user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "User deleted successfully",
	})
}

// PUT /admin/users/:id/enable
// Enable user (admin only)
// @Summary		Enable user
// @Description	Enable a disabled user account (admin only)
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id	path		int	true	"User ID"
// @Success		200	{object}	UserResponse
// @Failure		400	{object}	ErrorResponse	"Invalid user ID"
// @Failure		400	{object}	ErrorResponse	"User not found"
// @Failure		500	{object}	ErrorResponse	"Failed to enable user"
// @Router			/admin/users/{id}/enable [PUT]
func enableUserHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid user ID",
		})
	}

	user, err := updateUser(id, UpdateUserRequest{Status: "active"})
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to enable user",
		})
	}

	return c.JSON(UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Username:  user.Username,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// PUT /admin/users/:id/disable
// Disable user (admin only)
// @Summary		Disable user
// @Description	Disable a user account (admin only)
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id	path		int	true	"User ID"
// @Success		200	{object}	UserResponse
// @Failure		400	{object}	ErrorResponse	"Invalid user ID or cannot disable own account"
// @Failure		400	{object}	ErrorResponse	"User not found"
// @Failure		500	{object}	ErrorResponse	"Failed to disable user"
// @Router			/admin/users/{id}/disable [PUT]
func disableUserHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Invalid user ID",
		})
	}

	// Prevent disabling own account
	currentUserID := c.Locals("userID").(int)
	if id == currentUserID {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "Cannot disable your own account",
		})
	}

	user, err := updateUser(id, UpdateUserRequest{Status: "disabled"})
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Failed to disable user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Username:  user.Username,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// Setup API routes
func setupRoutes(app *fiber.App) {
	// Public routes
	app.Post("/auth/login", loginHandler)
	app.Post("/auth/refresh", refreshHandler)
	app.Post("/auth/logout", logoutHandler)

	// Protected routes (require authentication)
	app.Get("/user", authMiddleware, userHandler)

	// Admin routes (require admin role)
	admin := app.Group("/admin", authMiddleware, adminMiddleware)
	admin.Get("/users", getUsersHandler)
	admin.Post("/users", createUserHandler)
	admin.Get("/users/:id", getUserByIDHandler)
	admin.Put("/users/:id", updateUserHandler)
	admin.Delete("/users/:id", deleteUserHandler)
	admin.Put("/users/:id/enable", enableUserHandler)
	admin.Put("/users/:id/disable", disableUserHandler)
}
