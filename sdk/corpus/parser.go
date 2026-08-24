package corpus

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/nauticana/charter/sdk/model"
)

// Parser turns JSON bytes into documents with both envelope and decoded value.
type Parser struct{}

func (Parser) Parse(b []byte) (*model.Document, error) {
	d := &model.Document{Raw: b}
	if err := json.Unmarshal(b, &d.Envelope); err != nil {
		return nil, err
	}
	if d.ID == "" || d.Kind == "" {
		return nil, fmt.Errorf("document lacks id or kind")
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&d.Value); err != nil {
		return nil, err
	}
	return d, nil
}

// Decode unmarshals a document into its typed representation; generic, so it cannot be a method.
func Decode[T any](d *model.Document) (T, error) {
	var t T
	err := json.Unmarshal(d.Raw, &t)
	return t, err
}
