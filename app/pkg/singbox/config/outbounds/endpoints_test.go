package outbounds

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEndpointRefsFromRaw(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []EndpointRef
	}{
		{name: "absent section", raw: "", want: []EndpointRef{}},
		{name: "null section", raw: "null", want: []EndpointRef{}},
		{
			name: "wireguard endpoint",
			raw:  `[{"type":"wireguard","tag":"tparts-endpoint","address":["10.0.0.2/32"]}]`,
			want: []EndpointRef{{Tag: "tparts-endpoint", Type: "wireguard"}},
		},
		{
			// A type the pinned library does not know must still be listed:
			// the section is read raw so a newer core's endpoints survive.
			name: "unknown type and untagged entry",
			raw:  `[{"type":"future-endpoint","tag":"a"},{"type":"wireguard"},"junk"]`,
			want: []EndpointRef{{Tag: "a", Type: "future-endpoint"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := endpointRefsFromRaw(json.RawMessage(tc.raw))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
