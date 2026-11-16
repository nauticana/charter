package corpus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/nauticana/charter/sdk/model"
)

func parsed(t *testing.T, raw string) *model.Document {
	t.Helper()
	d, err := (Parser{}).Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestContentDigestCanonicalizesAsRFC8785(t *testing.T) {
	// The example of RFC 8785 section 3.2.2, wrapped as a document.
	d := parsed(t, `{"id":"X","kind":"Task","numbers":[333333333.33333329,1E30,4.50,2e-3,0.000000000000000000000000001],`+
		`"string":"\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/","literals":[null,true,false]}`)
	want := `{"id":"X","kind":"Task","literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],` +
		`"string":"€$\u000f\nA'B\"\\\\\"/"}`
	sum := sha256.Sum256([]byte(want))
	if got, err := ContentDigest(d); err != nil || got != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Fatalf("digest = %s, %v", got, err)
	}
}

func TestContentDigestIgnoresSerialization(t *testing.T) {
	a, err := ContentDigest(parsed(t, `{"id":"X","kind":"Task","n":1.0,"m":{"b":-0,"a":"<&>"}}`))
	if err != nil {
		t.Fatal(err)
	}
	stored := &model.Document{Envelope: model.Envelope{ID: "X"}, Value: map[string]any{"m": map[string]any{"a": "<&>", "b": 0}, "n": 1, "kind": "Task", "id": "X"}}
	if b, err := ContentDigest(stored); err != nil || b != a {
		t.Fatalf("value form %s, raw form %s, %v", b, a, err)
	}
	if c, _ := ContentDigest(parsed(t, `{"id":"X","kind":"Task","n":2}`)); c == a {
		t.Fatal("different content digests alike")
	}
	if _, err := ContentDigest(parsed(t, `{"id":"X","kind":"Task","n":1e400}`)); err == nil {
		t.Fatal("a number beyond a double digests")
	}
}

func TestCorpusFetchesTheVersionItHolds(t *testing.T) {
	c := New()
	task := parsed(t, `{"namespace":"a.example","id":"T","kind":"Task","taskDefinitionVersion":"2","lifecycleState":"active"}`)
	if err := c.Add(task); err != nil {
		t.Fatal(err)
	}
	key := DocumentKey{Namespace: "a.example", ID: "T"}
	for _, label := range []string{"", "2"} {
		if d, err := c.FetchVersion(context.Background(), key, label); err != nil || d != task {
			t.Fatalf("label %q: %v", label, err)
		}
	}
	if _, err := c.FetchVersion(context.Background(), key, "1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another version: %v", err)
	}
	if _, err := c.FetchVersion(context.Background(), DocumentKey{Namespace: "a.example", ID: "U"}, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown document: %v", err)
	}
}

func TestPublishedIsPastDrafting(t *testing.T) {
	for state, want := range map[string]bool{"proposed": false, "planned": false, "active": true, "retired": true, "": true} {
		if got := Published(&model.Document{Envelope: model.Envelope{LifecycleState: state}}); got != want {
			t.Errorf("%q: %t", state, got)
		}
	}
}
