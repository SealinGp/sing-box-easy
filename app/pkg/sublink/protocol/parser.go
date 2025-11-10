package protocol

import (
	"fmt"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
)

// Parser handles parsing of shadowsocks URIs
type Parser struct {
	uri string
}

// NewParser creates a new parser for shadowsocks URIs
func NewParser(uri string) (*Parser, error) {
	if uri == "" {
		return nil, fmt.Errorf("empty URI")
	}

	return &Parser{
		uri: strings.TrimSpace(uri),
	}, nil
}

func (p *Parser) extract(uri string) (string, string) {
	if index := strings.Index(uri, "://"); index != -1 {
		pl := uri[:index+3]
		content := uri[index+3:]
		return pl, content
	}
	return "", ""
}

// Parse parses the shadowsocks URI into a SubNode
func (p *Parser) Parse() (*node.SubNode, error) {
	schema, _ := p.extract(p.uri)

	ps, err := NewPBParser(schema)
	if err != nil {
		return nil, err
	}

	return ps.Parse(p.uri)
}
