package nodegroup

import (
	"reflect"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
)

func TestPruneEmptyGroupsCascadesWithoutChangingInput(t *testing.T) {
	before := []config.Outbound{makeSelector("managed", []string{"old"}, "")}
	after := []config.Outbound{
		makeSelector("parent", []string{"child", "endpoint"}, "child"),
		makeURLTest("child", []string{"empty"}),
		makeSelector("empty", nil, ""),
		makeSelector("removed-managed-parent", []string{"managed"}, ""),
		makeSelector("opaque-parent", []string{"opaque-outbound"}, ""),
	}
	want := []config.Outbound{
		makeSelector("parent", []string{"endpoint"}, ""),
		makeSelector("opaque-parent", []string{"opaque-outbound"}, ""),
	}
	got := PruneEmptyGroups(before, after)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if len(selectorMembers(t, after[0])) != 2 || selectorDefault(t, after[0]) != "child" || len(urlTestMembers(t, after[1])) != 1 {
		t.Fatal("input group options mutated")
	}
	if again := PruneEmptyGroups(nil, got); !reflect.DeepEqual(again, got) {
		t.Fatal("cleanup is not idempotent")
	}
}
