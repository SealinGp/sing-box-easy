package subscription

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultSubscriptionPath = "/etc/sing-box/subscriptions.json"
)

// Subscription represents a node subscription
type Subscription struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	AutoUpdate     bool      `json:"auto_update"`
	UpdateInterval string    `json:"update_interval"` // e.g., "24h"
	LastUpdate     time.Time `json:"last_update"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Manager manages subscriptions
type Manager struct {
	subscriptionPath string
}

// NewManager creates a new subscription manager
func NewManager(subscriptionPath string) *Manager {
	if subscriptionPath == "" {
		subscriptionPath = DefaultSubscriptionPath
	}
	return &Manager{
		subscriptionPath: subscriptionPath,
	}
}

// List returns all subscriptions
func (m *Manager) List() ([]Subscription, error) {
	// Check if file exists
	if _, err := os.Stat(m.subscriptionPath); os.IsNotExist(err) {
		// Return empty list if file doesn't exist
		return []Subscription{}, nil
	}

	data, err := os.ReadFile(m.subscriptionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscriptions file: %w", err)
	}

	var subscriptions []Subscription
	if err := json.Unmarshal(data, &subscriptions); err != nil {
		return nil, fmt.Errorf("failed to parse subscriptions file: %w", err)
	}

	return subscriptions, nil
}

// Get returns a subscription by ID
func (m *Manager) Get(id string) (*Subscription, error) {
	subscriptions, err := m.List()
	if err != nil {
		return nil, err
	}

	for _, sub := range subscriptions {
		if sub.ID == id {
			return &sub, nil
		}
	}

	return nil, fmt.Errorf("subscription not found")
}

// Add adds a new subscription
func (m *Manager) Add(sub Subscription) error {
	subscriptions, err := m.List()
	if err != nil {
		return err
	}

	// Generate ID if not provided
	if sub.ID == "" {
		sub.ID = fmt.Sprintf("sub_%d", time.Now().Unix())
	}

	// Set timestamps
	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now

	// Check for duplicate ID
	for _, existing := range subscriptions {
		if existing.ID == sub.ID {
			return fmt.Errorf("subscription with ID %s already exists", sub.ID)
		}
	}

	subscriptions = append(subscriptions, sub)
	return m.save(subscriptions)
}

// Update updates an existing subscription
func (m *Manager) Update(id string, sub Subscription) error {
	subscriptions, err := m.List()
	if err != nil {
		return err
	}

	found := false
	for i, existing := range subscriptions {
		if existing.ID == id {
			// Keep the original ID and CreatedAt
			sub.ID = id
			sub.CreatedAt = existing.CreatedAt
			sub.UpdatedAt = time.Now()
			subscriptions[i] = sub
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("subscription not found")
	}

	return m.save(subscriptions)
}

// Delete deletes a subscription by ID
func (m *Manager) Delete(id string) error {
	subscriptions, err := m.List()
	if err != nil {
		return err
	}

	newSubscriptions := make([]Subscription, 0)
	found := false
	for _, sub := range subscriptions {
		if sub.ID != id {
			newSubscriptions = append(newSubscriptions, sub)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("subscription not found")
	}

	return m.save(newSubscriptions)
}

// UpdateLastUpdate updates the last_update timestamp for a subscription
func (m *Manager) UpdateLastUpdate(id string) error {
	subscriptions, err := m.List()
	if err != nil {
		return err
	}

	found := false
	for i, sub := range subscriptions {
		if sub.ID == id {
			subscriptions[i].LastUpdate = time.Now()
			subscriptions[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("subscription not found")
	}

	return m.save(subscriptions)
}

// save saves subscriptions to file
func (m *Manager) save(subscriptions []Subscription) error {
	// Ensure directory exists
	dir := filepath.Dir(m.subscriptionPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(subscriptions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscriptions: %w", err)
	}

	if err := os.WriteFile(m.subscriptionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write subscriptions file: %w", err)
	}

	return nil
}
