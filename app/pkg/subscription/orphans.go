package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
)

type OrphanCleanupResult struct {
	RemovedNodes    int    `json:"removed_nodes"`
	BackupPath      string `json:"backup_path"`
	RestartRequired bool   `json:"restart_required"`
}

func (s *Service) knownSubscriptions() (map[string]bool, error) {
	subscriptions, err := s.repository.List()
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(subscriptions))
	for _, sub := range subscriptions {
		known[sub.ID] = true
	}
	return known, nil
}
func (s *Service) DetectOrphanSubscriptions() (OrphanPlan, error) {
	if s.configManager == nil {
		return OrphanPlan{}, errors.New("config manager unavailable")
	}
	doc, err := s.configManager.GetConfigDocument()
	if err != nil {
		return OrphanPlan{}, err
	}
	known, err := s.knownSubscriptions()
	if err != nil {
		return OrphanPlan{}, err
	}
	var sections map[string]json.RawMessage
	if err = json.Unmarshal(doc.Raw, &sections); err != nil {
		return OrphanPlan{}, err
	}
	plan, _, err := orphanPlan(sections, known)
	return plan, err
}
func (s *Service) CleanupOrphanSubscriptions(ctx context.Context, revision string) (OrphanCleanupResult, error) {
	result := OrphanCleanupResult{}
	preview, err := s.DetectOrphanSubscriptions()
	if err != nil {
		return result, err
	}
	if revision == "" || preview.Revision != revision {
		return result, fault.New(fault.Input, "configuration or subscriptions changed; refresh the cleanup preview")
	}
	// The same locks are held by create, edit, refresh and delete. Lock in sorted
	// ID order so a restored subscription cannot be removed by an older preview.
	for _, sub := range preview.Subscriptions {
		unlock := s.lockSubscription(sub.ID)
		defer unlock()
	}
	err = s.configManager.UpdateConfigSections(ctx, func(sections map[string]json.RawMessage) error {
		known, err := s.knownSubscriptions()
		if err != nil {
			return err
		}
		plan, updated, err := orphanPlan(sections, known)
		if err != nil {
			return err
		}
		if plan.Revision != revision {
			return fault.New(fault.Input, "configuration or subscriptions changed; refresh the cleanup preview")
		}
		if plan.NodeCount == 0 {
			return fault.New(fault.Input, "no orphan subscription nodes remain")
		}
		if len(plan.Blockers) > 0 {
			return fault.New(fault.Input, "orphan nodes are still referenced outside node groups; resolve the listed references first")
		}
		// A separate, private backup is mandatory even when version history is disabled.
		before, err := json.MarshalIndent(sections, "", "  ")
		if err != nil {
			return err
		}
		backup, err := os.CreateTemp(filepath.Dir(s.configManager.GetConfigPath()), "config.json.before-orphan-cleanup-*.bak")
		if err != nil {
			return fmt.Errorf("create cleanup backup: %w", err)
		}
		_, writeErr := backup.Write(before)
		syncErr := backup.Sync()
		closeErr := backup.Close()
		if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
			return err
		}
		result.BackupPath = backup.Name()
		result.RemovedNodes = plan.NodeCount
		sections["outbounds"] = updated
		return nil
	})
	if err != nil {
		return OrphanCleanupResult{}, err
	}
	// Config editing must not unexpectedly interrupt active network sessions.
	result.RestartRequired = true
	return result, nil
}
