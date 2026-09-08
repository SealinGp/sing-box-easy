package v1_13_0

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/identity"
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

// AuthStatusResponse describes how this deployment is set up, before any
// session exists. Public by necessity: the UI needs it to decide whether to
// show the login page and the profile/user-management views, and which
// navigation layout to render.
//
// SystemType is deliberately the only host detail exposed here — it is coarse
// (three possible values) and already inferable from AuthEnabled under the
// default "auto" mode. Everything else lives behind auth in /system/info.
type AuthStatusResponse struct {
	AuthEnabled bool `json:"auth_enabled"`
	// SystemType is the distribution family: "openwrt", "debian" or "unknown".
	// OpenWrt gets a top bar instead of a sidebar, because LuCI already owns
	// the left edge of the operator's screen.
	SystemType string `json:"system_type"`
}

// GetAuthStatus reports whether this deployment requires login, and on which
// platform family it runs.
func (h *Handler) GetAuthStatus(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, AuthStatusResponse{
		AuthEnabled: h.authEnabled,
		SystemType:  string(h.systemType),
	})
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

	u, err := user.UpdateAs(h.userManager, currentUser, id, req.Username, req.Password, req.Role)
	if err != nil {
		if isOperationFault(err) {
			respondOperationError(ctx, c, err)
		} else if errors.Is(err, user.ErrUserNotFound) {
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

	err = user.DeleteAs(h.userManager, currentUser, id)
	if err != nil {
		if isOperationFault(err) {
			respondOperationError(ctx, c, err)
		} else if errors.Is(err, user.ErrUserNotFound) {
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

// Preferences belong to the authenticated account, including viewers.
func (h *Handler) GetPreferences(ctx context.Context, c *app.RequestContext) {
	u, ok := GetCurrentUser(c)
	if !ok || u.ID == 0 {
		respErr(ctx, c, CodeUnauthorized, "A signed-in account is required")
		return
	}
	preferences, err := h.userManager.GetPreferences(u.ID)
	if err != nil {
		respErr(ctx, c, CodeInternalError, "Failed to load preferences")
		return
	}
	respOK(ctx, c, preferences)
}

func (h *Handler) UpdatePreferences(ctx context.Context, c *app.RequestContext) {
	u, ok := GetCurrentUser(c)
	if !ok || u.ID == 0 {
		respErr(ctx, c, CodeUnauthorized, "A signed-in account is required")
		return
	}
	var req user.Preferences
	if err := c.BindJSON(&req); err != nil || req.OverviewOrder == nil {
		respErr(ctx, c, CodeBadRequest, "overview_order must be an array of card IDs")
		return
	}
	preferences, err := h.userManager.UpdatePreferences(u.ID, req)
	if errors.Is(err, user.ErrInvalidPreferences) {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	if err != nil {
		respErr(ctx, c, CodeInternalError, "Failed to save preferences")
		return
	}
	respOK(ctx, c, preferences)
}
