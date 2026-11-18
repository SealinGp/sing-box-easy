package subscription

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/probe"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/repo"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// ConfigureProbes wires subscription-owned measurement and storage before startup.
func (s *Service) ConfigureProbes(e *xorm.Engine, settings *settings.ManagerXORM, nodes *nodehistory.Store) error {
	s.settingsManager = settings
	s.nodeHistory = nodes
	s.probeStore = repo.NewProbeStore(e)
	if err := s.probeStore.Init(); err != nil {
		return err
	}
	s.probeRunner = probe.NewRunner(s.probeStore, &probeEnvironment{subscriptions: s, configManager: s.configManager, settings: settings}, nodes)
	// A fresh, not-yet-configured install may have no outbound config yet.
	if _, err := s.configManager.GetOutboundsConfig(); err == nil {
		subs, err := s.List()
		if err != nil {
			return err
		}
		for _, sub := range subs {
			if err := s.syncNodeHistory(sub.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *Service) StartBackground(cron string) error {
	if store, ok := s.repository.(deletionRepository); ok {
		ids, err := store.PendingDeletions()
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := s.Delete(id); err != nil {
				logger.Warn("pending subscription deletion needs retry", zap.String("id", id), zap.Error(err))
			}
		}
	}
	if err := s.Start(cron); err != nil {
		return err
	}
	if s.probeRunner != nil {
		s.probeRunner.Start()
	}
	return nil
}
func (s *Service) StopBackground() {
	s.Stop()
	if s.probeRunner != nil {
		s.probeRunner.Stop()
	}
}
