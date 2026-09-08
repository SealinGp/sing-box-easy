// Package configuration owns lossless configuration editing and reference policies.
package configuration

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

type Service struct{ configManager *config.Manager }

func New(manager *config.Manager) *Service { return &Service{configManager: manager} }
func objectBody(body []byte) ([]byte, error) {
	if err := requireJSONObject(body); err != nil {
		return nil, err
	}
	return body, nil
}
func mutationResult(err error, message string, extra map[string]any) (any, error) {
	if err != nil {
		return nil, err
	}
	data := map[string]any{"message": message}
	for k, v := range extra {
		data[k] = v
	}
	return data, nil
}
func bindInbound(ctx context.Context, body []byte) ([]byte, option.Inbound, error) {
	var inbound option.Inbound
	if err := singjson.UnmarshalContext(config.CreateContext(ctx), body, &inbound); err != nil {
		return nil, inbound, fault.New(fault.Invalid, "invalid inbound configuration: "+err.Error())
	}
	return body, inbound, nil
}
