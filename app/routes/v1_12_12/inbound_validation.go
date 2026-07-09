package v1_13_0

import (
	"fmt"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func validateInbound(inbound option.Inbound) error {
	if strings.TrimSpace(inbound.Type) == "" {
		return fmt.Errorf("type is required")
	}
	if strings.TrimSpace(inbound.Tag) == "" {
		return fmt.Errorf("tag is required")
	}
	if inbound.Type != C.TypeTun && inboundListenPort(inbound) == 0 {
		return fmt.Errorf("listen_port is required")
	}

	switch options := inbound.Options.(type) {
	case *option.ShadowsocksInboundOptions:
		if strings.TrimSpace(options.Method) == "" {
			return fmt.Errorf("method is required")
		}
		if options.Method != "none" && options.Password == "" {
			return fmt.Errorf("password is required")
		}
	case *option.VMessInboundOptions:
		if len(options.Users) == 0 {
			return fmt.Errorf("users is required")
		}
		for i, user := range options.Users {
			if strings.TrimSpace(user.UUID) == "" {
				return fmt.Errorf("users[%d].uuid is required", i)
			}
		}
	}

	return nil
}

func inboundListenPort(inbound option.Inbound) uint16 {
	switch options := inbound.Options.(type) {
	case *option.RedirectInboundOptions:
		return options.ListenPort
	case *option.TProxyInboundOptions:
		return options.ListenPort
	case *option.DirectInboundOptions:
		return options.ListenPort
	case *option.SocksInboundOptions:
		return options.ListenPort
	case *option.HTTPMixedInboundOptions:
		return options.ListenPort
	case *option.ShadowsocksInboundOptions:
		return options.ListenPort
	case *option.VMessInboundOptions:
		return options.ListenPort
	case *option.TrojanInboundOptions:
		return options.ListenPort
	case *option.NaiveInboundOptions:
		return options.ListenPort
	case *option.HysteriaInboundOptions:
		return options.ListenPort
	case *option.Hysteria2InboundOptions:
		return options.ListenPort
	case *option.VLESSInboundOptions:
		return options.ListenPort
	case *option.TUICInboundOptions:
		return options.ListenPort
	case *option.ShadowTLSInboundOptions:
		return options.ListenPort
	default:
		return 0
	}
}
