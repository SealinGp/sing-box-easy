package subscription

import (
	"context"
	"errors"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/repo"
)

// NewService creates the subscription business service. CRUD, manual refreshes,
// and scheduled refreshes share the same repository and operation lock.
func NewService(cm *config.Manager, store repo.Repository, rules NodeRulesProvider, keywords InfoKeywordsProvider, restarter ServiceRestarter) *Service {
	return newService(cm, store, &feed.SubLink{}, rules, keywords, restarter)
}

var _ SubscriptionManager = (*Service)(nil)

func (au *Service) Init() error                          { return au.repository.Init() }
func (au *Service) List() ([]*Subscription, error)       { return au.repository.List() }
func (au *Service) Get(id string) (*Subscription, error) { return au.repository.Get(id) }
func (au *Service) add(sub Subscription) error {
	unlock := au.lockSubscription(sub.ID)
	defer unlock()
	if err := au.repository.Add(sub); err != nil {
		return err
	}
	if au.probeRunner != nil {
		au.probeRunner.Allow(sub.ID)
	}
	return nil
}

// Delete removes owned nodes and group references before deleting the record.
// Config and SQLite cannot commit together: retaining the record on config or
// restart failure lets the operator retry. Restart even when no nodes remain,
// since a previous attempt may have saved config but failed to apply it.
func (au *Service) Delete(id string) error {
	unlock := au.lockSubscription(id)
	defer unlock()
	if _, err := au.repository.Get(id); err != nil {
		return err
	}
	if au.configManager == nil || au.serviceRestarter == nil {
		return fmt.Errorf("subscription deletion requires config manager and service restarter")
	}
	wasPending := false
	if store, ok := au.repository.(deletionRepository); ok {
		var err error
		wasPending, err = store.IsDeleting(id)
		if err != nil {
			return err
		}
		if err := store.MarkDeleting(id); err != nil {
			return err
		}
	}
	if au.probeRunner != nil {
		au.probeRunner.BeginDelete(id)
	}
	if err := outbounds.NewReconciler(au.configManager, au.nodeRules).Apply(context.Background(), id, func(cfg *config.SingBoxConfig) outbounds.Changes {
		deleted := make(map[string]struct{})
		for _, n := range cfg.Outbounds {
			if TagBelongsToSubscription(n.Tag, id) {
				deleted[n.Tag] = struct{}{}
			}
		}
		return outbounds.Changes{Delete: deleted}
	}); err != nil {
		// Validation runs before committing the config. A rejected fresh delete
		// must not disable refresh/probes or become an automatic startup retry.
		// Keep older pending operations: they may already have committed config.
		var validation *config.ValidationError
		if !wasPending && errors.As(err, &validation) {
			if store, ok := au.repository.(deletionRepository); ok {
				if cancelErr := store.CancelDeletion(id); cancelErr != nil {
					return errors.Join(err, fmt.Errorf("failed to cancel pending deletion: %w", cancelErr))
				}
			}
			if au.probeRunner != nil {
				au.probeRunner.Allow(id)
			}
		}
		return fmt.Errorf("failed to remove subscription nodes: %w", err)
	}
	if err := au.restartService(); err != nil {
		return fmt.Errorf("subscription nodes removed but failed to restart sing-box; retry deletion: %w", err)
	}
	if au.nodeHistory != nil {
		if err := au.nodeHistory.SyncSubscription(id, nil); err != nil {
			return err
		}
	}
	if store, ok := au.repository.(deletionRepository); ok {
		if err := store.FinishDeletion(id); err != nil {
			return err
		}
	} else {
		if err := au.repository.Delete(id); err != nil {
			return err
		}
	}
	au.mutex.Lock()
	delete(au.updateStats, id)
	au.mutex.Unlock()
	return nil
}

type deletionRepository interface {
	MarkDeleting(string) error
	CancelDeletion(string) error
	PendingDeletions() ([]string, error)
	IsDeleting(string) (bool, error)
	FinishDeletion(string) error
}

func (s *Service) checkActive(id string) error {
	if store, ok := s.repository.(deletionRepository); ok {
		pending, err := store.IsDeleting(id)
		if err != nil {
			return err
		}
		if pending {
			return fmt.Errorf("subscription deletion is pending; retry deletion")
		}
	}
	return nil
}
