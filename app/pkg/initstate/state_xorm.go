package initstate

import (
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// ManagerXORM manages initialization state using XORM
type ManagerXORM struct{}

// NewManagerXORM creates a new XORM-backed initialization state manager
func NewManagerXORM() *ManagerXORM {
	return &ManagerXORM{}
}

// Init initializes the state manager
func (m *ManagerXORM) Init() error {
	logger.Info("State manager initialized with XORM")
	return nil
}

// GetState returns the current state
func (m *ManagerXORM) GetState() *State {
	engine, err := database.GetEngine()
	if err != nil {
		logger.Error("Failed to get database engine", zap.Error(err))
		return &State{}
	}

	var dbState database.InitState
	has, err := engine.ID(1).Get(&dbState)
	if err != nil {
		logger.Error("Failed to get init state", zap.Error(err))
		return &State{}
	}

	if !has {
		// Return default state if not found
		return &State{
			Initialized:        false,
			SingBoxInstalled:   false,
			ConfigGenerated:    false,
			DashboardInstalled: false,
			SingBoxVersion:     "",
			InitTime:           "",
		}
	}

	// Convert database model to State struct
	return &State{
		Initialized:        dbState.Initialized,
		SingBoxInstalled:   dbState.SingBoxInstalled,
		ConfigGenerated:    dbState.ConfigGenerated,
		DashboardInstalled: dbState.DashboardInstalled,
		SingBoxVersion:     dbState.SingBoxVersion,
		InitTime:           formatTime(dbState.InitTime),
	}
}

// SetSingBoxInstalled marks sing-box as installed
func (m *ManagerXORM) SetSingBoxInstalled(version string) error {
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	updates := &database.InitState{
		SingBoxInstalled: true,
		SingBoxVersion:   version,
	}

	_, err = engine.ID(1).Update(updates)
	if err != nil {
		return fmt.Errorf("failed to update sing_box_installed: %w", err)
	}

	logger.Info("Marked sing-box as installed", zap.String("version", version))
	return nil
}

// SetDashboardInstalled marks dashboard as installed
func (m *ManagerXORM) SetDashboardInstalled() error {
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	updates := &database.InitState{
		DashboardInstalled: true,
	}

	_, err = engine.ID(1).Update(updates)
	if err != nil {
		return fmt.Errorf("failed to update dashboard_installed: %w", err)
	}

	logger.Info("Marked dashboard as installed")
	return nil
}

// CompleteInitialization marks initialization as complete
func (m *ManagerXORM) CompleteInitialization() error {
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	now := time.Now().UTC()
	updates := &database.InitState{
		Initialized:      true,
		ConfigGenerated:  true,
		InitTime:         now,
	}

	_, err = engine.ID(1).Update(updates)
	if err != nil {
		return fmt.Errorf("failed to complete initialization: %w", err)
	}

	logger.Info("Initialization completed")
	return nil
}

// Reset resets the initialization state
func (m *ManagerXORM) Reset() error {
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	updates := &database.InitState{
		Initialized:         false,
		SingBoxInstalled:   false,
		ConfigGenerated:    false,
		DashboardInstalled: false,
		SingBoxVersion:     "",
		InitTime:           time.Time{},
	}

	_, err = engine.ID(1).Update(updates)
	if err != nil {
		return fmt.Errorf("failed to reset initialization: %w", err)
	}

	logger.Info("Initialization state reset")
	return nil
}

// formatTime converts time.Time to ISO format string
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
