// Package model holds schema-aligned document, reference, and value structs without behavior.
package model

// Document is any Charter document: its envelope, raw bytes, and decoded value.
type Document struct {
	Envelope
	Raw   []byte
	Value any
}
