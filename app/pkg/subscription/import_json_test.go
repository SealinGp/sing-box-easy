package subscription

import (
	"context"
	"testing"
)

func TestNativeNodeImport(t *testing.T) {
	const node = `{"type":"vmess","tag":"vmess-copy","server":"192.0.2.10","server_port":443,"uuid":"00000000-0000-4000-8000-000000000001","username":"display"}`
	for _, input := range []string{node, "[" + node + "]", `{"outbounds":[` + node + `]}`} {
		nodes, err := PreviewImport(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) != 1 || nodes[0].Type != "vmess" {
			t.Fatal("not imported")
		}
	}
	for _, input := range []string{`{`, `{"type":"vmess","listen_port":443}`, `[null]`} {
		if _, err := PreviewImport(context.Background(), input); err == nil {
			t.Fatalf("accepted invalid input %s", input)
		}
	}
}
