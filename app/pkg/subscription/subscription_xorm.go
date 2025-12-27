package subscription

import (
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/repo"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// ManagerXORM manages subscriptions using XORM
type ManagerXORM struct {
	e *xorm.Engine
}

// NewManagerXORM creates a new XORM-backed subscription manager
func NewManagerXORM() *ManagerXORM {
	e, err := database.GetEngine()
	if err != nil {
		logger.Fatal("Failed to get database engine", zap.Error(err))
	}

	return &ManagerXORM{
		e: e,
	}
}

// Init initializes the subscription manager and ensures database schema is ready
func (m *ManagerXORM) Init() error {
	logger.Info("Initializing subscription manager with XORM")

	// Sync subscription table
	if err := m.e.Sync2(new(repo.Subscription)); err != nil {
		logger.Error("Failed to sync subscription table", zap.Error(err))
		return fmt.Errorf("failed to sync subscription table: %w", err)
	}

	logger.Info("Subscription table synced successfully")
	return nil
}

// List returns all subscriptions
func (m *ManagerXORM) List() ([]*Subscription, error) {
	session := m.e.NewSession()
	defer session.Close()

	var dbSubscriptions []repo.Subscription
	err := session.Desc("created_at").Find(&dbSubscriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions: %w", err)
	}

	// Convert database models to Subscription structs
	subscriptions := make([]*Subscription, len(dbSubscriptions))
	for i, dbSub := range dbSubscriptions {
		subscriptions[i] = &Subscription{
			ID:             dbSub.ID,
			Name:           dbSub.Name,
			URL:            dbSub.URL,
			Enabled:        dbSub.Enabled,
			AutoUpdate:     dbSub.AutoUpdate,
			UpdateInterval: dbSub.UpdateInterval,
			LastUpdate:     dbSub.LastUpdate,
			CreatedAt:      dbSub.CreatedAt,
			UpdatedAt:      dbSub.UpdatedAt,
		}
	}

	return subscriptions, nil
}

// Get returns a subscription by ID
func (m *ManagerXORM) Get(id string) (*Subscription, error) {
	session := m.e.NewSession()
	defer session.Close()

	var dbSub repo.Subscription
	has, err := session.ID(id).Get(&dbSub)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if !has {
		return nil, fmt.Errorf("subscription not found")
	}

	sub := &Subscription{
		ID:             dbSub.ID,
		Name:           dbSub.Name,
		URL:            dbSub.URL,
		Enabled:        dbSub.Enabled,
		AutoUpdate:     dbSub.AutoUpdate,
		UpdateInterval: dbSub.UpdateInterval,
		LastUpdate:     dbSub.LastUpdate,
		CreatedAt:      dbSub.CreatedAt,
		UpdatedAt:      dbSub.UpdatedAt,
	}

	return sub, nil
}

// Add adds a new subscription
func (m *ManagerXORM) Add(sub Subscription) error {
	session := m.e.NewSession()
	defer session.Close()

	// Generate ID if not provided
	if sub.ID == "" {
		sub.ID = fmt.Sprintf("sub_%d", time.Now().Unix())
	}

	// Set timestamps (XORM will handle created_at and updated_at automatically)
	dbSub := &repo.Subscription{
		ID:             sub.ID,
		Name:           sub.Name,
		URL:            sub.URL,
		Enabled:        sub.Enabled,
		AutoUpdate:     sub.AutoUpdate,
		UpdateInterval: sub.UpdateInterval,
		LastUpdate:     sub.LastUpdate,
	}

	_, err := session.Insert(dbSub)
	if err != nil {
		return fmt.Errorf("failed to add subscription: %w", err)
	}

	logger.Info("Subscription added", zap.String("id", sub.ID), zap.String("name", sub.Name))
	return nil
}

// Update updates an existing subscription
func (m *ManagerXORM) Update(id string, sub Subscription) error {
	session := m.e.NewSession()
	defer session.Close()

	// Check if subscription exists
	has, err := session.ID(id).Exist(&repo.Subscription{})
	if err != nil {
		return fmt.Errorf("failed to check subscription existence: %w", err)
	}

	if !has {
		return fmt.Errorf("subscription not found")
	}

	// Update subscription (XORM will handle updated_at automatically)
	updates := map[string]interface{}{
		"name":            sub.Name,
		"url":             sub.URL,
		"enabled":         sub.Enabled,
		"auto_update":     sub.AutoUpdate,
		"update_interval": sub.UpdateInterval,
	}

	// Only update last_update if it's provided
	if !sub.LastUpdate.IsZero() {
		updates["last_update"] = sub.LastUpdate
	}

	_, err = session.ID(id).Update(&repo.Subscription{}, updates)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	logger.Info("Subscription updated", zap.String("id", id), zap.String("name", sub.Name))
	return nil
}

// Delete deletes a subscription by ID
func (m *ManagerXORM) Delete(id string) error {
	session := m.e.NewSession()
	defer session.Close()

	affected, err := session.ID(id).Delete(&repo.Subscription{})
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("subscription not found")
	}

	logger.Info("Subscription deleted", zap.String("id", id))
	return nil
}

// UpdateLastUpdate updates the last_update timestamp for a subscription
func (m *ManagerXORM) UpdateLastUpdate(id string) error {
	// Use a session for the update
	session := m.e.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// First check if subscription exists
	var count int64
	count, err := session.ID(id).Count(&repo.Subscription{})
	if err != nil {
		session.Rollback()
		return fmt.Errorf("failed to check subscription existence: %w", err)
	}

	if count == 0 {
		session.Rollback()
		return fmt.Errorf("subscription not found")
	}

	// Update using SQL
	sql := "UPDATE subscriptions SET last_update = ? WHERE id = ?"
	_, err = session.Exec(sql, time.Now(), id)
	if err != nil {
		session.Rollback()
		return fmt.Errorf("failed to update last_update: %w", err)
	}

	if err := session.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Subscription last_update timestamp updated", zap.String("id", id))
	return nil
}
