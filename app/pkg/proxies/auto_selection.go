package proxies

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/autoselect"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
)

func (s *Service) ConfigureAutoSelection(store autoselect.Settings) error {
	s.automatic = autoselect.New(store, s.history, func() (autoselect.Runtime, error) { return s.runtime() })
	return s.automatic.Init()
}
func (s *Service) StartAutoSelection() {
	if s.automatic != nil {
		s.automatic.Start()
	}
}
func (s *Service) StopAutoSelection() {
	if s.automatic != nil {
		s.automatic.Close()
	}
}
func (s *Service) SetAutoSelection(ctx context.Context, group string, policy autoselect.Policy) (autoselect.Status, error) {
	if s.automatic == nil {
		return autoselect.Status{}, fault.New(fault.Failed, "quality auto-selection is unavailable")
	}
	return s.automatic.Configure(ctx, group, policy)
}
