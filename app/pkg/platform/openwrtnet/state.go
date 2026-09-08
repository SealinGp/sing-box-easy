package openwrtnet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// state records what Apply changed, so Revert can put the host back.
//
// It lives in a file rather than memory because the panel and sing-box have
// independent lifetimes: restarting sing-box-easy while sing-box keeps running
// must not lose the knowledge of what to undo later.
type state struct {
	// ZoneCreated means the firewall zone is ours to remove.
	ZoneCreated bool `json:"zone_created"`
	// DNSChanged means dnsmasq's upstream was replaced.
	DNSChanged bool `json:"dns_changed"`
	// PriorServers is the dnsmasq upstream list from before the change. Empty
	// is meaningful: it means no upstream was configured and Revert should
	// clear the option rather than write one.
	PriorServers []string `json:"prior_servers,omitempty"`
	// RebindDomains are the rebind-protection exemptions this package added.
	// Recording the names we wrote — rather than re-deriving them from the
	// config at revert time — is what keeps a later config edit from stranding
	// an exemption, or from deleting one the operator added by hand.
	RebindDomains []string `json:"rebind_domains,omitempty"`
}

func (m *Manager) loadState() (state, error) {
	var st state
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return st, fmt.Errorf("failed to read the network state file: %w", err)
	}
	if err := json.Unmarshal(data, &st); err != nil {
		// A corrupt state file must not wedge the service permanently. Treat
		// it as "nothing recorded" — the worst case is a leftover zone the
		// operator can delete, rather than a start that always fails.
		return state{}, nil
	}
	return st, nil
}

func (m *Manager) saveState(st state) error {
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to encode the network state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0o755); err != nil {
		return fmt.Errorf("failed to create the network state directory: %w", err)
	}
	if err := os.WriteFile(m.statePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write the network state file: %w", err)
	}
	return nil
}

func (m *Manager) clearState() error {
	if err := os.Remove(m.statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear the network state file: %w", err)
	}
	return nil
}
