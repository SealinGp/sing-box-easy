package user

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.InitDefault()
	dir, err := os.MkdirTemp("", "user_test")
	if err != nil {
		panic(err)
	}
	if err := database.Init(filepath.Join(dir, "test.db")); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = database.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func newTestManager(t *testing.T) *ManagerXORM {
	t.Helper()
	m := NewManagerXORM("", "")
	if err := m.Init(); err != nil {
		t.Fatalf("failed to init user manager: %v", err)
	}
	// Truncate tables to ensure isolated tests
	e, err := database.GetEngine()
	if err != nil {
		t.Fatalf("failed to get engine: %v", err)
	}
	if _, err := e.Exec("DELETE FROM users"); err != nil {
		t.Fatalf("failed to truncate users: %v", err)
	}
	if _, err := e.Exec("DELETE FROM user_sessions"); err != nil {
		t.Fatalf("failed to truncate user_sessions: %v", err)
	}
	return m
}

func TestCreateAndAuthenticateUser(t *testing.T) {
	m := newTestManager(t)

	// Create user
	u, err := m.CreateUser("bob", "secret123", "viewer")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if u.Username != "bob" {
		t.Errorf("expected username bob, got %q", u.Username)
	}
	if u.Role != "viewer" {
		t.Errorf("expected role viewer, got %q", u.Role)
	}

	// Authenticate success
	authenticatedUser, token, err := m.Authenticate("bob", "secret123")
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}
	if authenticatedUser.ID != u.ID {
		t.Errorf("expected authenticated user ID %d, got %d", u.ID, authenticatedUser.ID)
	}
	if token == "" {
		t.Errorf("expected non-empty token")
	}

	// Validate session
	validatedUser, err := m.ValidateSession(token)
	if err != nil {
		t.Fatalf("failed to validate session: %v", err)
	}
	if validatedUser.ID != u.ID {
		t.Errorf("expected validated user ID %d, got %d", u.ID, validatedUser.ID)
	}

	// Authenticate fail (wrong password)
	_, _, err = m.Authenticate("bob", "wrongpassword")
	if err != ErrInvalidPassword {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}

	// Authenticate fail (non-existent user)
	_, _, err = m.Authenticate("nobody", "secret123")
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	// Logout
	if err := m.Logout(token); err != nil {
		t.Fatalf("failed to logout: %v", err)
	}

	// Validate session after logout
	_, err = m.ValidateSession(token)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestUserManagementConstraints(t *testing.T) {
	m := newTestManager(t)

	// Seed an admin first
	admin, err := m.CreateUser("admin", "adminpass", "admin")
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Cannot delete the last administrator
	err = m.DeleteUser(admin.ID)
	if err != ErrLastAdminDeletion {
		t.Errorf("expected ErrLastAdminDeletion when deleting last admin, got %v", err)
	}

	// Demote attempt on last admin
	_, err = m.UpdateUser(admin.ID, "", "", "viewer")
	if err == nil || err.Error() != "cannot demote the last administrator" {
		t.Errorf("expected error demoting last admin, got %v", err)
	}

	// Create second admin
	admin2, err := m.CreateUser("admin2", "adminpass", "admin")
	if err != nil {
		t.Fatalf("failed to create second admin: %v", err)
	}
	_ = admin2

	// Now can delete the first admin
	err = m.DeleteUser(admin.ID)
	if err != nil {
		t.Errorf("expected successful deletion of first admin, got %v", err)
	}
}

func TestSeedCustomAdmin(t *testing.T) {
	// Clean database first
	e, err := database.GetEngine()
	if err != nil {
		t.Fatalf("failed to get engine: %v", err)
	}
	if _, err := e.Exec("DELETE FROM users"); err != nil {
		t.Fatalf("failed to truncate users: %v", err)
	}
	if _, err := e.Exec("DELETE FROM user_sessions"); err != nil {
		t.Fatalf("failed to truncate user_sessions: %v", err)
	}

	m := NewManagerXORM("myadmin", "mypass123")
	if err := m.Init(); err != nil {
		t.Fatalf("failed to init user manager: %v", err)
	}

	// Verify custom admin can authenticate
	u, token, err := m.Authenticate("myadmin", "mypass123")
	if err != nil {
		t.Fatalf("failed to authenticate custom admin: %v", err)
	}
	if u.Username != "myadmin" {
		t.Errorf("expected username myadmin, got %q", u.Username)
	}
	if token == "" {
		t.Errorf("expected non-empty token")
	}
}

