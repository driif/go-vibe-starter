// Package apperr is the error the domain returns to callers. It knows no transport: the error
// interceptor maps Kind to a Connect code.
package apperr

// Kind classifies an Error. The zero value is Internal.
type Kind int

// The kinds an Error can have.
const (
	Internal Kind = iota
	InvalidArgument
	NotFound
	AlreadyExists
	PermissionDenied
	FailedPrecondition
)

// Error is a failure the client is told about.
type Error struct {
	Kind Kind
	// Reason is stable and UPPER_SNAKE, e.g. "NOTE_NOT_FOUND". Clients switch on it.
	Reason string
	// Message is for developers and is sent to the client.
	Message string
}

func (e *Error) Error() string { return e.Reason + ": " + e.Message }

// New returns an Error with the given kind, reason and message.
func New(kind Kind, reason, message string) *Error {
	return &Error{Kind: kind, Reason: reason, Message: message}
}
