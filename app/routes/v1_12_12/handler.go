package v1_13_0

import (
	"context"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/appupdate"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/configversion"
	"github.com/SealinGp/sing-box-easy/app/pkg/githubauth"
	"github.com/SealinGp/sing-box-easy/app/pkg/initstate"
	"github.com/SealinGp/sing-box-easy/app/pkg/installer"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/noderules"
	"github.com/SealinGp/sing-box-easy/app/pkg/service"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/SealinGp/sing-box-easy/app/pkg/user"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sagernet/sing/common/json"
	"go.uber.org/zap"
)

// Handler holds all dependencies for v1.12.12 API handlers
type Handler struct {
	configManager       *config.Manager
	serviceController   *service.Controller
	subscriptionManager subscription.SubscriptionManager
	sublink             *sublink.SubLink
	installer           *installer.Manager
	dashboardManager    *installer.DashboardManager
	initStateManager    initstate.InitStateManager
	autoUpdater         *subscription.AutoUpdater
	schedulerHandler    *schedulerHandler
	versionStore        *configversion.StoreXORM
	versionCleaner      *configversion.Cleaner
	settingsManager     *settings.ManagerXORM
	nodeRulesManager    *noderules.ManagerXORM
	userManager         user.UserManager
	updater             *appupdate.Updater
	githubAuth          *githubauth.Manager
	// authEnabled is the resolved login requirement (server.auth × platform).
	// false means every request runs as an administrator.
	authEnabled bool
	// systemType is the detected distribution family, probed once at startup.
	// It drives both the auth default and the frontend's navigation layout.
	systemType service.SystemType
}

// NewHandler creates a new v1.12.12 handler using XORM-backed managers.
// authMode is the raw server.auth config value (auto/enabled/disabled).
func NewHandler(
	configPath, singBoxPath string,
	adminUser, adminPass string,
	sublinkParser *sublink.SubLink,
	githubConfig appconfig.GitHubConfig,
	authMode string,
) *Handler {
	configManager := config.NewManager(configPath, singBoxPath, "") // Use default template path
	serviceController := service.NewController(configManager, singBoxPath)

	// Use XORM-backed managers
	subscriptionManager := subscription.NewManagerXORM()
	if err := subscriptionManager.Init(); err != nil {
		logger.Fatal("Failed to initialize subscription manager", zap.Error(err))
	}
	initStateManager := initstate.NewManagerXORM()

	// Config version history (DB-backed) + application settings.
	versionStore := configversion.NewStoreXORM()
	versionCleaner := configversion.NewCleaner(versionStore, configversion.DefaultMaxAge)
	settingsManager := settings.NewManagerXORM()
	configManager.SetVersionStore(versionStore)

	// Pass initStateManager and configManager to installer
	installerManager := installer.NewManager(initStateManager, configManager)
	dashboardManager := installer.NewDashboardManager(initStateManager)

	// Outbound Node Rules manager (Filters + Groups) — drives auto-grouping of
	// subscription nodes.
	nodeRulesManager := noderules.NewManagerXORM()

	// Initialize auto-updater (rules-aware)
	autoUpdater := subscription.NewAutoUpdater(configManager, subscriptionManager, sublinkParser, nodeRulesManager, settingsManager)
	schedulerHandler := newSchedulerHandler(autoUpdater)

	// Initialize user manager
	userManager := user.NewManagerXORM(adminUser, adminPass)

	// Self-update manager (GitHub releases -> binary + frontend swap + restart).
	// The token is resolved per request from settings, so signing in through
	// the UI lifts the GitHub rate limit without a restart.
	updater := appupdate.NewUpdater(githubConfig.Proxy, settingsManager.GetGitHubToken)

	// GitHub sign-in (OAuth device flow) — issues the token the updater reads.
	//
	// The client ID is resolved per call, database first: an operator can paste
	// one into Settings and sign in immediately, with app.yml /
	// GITHUB_OAUTH_CLIENT_ID remaining the fallback for headless deployments
	// that prefer to bake it in.
	fallbackClientID := strings.TrimSpace(githubConfig.OAuthClientID)
	githubAuth := githubauth.NewManager(
		func() string {
			if stored := settingsManager.GetGitHubOAuthClientID(); stored != "" {
				return stored
			}
			return fallbackClientID
		},
		githubConfig.Proxy,
		settingsManager,
	)

	systemType := service.DetectSystemType()
	authEnabled := ResolveAuthEnabled(authMode, systemType)
	if !authEnabled {
		logger.Warn("==================================================================")
		logger.Warn("AUTHENTICATION IS DISABLED — every visitor has admin access.")
		logger.Warn("Anyone who can reach this panel's port can control sing-box.")
		logger.Warn("Set server.auth: enabled in app.yml to require login.")
		logger.Warn("==================================================================")
	}

	return &Handler{
		configManager:       configManager,
		serviceController:   serviceController,
		subscriptionManager: subscriptionManager,
		sublink:             sublinkParser,
		installer:           installerManager,
		dashboardManager:    dashboardManager,
		initStateManager:    initStateManager,
		autoUpdater:         autoUpdater,
		schedulerHandler:    schedulerHandler,
		versionStore:        versionStore,
		versionCleaner:      versionCleaner,
		settingsManager:     settingsManager,
		nodeRulesManager:    nodeRulesManager,
		userManager:         userManager,
		updater:             updater,
		githubAuth:          githubAuth,
		authEnabled:         authEnabled,
		systemType:          systemType,
	}
}

// Init initializes all components and returns error if any fails
func (h *Handler) Init() error {
	// Initialize state manager
	if err := h.initStateManager.Init(); err != nil {
		return err
	}

	// Initialize installer manager
	if err := h.installer.Init(); err != nil {
		return err
	}

	// Initialize config version store + settings, then apply the configured
	// retention count to the config manager.
	if err := h.versionStore.Init(); err != nil {
		return err
	}
	if err := h.settingsManager.Init(); err != nil {
		return err
	}
	h.configManager.SetKeepVersions(h.settingsManager.GetConfigVersionsKeep())

	// Initialize node-rules tables + seed the mandatory fallback Filter.
	if err := h.nodeRulesManager.Init(); err != nil {
		return err
	}

	// Initialize user manager (sync tables, seed admin)
	if err := h.userManager.Init(); err != nil {
		return err
	}

	return nil
}

// StartAutoUpdater starts the auto-updater with the given cron expression
func (h *Handler) StartAutoUpdater(cronExpression string) error {
	if h.autoUpdater == nil {
		return nil // Auto-updater not initialized, skip silently
	}
	return h.autoUpdater.Start(cronExpression)
}

// StopAutoUpdater stops the auto-updater
func (h *Handler) StopAutoUpdater() {
	if h.autoUpdater != nil {
		h.autoUpdater.Stop()
	}
}

// StartVersionCleaner starts the daily retention sweep that deletes config
// versions older than the configured max age.
func (h *Handler) StartVersionCleaner() error {
	if h.versionCleaner == nil {
		return nil
	}
	return h.versionCleaner.Start(configversion.DefaultCleanupCron)
}

// StopVersionCleaner stops the retention sweep.
func (h *Handler) StopVersionCleaner() {
	if h.versionCleaner != nil {
		h.versionCleaner.Stop()
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
