package user

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/user/repo"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrSessionExpired    = errors.New("session expired")
	ErrSessionNotFound   = errors.New("session not found")
	ErrUsernameExists    = errors.New("username already exists")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrLastAdminDeletion = errors.New("cannot delete the last administrator")
)

// UserManager defines interface for managing users and sessions
type UserManager interface {
	Init() error
	Authenticate(username, password string) (*repo.User, string, error)
	ValidateSession(token string) (*repo.User, error)
	Logout(token string) error
	CreateUser(username, password, role string) (*repo.User, error)
	UpdateUser(id int64, username, password, role string) (*repo.User, error)
	DeleteUser(id int64) error
	GetUserByID(id int64) (*repo.User, error)
	ListUsers() ([]repo.User, error)
}

// ManagerXORM implements UserManager using XORM and SQLite
type ManagerXORM struct {
	e         *xorm.Engine
	adminUser string
	adminPass string
}

// NewManagerXORM creates a new XORM-backed user manager with optional custom admin user and pass
func NewManagerXORM(adminUser, adminPass string) *ManagerXORM {
	e, err := database.GetEngine()
	if err != nil {
		logger.Fatal("Failed to get database engine", zap.Error(err))
	}
	return &ManagerXORM{
		e:         e,
		adminUser: adminUser,
		adminPass: adminPass,
	}
}

// Init ensures tables are synchronized and seeds default admin if empty
func (m *ManagerXORM) Init() error {
	logger.Info("Initializing user manager tables")
	if err := m.e.Sync2(new(repo.User), new(repo.UserSession)); err != nil {
		logger.Error("Failed to sync user tables", zap.Error(err))
		return err
	}

	return m.seedDefaultAdmin()
}

func (m *ManagerXORM) seedDefaultAdmin() error {
	count, err := m.e.Count(new(repo.User))
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count == 0 {
		logger.Warn("No users found in database. Seeding default administrator account.")
		
		adminUser := m.adminUser
		if adminUser == "" {
			adminUser = "admin"
		}
		adminPass := m.adminPass
		if adminPass == "" {
			adminPass = "admin"
		}

		hashedPassword, err := hashPassword(adminPass)
		if err != nil {
			return fmt.Errorf("failed to hash default password: %w", err)
		}

		defaultAdmin := &repo.User{
			Username:     adminUser,
			PasswordHash: hashedPassword,
			Role:         "admin",
		}

		if _, err := m.e.Insert(defaultAdmin); err != nil {
			return fmt.Errorf("failed to insert default admin: %w", err)
		}

		logger.Warn("==================================================================")
		logger.Warn("DEFAULT ADMINISTRATOR SEEDED!")
		logger.Warn(fmt.Sprintf("Username: %s", adminUser))
		logger.Warn(fmt.Sprintf("Password: %s", adminPass))
		logger.Warn("PLEASE CHANGE THIS PASSWORD IMMEDIATELY AFTER YOUR FIRST LOGIN!")
		logger.Warn("==================================================================")
	}

	return nil
}

// Authenticate verifies password and creates a new session token
func (m *ManagerXORM) Authenticate(username, password string) (*repo.User, string, error) {
	var u repo.User
	has, err := m.e.Where("username = ?", username).Get(&u)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch user: %w", err)
	}
	if !has {
		return nil, "", ErrUserNotFound
	}

	if !checkPasswordHash(password, u.PasswordHash) {
		return nil, "", ErrInvalidPassword
	}

	// Generate session token
	token, err := generateRandomToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}

	// Session expires in 24 hours
	session := &repo.UserSession{
		Token:     token,
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if _, err := m.e.Insert(session); err != nil {
		return nil, "", fmt.Errorf("failed to save session: %w", err)
	}

	return &u, token, nil
}

