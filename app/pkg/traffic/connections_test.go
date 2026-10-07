package traffic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

func TestConnectionsFrameReversesChainsToExitFirst(t *testing.T) {
	leafFirst := []string{"香港 01", "自动选择", "Media"}
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "192.168.1.2", "example.com", "final", leafFirst, 1, 2),
	}}
	live, _ := NewDiffer().Step(time.Unix(100, 0), snapshot)

	row := BuildConnectionsFrame(time.Unix(100, 0), snapshot, live).Connections[0]
	if len(row.Chains) != 3 || row.Chains[0] != "Media" || row.Chains[2] != "香港 01" {
		t.Fatalf("chains = %v, want exit first", row.Chains)
	}
	if leafFirst[0] != "香港 01" {
		t.Fatal("the snapshot's own slice was reversed in place")
	}
}

func TestConnectionsFrameCarriesRatesAndTotals(t *testing.T) {
	differ := NewDiffer()
	first := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "192.168.1.2", "a.com", "final", []string{"direct"}, 100, 1000),
		conn("b", "tun/tun-in", "192.168.1.3", "b.com", "final", []string{"direct"}, 0, 0),
	}}
	live, _ := differ.Step(time.Unix(100, 0), first)
	opening := BuildConnectionsFrame(time.Unix(100, 0), first, live)
	if !opening.Connections[0].Fresh || opening.Totals.DownRate != 0 {
		t.Fatalf("first frame must be all Fresh with no rate: %+v", opening)
	}

	second := &clashapi.Snapshot{
		DownloadTotal: 9000, UploadTotal: 900, Memory: 42,
		Connections: []clashapi.Connection{
			conn("b", "tun/tun-in", "192.168.1.3", "b.com", "final", []string{"direct"}, 50, 500),
			conn("a", "tun/tun-in", "192.168.1.2", "a.com", "final", []string{"direct"}, 300, 5000),
		},
	}
	live, _ = differ.Step(time.Unix(102, 0), second)
	frame := BuildConnectionsFrame(time.Unix(102, 0), second, live)

	if frame.Connections[0].ID != "a" || frame.Connections[1].ID != "b" {
		t.Fatalf("rows must be ordered by id, got %s, %s", frame.Connections[0].ID, frame.Connections[1].ID)
	}
	a := frame.Connections[0]
	// 4000 bytes over TWO seconds: per second, not per tick.
	if a.DownRate != 2000 || a.UpRate != 100 || a.Fresh {
		t.Errorf("a = %+v", a)
	}
	if frame.Totals.DownRate != 2250 || frame.Totals.UpRate != 125 {
		t.Errorf("rate totals = %+v", frame.Totals)
	}
	if frame.Totals.Connections != 2 || frame.Totals.DownloadTotal != 9000 || frame.Totals.Memory != 42 {
		t.Errorf("totals = %+v", frame.Totals)
	}
	if frame.At != time.Unix(102, 0).UnixMilli() {
		t.Errorf("at = %d", frame.At)
	}
}

func TestConnectionsFrameEmptyIsAnEmptyArray(t *testing.T) {
	frame := BuildConnectionsFrame(time.Unix(1, 0), &clashapi.Snapshot{}, nil)
	if frame.Connections == nil || len(frame.Connections) != 0 {
		t.Fatalf("connections = %#v, want an empty non-nil slice", frame.Connections)
	}
}

func TestConnectionsFrameZeroStartStaysZero(t *testing.T) {
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "192.168.1.2", "a.com", "final", nil, 0, 0),
	}}
	live, _ := NewDiffer().Step(time.Unix(1, 0), snapshot)
	if row := BuildConnectionsFrame(time.Unix(1, 0), snapshot, live).Connections[0]; row.Start != 0 || row.Chains == nil {
		t.Errorf("row = %+v", row)
	}
}

func TestConnectionsFrameSummaryKeepsTotalsAndDropsRows(t *testing.T) {
	snapshot := &clashapi.Snapshot{DownloadTotal: 7, Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "192.168.1.2", "a.com", "final", []string{"direct"}, 1, 2),
	}}
	live, _ := NewDiffer().Step(time.Unix(5, 0), snapshot)
	full := BuildConnectionsFrame(time.Unix(5, 0), snapshot, live)

	summary := full.Summary()
	if summary.Totals != full.Totals || summary.At != full.At {
		t.Errorf("summary = %+v, want the frame's own totals and time", summary)
	}
	if summary.Connections == nil || len(summary.Connections) != 0 {
		t.Errorf("connections = %#v, want an empty non-nil slice", summary.Connections)
	}
	if len(full.Connections) != 1 {
		t.Error("Summary must not modify the frame it was taken from")
	}
}

type snapshotSource struct {
	calls int
	err   error
}

func (s *snapshotSource) Connections(context.Context) (*clashapi.Snapshot, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return &clashapi.Snapshot{}, nil
}

func TestRunConnectionsEmitsImmediatelyAndStopsWhenTheClientGoes(t *testing.T) {
	source := &snapshotSource{}
	gone := errors.New("client went away")
	frames := 0
	err := RunConnections(context.Background(), source, time.Millisecond, func(*ConnectionsFrame) error {
		frames++
		if frames == 3 {
			return gone
		}
		return nil
	})
	if !errors.Is(err, gone) || frames != 3 || source.calls != 3 {
		t.Fatalf("err = %v, frames = %d, polls = %d", err, frames, source.calls)
	}
}

func TestRunConnectionsReturnsSourceFailure(t *testing.T) {
	down := errors.New("connection refused")
	err := RunConnections(context.Background(), &snapshotSource{err: down}, time.Millisecond, func(*ConnectionsFrame) error {
		t.Fatal("nothing should be emitted when the first poll fails")
		return nil
	})
	if !errors.Is(err, down) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunConnectionsEndsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	err := RunConnections(ctx, &snapshotSource{}, time.Millisecond, func(*ConnectionsFrame) error {
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
