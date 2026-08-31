package protocol

import (
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
)

// SubNodeParserFactory creates a new instance of a SubNodeParser
type SubNodeParserFactory func() node.SubNodeParser

var ppMap = map[string]SubNodeParserFactory{
	"ss://":        func() node.SubNodeParser { return new(Shadowsocks) },
	"vmess://":     func() node.SubNodeParser { return new(Vmess) },
	"trojan://":    func() node.SubNodeParser { return new(Trojan) },
	"vless://":     func() node.SubNodeParser { return new(VLESS) },
	"hysteria2://": func() node.SubNodeParser { return new(Hysteria2) },
	"anytls://":    func() node.SubNodeParser { return new(AnyTLS) },
}

func NewPBParser(schema string) (node.SubNodeParser, error) {
	factory, ok := ppMap[schema]
	if !ok {
		return nil, fmt.Errorf("protocol %s not supported", schema)
	}

	// Create a new instance for each parse operation
	return factory(), nil
}
