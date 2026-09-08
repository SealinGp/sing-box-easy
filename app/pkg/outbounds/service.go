// Package outbounds owns node editing and managed group policies.
package outbounds

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/rules"
)

type Service struct {
	configManager    *config.Manager
	nodeRulesManager *noderules.ManagerXORM
}

func New(cm *config.Manager, rules *noderules.ManagerXORM) *Service {
	return &Service{configManager: cm, nodeRulesManager: rules}
}
