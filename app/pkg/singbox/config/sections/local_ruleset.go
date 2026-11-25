package sections

import (
	"context"
	"encoding/json"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
)

func (s *Service) ReadLocalRuleFile(tag string) (any, error) {
	result, err := s.configManager.ReadLocalRuleFile(tag)
	if err != nil {
		return nil, fault.New(fault.Input, err.Error())
	}
	return result, nil
}
func (s *Service) WriteLocalRuleFile(ctx context.Context, body []byte) (any, error) {
	var req config.LocalRuleWrite
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fault.New(fault.Input, err.Error())
	}
	result, err := s.configManager.WriteLocalRuleFile(ctx, req)
	if err != nil {
		return nil, fault.New(fault.Input, err.Error())
	}
	return result, nil
}
