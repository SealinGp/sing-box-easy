package protocol

import (
	"testing"
)

func TestTrojan_Parse(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{
			name:    "basic trojan",
			uri:     "trojan://603dc34d-1070-4e59-93ff-59f98250efe5@lm-allnodes.lma1b2.com:53438#test-node",
			wantErr: false,
		},
		{
			name:    "trojan with allowInsecure",
			uri:     "trojan://603dc34d-1070-4e59-93ff-59f98250efe5@lm-allnodes.lma1b2.com:53438?allowInsecure=1#%E6%96%B0%E5%8A%A0%E5%9D%A1%2008",
			wantErr: false,
		},
		{
			name:    "trojan with websocket",
			uri:     "trojan://password@example.com:443?type=ws&path=/ws&host=example.com#ws-node",
			wantErr: false,
		},
		{
			name:    "trojan with grpc",
			uri:     "trojan://password@example.com:443?type=grpc&serviceName=TrojanService&sni=example.com#grpc-node",
			wantErr: false,
		},
		{
			name:    "invalid format - missing @",
			uri:     "trojan://passwordexample.com:443#test",
			wantErr: true,
		},
		{
			name:    "invalid format - missing port",
			uri:     "trojan://password@example.com#test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trojan{}
			node, err := tr.Parse(tt.uri)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if node == nil {
					t.Error("Parse() returned nil node")
					return
				}

				if node.Type != "trojan" {
					t.Errorf("Parse() node.Type = %v, want trojan", node.Type)
				}

				if node.Tag == "" {
					t.Error("Parse() node.Tag is empty")
				}

				t.Logf("Successfully parsed node: %s (server: %s:%d)", node.Tag, tr.Server, tr.ServerPort)
			}
		})
	}
}

func TestTrojan_TypeName(t *testing.T) {
	tr := &Trojan{}
	if got := tr.TypeName(); got != "trojan" {
		t.Errorf("TypeName() = %v, want trojan", got)
	}
}

func TestTrojan_Schema(t *testing.T) {
	tr := &Trojan{}
	if got := tr.Schema(); got != "trojan://" {
		t.Errorf("Schema() = %v, want trojan://", got)
	}
}
