package main

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Role      string `json:"role"`   // "admin" or "user"
	Status    string `json:"status"` // "active" or "disabled"
	Password  string `json:"-"`      // Don't include in JSON responses
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

type Menu struct {
	Route    string `json:"route"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Icon     string `json:"icon"`
	Badge    string `json:"badge,omitempty"`
	Children []Menu `json:"children,omitempty"`
}

type MenuResponse struct {
	Menu []Menu `json:"menu"`
}

type UserResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"` // "admin" or "user"
	Avatar   string `json:"avatar,omitempty"`
}

type UpdateUserRequest struct {
	Email  string `json:"email,omitempty"`
	Name   string `json:"name,omitempty"`
	Role   string `json:"role,omitempty"`
	Status string `json:"status,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

type UsersListResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}

// Database connection
var db *sql.DB

// Initialize database and create tables
func initDatabase() error {
	var err error
	db, err = sql.Open("sqlite3", "./webadmin.db")
	if err != nil {
		return err
	}

	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		avatar TEXT DEFAULT '',
		role TEXT NOT NULL DEFAULT 'user',
		status TEXT NOT NULL DEFAULT 'active',
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create refresh_tokens table
	createTokensTable := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	if _, err := db.Exec(createUsersTable); err != nil {
		return err
	}

	if _, err := db.Exec(createTokensTable); err != nil {
		return err
	}

	// Add migration for existing databases to add role and status columns
	addRoleColumn := `ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'user';`
	addStatusColumn := `ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active';`
	addUpdatedAtColumn := `ALTER TABLE users ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;`

	// These will fail if columns already exist, which is fine
	db.Exec(addRoleColumn)
	db.Exec(addStatusColumn)
	db.Exec(addUpdatedAtColumn)

	// Create default admin user if not exists
	return createDefaultUser()
}

// Create default admin user
func createDefaultUser() error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("adminpwd"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = db.Exec(`
			INSERT INTO users (username, email, name, avatar, role, status, password) 
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"admin", "admin@ng-matero.com", "Administrator", "/images/avatar-default.jpg", "admin", "active", string(hashedPassword))
		return err
	}

	return nil
}

// User database operations
func getUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, username, email, name, avatar, role, status, password, 
		       created_at, updated_at
		FROM users WHERE username = ? AND status = 'active'`, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.Name, &user.Avatar,
		&user.Role, &user.Status, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func getUserByID(id int) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, username, email, name, avatar, role, status, 
		       created_at, updated_at
		FROM users WHERE id = ?`, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Name, &user.Avatar,
		&user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func getAllUsers(limit, offset int) ([]User, int, error) {
	var users []User
	var total int

	// Get total count
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get users with pagination
	rows, err := db.Query(`
		SELECT id, username, email, name, avatar, role, status, 
		       created_at, updated_at
		FROM users 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Name,
			&user.Avatar, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
}

func createUser(req CreateUserRequest) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if req.Role == "" {
		req.Role = "user"
	}

	if req.Avatar == "" {
		req.Avatar = "/images/avatar-default.jpg"
	}

	result, err := db.Exec(`
		INSERT INTO users (username, email, name, avatar, role, status, password) 
		VALUES (?, ?, ?, ?, ?, 'active', ?)`,
		req.Username, req.Email, req.Name, req.Avatar, req.Role, string(hashedPassword))
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return getUserByID(int(id))
}

func updateUser(id int, req UpdateUserRequest) (*User, error) {
	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}

	if req.Email != "" {
		setParts = append(setParts, "email = ?")
		args = append(args, req.Email)
	}
	if req.Name != "" {
		setParts = append(setParts, "name = ?")
		args = append(args, req.Name)
	}
	if req.Role != "" {
		setParts = append(setParts, "role = ?")
		args = append(args, req.Role)
	}
	if req.Status != "" {
		setParts = append(setParts, "status = ?")
		args = append(args, req.Status)
	}
	if req.Avatar != "" {
		setParts = append(setParts, "avatar = ?")
		args = append(args, req.Avatar)
	}

	if len(setParts) == 0 {
		return getUserByID(id)
	}

	setParts = append(setParts, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE users SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return getUserByID(id)
}

func deleteUser(id int) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func saveRefreshToken(userID int, token string, expiresAt time.Time) error {
	_, err := db.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at) 
		VALUES (?, ?, ?)`, userID, token, expiresAt)
	return err
}

func validateRefreshToken(token string) (int, error) {
	var userID int
	var expiresAt time.Time
	err := db.QueryRow(`
		SELECT user_id, expires_at 
		FROM refresh_tokens 
		WHERE token = ?`, token).Scan(&userID, &expiresAt)
	if err != nil {
		return 0, err
	}

	if time.Now().After(expiresAt) {
		// Token expired, delete it
		db.Exec("DELETE FROM refresh_tokens WHERE token = ?", token)
		return 0, sql.ErrNoRows
	}

	return userID, nil
}

func deleteRefreshToken(token string) error {
	_, err := db.Exec("DELETE FROM refresh_tokens WHERE token = ?", token)
	return err
}
