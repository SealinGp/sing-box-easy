package v1_13_0

import (
	"context"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/user"
	"github.com/SealinGp/sing-box-easy/app/pkg/user/repo"
	"github.com/cloudwego/hertz/pkg/app"
)

const (
	UserContextKey = "current_user"
)

// AuthMiddleware creates a middleware that checks the session token
func AuthMiddleware(userManager user.UserManager) app.HandlerFunc {
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
