package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

func parseNativeNodes(ctx context.Context, input string) ([]option.Outbound, error) {
	var raw json.RawMessage = []byte(input)
	if strings.HasPrefix(input, "{") {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		if nodes, ok := object["outbounds"]; ok {
			raw = nodes
		} else {
			raw = append(append([]byte{'['}, raw...), ']')
		}
	}
	var objects []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &objects); err != nil {
		return nil, fmt.Errorf("invalid outbound JSON: %w", err)
	}
	result := make([]option.Outbound, 0, len(objects))
	for i, node := range objects {
		if node == nil {
			return nil, fmt.Errorf("outbounds[%d] must be an object", i)
		}
		var kind string
		_ = json.Unmarshal(node["type"], &kind)
		// Older inbound-copy versions emitted display names as username on every
		// protocol. Only protocols whose authentication actually uses it retain it.
		if kind != "socks" && kind != "http" && kind != "naive" {
			delete(node, "username")
		}
		if _, ok := node["tag"]; !ok {
			node["tag"], _ = json.Marshal(fmt.Sprintf("%s-import-%d", kind, i+1))
		}
		encoded, err := json.Marshal(node)
		if err != nil {
			return nil, err
		}
		var outbound option.Outbound
		if err = singjson.UnmarshalContext(config.CreateContext(ctx), encoded, &outbound); err != nil {
			return nil, fmt.Errorf("outbounds[%d]: %w", i, err)
		}
		result = append(result, outbound)
	}
	return result, nil
}
