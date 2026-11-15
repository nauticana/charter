// Package model holds schema-aligned document, reference, and value structs without behavior.
package model

import "encoding/json"

// Document is any Charter document: its envelope, raw bytes, and decoded value.
type Document struct {
	Envelope
	Raw   []byte
	Value any
}

// MarshalJSON writes the document itself: its raw bytes, or its value when it has none.
func (d Document) MarshalJSON() ([]byte, error) {
	if len(d.Raw) > 0 {
		return json.Marshal(json.RawMessage(d.Raw))
	}
	return json.Marshal(d.Value)
}
