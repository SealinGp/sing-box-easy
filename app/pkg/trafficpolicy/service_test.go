package trafficpolicy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/core"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

var ctx = context.Background()

type testCore struct{ fail bool }

func (c *testCore) Version(context.Context) (core.CoreVersion, error) {
	return core.CoreVersion{Major: 1, Minor: 14, Raw: "1.14.0"}, nil
}
func (c *testCore) Validate(context.Context, string) error {
	if c.fail {
		return errors.New("invalid core candidate")
	}
	return nil
}
func setup(t *testing.T) (*Service, *config.Manager, *testCore) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{"future":{"keep":true},"outbounds":[{"type":"direct","tag":"direct"}],"dns":{"servers":[{"type":"local","tag":"a"},{"type":"local","tag":"b"},{"type":"local","tag":"c"}],"rules":[]},"route":{"rules":[]}}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := xorm.NewEngine("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	c := &testCore{}
	m := config.NewManagerWithCore(path, c)
	s, err := New(db, m)
	if err != nil {
		t.Fatal(err)
	}
	return s, m, c
}
func apply(t *testing.T, s *Service, c Command) Plan {
	t.Helper()
	p, err := s.Plan(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Commit(ctx, p.Command); err != nil {
		t.Fatal(err)
	}
	return p
}
func addGroup(t *testing.T, s *Service, name string) Group {
	p := apply(t, s, Command{Action: "save_group", Group: &Group{Name: name, Servers: []string{"a", "b"}, Timeout: 3, Fallback: "strict"}})
	return *p.Command.Group
}
func addPolicy(t *testing.T, s *Service, g Group) Policy {
	p := apply(t, s, Command{Action: "save_policy", Policy: &Policy{Name: "Sites", GroupID: g.ID, Outbound: "direct", Mode: "or", Conditions: []Condition{{Kind: "domain_suffix", Values: []string{"one.test", "two.test"}}}}})
	return *p.Command.Policy
}
func TestDependenciesReplacementAndOwnedRemoval(t *testing.T) {
	s, m, _ := setup(t)
	g := addGroup(t, s, "Overseas")
	p := addPolicy(t, s, g)
	if _, err := s.Plan(ctx, Command{Action: "delete_group", ID: g.ID}); err == nil {
		t.Fatal("referenced group deletion allowed")
	}
	replacement := addGroup(t, s, "Other")
	apply(t, s, Command{Action: "delete_group", ID: g.ID, Replacement: replacement.ID})
	snap, _ := s.Snapshot(ctx)
	if len(snap.Groups) != 1 || snap.Policies[0].GroupID != replacement.ID || snap.Policies[0].Status != "synced" {
		t.Fatalf("bad replacement: %+v", snap)
	}
	apply(t, s, Command{Action: "delete_policy", ID: p.ID})
	doc, _ := m.GetConfigDocument()
	d, _ := decode(doc.Raw)
	if len(rules(d, "dns")) != 0 || len(rules(d, "route")) != 0 || d["future"] == nil {
		t.Fatal("unowned document changed or owned members retained")
	}
}
func TestRawEditDriftRequiresAcknowledgmentAndNeverResurrects(t *testing.T) {
	s, m, _ := setup(t)
	g := addGroup(t, s, "Group")
	p := addPolicy(t, s, g)
	doc, _ := m.GetConfigDocument()
	d, _ := decode(doc.Raw)
	putRules(d, "dns", []Rule{})
	putRules(d, "route", []Rule{})
	if err := s.SaveRaw(ctx, encoded(d), ""); err == nil {
		t.Fatal("unacknowledged raw deletion allowed")
	}
	impact, err := s.ImpactPlan(ctx, encoded(d))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SaveRaw(ctx, encoded(d), impact["revision"].(string)); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(ctx)
	if snap.Policies[0].Status != "missing" || len(snap.Groups) != 1 {
		t.Fatal("lost metadata")
	}
	if _, err = s.Plan(ctx, Command{Action: "delete_policy", ID: p.ID}); err == nil {
		t.Fatal("drift deletion allowed")
	}
	apply(t, s, Command{Action: "restore_policy", ID: p.ID})
	snap, _ = s.Snapshot(ctx)
	if snap.Policies[0].Status != "synced" {
		t.Fatal("restore failed")
	}
}
func TestStaleRevisionAndCoreFailureLeaveBothStoresUntouched(t *testing.T) {
	s, m, c := setup(t)
	g := addGroup(t, s, "Group")
	plan, err := s.Plan(ctx, Command{Action: "save_policy", Policy: &Policy{Name: "X", GroupID: g.ID, Outbound: "direct", Mode: "or", Conditions: []Condition{{Kind: "domain", Values: []string{"x.test"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := m.GetConfigDocument()
	c.fail = true
	if err = s.Commit(ctx, plan.Command); err == nil {
		t.Fatal("core error ignored")
	}
	after, _ := m.GetConfigDocument()
	if string(before.Raw) != string(after.Raw) {
		t.Fatal("partial file write")
	}
	snap, _ := s.Snapshot(ctx)
	if len(snap.Policies) != 0 {
		t.Fatal("partial metadata write")
	}
	c.fail = false
	addGroup(t, s, "Second")
	if err = s.Commit(ctx, plan.Command); err == nil {
		t.Fatal("stale revision accepted")
	}
}
func TestRecoveryBeforeAndAfterPromotion(t *testing.T) {
	for _, promoted := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "after"}[promoted], func(t *testing.T) {
			s, m, _ := setup(t)
			g := addGroup(t, s, "Group")
			doc, _ := m.GetConfigDocument()
			state, _ := s.load(doc.Raw)
			p, err := build(doc.Raw, state, Command{Action: "save_policy", Policy: &Policy{Name: "X", GroupID: g.ID, Outbound: "direct", Mode: "or", Conditions: []Condition{{Kind: "domain", Values: []string{"x.test"}}}}})
			if err != nil {
				t.Fatal(err)
			}
			r, _ := s.record()
			before, _ := decode(doc.Raw)
			after, _ := decode(p.document)
			r.Pending = string(encoded(p.state))
			r.BeforeHash = hash(before)
			r.AfterHash = hash(after)
			if err = s.write(r); err != nil {
				t.Fatal(err)
			}
			if promoted {
				if err = os.WriteFile(m.GetConfigPath(), p.document, 0600); err != nil {
					t.Fatal(err)
				}
			}
			snap, err := s.Snapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if promoted {
				want = 1
			}
			if len(snap.Policies) != want {
				t.Fatal("recovery chose wrong state")
			}
			r, _ = s.record()
			if r.Pending != "" {
				t.Fatal("pending operation not finalized")
			}
		})
	}
}
func TestExternalReferenceAndPartialEditBlockDestruction(t *testing.T) {
	s, m, _ := setup(t)
	p := addPolicy(t, s, addGroup(t, s, "Group"))
	doc, _ := m.GetConfigDocument()
	d, _ := decode(doc.Raw)
	all := rules(d, "dns")
	all = append(all, Rule{"action": "respond", "match_response": p.Prefix + "_1"})
	putRules(d, "dns", all)
	if err := m.SaveRawConfig(ctx, encoded(d)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Plan(ctx, Command{Action: "delete_policy", ID: p.ID}); err == nil {
		t.Fatal("dangling external reference allowed")
	}
	all = all[:len(all)-1]
	all[0]["timeout"] = "9s"
	putRules(d, "dns", all)
	if err := m.SaveRawConfig(ctx, encoded(d)); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(ctx)
	if snap.Policies[0].Status != "modified" {
		t.Fatal("manual edit undetected")
	}
	apply(t, s, Command{Action: "detach_policy", ID: p.ID})
	after, _ := m.GetConfigDocument()
	got, _ := decode(after.Raw)
	if hash(got) != hash(d) {
		t.Fatal("detach changed config")
	}
}
func TestGeneratedRulesOrderAndMatcherSemantics(t *testing.T) {
	s, m, _ := setup(t)
	g := addGroup(t, s, "Group")
	doc, _ := m.GetConfigDocument()
	d, _ := decode(doc.Raw)
	p := Policy{Name: "X", GroupID: g.ID, Outbound: "direct", Mode: "or", Prefix: "dns_test", Conditions: []Condition{{Kind: "domain_suffix", Values: []string{"a.test"}}, {Kind: "domain_suffix", Values: []string{"b.test"}}}}
	if err := generate(&p, g, d); err != nil {
		t.Fatal(err)
	}
	if len(p.Route[0]["domain_suffix"].([]string)) != 2 {
		t.Fatal("OR values not merged")
	}
	if p.DNS[0]["action"] != "evaluate" || p.DNS[1]["action"] != "evaluate" || p.DNS[2]["action"] != "respond" || p.DNS[4]["action"] != "reject" {
		t.Fatal("incorrect DNS race order")
	}
	p.Mode = "and"
	if err := generate(&p, g, d); err != nil {
		t.Fatal(err)
	}
	if p.Route[0]["type"] != "logical" {
		t.Fatal("AND rows merged")
	}
}

func TestAcceptCompatibleSettingsUpdatesDependents(t *testing.T) {
	s, m, _ := setup(t)
	g := addGroup(t, s, "Group")
	p := addPolicy(t, s, g)
	second := apply(t, s, Command{Action: "save_policy", Policy: &Policy{Name: "Other", GroupID: g.ID, Outbound: "direct", Mode: "or", Conditions: []Condition{{Kind: "domain", Values: []string{"other.test"}}}, RouteIndex: 1, DNSIndex: 5}}).Command.Policy
	doc, _ := m.GetConfigDocument()
	d, _ := decode(doc.Raw)
	all := rules(d, "dns")
	for _, r := range all {
		if r["tag"] == p.Prefix+"_1" || r["tag"] == p.Prefix+"_2" {
			r["timeout"] = "7s"
		}
	}
	putRules(d, "dns", all)
	if err := m.SaveRawConfig(ctx, encoded(d)); err != nil {
		t.Fatal(err)
	}
	apply(t, s, Command{Action: "accept_policy", ID: p.ID})
	snap, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Groups[0].Timeout != 7 {
		t.Fatal("settings not adopted")
	}
	for _, policy := range snap.Policies {
		if policy.Status != "synced" {
			t.Fatalf("dependent drift: %s", policy.Name)
		}
		if policy.ID == second.ID && policy.DNS[0]["timeout"] != "7s" {
			t.Fatal("dependent not updated")
		}
	}
}
func TestAmbiguousRecoveryKeepsPanelAndManualConfig(t *testing.T) {
	s, m, _ := setup(t)
	g := addGroup(t, s, "Group")
	doc, _ := m.GetConfigDocument()
	state, _ := s.load(doc.Raw)
	p, err := build(doc.Raw, state, Command{Action: "save_policy", Policy: &Policy{Name: "Pending", GroupID: g.ID, Outbound: "direct", Mode: "or", Conditions: []Condition{{Kind: "domain", Values: []string{"pending.test"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := s.record()
	before, _ := decode(doc.Raw)
	after, _ := decode(p.document)
	r.Pending = string(encoded(p.state))
	r.BeforeHash = hash(before)
	r.AfterHash = hash(after)
	if err = s.write(r); err != nil {
		t.Fatal(err)
	}
	after["future"] = Rule{"manual": true}
	if err = os.WriteFile(m.GetConfigPath(), encoded(after), 0600); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(s.db, m)
	if err != nil {
		t.Fatal("panel must remain available", err)
	}
	snap, err := restarted.Snapshot(ctx)
	if err != nil || !snap.Recovery {
		t.Fatal("missing recovery state", err)
	}
	if err = m.SaveRawConfig(ctx, encoded(before)); !errors.Is(err, ErrRecovery) {
		t.Fatal("write bypassed recovery barrier", err)
	}
	apply(t, restarted, Command{Action: "recover_accept"})
	snap, err = restarted.Snapshot(ctx)
	if err != nil || snap.Recovery || len(snap.Policies) != 1 {
		t.Fatal("explicit recovery failed", err)
	}
	current, _ := m.GetConfigDocument()
	got, _ := decode(current.Raw)
	if hash(got) != hash(after) {
		t.Fatal("recovery rewrote manual configuration")
	}
}

func TestSingleServerAndGroupDetachPreserveConfiguration(t *testing.T) {
	s, m, _ := setup(t)
	plan := apply(t, s, Command{Action: "save_policy", Policy: &Policy{Name: "Single", DNSServer: "a", Timeout: 3, Fallback: "strict", Outbound: "direct", Mode: "or", Conditions: []Condition{{Kind: "domain_suffix", Values: []string{"single.test"}}}}})
	p := plan.Command.Policy
	if len(p.DNS) != 3 {
		t.Fatal("expected evaluate, respond and guard")
	}
	if _, ok := p.DNS[1]["race"]; ok {
		t.Fatal("single server must not race")
	}
	g := addGroup(t, s, "Group")
	_ = apply(t, s, Command{Action: "save_policy", Policy: &Policy{Name: "Grouped", GroupID: g.ID, Outbound: "direct", Mode: "or", Conditions: []Condition{{Kind: "domain_suffix", Values: []string{"grouped.test"}}}, RouteIndex: 1, DNSIndex: 3}})
	before, _ := m.GetConfigDocument()
	apply(t, s, Command{Action: "detach_group", ID: g.ID})
	after, _ := m.GetConfigDocument()
	if string(before.Raw) != string(after.Raw) {
		t.Fatal("stop managing rewrote configuration")
	}
	snap, _ := s.Snapshot(ctx)
	if len(snap.Groups) != 0 || len(snap.Policies) != 1 {
		t.Fatal("wrong metadata detached")
	}
}
func TestRawServerRemovalReportsAffectedPolicy(t *testing.T) {
	s, m, _ := setup(t)
	addPolicy(t, s, addGroup(t, s, "Group"))
	doc, _ := m.GetConfigDocument()
	d, _ := decode(doc.Raw)
	section(d, "dns")["servers"] = []any{Rule{"type": "local", "tag": "c"}}
	impact, err := s.ImpactPlan(ctx, encoded(d))
	if err != nil {
		t.Fatal(err)
	}
	policies := impact["policies"].([]Policy)
	if len(policies) != 1 || policies[0].Status != "missing_dependency" {
		t.Fatal("server removal impact not reported")
	}
}
