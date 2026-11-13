package trafficpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"xorm.io/xorm"
)

// One row atomically stores metadata plus the write-ahead intent. Config bytes
// are not duplicated here: recovery compares canonical before/after hashes.
type Record struct {
	ID         int64  `xorm:"pk"`
	State      string `xorm:"TEXT"`
	Pending    string `xorm:"TEXT"`
	BeforeHash string
	AfterHash  string
}

func (Record) TableName() string { return "traffic_policy_state" }

type Service struct {
	db       *xorm.Engine
	manager  *config.Manager
	applying bool
}

func New(db *xorm.Engine, m *config.Manager) (*Service, error) {
	if err := db.Sync2(new(Record)); err != nil {
		return nil, err
	}
	s := &Service{db: db, manager: m}
	m.SetWriteBarrier(func(raw []byte) error {
		if s.applying {
			return nil
		}
		return s.recover(raw)
	})
	// Reconcile pending operations at startup, including interrupted finalization.
	if raw, err := os.ReadFile(m.GetConfigPath()); err == nil {
		if err = s.recover(raw); err != nil && !errors.Is(err, ErrRecovery) {
			return nil, err
		}
	}
	return s, nil
}
func (s *Service) record() (Record, error) {
	r := Record{ID: 1}
	ok, err := s.db.ID(1).Get(&r)
	if err != nil {
		return r, err
	}
	if !ok {
		r.State = string(encoded(State{Groups: []Group{}, Policies: []Policy{}}))
		_, err = s.db.Insert(&r)
	}
	return r, err
}
func (s *Service) write(r Record) error { _, err := s.db.ID(1).AllCols().Update(&r); return err }
func (s *Service) recover(raw []byte) error {
	r, err := s.record()
	if err != nil {
		return err
	}
	if r.Pending == "" {
		return nil
	}
	d, err := decode(raw)
	if err != nil {
		return err
	}
	h := hash(d)
	switch h {
	case r.AfterHash:
		r.State = r.Pending
	case r.BeforeHash:
	default:
		return ErrRecovery
	}
	r.Pending = ""
	r.BeforeHash = ""
	r.AfterHash = ""
	return s.write(r)
}
func (s *Service) load(raw []byte) (State, error) {
	if err := s.recover(raw); err != nil {
		return State{}, err
	}
	r, err := s.record()
	if err != nil {
		return State{}, err
	}
	var state State
	err = json.Unmarshal([]byte(r.State), &state)
	return state, err
}
func revision(d Rule, state State) string { return hash([]any{d, state}) }
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	var out Snapshot
	err := s.manager.WithDocumentLock(ctx, func(doc config.ConfigDocument, _ func(config.ConfigDocument) error) error {
		state, err := s.load(doc.Raw)
		recovery := errors.Is(err, ErrRecovery)
		if recovery {
			r, readErr := s.record()
			if readErr != nil {
				return readErr
			}
			err = json.Unmarshal([]byte(r.State), &state)
		}
		if err != nil {
			return err
		}
		d, err := decode(doc.Raw)
		if err != nil {
			return err
		}
		out = Snapshot{Recovery: recovery, State: cloneState(state), Revision: revision(d, state), Servers: tags(section(d, "dns")["servers"]), Outbounds: tags(d["outbounds"]), RuleSets: tags(section(d, "route")["rule_set"]), RouteRules: rules(d, "route"), DNSRules: rules(d, "dns")}
		for i := range out.Policies {
			out.Policies[i].Status = reportedStatus(out.Policies[i], state, d)
		}
		return nil
	})
	return out, err
}
func (s *Service) Plan(ctx context.Context, c Command) (Plan, error) {
	var p Plan
	err := s.manager.WithDocumentLock(ctx, func(doc config.ConfigDocument, _ func(config.ConfigDocument) error) error {
		if isRecovery(c) {
			var err error
			p, err = s.recoveryPlan(doc.Raw, c)
			return err
		}
		state, err := s.load(doc.Raw)
		if err != nil {
			return err
		}
		p, err = build(doc.Raw, state, c)
		return err
	})
	return p, err
}
func (s *Service) Commit(ctx context.Context, c Command) error {
	if isRecovery(c) {
		return s.commitRecovery(ctx, c)
	}
	return s.manager.WithDocumentLock(ctx, func(doc config.ConfigDocument, save func(config.ConfigDocument) error) error {
		state, err := s.load(doc.Raw)
		if err != nil {
			return err
		}
		d, err := decode(doc.Raw)
		if err != nil {
			return err
		}
		if c.Revision == "" || c.Revision != revision(d, state) {
			return fmt.Errorf("configuration or dependencies changed; refresh and preview again")
		}
		p, err := build(doc.Raw, state, c)
		if err != nil {
			return err
		}
		next, _ := decode(p.document)
		if hash(d) == hash(next) {
			r, err := s.record()
			if err != nil {
				return err
			}
			r.State = string(encoded(p.state))
			return s.write(r)
		}
		// Both actions are checked by the installed core during save. The explicit
		// gate prevents silently emitting features absent from older core versions.
		caps, err := s.manager.CoreCapabilities(ctx)
		if err != nil {
			return err
		}
		if len(rules(next, "dns")) > 0 && addsDNS(p) && (!caps.DNSEvaluate || !caps.DNSRespond || !caps.DNSRace || !caps.DNSMatchResponse) {
			return fmt.Errorf("installed core does not support parallel DNS evaluation")
		}
		r, err := s.record()
		if err != nil {
			return err
		}
		r.Pending = string(encoded(p.state))
		r.BeforeHash = hash(d)
		r.AfterHash = hash(next)
		if err = s.write(r); err != nil {
			return err
		}
		s.applying = true
		err = save(config.ConfigDocument{Raw: p.document})
		s.applying = false
		if err != nil { // Recover against what actually reached disk; never guess.
			raw, readErr := os.ReadFile(s.manager.GetConfigPath())
			if readErr == nil {
				if recoveryErr := s.recover(raw); recoveryErr != nil {
					return fmt.Errorf("%w; recovery pending: %v", err, recoveryErr)
				}
			}
			return err
		}
		if err = s.recover(p.document); err != nil {
			return fmt.Errorf("configuration saved; metadata finalization pending, retry to recover: %w", err)
		}
		return nil
	})
}
func addsDNS(p Plan) bool {
	return hash(p.Before["dns"]) != hash(p.After["dns"]) && (p.Command.Action == "save_policy" || p.Command.Action == "save_group" || p.Command.Action == "delete_group" || p.Command.Action == "restore_policy" || p.Command.Action == "accept_policy")
}
func groupByID(state State, id string) (Group, error) {
	if id == "" {
		return Group{}, nil
	}
	for _, g := range state.Groups {
		if g.ID == id {
			return g, nil
		}
	}
	return Group{}, fmt.Errorf("DNS group not found")
}
func build(raw []byte, state State, c Command) (Plan, error) {
	d, err := decode(raw)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Revision: revision(d, state), Before: ruleView(d), Affected: []string{}}
	// Before/after must be independent; mutation helpers reuse section maps.
	var before Rule
	_ = json.Unmarshal(encoded(p.Before), &before)
	p.Before = before
	state = cloneState(state)
	findPolicy := func(id string) int {
		for i, x := range state.Policies {
			if x.ID == id {
				return i
			}
		}
		return -1
	}
	rebuild := func(index int) error {
		old := state.Policies[index]
		r := locate(rules(d, "route"), old.Route)
		n := locate(rules(d, "dns"), old.DNS)
		if err := removeOwned(d, old); err != nil {
			return err
		}
		g, err := groupByID(state, old.GroupID)
		if err != nil {
			return err
		}
		if old.GroupID != "" {
			if err = validateGroup(&g, d, state); err != nil {
				return err
			}
		}
		old.RouteIndex = r
		old.DNSIndex = n
		if err = generate(&old, g, d); err != nil {
			return err
		}
		if err = insert(d, old); err != nil {
			return err
		}
		state.Policies[index] = old
		p.Affected = append(p.Affected, old.Name)
		return nil
	}
	switch c.Action {
	case "save_group":
		if c.Group == nil {
			return p, fmt.Errorf("group required")
		}
		g := *c.Group
		index := -1
		for i, x := range state.Groups {
			if x.ID == g.ID {
				index = i
			}
		}
		if g.ID == "" {
			g.ID, err = newID()
			if err != nil {
				return p, err
			}
		} else if index < 0 && !validID(g.ID) {
			return p, fmt.Errorf("invalid group ID")
		}
		if err = validateGroup(&g, d, state); err != nil {
			return p, err
		}
		if index < 0 {
			state.Groups = append(state.Groups, g)
		} else {
			state.Groups[index] = g
			for i, x := range state.Policies {
				if x.GroupID == g.ID {
					if err = rebuild(i); err != nil {
						return p, err
					}
				}
			}
		}
		c.Group = &g
	case "delete_group", "detach_group":
		index := -1
		for i, g := range state.Groups {
			if g.ID == c.ID {
				index = i
			}
		}
		if index < 0 {
			return p, fmt.Errorf("group not found")
		}
		if c.Action == "detach_group" {
			kept := []Policy{}
			for _, x := range state.Policies {
				if x.GroupID == c.ID {
					p.Affected = append(p.Affected, x.Name)
				} else {
					kept = append(kept, x)
				}
			}
			state.Policies = kept
			state.Groups = append(state.Groups[:index], state.Groups[index+1:]...)
			break
		}
		for i, x := range state.Policies {
			if x.GroupID != c.ID {
				continue
			}
			if c.Replacement == "" || c.Replacement == c.ID {
				return p, fmt.Errorf("group is used by %s; choose a replacement first", x.Name)
			}
			if _, err = groupByID(state, c.Replacement); err != nil {
				return p, err
			}
			state.Policies[i].GroupID = c.Replacement
			if err = rebuild(i); err != nil {
				return p, err
			}
		}
		state.Groups = append(state.Groups[:index], state.Groups[index+1:]...)
	case "save_policy":
		if c.Policy == nil {
			return p, fmt.Errorf("policy required")
		}
		next := *c.Policy
		index := findPolicy(next.ID)
		if next.ID == "" {
			next.ID, err = newID()
			if err != nil {
				return p, err
			}
			next.Prefix = "dns_" + next.ID
		} else if index < 0 {
			if !validID(next.ID) {
				return p, fmt.Errorf("invalid policy ID")
			}
			next.Prefix = "dns_" + next.ID
		} else {
			old := state.Policies[index]
			next.Prefix = old.Prefix
			if err = removeOwned(d, old); err != nil {
				return p, err
			}
		}
		g, err := groupByID(state, next.GroupID)
		if err != nil {
			return p, err
		}
		if next.GroupID != "" {
			if err = validateGroup(&g, d, state); err != nil {
				return p, err
			}
		}
		if err = generate(&next, g, d); err != nil {
			return p, err
		}
		for _, r := range rules(d, "dns") {
			if tag, ok := r["tag"].(string); ok && strings.HasPrefix(tag, next.Prefix+"_") {
				return p, fmt.Errorf("generated DNS tag collision; preview again")
			}
		}
		if err = insert(d, next); err != nil {
			return p, err
		}
		// Reject indistinguishable owned blocks rather than risking another rule.
		if policyStatus(next, d) != "synced" {
			return p, fmt.Errorf("an identical rule already exists; manage it manually rather than duplicating ownership")
		}
		if index < 0 {
			state.Policies = append(state.Policies, next)
		} else {
			state.Policies[index] = next
		}
		c.Policy = &next
		p.Affected = append(p.Affected, next.Name)
	case "delete_policy", "detach_policy", "restore_policy", "accept_policy":
		index := findPolicy(c.ID)
		if index < 0 {
			return p, fmt.Errorf("policy not found")
		}
		old := state.Policies[index]
		p.Affected = append(p.Affected, old.Name)
		if c.Action == "delete_policy" {
			if err = removeOwned(d, old); err != nil {
				return p, err
			}
		}
		if c.Action == "restore_policy" {
			if policyStatus(old, d) != "missing" {
				return p, fmt.Errorf("only completely missing policies can be restored automatically; review modified rules or stop managing first")
			}
			// A changed route can lose all identity. Restoration never deletes it;
			// user reviews the explicit additions and insertion positions.
			g, err := groupByID(state, old.GroupID)
			if err != nil {
				return p, err
			}
			if old.GroupID != "" {
				if err = validateGroup(&g, d, state); err != nil {
					return p, err
				}
			}
			if err = generate(&old, g, d); err != nil {
				return p, err
			}
			if err = insert(d, old); err != nil {
				return p, err
			}
			state.Policies[index] = old
		} else if c.Action == "accept_policy" {
			next, group, err := acceptSettings(old, state, d)
			if err != nil {
				return p, err
			}
			for i, g := range state.Groups {
				if g.ID == group.ID {
					state.Groups[i] = group
				}
			}
			state.Policies[index] = next
			for i, other := range state.Policies {
				if i != index && other.GroupID == group.ID {
					if err = rebuild(i); err != nil {
						return p, err
					}
				}
			}

		} else {
			state.Policies = append(state.Policies[:index], state.Policies[index+1:]...)
		}
	default:
		return p, fmt.Errorf("unknown managed policy action")
	}
	p.Command = c
	p.Command.Revision = p.Revision
	p.After = ruleView(d)
	p.state = state
	p.document = encoded(d)
	return p, nil
}

