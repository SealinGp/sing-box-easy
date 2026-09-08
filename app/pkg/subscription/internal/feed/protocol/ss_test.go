package protocol

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestShadowsocksParseWithQuery(t *testing.T) {
	// A valid Shadowsocks 2022 URI with query parameters and a tag
	uri := "ss://MjAyMi1ibGFrZTMtYWVzLTI1Ni1nY206SVhiRGp3RENxbGRiV3NPRHc3VUR3bzNEcWNPUlZsWERzaUhEbEV6Q2dzT3BKMnc3d3JsU0pFM0Rxbk53OkhNS1V3bzhvd3BzbmJuYzNkOE9nZDhPRFhzS0l3NDNEbE1PRndwWERpTUtUd3FCVFg4Sy93N1RDa3NLR2VoQkdFdz09@12.19.136.164:20674?type=tcp#claude-1-6j6q6w7k"

	ss := new(Shadowsocks)
	n, err := ss.Parse(uri)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if n.Type != "shadowsocks" {
		t.Errorf("parsed type = %q, want %q", n.Type, "shadowsocks")
	}

	if n.Tag != "claude-1-6j6q6w7k" {
		t.Errorf("parsed tag = %q, want %q", n.Tag, "claude-1-6j6q6w7k")
	}

	opts, ok := n.Options.(option.ShadowsocksROutboundOptions)
	if !ok {
		t.Fatalf("Options is %T, want option.ShadowsocksROutboundOptions", n.Options)
	}

	if opts.Server != "12.19.136.164" {
		t.Errorf("parsed server = %q, want %q", opts.Server, "12.19.136.164")
	}

	if opts.ServerPort != 20674 {
		t.Errorf("parsed port = %d, want %d", opts.ServerPort, 20674)
	}

	if opts.Method != "2022-blake3-aes-256-gcm" {
		t.Errorf("parsed method = %q, want %q", opts.Method, "2022-blake3-aes-256-gcm")
	}
}
