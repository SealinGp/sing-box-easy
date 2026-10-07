package traffic

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

// ConnectionSource is the slice of the Clash API the connections table reads.
// It is narrower than Source on purpose: the table reports each connection's
// rule STRING as sing-box wrote it, so it has no use for `/rules`.
type ConnectionSource interface {
	Connections(ctx context.Context) (*clashapi.Snapshot, error)
}

// ConnectionRow is one live connection as the Connections page shows it.
//
// It is the raw connection plus the one thing the payload lacks — a rate — and
// with the one thing the payload gets backwards put right: Chains here runs
// EXIT FIRST (the outbound the rule named), ending at the leaf actually
// dialled. sing-box serialises it leaf first; correcting it once here means no
// client has to remember to.
type ConnectionRow struct {
	ID              string   `json:"id"`
	Network         string   `json:"network"`
	Inbound         string   `json:"inbound"`
	SourceIP        string   `json:"source_ip"`
	SourcePort      string   `json:"source_port"`
	DestinationIP   string   `json:"destination_ip"`
	DestinationPort string   `json:"destination_port"`
	Host            string   `json:"host"`
	ProcessPath     string   `json:"process_path"`
	Rule            string   `json:"rule"`
	Chains          []string `json:"chains"`
	// Start is when the connection opened, unix milliseconds.
	Start    int64 `json:"start"`
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`
	// UpRate and DownRate are bytes per second since the previous frame. Both
	// are 0 on a connection's first frame — see Fresh.
	UpRate   float64 `json:"up_rate"`
	DownRate float64 `json:"down_rate"`
	// Fresh marks a first sighting: no baseline yet, so a zero rate means
	// "unknown", not "idle".
	Fresh bool `json:"fresh"`
}

// ConnectionTotals are the frame-wide figures above the table.
type ConnectionTotals struct {
	Connections int     `json:"connections"`
	DownRate    float64 `json:"down_rate"`
	UpRate      float64 `json:"up_rate"`
	// DownloadTotal and UploadTotal are sing-box's lifetime counters, which
	// include connections that have already closed.
	DownloadTotal int64  `json:"download_total"`
	UploadTotal   int64  `json:"upload_total"`
	Memory        uint64 `json:"memory"`
}

// ConnectionsFrame is one tick of `GET /runtime/connections/stream`.
type ConnectionsFrame struct {
	// At is the sample time, unix milliseconds.
	At          int64            `json:"at"`
	Totals      ConnectionTotals `json:"totals"`
	Connections []ConnectionRow  `json:"connections"`
}

// BuildConnectionsFrame shapes one differ step into a frame.
//
// Rows are ordered by id so two frames of an unchanged table are byte-for-byte
// comparable; the page sorts by whatever column the operator picked.
func BuildConnectionsFrame(at time.Time, snapshot *clashapi.Snapshot, live []Live) *ConnectionsFrame {
	rows := make([]ConnectionRow, 0, len(live))
	totals := ConnectionTotals{
		Connections:   len(live),
		DownloadTotal: snapshot.DownloadTotal,
		UploadTotal:   snapshot.UploadTotal,
		Memory:        snapshot.Memory,
	}
	for _, conn := range live {
		totals.DownRate += conn.DownRate
		totals.UpRate += conn.UpRate
		rows = append(rows, ConnectionRow{
			ID:              conn.ID,
			Network:         conn.Metadata.Network,
			Inbound:         conn.Metadata.Type,
			SourceIP:        conn.Metadata.SourceIP,
			SourcePort:      conn.Metadata.SourcePort,
			DestinationIP:   conn.Metadata.DestinationIP,
			DestinationPort: conn.Metadata.DestinationPort,
			Host:            conn.Metadata.Host,
			ProcessPath:     conn.Metadata.ProcessPath,
			Rule:            conn.Rule,
			Chains:          exitFirst(conn.Chains),
			Start:           startMillis(conn.Start),
			Upload:          conn.Upload,
			Download:        conn.Download,
			UpRate:          conn.UpRate,
			DownRate:        conn.DownRate,
			Fresh:           conn.Fresh,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return &ConnectionsFrame{At: at.UnixMilli(), Totals: totals, Connections: rows}
}

// Summary returns the frame's totals without its rows.
//
// The Overview card charts three numbers a second and has no use for the
// table; sending it every connection as well — roughly 400 bytes each, on a
// router with hundreds — would spend most of the stream on data that is
// dropped on arrival. Connections is an empty slice, not nil, so the client
// decodes one shape either way.
func (f *ConnectionsFrame) Summary() *ConnectionsFrame {
	return &ConnectionsFrame{At: f.At, Totals: f.Totals, Connections: []ConnectionRow{}}
}

// exitFirst returns a reversed copy; the input belongs to the snapshot.
func exitFirst(chains []string) []string {
	reversed := make([]string, len(chains))
	for i, name := range chains {
		reversed[len(chains)-1-i] = name
	}
	return reversed
}

func startMillis(start time.Time) int64 {
	if start.IsZero() {
		return 0
	}
	return start.UnixMilli()
}

// RunConnections samples the source on a ticker and emits a frame per sample
// until the context ends, the source fails, or the emitter returns an error.
// Same contract as Run, including the first frame being emitted immediately.
func RunConnections(ctx context.Context, source ConnectionSource, interval time.Duration, emit func(*ConnectionsFrame) error) error {
	if interval <= 0 {
		interval = defaultInterval
	}
	differ := NewDiffer()

	sample := func() error {
		snapshot, err := source.Connections(ctx)
		if err != nil {
			return err
		}
		now := time.Now()
		live, _ := differ.Step(now, snapshot)
		return emit(BuildConnectionsFrame(now, snapshot, live))
	}

	if err := sample(); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := sample(); err != nil {
				return err
			}
		}
	}
}

// ConnectionStream is a prepared connections stream.
type ConnectionStream struct {
	client *clashapi.Client
}

// PrepareConnections resolves the controller before the response is switched
// to streaming, so a missing clash_api is an ordinary error envelope rather
// than a stream that opens and immediately dies.
func (s *Service) PrepareConnections() (*ConnectionStream, error) {
	client, err := s.controller("the connections page")
	if err != nil {
		return nil, err
	}
	return &ConnectionStream{client: client}, nil
}

// Run streams frames until the context ends.
func (s *ConnectionStream) Run(ctx context.Context, emit func(*ConnectionsFrame) error) error {
	err := RunConnections(ctx, s.client, 0, emit)
	if err == nil || errors.Is(err, context.Canceled) {
		return err
	}
	return unreachable(err)
}

// CloseConnection closes one connection in the running sing-box.
func (s *Service) CloseConnection(ctx context.Context, id string) error {
	if id == "" {
		return fault.New(fault.Input, "connection id is required")
	}
	client, err := s.controller("closing a connection")
	if err != nil {
		return err
	}
	if err := client.CloseConnection(ctx, id); err != nil {
		return unreachable(err)
	}
	return nil
}

// CloseAllConnections closes every connection in the running sing-box.
func (s *Service) CloseAllConnections(ctx context.Context) error {
	client, err := s.controller("closing connections")
	if err != nil {
		return err
	}
	if err := client.CloseAllConnections(ctx); err != nil {
		return unreachable(err)
	}
	return nil
}

func (s *Service) controller(feature string) (*clashapi.Client, error) {
	settings, err := s.config.GetClashAPISettings()
	if err != nil {
		return nil, err
	}
	client, err := clashapi.NewFromValues(settings.ExternalController, settings.Secret)
	if err != nil {
		if errors.Is(err, clashapi.ErrDisabled) {
			return nil, fault.New(fault.Unavailable, feature+" needs experimental.clash_api.external_controller to be set")
		}
		return nil, err
	}
	return client, nil
}

func unreachable(err error) error {
	message := err.Error()
	if !errors.Is(err, clashapi.ErrUnauthorized) {
		message = "sing-box is not reachable: " + message
	}
	return &fault.Error{Kind: fault.Unavailable, Message: message, Cause: err}
}
