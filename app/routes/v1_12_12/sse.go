package v1_13_0

// Server-Sent Events, for the endpoints where the server has something to say
// before the client thinks to ask: the live log tail, and the phases of a DNS
// probe.
//
// WHY SSE AND NOT WEBSOCKETS
// ──────────────────────────
// Both of these flows are one-directional — the server talks, the client
// listens. WebSockets would add a dependency (hertz-contrib/websocket), an
// upgrade handshake, and a framing layer to carry text that is already framed.
// SSE is plain chunked HTTP: it needs nothing that hertz v0.10.3 does not
// already have (protocol/http1/resp.NewChunkedBodyWriter), it reuses the
// existing Bearer auth middleware unchanged, and it reconnects on its own.
//
// WHY NOT EventSource ON THE CLIENT
// ─────────────────────────────────
// The browser's EventSource cannot set request headers, and this API
// authenticates with `Authorization: Bearer <token>` from localStorage. The
// alternative — passing the token as a query parameter — writes a live session
// credential into every access log, proxy log and Referer header on the path.
// The frontend therefore reads these streams with fetch() + ReadableStream,
// which does send headers. See services/stream.ts.
//
// THE ENVELOPE SURVIVES
// ─────────────────────
// CLAUDE.md's rule is that every response carries BasicResponse{code,data,msg}
// at HTTP 200. A stream cannot carry one envelope for the whole response, so
// each EVENT carries its own. Clients branch on `code` exactly as they do for a
// unary call, and an error that occurs mid-stream is reportable rather than
// being an unexplained disconnect.

import (
	"context"
	"fmt"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	hresp "github.com/cloudwego/hertz/pkg/protocol/http1/resp"
	"github.com/sagernet/sing/common/json"
	"go.uber.org/zap"
)

// SSEStream writes SSE frames to one client for the life of a request.
type SSEStream struct {
	c *app.RequestContext
}

// NewSSEStream puts the response into chunked streaming mode and writes the
// headers.
//
// `X-Accel-Buffering: no` is not optional. Any buffering proxy in front of this
// — nginx in the usual self-hosted setup, and LuCI's uhttpd on OpenWrt — will
// otherwise hold events until its buffer fills, which for a quiet log means
// they arrive minutes late or not at all, and looks exactly like the stream
// being broken.
func NewSSEStream(c *app.RequestContext) *SSEStream {
	c.SetStatusCode(consts.StatusOK)
	c.Response.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Response.Header.Set("Cache-Control", "no-cache, no-transform")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")

	c.Response.HijackWriter(hresp.NewChunkedBodyWriter(&c.Response, c.GetWriter()))
	return &SSEStream{c: c}
}

// Event writes one named event carrying the standard envelope.
//
// Returns an error when the client has gone; every caller must treat that as
// "stop working", not as something to log and continue past — a log follower
// that ignores it keeps a journalctl child alive for a browser tab that closed.
func (s *SSEStream) Event(name string, data any) error {
	payload, err := json.Marshal(BasicResponse[any]{Code: CodeSuccess, Data: data, Msg: "success"})
	if err != nil {
		return fmt.Errorf("marshal sse payload: %w", err)
	}
	return s.write(name, payload)
}

// Error writes a terminal error event. The stream should be closed after it.
func (s *SSEStream) Error(code Code, msg string) error {
	payload, err := json.Marshal(BasicResponse[any]{Code: code, Data: nil, Msg: msg})
	if err != nil {
		return fmt.Errorf("marshal sse error: %w", err)
	}
	return s.write("error", payload)
}

// Comment writes an SSE comment frame — a keep-alive that carries no data.
//
// Needed because a silent stream is indistinguishable from a dead one: an idle
// proxy will drop a connection that has sent nothing for its timeout window,
// and sing-box at `level: info` can easily log nothing for minutes.
func (s *SSEStream) Comment() error {
	return s.raw(": keep-alive\n\n")
}

func (s *SSEStream) write(name string, payload []byte) error {
	return s.raw(formatSSEFrame(name, payload))
}

// formatSSEFrame builds `event: <name>` + one `data:` line per line of payload.
//
// The per-line split is the part that must not be simplified away. A payload
// containing a raw newline — which happens whenever sing-box logs a multi-line
// message, since the line is carried inside a JSON string — would otherwise
// terminate the frame early, and the client would parse a truncated event and
// silently drop it.
//
// Kept pure so the framing can be tested without an HTTP connection.
func formatSSEFrame(name string, payload []byte) string {
	var b strings.Builder
	b.WriteString("event: ")
	b.WriteString(name)
	b.WriteString("\n")
	for _, line := range strings.Split(string(payload), "\n") {
		b.WriteString("data: ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (s *SSEStream) raw(frame string) error {
	if _, err := s.c.Write([]byte(frame)); err != nil {
		return err
	}
	return s.c.Flush()
}

// Done reports whether the request context has been cancelled — the client
// disconnected, or the server is shutting down.
func Done(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// logStreamEnd records why a stream finished, at debug level.
//
// A client closing a tab is the ordinary way for these to end, so it must not
// look like a fault in the logs — but a stream that ends for another reason is
// worth being able to find.
func logStreamEnd(name string, err error) {
	if err == nil {
		return
	}
	logger.L.Debug("SSE stream ended", zap.String("stream", name), zap.Error(err))
}