// ValidateSession retrieves and validates a session, returning the associated user
func (m *ManagerXORM) ValidateSession(token string) (*repo.User, error) {
	var s repo.UserSession
	has, err := m.e.ID(token).Get(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session: %w", err)
	}
	if !has {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(s.ExpiresAt) {
		// Clean up expired session
		_, _ = m.e.ID(token).Delete(new(repo.UserSession))
		return nil, ErrSessionExpired
	}

	var u repo.User
	hasUser, err := m.e.ID(s.UserID).Get(&u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session user: %w", err)
	}
	if !hasUser {
		return nil, ErrUserNotFound
	}

	return &u, nil
}

// Logout deletes the session token
func (m *ManagerXORM) Logout(token string) error {
	_, err := m.e.ID(token).Delete(new(repo.UserSession))
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// CreateUser creates a new user with hashed password
func (m *ManagerXORM) CreateUser(username, password, role string) (*repo.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("username and password cannot be empty")
	}

	// Check unique username
	has, err := m.e.Where("username = ?", username).Exist(new(repo.User))
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if has {
		return nil, ErrUsernameExists
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if role != "admin" && role != "viewer" {
		role = "viewer"
	}

	u := &repo.User{
		Username:     username,
		PasswordHash: hashedPassword,
		Role:         role,
	}

	if _, err := m.e.Insert(u); err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return u, nil
}

// UpdateUser updates user fields
func (m *ManagerXORM) UpdateUser(id int64, username, password, role string) (*repo.User, error) {
	var u repo.User
	has, err := m.e.ID(id).Get(&u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user for update: %w", err)
	}
	if !has {
		return nil, ErrUserNotFound
	}

	session := m.e.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return nil, err
	}

	cols := []string{}

	if username != "" && username != u.Username {
		// Check uniqueness
		exists, err := session.Where("username = ? AND id != ?", username, id).Exist(new(repo.User))
		if err != nil {
			session.Rollback()
			return nil, err
		}
		if exists {
			session.Rollback()
			return nil, ErrUsernameExists
		}
		u.Username = username
		cols = append(cols, "username")
	}

	if password != "" {
		hashedPassword, err := hashPassword(password)
		if err != nil {
			session.Rollback()
			return nil, err
		}
		u.PasswordHash = hashedPassword
		cols = append(cols, "password_hash")
	}

	if role != "" && role != u.Role {
		// If demoting an admin, ensure they aren't the last admin
		if u.Role == "admin" && role != "admin" {
			count, err := session.Where("role = ?", "admin").Count(new(repo.User))
			if err != nil {
				session.Rollback()
				return nil, err
			}
			if count <= 1 {
				session.Rollback()
				return nil, errors.New("cannot demote the last administrator")
			}
		}
		if role != "admin" && role != "viewer" {
			role = "viewer"
		}
		u.Role = role
		cols = append(cols, "role")
	}

	if len(cols) > 0 {
		if _, err := session.ID(id).Cols(cols...).Update(&u); err != nil {
			session.Rollback()
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	if err := session.Commit(); err != nil {
		return nil, err
	}

	return &u, nil
}

// DeleteUser deletes a user and cleans up their sessions
func (m *ManagerXORM) DeleteUser(id int64) error {
	var u repo.User
	has, err := m.e.ID(id).Get(&u)
	if err != nil {
		return fmt.Errorf("failed to fetch user for deletion: %w", err)
	}
	if !has {
		return ErrUserNotFound
	}

	session := m.e.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return err
	}

	// Ensure we're not deleting the last admin
	if u.Role == "admin" {
		count, err := session.Where("role = ?", "admin").Count(new(repo.User))
		if err != nil {
			session.Rollback()
			return err
		}
		if count <= 1 {
			session.Rollback()
			return ErrLastAdminDeletion
		}
	}

	// Delete sessions
	if _, err := session.Where("user_id = ?", id).Delete(new(repo.UserSession)); err != nil {
		session.Rollback()
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	// Delete user
	if _, err := session.ID(id).Delete(new(repo.User)); err != nil {
		session.Rollback()
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return session.Commit()
}

// GetUserByID fetches a user by their ID
func (m *ManagerXORM) GetUserByID(id int64) (*repo.User, error) {
	var u repo.User
	has, err := m.e.ID(id).Get(&u)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// ListUsers lists all users
func (m *ManagerXORM) ListUsers() ([]repo.User, error) {
	var users []repo.User
	err := m.e.Find(&users)
	return users, err
}

// Helper functions for hashing and tokens
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
