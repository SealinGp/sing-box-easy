package repo

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/model"
	"reflect"
	"testing"
)

// TestMarshalUnmarshalInfo verifies the JSON column round-trips, and that
// empty/invalid encodings decode to nil rather than erroring.
func TestMarshalUnmarshalInfo(t *testing.T) {
	logger.InitDefault()
	in := []model.SubInfo{{Key: "剩余流量", Value: "4.59 TB"}, {Key: "套餐到期", Value: "2026-10-19"}}
	encoded, err := marshalInfo(in)
	if err != nil {
		t.Fatalf("marshalInfo error: %v", err)
	}
	if got := unmarshalInfo(encoded); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}

	if got := unmarshalInfo(""); got != nil {
		t.Errorf("unmarshalInfo(\"\") = %+v, want nil", got)
	}
	if got := unmarshalInfo("not json"); got != nil {
		t.Errorf("unmarshalInfo(invalid) = %+v, want nil", got)
	}

	// nil marshals to an empty JSON array (clears stale info).
	if encoded, _ := marshalInfo(nil); encoded != "[]" {
		t.Errorf("marshalInfo(nil) = %q, want %q", encoded, "[]")
	}
}
