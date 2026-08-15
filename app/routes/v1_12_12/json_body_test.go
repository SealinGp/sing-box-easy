package v1_13_0

import (
	"encoding/json"
	"testing"
)

func TestDropEmptyJSONFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		keys []string
		want map[string]any
	}{
		{
			// The exact payload the wizard submits with a blank duration field.
			name: "drops an empty duration",
			body: `{"enabled":true,"path":"/etc/sing-box/cache.db","cache_id":"","store_fakeip":false,"store_rdrc":false,"rdrc_timeout":""}`,
			keys: []string{"rdrc_timeout"},
			want: map[string]any{
				"enabled": true, "path": "/etc/sing-box/cache.db",
				"cache_id": "", "store_fakeip": false, "store_rdrc": false,
			},
		},
		{
			name: "keeps a populated duration",
			body: `{"enabled":true,"rdrc_timeout":"12h"}`,
			keys: []string{"rdrc_timeout"},
			want: map[string]any{"enabled": true, "rdrc_timeout": "12h"},
		},
		{
			name: "only touches the named keys",
			body: `{"cache_id":"","rdrc_timeout":""}`,
			keys: []string{"rdrc_timeout"},
			want: map[string]any{"cache_id": ""},
		},
		{
			name: "absent key is a no-op",
			body: `{"enabled":true}`,
			keys: []string{"rdrc_timeout"},
			want: map[string]any{"enabled": true},
		},
		{
			// Only an empty *string* means "unset" — null, 0 and false are
			// values the caller chose and must reach the binder untouched.
			name: "does not drop null or zero",
			body: `{"a":null,"b":0,"c":false}`,
			keys: []string{"a", "b", "c"},
			want: map[string]any{"a": nil, "b": float64(0), "c": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dropEmptyJSONFields([]byte(tt.body), tt.keys...)

			var decoded map[string]any
			if err := json.Unmarshal(got, &decoded); err != nil {
				t.Fatalf("result is not valid JSON: %v (%s)", err, got)
			}
			if len(decoded) != len(tt.want) {
				t.Fatalf("got %v, want %v", decoded, tt.want)
			}
			for k, want := range tt.want {
				if decoded[k] != want {
					t.Errorf("key %q = %v, want %v", k, decoded[k], want)
				}
			}
		})
	}
}

func TestDropEmptyJSONFieldsPassesThroughUnparseableBodies(t *testing.T) {
	// Malformed or non-object bodies must reach the binder unchanged so the
	// caller sees the real parse error rather than a confusing rewrite.
	for _, body := range []string{"", "not json", "[1,2,3]", `"a string"`} {
		got := dropEmptyJSONFields([]byte(body), "rdrc_timeout")
		if string(got) != body {
			t.Errorf("dropEmptyJSONFields(%q) = %q, want it unchanged", body, got)
		}
	}
}
