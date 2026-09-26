// Package outbounds owns node editing and managed group policies.
package outbounds

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	noderules "github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds/rules"
)

type Service struct {
	configManager    *config.Manager
	nodeRulesManager *noderules.ManagerXORM
}

func New(cm *config.Manager, rules *noderules.ManagerXORM) *Service {
	return &Service{configManager: cm, nodeRulesManager: rules}
}
