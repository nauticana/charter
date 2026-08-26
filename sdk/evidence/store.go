// Package evidence appends action records, evidence records, exceptions, escalations, and bundles without ever
// replacing history, and links corrections through supersession and bundles through verifiable integrity.
package evidence

import (
	"context"
	"errors"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrDuplicate    = errors.New("evidence already exists; evidence is append-only")
	ErrNoStore      = errors.New("no evidence store")
	ErrSupersedes   = errors.New("superseded record must be an existing evidence record")
	ErrBundleRecord = errors.New("bundle references a record that is not stored")
	ErrIntegrity    = errors.New("bundle integrity does not match its records")
)

// Store is the abstract storage behind a sink: it accepts new documents and serves them back as a corpus.Source.
type Store interface {
	corpus.Source
	Append(ctx context.Context, d *model.Document) error
}
