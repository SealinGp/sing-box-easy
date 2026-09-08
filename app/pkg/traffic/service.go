package trafficflow

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/integrations/clashapi"
	"strings"
)

type Service struct{ config *config.Manager }

func NewService(cm *config.Manager) *Service { return &Service{cm} }

type Stream struct {
	client Source
	filter Filter
}

func (s *Service) Prepare(filter Filter) (*Stream, error) {
	settings, err := s.config.GetClashAPISettings()
	if err != nil {
		return nil, err
	}
	client, err := clashapi.NewFromValues(settings.ExternalController, settings.Secret)
	if err != nil {
		if errors.Is(err, clashapi.ErrDisabled) {
			return nil, fault.New(fault.Unavailable, "live traffic needs experimental.clash_api.external_controller to be set")
		}
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
	message := err.Error()
	if !errors.Is(err, clashapi.ErrUnauthorized) {
		message = "sing-box is not reachable: " + message
	}
	return &fault.Error{Kind: fault.Unavailable, Message: message, Cause: err}
}
