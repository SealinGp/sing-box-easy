package trafficpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
)

var ErrRecovery = errors.New("an interrupted managed operation conflicts with manual configuration changes; choose which metadata to retain in Traffic policies")

func isRecovery(c Command) bool { return strings.HasPrefix(c.Action, "recover_") }

// Explicit recovery never overwrites manual config edits. The operator chooses
// which metadata generation to retain, then reviews the resulting drift.
func (s *Service) recoveryPlan(raw []byte, c Command) (Plan, error) {
	r, err := s.record()
	if err != nil {
		return Plan{}, err
	}
	if r.Pending == "" {
		return Plan{}, fmt.Errorf("no pending operation")
	}
	d, err := decode(raw)
	if err != nil {
		return Plan{}, err
	}
	stateJSON := r.State
	if c.Action == "recover_accept" {
		stateJSON = r.Pending
	} else if c.Action != "recover_discard" {
		return Plan{}, fmt.Errorf("unknown recovery action")
	}
	var state State
	if err = json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return Plan{}, err
	}
	rev := hash([]any{d, r})
	c.Revision = rev
	names := []string{}
	for _, p := range state.Policies {
		names = append(names, p.Name)
	}
	return Plan{Revision: rev, Command: c, Before: ruleView(d), After: ruleView(d), Affected: names, state: state, document: raw}, nil
}
func (s *Service) commitRecovery(ctx context.Context, c Command) error {
	return s.manager.WithDocumentLock(ctx, func(doc config.ConfigDocument, _ func(config.ConfigDocument) error) error {
		p, err := s.recoveryPlan(doc.Raw, c)
		if err != nil {
			return err
		}
		if c.Revision != p.Revision {
			return fmt.Errorf("recovery state changed; preview again")
		}
		r, err := s.record()
		if err != nil {
			return err
		}
		r.State = string(encoded(p.state))
		r.Pending = ""
		r.BeforeHash = ""
		r.AfterHash = ""
		return s.write(r)
	})
}
