package v1_13_0

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/initstate"
	"github.com/SealinGp/sing-box-easy/app/pkg/installer"
	"github.com/SealinGp/sing-box-easy/app/pkg/service"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
)

// Handler holds all dependencies for v1.12.12 API handlers
type Handler struct {
	configManager       *config.Manager
	serviceController   *service.Controller
	subscriptionManager *subscription.Manager
	sublink             *sublink.SubLink
	installer           *installer.Manager
	dashboardManager    *installer.DashboardManager
	initStateManager    *initstate.Manager
}

// NewHandler creates a new v1.12.12 handler
func NewHandler(configPath, singBoxPath, subscriptionPath, initStatePath string) *Handler {
	configManager := config.NewManager(configPath, singBoxPath, "") // Use default template path
	serviceController := service.NewController(configManager, singBoxPath)
	subscriptionManager := subscription.NewManager(subscriptionPath)
	sublinkParser := new(sublink.SubLink)

	initStateManager := initstate.NewManager(initStatePath)

	// Pass initStateManager and configManager to installer
	installerManager := installer.NewManager(initStateManager, configManager)

	dashboardManager := installer.NewDashboardManager(initStateManager)

	return &Handler{
		configManager:       configManager,
		serviceController:   serviceController,
		subscriptionManager: subscriptionManager,
		sublink:             sublinkParser,
		installer:           installerManager,
		dashboardManager:    dashboardManager,
		initStateManager:    initStateManager,
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

	return nil
}
