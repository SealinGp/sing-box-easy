package bootstrap

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/configuration"
	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/repo"
	"github.com/SealinGp/sing-box-easy/app/pkg/system"
	"github.com/SealinGp/sing-box-easy/app/pkg/traffic"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/appupdate"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/config/history"
	"github.com/SealinGp/sing-box-easy/app/pkg/githubauth"
	"github.com/SealinGp/sing-box-easy/app/pkg/identity"
	"github.com/SealinGp/sing-box-easy/app/pkg/installation"
	"github.com/SealinGp/sing-box-easy/app/pkg/installation/state"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/rules"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
)

// Modules contains application-owned dependencies and background lifecycles.
type Modules struct {
	System              *system.Service
	Installation        *installer.Service
	TrafficService      *trafficflow.Service
	SettingsService     *settings.Service
	Diagnostics         *diagnostics.Service
	Outbounds           *outbounds.Service
	Configuration       *configuration.Service
	ConfigManager       *config.Manager
	ServiceController   *service.Controller
	SubscriptionManager *subscription.Service
	Installer           *installer.Manager
	DashboardManager    *installer.DashboardManager
	InitStateManager    initstate.InitStateManager
	VersionStore        *configversion.StoreXORM
	VersionCleaner      *configversion.Cleaner
	SettingsManager     *settings.ManagerXORM
	NodeRulesManager    *noderules.ManagerXORM
	UserManager         user.UserManager
	Updater             *appupdate.Updater
	GithubAuth          *githubauth.Manager
	// authEnabled is the resolved login requirement (server.auth × platform).
	// false means every request runs as an administrator.
	AuthEnabled bool
	// systemType is the detected distribution family, probed once at startup.
	// It drives both the auth default and the frontend's navigation layout.
	SystemType service.SystemType
}

// New constructs dependencies with one explicitly shared database engine.
// authMode is the raw server.auth config value (auto/enabled/disabled).
func New(
	configPath, singBoxPath string,
	adminUser, adminPass string,
	githubConfig appconfig.GitHubConfig,
	authMode string,
) (*Modules, error) {
	e, err := database.GetEngine()
	if err != nil {
		return nil, err
	}
	configManager := config.NewManager(configPath, singBoxPath, "") // Use default template path
	serviceController := service.NewController(configManager, singBoxPath)

	initStateManager := initstate.NewManagerXORM(e)

	// Config version history (DB-backed) + application settings.
	versionStore := configversion.NewStoreXORM(e)
	versionCleaner := configversion.NewCleaner(versionStore, configversion.DefaultMaxAge)
	settingsManager := settings.NewManagerXORM(e)
	configManager.SetVersionStore(versionStore)

	// Pass initStateManager and configManager to installer
	installerManager := installer.NewManager(initStateManager, configManager)
	dashboardManager := installer.NewDashboardManager(initStateManager, configManager)

	// Outbound Node Rules manager (Filters + Groups) — drives auto-grouping of
	// subscription nodes.
	nodeRulesManager := noderules.NewManagerXORM(e)

	subscriptionManager := subscription.NewService(configManager, repo.NewStore(e), nodeRulesManager, settingsManager, serviceController)
	if err := subscriptionManager.Init(); err != nil {
		return nil, err
	}
	if err := subscriptionManager.ConfigureProbes(e, settingsManager); err != nil {
		return nil, err
	}

	// Initialize user manager
	userManager := user.NewManagerXORM(e, adminUser, adminPass)

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
	authEnabled := resolveAuthEnabled(authMode, systemType)
	if !authEnabled {
		logger.Warn("==================================================================")
		logger.Warn("AUTHENTICATION IS DISABLED — every visitor has admin access.")
		logger.Warn("Anyone who can reach this panel's port can control sing-box.")
		logger.Warn("Set server.auth: enabled in app.yml to require login.")
		logger.Warn("==================================================================")
	}

	return &Modules{
		System:              system.New(configManager.GetConfigPath(), database.Path(), serviceController),
		Installation:        installer.NewService(installerManager, dashboardManager, configManager, initStateManager),
		TrafficService:      trafficflow.NewService(configManager),
		SettingsService:     settings.NewService(settingsManager, configManager, githubAuth.Configured),
		Diagnostics:         diagnostics.New(configManager, serviceController),
		Outbounds:           outbounds.New(configManager, nodeRulesManager),
		Configuration:       configuration.New(configManager),
		ConfigManager:       configManager,
		ServiceController:   serviceController,
		SubscriptionManager: subscriptionManager,
		Installer:           installerManager,
		DashboardManager:    dashboardManager,
		InitStateManager:    initStateManager,
		VersionStore:        versionStore,
		VersionCleaner:      versionCleaner,
		SettingsManager:     settingsManager,
		NodeRulesManager:    nodeRulesManager,
		UserManager:         userManager,
		Updater:             updater,
		GithubAuth:          githubAuth,
		AuthEnabled:         authEnabled,
		SystemType:          systemType,
	}, nil
}

// Init initializes all components and returns error if any fails
func (h *Modules) Init() error {
	// Initialize state manager
	if err := h.InitStateManager.Init(); err != nil {
		return err
	}

	// Initialize installer manager
	if err := h.Installer.Init(); err != nil {
		return err
	}

	// Initialize config version store + settings, then apply the configured
	// retention count to the config manager.
	if err := h.VersionStore.Init(); err != nil {
		return err
	}
	if err := h.SettingsManager.Init(); err != nil {
		return err
	}
	h.ConfigManager.SetKeepVersions(h.SettingsManager.GetConfigVersionsKeep())

	// Initialize node-rules tables + seed the mandatory fallback Filter.
	if err := h.NodeRulesManager.Init(); err != nil {
		return err
	}

	// Initialize user manager (sync tables, seed admin)
	if err := h.UserManager.Init(); err != nil {
		return err
	}

	return nil
}

// Start starts background work after every repository has been initialized.
func (h *Modules) Start() error {
	if err := h.SubscriptionManager.StartBackground("*/5 * * * *"); err != nil {
		return err
	}
	if err := h.VersionCleaner.Start(configversion.DefaultCleanupCron); err != nil {
		h.SubscriptionManager.StopBackground()
		return err
	}
	return nil
}

// Close drains workers before the caller closes the database.
func (h *Modules) Close() { h.SubscriptionManager.StopBackground(); h.VersionCleaner.Stop() }
func resolveAuthEnabled(mode string, systemType service.SystemType) bool {
	switch mode {
	case appconfig.AuthDisabled:
		return false
	case appconfig.AuthAuto:
		return systemType != service.SystemOpenWRT
	default: // appconfig.AuthEnabled and anything unexpected
		return true
	}
}
