package corpus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"unicode/utf16"

	"github.com/nauticana/charter/sdk/model"
)

// ContentDigest returns the sha256 digest of a document's JSON canonicalized as RFC 8785 specifies, the content digest
// a derivation records (CHR-CONF-015); equal documents digest alike however they were stored or serialized.
func ContentDigest(d *model.Document) (string, error) {
	raw, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("digest %s:%s: %w", d.Namespace, d.ID, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", fmt.Errorf("digest %s:%s: %w", d.Namespace, d.ID, err)
	}
	var b bytes.Buffer
	if err := canonical(&b, v); err != nil {
		return "", fmt.Errorf("digest %s:%s: %w", d.Namespace, d.ID, err)
	}
	sum := sha256.Sum256(b.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func canonical(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case json.Number:
		f, err := x.Float64()
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return fmt.Errorf("number %s is not a finite IEEE 754 double", x)
		}
		if f == 0 {
			f = 0 // RFC 8785 writes negative zero as 0.
		}
		// encoding/json formats a float64 as ECMAScript does, the form RFC 8785 requires.
		out, err := json.Marshal(f)
		if err != nil {
			return err
		}
		b.Write(out)
	case string:
		canonicalString(b, x)
	case []any:
		b.WriteByte('[')
		for i, item := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := canonical(b, item); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		// RFC 8785 orders members by the UTF-16 code units of their names.
		slices.SortFunc(keys, func(a, c string) int { return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(c))) })
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			canonicalString(b, k)
			b.WriteByte(':')
			if err := canonical(b, x[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		return fmt.Errorf("unsupported JSON value %T", v)
	}
	return nil
}

// canonicalString escapes only the quote, the backslash, and control characters, as RFC 8785 requires.
func canonicalString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}
