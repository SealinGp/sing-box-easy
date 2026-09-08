package subscription

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
)

func (h *Service) GetSubscriptionInfoKeywords(ctx context.Context) (any, error) {
	configured := NormalizeInfoKeywords(h.settingsManager.GetSubscriptionInfoKeywords())

	return map[string]any{
		"keywords":       EffectiveInfoKeywords(configured),
		"defaults":       DefaultInfoLabelKeywords,
		"using_defaults": len(configured) == 0,
		"limits": map[string]int{
			"max_keywords": settings.MaxInfoKeywords,
			"max_length":   settings.MaxInfoKeywordLen,
		},
	}, nil
}

type UpdateSubscriptionInfoKeywordsCommand struct {
	Keywords []string `json:"keywords"`
}

func (h *Service) UpdateSubscriptionInfoKeywords(ctx context.Context, req UpdateSubscriptionInfoKeywordsCommand) (any, error) {

	normalized := NormalizeInfoKeywords(req.Keywords)
	if err := h.settingsManager.SetSubscriptionInfoKeywords(normalized); err != nil {
		return nil, fault.New(fault.Invalid, err.Error())
	}

	return map[string]any{
		"keywords":       EffectiveInfoKeywords(normalized),
		"defaults":       DefaultInfoLabelKeywords,
		"using_defaults": len(normalized) == 0,
	}, nil
}
