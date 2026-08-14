package v1_13_0

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/user"
	"github.com/cloudwego/hertz/pkg/app"
)

// LoginRequest defines parameters for logging in
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateUserRequest defines parameters for creating a new user
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateUserRequest defines parameters for updating a user
type UpdateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// AuthStatusResponse tells the frontend whether login is required. Public:
// the UI needs it before any session exists to decide whether to show the
// login page and the profile/user-management views.
type AuthStatusResponse struct {
	AuthEnabled bool `json:"auth_enabled"`
}

// GetAuthStatus reports whether this deployment requires login.
func (h *Handler) GetAuthStatus(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, AuthStatusResponse{AuthEnabled: h.authEnabled})
}

// Login authenticates credentials and returns a session token
func (h *Handler) Login(ctx context.Context, c *app.RequestContext) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "Invalid request format")
		return
	}

	u, token, err := h.userManager.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) || errors.Is(err, user.ErrInvalidPassword) {
			respErr(ctx, c, CodeUnauthorized, "Incorrect username or password")
		} else {
			respErr(ctx, c, CodeInternalError, "Authentication error: "+err.Error())
		}
		return
	}

	respOK(ctx, c, map[string]any{
		"token": token,
		"user":  u,
	})
}

// Logout terminates the current session
func (h *Handler) Logout(ctx context.Context, c *app.RequestContext) {
	authHeader := string(c.GetHeader("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		token := parts[1]
		if err := h.userManager.Logout(token); err != nil {
			respErr(ctx, c, CodeInternalError, "Logout failed: "+err.Error())
			return
		}
	}

	respOK(ctx, c, map[string]any{"message": "Logged out successfully"})
}

// GetMe returns the currently logged in user details
func (h *Handler) GetMe(ctx context.Context, c *app.RequestContext) {
	u, ok := GetCurrentUser(c)
	if !ok {
		respErr(ctx, c, CodeUnauthorized, "Not logged in")
		return
	}
	respOK(ctx, c, u)
}

// ListUsers lists all registered users (admin only)
func (h *Handler) ListUsers(ctx context.Context, c *app.RequestContext) {
	users, err := h.userManager.ListUsers()
	if err != nil {
		respErr(ctx, c, CodeInternalError, "Failed to retrieve users: "+err.Error())
		return
	}
	respOK(ctx, c, users)
}

// CreateUser registers a new user (admin only)
func (h *Handler) CreateUser(ctx context.Context, c *app.RequestContext) {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "Invalid request format")
		return
	}

	if req.Username == "" || req.Password == "" {
		respErr(ctx, c, CodeBadRequest, "Username and password cannot be empty")
		return
	}

	u, err := h.userManager.CreateUser(req.Username, req.Password, req.Role)
	if err != nil {
		if errors.Is(err, user.ErrUsernameExists) {
			respErr(ctx, c, CodeConflict, "Username is already taken")
		} else {
			respErr(ctx, c, CodeInternalError, "Failed to create user: "+err.Error())
		}
		return
	}

	respOK(ctx, c, u)
}

// UpdateUser modifies an existing user
func (h *Handler) UpdateUser(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "Invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "Invalid request format")
		return
	}

	currentUser, ok := GetCurrentUser(c)
	if !ok {
		respErr(ctx, c, CodeUnauthorized, "Not logged in")
		return
	}

	// Security Policy: Non-admins can only update themselves, and cannot change their role
	if currentUser.Role != "admin" {
		if currentUser.ID != id {
			respErr(ctx, c, CodeForbidden, "You can only update your own profile")
			return
		}
		// Clear role change attempt for non-admin
		req.Role = ""
	}

	u, err := h.userManager.UpdateUser(id, req.Username, req.Password, req.Role)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			respErr(ctx, c, CodeNotFound, "User not found")
		} else if errors.Is(err, user.ErrUsernameExists) {
			respErr(ctx, c, CodeConflict, "Username is already taken")
		} else if strings.Contains(err.Error(), "demote the last administrator") {
			respErr(ctx, c, CodeForbidden, err.Error())
		} else {
			respErr(ctx, c, CodeInternalError, "Failed to update user: "+err.Error())
		}
		return
	}

	respOK(ctx, c, u)
}

// DeleteUser removes a user (admin only)
func (h *Handler) DeleteUser(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "Invalid user ID")
		return
	}

	currentUser, ok := GetCurrentUser(c)
	if !ok {
		respErr(ctx, c, CodeUnauthorized, "Not logged in")
		return
	}

	// Cannot delete yourself
	if currentUser.ID == id {
		respErr(ctx, c, CodeForbidden, "You cannot delete your own account")
		return
	}

	err = h.userManager.DeleteUser(id)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			respErr(ctx, c, CodeNotFound, "User not found")
		} else if errors.Is(err, user.ErrLastAdminDeletion) {
			respErr(ctx, c, CodeForbidden, "Cannot delete the last administrator account")
		} else {
			respErr(ctx, c, CodeInternalError, "Failed to delete user: "+err.Error())
		}
		return
	}

	respOK(ctx, c, map[string]any{"message": "User deleted successfully"})
}
