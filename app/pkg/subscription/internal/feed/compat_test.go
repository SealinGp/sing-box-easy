package sublink

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/parser"
)

func (l *SubLink) parseNonBase64Body(body []byte, source string) ([]*node.SubNode, error) {
	return parser.ParsePlain(body)
}
