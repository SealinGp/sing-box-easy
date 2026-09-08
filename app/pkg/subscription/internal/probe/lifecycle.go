package subprobe

import "context"

type targetRun struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// admitTarget also checks blocked targets from a sweep's stale target snapshot.
func (r *Runner) admitTarget(ctx context.Context, id string) (context.Context, func(), error) {
	r.targetsMu.Lock()
	defer r.targetsMu.Unlock()
	if r.blocked[id] {
		return ctx, nil, ErrNoSuchTarget
	}
	if r.targetRuns == nil {
		r.targetRuns = make(map[string]*targetRun)
	}
	ctx, cancel := context.WithCancel(ctx)
	run := &targetRun{cancel: cancel, done: make(chan struct{})}
	r.targetRuns[id] = run
	return ctx, func() { r.targetsMu.Lock(); delete(r.targetRuns, id); cancel(); close(run.done); r.targetsMu.Unlock() }, nil
}

// BeginDelete excludes new measurements and drains any publisher before history deletion.
func (r *Runner) BeginDelete(id string) {
	r.targetsMu.Lock()
	if r.blocked == nil {
		r.blocked = make(map[string]bool)
	}
	r.blocked[id] = true
	run := r.targetRuns[id]
	if run != nil {
		run.cancel()
	}
	r.targetsMu.Unlock()
	if run != nil {
		<-run.done
	}
	r.mu.Lock()
	delete(r.snapshots, id)
	r.mu.Unlock()
}
func (r *Runner) Allow(id string) { r.targetsMu.Lock(); delete(r.blocked, id); r.targetsMu.Unlock() }
