package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/bootstrap"
	"github.com/SealinGp/sing-box-easy/app/pkg/configuration"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds"
	"github.com/SealinGp/sing-box-easy/app/pkg/system"
	"github.com/SealinGp/sing-box-easy/app/pkg/traffic"

	"github.com/SealinGp/sing-box-easy/app/pkg/appupdate"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/githubauth"
	"github.com/SealinGp/sing-box-easy/app/pkg/identity"
	"github.com/SealinGp/sing-box-easy/app/pkg/installation"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sagernet/sing/common/json"
)

// Handler holds all dependencies for v1.12.12 API handlers
type Handler struct {
	system                *system.Service
	installationModule    *installer.Service
	trafficServiceModule  *trafficflow.Service
	settingsServiceModule *settings.Service
	diagnosticsModule     *diagnostics.Service
	outboundsModule       *outbounds.Service
	configurationModule   *configuration.Service
	subscriptions         *subscription.Service
	configManager         *config.Manager
	serviceController     *service.Controller
	schedulerHandler      *schedulerHandler
	userManager           user.UserManager
	updater               *appupdate.Updater
	githubAuth            *githubauth.Manager
	// authEnabled is the resolved login requirement (server.auth × platform).
	// false means every request runs as an administrator.
	authEnabled bool
	// systemType is the detected distribution family, probed once at startup.
	// It drives both the auth default and the frontend's navigation layout.
	systemType service.SystemType
}

func NewHandler(m *bootstrap.Modules) *Handler {
	return &Handler{
		system:                m.System,
		installationModule:    m.Installation,
		trafficServiceModule:  m.TrafficService,
		settingsServiceModule: m.SettingsService,
		diagnosticsModule:     m.Diagnostics,
		outboundsModule:       m.Outbounds,
		configurationModule:   m.Configuration,
		subscriptions:         m.SubscriptionManager,
		configManager:         m.ConfigManager,
		serviceController:     m.ServiceController,
		userManager:           m.UserManager,
		updater:               m.Updater,
		githubAuth:            m.GithubAuth,
		authEnabled:           m.AuthEnabled,
		systemType:            m.SystemType,
		schedulerHandler:      newSchedulerHandler(m.SubscriptionManager),
	}
}

// Code represents business response codes
type Code uint8

// Business response codes
const (
	CodeSuccess         Code = iota // Operation successful
	CodeBadRequest                  // Invalid request parameters
	CodeNotFound                    // Resource not found
	CodeInternalError               // Internal server error
	CodeValidationError             // Validation failed
	CodeConflict                    // Resource conflict (e.g., duplicate)
	CodeUnauthorized                // Unauthorized access
	CodeForbidden                   // Forbidden operation
	CodeServiceError                // External service error
	CodeConfigError                 // Configuration error
	CodeOperationFailed             // Operation failed
)

// BasicResponse is the standard response structure for frontend
type BasicResponse[T any] struct {
	Code Code   `json:"code"` // business code
	Data T      `json:"data"`
	Msg  string `json:"msg"`
}

// resp serializes data wrapped in BasicResponse using sing-box JSON serialization
// Note: This is a standalone generic function because Go doesn't support generic methods on structs
func resp[T any](ctx context.Context, c *app.RequestContext, code Code, data T, msg string) {
	res := &BasicResponse[T]{
		Code: code,
		Data: data,
		Msg:  msg,
	}
	jsonCtx := config.CreateContext(ctx)
	responseJSON, err := json.MarshalContext(jsonCtx, res)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"code": CodeInternalError,
			"msg":  "failed to serialize response: " + err.Error(),
		})
		return
	}

	c.Data(consts.StatusOK, "application/json; charset=utf-8", responseJSON)
}

// respOK is a shorthand for successful response with code 0
func respOK[T any](ctx context.Context, c *app.RequestContext, data T) {
	resp(ctx, c, CodeSuccess, data, "success")
}

// respErr is a shorthand for error response
func respErr(ctx context.Context, c *app.RequestContext, code Code, msg string) {
	resp[any](ctx, c, code, nil, msg)
}
