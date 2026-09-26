package outbounds

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/nodetag"
	"github.com/sagernet/sing-box/option"
)

// countingCore stands in for the sing-box binary: it accepts every config and
// counts how often it was asked, so a test can tell "wrote nothing" apart
// from "wrote the same thing".
type countingCore struct{ validations int }

func (c *countingCore) Version(context.Context) (config.CoreVersion, error) {
	return config.CoreVersion{Major: 1, Minor: 12, Patch: 12, Raw: "1.12.12"}, nil
}

func (c *countingCore) Validate(context.Context, string) error {
	c.validations++
	return nil
}

func newManager(t *testing.T, raw string) (*config.Manager, *countingCore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	core := &countingCore{}
	return config.NewManagerWithCore(path, core), core, path
}

func trojan(tag, server string, port uint16) config.Outbound {
	return config.Outbound{
		Type: "trojan",
		Tag:  tag,
		Options: &option.TrojanOutboundOptions{
			ServerOptions: option.ServerOptions{Server: server, ServerPort: port},
			Password:      "secret",
		},
	}
}

func outboundTags(t *testing.T, m *config.Manager) []string {
	t.Helper()
	cfg, err := m.GetOutboundsConfig()
	if err != nil {
		t.Fatal(err)
	}
	tags := make([]string, 0, len(cfg.Outbounds))
	for _, ob := range cfg.Outbounds {
		tags = append(tags, ob.Tag)
	}
	return tags
}

func members(t *testing.T, m *config.Manager, tag string) []string {
	t.Helper()
	cfg, err := m.GetOutboundsConfig()
	if err != nil {
		t.Fatal(err)
	}
	for _, ob := range cfg.Outbounds {
		if ob.Tag != tag {
			continue
		}
		switch o := ob.Options.(type) {
		case *option.SelectorOutboundOptions:
			return o.Outbounds
		case *option.URLTestOutboundOptions:
			return o.Outbounds
		}
		t.Fatalf("%q is not a group: %T", tag, ob.Options)
	}
	t.Fatalf("no outbound %q", tag)
	return nil
}

func TestAddOutboundsSkipsANodeStoredUnderAnOlderTagShape(t *testing.T) {
	existing := trojan("香港 09", "s4.example.com", 37219)
	legacyTag := nodetag.GenerateUniqueTag("香港 09", existing)
	m, _, _ := newManager(t, `{"outbounds":[
		{"type":"trojan","tag":"`+legacyTag+`","server":"s4.example.com","server_port":37219,"password":"secret"}
	]}`)

	fresh := trojan("日本 01", "jp.example.com", 443)
	added, skipped, err := addOutbounds(context.Background(), m, []config.Outbound{existing, fresh})
	if err != nil {
		t.Fatal(err)
	}

	// Re-pasting a link added before tags were fingerprinted is a skip, and it
	// is reported under the tag it is actually stored as.
	if !slices.Equal(skipped, []string{legacyTag}) {
		t.Errorf("skipped = %v, want [%s]", skipped, legacyTag)
	}
	wantNew := nodetag.OutboundTagCandidates("日本 01", fresh)[0]
	if !slices.Equal(added, []string{wantNew}) {
		t.Errorf("added = %v, want [%s]", added, wantNew)
	}
	if got := outboundTags(t, m); !slices.Equal(got, []string{legacyTag, wantNew}) {
		t.Errorf("outbounds = %v, want [%s %s]", got, legacyTag, wantNew)
	}
}

func TestAddOutboundsWritesNothingWhenEveryNodeExists(t *testing.T) {
	node := trojan("香港 09", "s4.example.com", 37219)
	tag := nodetag.OutboundTagCandidates("香港 09", node)[0]
	m, core, path := newManager(t, `{"outbounds":[
		{"type":"trojan","tag":"`+tag+`","server":"s4.example.com","server_port":37219,"password":"secret"}
	]}`)
	before, _ := os.ReadFile(path)

	added, skipped, err := addOutbounds(context.Background(), m, []config.Outbound{node})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 0 || !slices.Equal(skipped, []string{tag}) {
		t.Errorf("added=%v skipped=%v, want none added and [%s] skipped", added, skipped, tag)
	}
	// No validate-then-write cycle for a config that did not change: running
	// one would surface unrelated pre-existing validation failures.
	if core.validations != 0 {
		t.Errorf("sing-box check ran %d times for a no-op add", core.validations)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Error("config.json was rewritten by a no-op add")
	}
}

func TestReconcilerApplyCarriesRenamesAndDeletesIntoGroups(t *testing.T) {
	m, _, _ := newManager(t, `{"outbounds":[
		{"type":"trojan","tag":"hk-old","server":"hk.example.com","server_port":443,"password":"secret"},
		{"type":"trojan","tag":"jp","server":"jp.example.com","server_port":443,"password":"secret"},
		{"type":"selector","tag":"proxy","outbounds":["hk-old","jp"],"default":"hk-old"},
		{"type":"urltest","tag":"auto","outbounds":["hk-old","jp"]}
	]}`)

	renamed := trojan("hk-new", "hk.example.com", 443)
	err := NewReconciler(m, nil).Apply(context.Background(), "sub_test", func(*config.SingBoxConfig) Changes {
		return Changes{
			Update: map[string]config.Outbound{"hk-old": renamed},
			Delete: map[string]struct{}{"jp": {}},
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	// A rename that was not carried into the groups would leave them naming
	// a tag that no longer exists; a delete that was not would do the same.
	for _, group := range []string{"proxy", "auto"} {
		if got := members(t, m, group); !slices.Equal(got, []string{"hk-new"}) {
			t.Errorf("%s members = %v, want [hk-new]", group, got)
		}
	}
	// Membership, not position: the raw-preserving writer keeps records by
	// tag, so a renamed node is re-emitted after the records it did not touch.
	got := outboundTags(t, m)
	slices.Sort(got)
	if !slices.Equal(got, []string{"auto", "hk-new", "proxy"}) {
		t.Errorf("outbounds = %v, want {auto hk-new proxy}", got)
	}
}

func TestReconcilerApplyWithNoChangesDoesNotWrite(t *testing.T) {
	m, core, _ := newManager(t, `{"outbounds":[{"type":"direct","tag":"direct"}]}`)
	err := NewReconciler(m, nil).Apply(context.Background(), "sub_test", func(*config.SingBoxConfig) Changes {
		return Changes{}
	})
	if err != nil {
		t.Fatal(err)
	}
	if core.validations != 0 {
		t.Errorf("sing-box check ran %d times for an empty change set", core.validations)
	}
}