// Impact compares the candidate raw edit against current managed ownership.
// It is read-only; reconciliation is derived from the next snapshot, including
// after rollbacks and changes made outside the panel.
func (s *Service) Impact(ctx context.Context, raw []byte) ([]Policy, error) {
	candidate, err := decode(raw)
	if err != nil {
		return nil, err
	}
	result := []Policy{}
	err = s.manager.WithDocumentLock(ctx, func(doc config.ConfigDocument, _ func(config.ConfigDocument) error) error {
		state, err := s.load(doc.Raw)
		if err != nil {
			return err
		}
		current, _ := decode(doc.Raw)
		for _, p := range state.Policies {
			status := reportedStatus(p, state, candidate)
			if status != reportedStatus(p, state, current) {
				p.Status = status
				result = append(result, p)
			}
		}
		return nil
	})
	return result, err
}

func validID(id string) bool { return regexp.MustCompile(`^[A-Za-z0-9_-]{12}$`).MatchString(id) }

// SaveRaw prevents a raw edit from hiding managed-policy impact. The revision
// returned by ImpactPlan is the explicit acknowledgment of that exact snapshot.
func (s *Service) SaveRaw(ctx context.Context, raw []byte, expected string) error {
	return s.manager.WithDocumentLock(ctx, func(doc config.ConfigDocument, save func(config.ConfigDocument) error) error {
		state, err := s.load(doc.Raw)
		if err != nil {
			return err
		}
		current, err := decode(doc.Raw)
		if err != nil {
			return err
		}
		candidate, err := decode(raw)
		if err != nil {
			return err
		}
		if expected != "" && expected != revision(current, state) {
			return fmt.Errorf("configuration changed; review the raw edit again")
		}
		for _, p := range state.Policies {
			if reportedStatus(p, state, current) != reportedStatus(p, state, candidate) && expected == "" {
				return fmt.Errorf("raw edit affects managed traffic policies; review its impact before saving")
			}
		}
		return save(config.ConfigDocument{Raw: raw})
	})
}
func (s *Service) ImpactPlan(ctx context.Context, raw []byte) (map[string]any, error) {
	var out map[string]any
	candidate, err := decode(raw)
	if err != nil {
		return nil, err
	}
	err = s.manager.WithDocumentLock(ctx, func(doc config.ConfigDocument, _ func(config.ConfigDocument) error) error {
		state, err := s.load(doc.Raw)
		if err != nil {
			return err
		}
		current, _ := decode(doc.Raw)
		affected := []Policy{}
		for _, p := range state.Policies {
			status := reportedStatus(p, state, candidate)
			if status != reportedStatus(p, state, current) {
				p.Status = status
				affected = append(affected, p)
			}
		}
		out = map[string]any{"revision": revision(current, state), "policies": affected}
		return nil
	})
	return out, err
}
