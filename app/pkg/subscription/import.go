package subscription

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed"
	"github.com/sagernet/sing-box/option"
	"strings"
)

// ImportPreview resolves pasted feed inputs without persisting a subscription.
func (s *Service) ImportPreview(ctx context.Context, input string) ([]option.Outbound, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "{") || strings.HasPrefix(input, "[") {
		return parseNativeNodes(ctx, input)
	}
	nodes, _, err := s.sublinkManager.Resolve(ctx, strings.Split(input, "\n"), feed.FetchOptions{})
	if err != nil {
		return nil, err
	}
	result := make([]option.Outbound, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, option.Outbound{Tag: n.Tag, Type: n.Type, Options: n.Options})
	}
	return result, nil
}

// PreviewImport is the standalone, read-only import path used before the panel
// database exists (for example by the installation wizard).
func PreviewImport(ctx context.Context, input string) ([]option.Outbound, error) {
	s := &Service{sublinkManager: &feed.SubLink{}}
	return s.ImportPreview(ctx, input)
}
