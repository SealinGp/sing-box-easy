package applog

import (
	"net/url"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// Scheme is the URL scheme the zap sink registers under. `logger.Init` adds
// "applog://ring" to OutputPaths, so the ring fills as a side effect of normal
// logging rather than through a second call at every log site.
const Scheme = "applog"

var (
	defaultRing = New(DefaultCapacity)
	registerNow sync.Once
	registerErr error
)

// Default is the process-wide ring the zap sink writes into.
func Default() *Ring { return defaultRing }

// Register installs the zap sink. Safe to call more than once.
//
// Guarded by a Once because zap.RegisterSink returns an error on a duplicate
// scheme, and Init is called again by the tests and by the config-reload path.
// Re-registering is not an error worth propagating — the sink is already there.
func Register() error {
	registerNow.Do(func() {
		registerErr = zap.RegisterSink(Scheme, func(*url.URL) (zap.Sink, error) {
			return sink{ring: defaultRing}, nil
		})
	})
	return registerErr
}

// sink adapts the ring to zap's io.WriteCloser + Sync contract.
type sink struct{ ring *Ring }

// Write splits an encoded entry into lines and appends them.
//
// zap hands over one entry per call, newline-terminated. Splitting matters for
// the development encoder, which puts a multi-line stack trace in the same
// write — stored whole, one buffer slot would hold a 30-line blob and the
// viewer would render it as a single unbreakable row.
func (s sink) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\n")
	if text == "" {
		return len(p), nil
	}
	s.ring.Append(strings.Split(text, "\n")...)
	return len(p), nil
}

// Sync is a no-op: the ring is memory, there is nothing to flush.
func (sink) Sync() error { return nil }

// Close is a no-op: the ring outlives any single logger built on it, so closing
// the sink must not discard what it holds.
func (sink) Close() error { return nil }
