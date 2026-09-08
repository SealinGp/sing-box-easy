package subscription

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds"
)

func (s *Service) applyChanges(deleted map[string]struct{}, added []config.Outbound, updated map[string]config.Outbound, id string) error {
	return outbounds.NewReconciler(s.configManager, s.nodeRules).Apply(context.Background(), id, func(*config.SingBoxConfig) outbounds.Changes {
		return outbounds.Changes{Delete: deleted, Add: added, Update: updated}
	})
}
func (s *Service) rebuildNodeRules(nodes []config.Outbound, id string) ([]config.Outbound, error) {
	return outbounds.NewReconciler(s.configManager, s.nodeRules).Rebuild(nodes, id)
}
