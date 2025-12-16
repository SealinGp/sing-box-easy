package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/initstate"
	"github.com/SealinGp/sing-box-easy/app/pkg/installer"
	"github.com/SealinGp/sing-box-easy/app/pkg/service"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sagernet/sing/common/json"
)

// Handler holds all dependencies for v1.12.12 API handlers
type Handler struct {
	configManager       *config.Manager
	serviceController   *service.Controller
	subscriptionManager interface {
		List() ([]subscription.Subscription, error)
		Get(id string) (*subscription.Subscription, error)
		Add(sub subscription.Subscription) error
		Update(id string, sub subscription.Subscription) error
		Delete(id string) error
		UpdateLastUpdate(id string) error
	}
	sublink          *sublink.SubLink
	installer        *installer.Manager
	dashboardManager *installer.DashboardManager
	initStateManager interface {
		Init() error
		GetState() *initstate.State
		SetSingBoxInstalled(version string) error
		SetDashboardInstalled() error
		CompleteInitialization() error
		Reset() error
	}
}

// NewHandler creates a new v1.12.12 handler using XORM-backed managers
func NewHandler(configPath, singBoxPath string) *Handler {
	configManager := config.NewManager(configPath, singBoxPath, "") // Use default template path
	serviceController := service.NewController(configManager, singBoxPath)

	// Use XORM-backed managers
	subscriptionManager := subscription.NewManagerXORM()
	sublinkParser := new(sublink.SubLink)
	initStateManager := initstate.NewManagerXORM()

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
