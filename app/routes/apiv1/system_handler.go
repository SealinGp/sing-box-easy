package apiv1

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetSystemInfo(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, h.system.Info())
}

// GetLANClients lists LAN devices (DHCP leases, static hosts, neighbour table)
// so a route rule's source_mac_address / source_hostname can be picked rather
// than typed. `supported: false` off OpenWrt.
func (h *Handler) GetLANClients(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, h.system.LANClients())
}
