package v1_13_0

import (
	"context"
	"errors"

	"github.com/SealinGp/sing-box-easy/app/pkg/githubauth"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetGitHubAuthStatus reports whether GitHub sign-in is available and whether
// an account is currently connected.
//
// GET /github/auth/status
func (h *Handler) GetGitHubAuthStatus(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, h.githubAuth.Status())
}

// StartGitHubLogin begins an OAuth device-flow login and returns the code the
// user types on github.com. The client then polls GetGitHubLoginSession.
//
// Admin-only: the resulting token is stored server-side and used for all
// outbound GitHub calls, so it is an instance-wide credential.
//
// POST /github/auth/device
func (h *Handler) StartGitHubLogin(ctx context.Context, c *app.RequestContext) {
	session, err := h.githubAuth.StartLogin(ctx)
	if err != nil {
		if errors.Is(err, githubauth.ErrNotConfigured) {
			respErr(ctx, c, CodeConfigError, err.Error())
			return
		}
		respErr(ctx, c, CodeServiceError, err.Error())
		return
	}

	respOK(ctx, c, session.View())
}

// GetGitHubLoginSession reports progress of a pending login. The client polls
// this until status leaves "pending".
//
// GET /github/auth/device/:session_id
func (h *Handler) GetGitHubLoginSession(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		respErr(ctx, c, CodeBadRequest, "session_id is required")
		return
	}

	session, err := h.githubAuth.GetSession(sessionID)
	if err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	respOK(ctx, c, session.View())
}

// CancelGitHubLogin aborts a pending login.
//
// DELETE /github/auth/device/:session_id
func (h *Handler) CancelGitHubLogin(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		respErr(ctx, c, CodeBadRequest, "session_id is required")
		return
	}

	if err := h.githubAuth.CancelLogin(sessionID); err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	respOK(ctx, c, h.githubAuth.Status())
}

// SignOutGitHub clears the stored credential, returning update checks to
// anonymous (rate-limited) access.
//
// DELETE /github/auth
func (h *Handler) SignOutGitHub(ctx context.Context, c *app.RequestContext) {
	if err := h.githubAuth.SignOut(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, h.githubAuth.Status())
}
