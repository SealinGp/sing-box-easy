package subscription

import (
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// ManagerXORM manages subscriptions using XORM
type ManagerXORM struct{}

// NewManagerXORM creates a new XORM-backed subscription manager
func NewManagerXORM() *ManagerXORM {
	return &ManagerXORM{}
}

// List returns all subscriptions
func (m *ManagerXORM) List() ([]Subscription, error) {
	engine, err := database.GetEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to get database engine: %w", err)
	}

	var dbSubscriptions []database.Subscription
	err = engine.Desc("created_at").Find(&dbSubscriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions: %w", err)
	}

	// Convert database models to Subscription structs
	subscriptions := make([]Subscription, len(dbSubscriptions))
	for i, dbSub := range dbSubscriptions {
		subscriptions[i] = Subscription{
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
	engine, err := database.GetEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to get database engine: %w", err)
	}

	var dbSub database.Subscription
	has, err := engine.ID(id).Get(&dbSub)
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
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	// Generate ID if not provided
	if sub.ID == "" {
		sub.ID = fmt.Sprintf("sub_%d", time.Now().Unix())
	}

	// Set timestamps (XORM will handle created_at and updated_at automatically)
	dbSub := &database.Subscription{
		ID:             sub.ID,
		Name:           sub.Name,
		URL:            sub.URL,
		Enabled:        sub.Enabled,
		AutoUpdate:     sub.AutoUpdate,
		UpdateInterval: sub.UpdateInterval,
		LastUpdate:     sub.LastUpdate,
	}

	_, err = engine.Insert(dbSub)
	if err != nil {
		return fmt.Errorf("failed to add subscription: %w", err)
	}

	logger.Info("Subscription added", zap.String("id", sub.ID), zap.String("name", sub.Name))
	return nil
}

// Update updates an existing subscription
func (m *ManagerXORM) Update(id string, sub Subscription) error {
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	// Check if subscription exists
	has, err := engine.ID(id).Exist(&database.Subscription{})
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

	_, err = engine.ID(id).Update(&database.Subscription{}, updates)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	logger.Info("Subscription updated", zap.String("id", id), zap.String("name", sub.Name))
	return nil
}

// Delete deletes a subscription by ID
func (m *ManagerXORM) Delete(id string) error {
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	affected, err := engine.ID(id).Delete(&database.Subscription{})
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
	engine, err := database.GetEngine()
	if err != nil {
		return fmt.Errorf("failed to get database engine: %w", err)
	}

	// Use a session for the update
	session := engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// First check if subscription exists
	var count int64
	count, err = session.ID(id).Count(&database.Subscription{})
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
