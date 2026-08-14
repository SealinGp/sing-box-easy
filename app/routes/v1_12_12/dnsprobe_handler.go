package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/dnsprobe"
	"github.com/cloudwego/hertz/pkg/app"
)

// DNSProbeRequest asks what this deployment does with one domain.
type DNSProbeRequest struct {
	Domain string `json:"domain"`
	// Type is a DNS record type; defaults to A.
	Type string `json:"type"`
	// CompareServers additionally queries each directly reachable configured
	// resolver so their answers can be compared. Off by default because it
	// sends real traffic to every upstream.
	CompareServers bool `json:"compare_servers"`
}

// ProbeDNS resolves a domain through sing-box and explains the routing.
//
// POST /dns/probe
func (h *Handler) ProbeDNS(ctx context.Context, c *app.RequestContext) {
	var req DNSProbeRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeConfigError, err.Error())
		return
	}

	result, err := dnsprobe.Run(&cfg.Options, dnsprobe.Options{
		Domain:         req.Domain,
		QueryType:      req.Type,
		CompareServers: req.CompareServers,
		Tailer:         h.logTailer(),
	})
	if err != nil {
		// The only errors here are invalid input; everything else degrades
		// into the result itself.
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}

	respOK(ctx, c, result)
}

// logTailer adapts the service controller's log tailing to what the probe
// needs, so the probe package does not depend on the service package.
func (h *Handler) logTailer() dnsprobe.LogTailer {
	return func(lines int, afterCursor string) ([]string, string, error) {
		chunk, err := h.serviceController.TailLogs(lines, afterCursor)
		if err != nil {
			return nil, "", err
		}
		return chunk.Lines, chunk.Cursor, nil
	}
}
