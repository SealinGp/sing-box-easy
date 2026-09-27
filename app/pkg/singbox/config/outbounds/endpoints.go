package outbounds

import (
	"encoding/json"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// EndpointRef names one entry of the top-level `endpoints` section.
//
// Endpoints (wireguard, tailscale, …) are dialable by tag exactly like an
// outbound — a route rule's `outbound` may name one — but they live in their
// own section, so a picker fed only by `outbounds` could not offer them and a
// rule already routed to one read as "missing".
type EndpointRef struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

// endpointRefs lists the configured endpoints. Read as raw JSON rather than
// through the pinned option structs, so an endpoint type added by a newer core
// is still listed. Failure degrades to an empty list: the outbound list it
// accompanies is the primary payload and must not fail with it.
func (h *Service) endpointRefs() []EndpointRef {
	raw, _, err := h.configManager.GetConfigSection("endpoints")
	if err != nil {
		logger.Warn("failed to read endpoints section", zap.Error(err))
		return []EndpointRef{}
	}
	return endpointRefsFromRaw(raw)
}

func endpointRefsFromRaw(raw json.RawMessage) []EndpointRef {
	refs := []EndpointRef{}
	if len(raw) == 0 {
		return refs
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return refs
	}
	for _, entry := range entries {
		var head struct {
			Tag  string `json:"tag"`
			Type string `json:"type"`
		}
		if json.Unmarshal(entry, &head) != nil || head.Tag == "" {
			continue
		}
		refs = append(refs, EndpointRef{Tag: head.Tag, Type: head.Type})
	}
	return refs
}
