package traffic

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"strings"
)

type Service struct{ config *config.Manager }

func NewService(cm *config.Manager) *Service { return &Service{cm} }

type Stream struct {
	client Source
	filter Filter
}

func (s *Service) Prepare(filter Filter) (*Stream, error) {
	client, err := s.controller("live traffic")
	if err != nil {
		return nil, err
	}
	filter.SourceIP = strings.TrimSpace(filter.SourceIP)
	filter.Host = strings.TrimSpace(filter.Host)
	return &Stream{client, filter}, nil
}
func (s *Stream) Run(ctx context.Context, emit func(*Frame) error) error {
	err := Run(ctx, s.client, Options{Filter: s.filter}, emit)
	if err == nil || errors.Is(err, context.Canceled) {
		return err
	}
	return unreachable(err)
}
