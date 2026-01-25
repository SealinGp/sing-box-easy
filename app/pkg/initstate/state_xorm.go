package initstate

import (
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/initstate/repo"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// ManagerXORM manages initialization state using XORM
type ManagerXORM struct {
	e *xorm.Engine
}

// NewManagerXORM creates a new XORM-backed initialization state manager
func NewManagerXORM() *ManagerXORM {
	e, err := database.GetEngine()
	if err != nil {
		logger.Fatal("Failed to get database engine", zap.Error(err))
	}

	return &ManagerXORM{
		e: e,
	}
}

// Init initializes the state manager
func (m *ManagerXORM) Init() error {
	logger.Info("State manager initialized with XORM")

	// Ensure the init_state table exists
	if err := m.e.Sync2(new(repo.InitState)); err != nil {
		logger.Error("Failed to sync init_state table", zap.Error(err))
		return err
	}

	if err := m.initDefaultData(); err != nil {
		return err
	}

	return nil
}

// initDefaultData initializes default data
func (m *ManagerXORM) initDefaultData() error {
	session := m.e.NewSession()
	defer session.Close()

	// Check if init_state has data
	count, err := session.Count(new(repo.InitState))
	if err != nil {
		return fmt.Errorf("failed to count init_state: %w", err)
	}

	// Insert default init_state if not exists
	if count == 0 {
		initState := &repo.InitState{
			Initialized:        false,
			SingBoxInstalled:   false,
			ConfigGenerated:    false,
			DashboardInstalled: false,
			SingBoxVersion:     "",
		}
		if _, err := session.Insert(initState); err != nil {
			return fmt.Errorf("failed to insert default init_state: %w", err)
		}
		logger.Info("Default init_state created")
	}

	return nil
}

// GetState returns the current state
func (m *ManagerXORM) GetState() *State {
	session := m.e.NewSession()
	defer session.Close()

	var dbState repo.InitState
	has, err := session.ID(1).Get(&dbState)
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
	session := m.e.NewSession()
	defer session.Close()

	updates := &repo.InitState{
		SingBoxInstalled: true,
		SingBoxVersion:   version,
	}

	_, err := session.ID(1).Cols("sing_box_installed", "sing_box_version").Update(updates)
	if err != nil {
		return fmt.Errorf("failed to update sing_box_installed: %w", err)
	}

	logger.Info("Marked sing-box as installed", zap.String("version", version))
	return nil
}

// SetDashboardInstalled marks dashboard as installed
func (m *ManagerXORM) SetDashboardInstalled() error {
	session := m.e.NewSession()
	defer session.Close()

	updates := &repo.InitState{
		DashboardInstalled: true,
	}

	_, err := session.ID(1).Cols("dashboard_installed").Update(updates)
	if err != nil {
		return fmt.Errorf("failed to update dashboard_installed: %w", err)
	}

	logger.Info("Marked dashboard as installed")
	return nil
}

// CompleteInitialization marks initialization as complete
func (m *ManagerXORM) CompleteInitialization() error {
	session := m.e.NewSession()
	defer session.Close()

	now := time.Now().UTC()
	updates := &repo.InitState{
		Initialized:     true,
		ConfigGenerated: true,
		InitTime:        now,
	}

	_, err := session.ID(1).Cols("initialized", "config_generated", "init_time").Update(updates)
	if err != nil {
		return fmt.Errorf("failed to complete initialization: %w", err)
	}

	logger.Info("Initialization completed")
	return nil
}

// Reset resets the initialization state
func (m *ManagerXORM) Reset() error {
	session := m.e.NewSession()
	defer session.Close()

	updates := &repo.InitState{
		Initialized:        false,
		SingBoxInstalled:   false,
		ConfigGenerated:    false,
		DashboardInstalled: false,
		SingBoxVersion:     "",
		InitTime:           time.Time{},
	}

	_, err := session.ID(1).Cols("initialized", "sing_box_installed", "config_generated", "dashboard_installed", "sing_box_version", "init_time").Update(updates)
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
