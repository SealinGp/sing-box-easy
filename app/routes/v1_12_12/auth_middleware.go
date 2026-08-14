package v1_13_0

import (
	"context"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/service"
	"github.com/SealinGp/sing-box-easy/app/pkg/user"
	"github.com/SealinGp/sing-box-easy/app/pkg/user/repo"
	"github.com/cloudwego/hertz/pkg/app"
)

const (
	UserContextKey = "current_user"
)

// ResolveAuthEnabled maps the server.auth config mode onto an effective
// login requirement for this host. "auto" disables login only on OpenWrt
// (router-LAN deployments); any unknown value fails safe to enabled.
func ResolveAuthEnabled(mode string, systemType service.SystemType) bool {
	switch mode {
	case appconfig.AuthDisabled:
		return false
	case appconfig.AuthAuto:
		return systemType != service.SystemOpenWRT
	default: // appconfig.AuthEnabled and anything unexpected
		return true
	}
}

// noAuthUser is the synthetic identity injected when login is disabled: every
// request acts as an administrator so downstream RequireAdmin checks and
// self-service handlers keep working unchanged.
var noAuthUser = &repo.User{ID: 0, Username: "admin", Role: "admin"}

// AuthMiddleware creates a middleware that checks the session token. When
// enabled is false (server.auth: disabled, or "auto" on OpenWrt) the check is
// bypassed and every request runs as an administrator.
func AuthMiddleware(userManager user.UserManager, enabled bool) app.HandlerFunc {
	if !enabled {
		return func(ctx context.Context, c *app.RequestContext) {
			c.Set(UserContextKey, noAuthUser)
			c.Next(ctx)
		}
	}
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			respErr(ctx, c, CodeUnauthorized, "Authorization header is missing")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			respErr(ctx, c, CodeUnauthorized, "Invalid authorization format, must be Bearer <token>")
			c.Abort()
			return
		}

		token := parts[1]
		u, err := userManager.ValidateSession(token)
		if err != nil {
			respErr(ctx, c, CodeUnauthorized, "Session is invalid or expired: "+err.Error())
			c.Abort()
			return
		}

		// Set the user in request context for down-stream handlers
		c.Set(UserContextKey, u)
		c.Next(ctx)
	}
}

// RequireAdmin middleware ensures the authenticated user is an administrator
func RequireAdmin() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		uVal, exists := c.Get(UserContextKey)
		if !exists {
			respErr(ctx, c, CodeUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		u, ok := uVal.(*repo.User)
		if !ok || u.Role != "admin" {
			respErr(ctx, c, CodeForbidden, "Administrator privilege required")
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// GetCurrentUser retrieves the user object set by AuthMiddleware
func GetCurrentUser(c *app.RequestContext) (*repo.User, bool) {
	uVal, exists := c.Get(UserContextKey)
	if !exists {
		return nil, false
	}
	u, ok := uVal.(*repo.User)
	return u, ok
}
