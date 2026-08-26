package corpus

import "errors"

var (
	ErrNotFound     = errors.New("document not found")
	ErrKindMismatch = errors.New("document kind mismatch")
	ErrExternal     = errors.New("external reference does not resolve to a Charter document")
	ErrNoSource     = errors.New("no document source")
)
