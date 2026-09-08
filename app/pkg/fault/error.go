// Package fault defines operation failures independent of any transport.
package fault

type Kind string

const (
	Input         Kind = "input"
	Internal      Kind = "internal"
	Configuration Kind = "configuration"
	Invalid       Kind = "invalid"
	Missing       Kind = "missing"
	Conflict      Kind = "conflict"
	Forbidden     Kind = "forbidden"
	Unavailable   Kind = "unavailable"
	Failed        Kind = "failed"
)

type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string            { return e.Message }
func (e *Error) Unwrap() error            { return e.Cause }
func New(kind Kind, message string) error { return &Error{Kind: kind, Message: message} }
