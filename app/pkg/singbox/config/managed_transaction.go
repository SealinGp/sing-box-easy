package config

import (
	"context"
	"os"
)

// SetWriteBarrier installs the managed-operation recovery check at startup.
// The callback runs under mutationMu and must not re-enter Manager methods.
func (m *Manager) SetWriteBarrier(check func([]byte) error) { m.writeBarrier = check }

// WithDocumentLock shares the same lock as raw edits, rollbacks and section
// mutations. A nil save is never needed: callbacks may simply return to read.
func (m *Manager) WithDocumentLock(ctx context.Context, fn func(ConfigDocument, func(ConfigDocument) error) error) error {
	m.mutationMu.Lock()
	defer m.mutationMu.Unlock()
	raw, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}
	doc, err := NewConfigDocument(raw)
	if err != nil {
		return err
	}
	return fn(doc, func(next ConfigDocument) error { return m.saveDocumentLocked(ctx, next) })
}
