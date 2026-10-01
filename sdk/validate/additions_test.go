package validate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func position(t *testing.T, id, unit, spec string) *model.Document {
	t.Helper()
	d, err := (corpus.Parser{}).Parse([]byte(`{"charterSpecVersion":"` + spec + `","namespace":"harbor.example","kind":"Position","id":"` + id +
		`","enterpriseId":"ENT-HARBOR","name":"Planner","organizationUnitId":"` + unit + `","lifecycleState":"active","validity":{"from":"2024-01-01"},"positionTypeId":"PT-FUNCTIONAL-MANAGER"}`))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func check(t *testing.T, a *Additions, source corpus.Source, docs ...*model.Document) []Finding {
	t.Helper()
	f, err := a.Validate(context.Background(), source, docs...)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// countingSource records every key fetched from the corpus it wraps.
type countingSource struct {
	*corpus.Corpus
	fetched map[corpus.DocumentKey]int
	err     error
}

func (s *countingSource) Fetch(ctx context.Context, key corpus.DocumentKey) (*model.Document, error) {
	s.fetched[key]++
	if s.err != nil {
		return nil, s.err
	}
	return s.Corpus.Fetch(ctx, key)
}

func (s *countingSource) List(context.Context, model.Kind) ([]*model.Document, error) {
	return nil, errors.New("additions must not list the source")
}

func TestAdditionsFetchOnlyWhatTheWriteReferences(t *testing.T) {
	base, err := corpus.NewDirLoader(harbor).Load()
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewAdditions()
	if err != nil {
		t.Fatal(err)
	}
	source := &countingSource{Corpus: base, fetched: map[corpus.DocumentKey]int{}}
	if f := check(t, a, source, position(t, "POS-NEW", "OU-CREDIT-CONTROL", "1.2.0")); len(f) != 0 {
		t.Fatalf("valid addition: %+v", f)
	}
	want := []string{"ENT-HARBOR", "OU-CREDIT-CONTROL", "PT-FUNCTIONAL-MANAGER"}
	if len(source.fetched) != len(want) {
		t.Fatalf("fetched %v, want only %v of %d documents", source.fetched, want, base.Len())
	}
	for _, id := range want {
		if source.fetched[corpus.DocumentKey{Namespace: "harbor.example", ID: id}] != 1 {
			t.Errorf("%s fetched %d times", id, source.fetched[corpus.DocumentKey{Namespace: "harbor.example", ID: id}])
		}
	}
	source.err = errors.New("store unavailable")
	if f, err := a.Validate(context.Background(), source, position(t, "POS-NEW", "OU-CREDIT-CONTROL", "1.2.0")); err == nil || f != nil {
		t.Fatalf("a failing source must be an error, not a dangling reference: %+v %v", f, err)
	}
}

func TestAdditionsJudgeOnlyTheDocumentsWritten(t *testing.T) {
	base, err := corpus.NewDirLoader(harbor).Load()
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewAdditions()
	if err != nil {
		t.Fatal(err)
	}
	if f := check(t, a, base, position(t, "POS-NEW", "OU-CREDIT-CONTROL", "1.2.0")); len(f) != 0 {
		t.Fatalf("valid addition: %+v", f)
	}
	if f := check(t, a, nil, position(t, "POS-NEW", "OU-CREDIT-CONTROL", "1.2.0")); len(f) == 0 || f[0].RuleID != "CHR-RULE-CONF-001" {
		t.Fatalf("references must resolve against the corpus written into: %+v", f)
	}
	dangling := check(t, a, base, position(t, "POS-NEW", "OU-NONE", "1.2.0"))
	if len(dangling) != 1 || dangling[0].RuleID != "CHR-RULE-CONF-001" || dangling[0].DocumentID != "POS-NEW" {
		t.Fatalf("dangling reference: %+v", dangling)
	}
	if f := check(t, a, base, position(t, "POS-NEW", "ENT-HARBOR", "1.2.0")); len(f) != 1 || f[0].RuleID != "CHR-RULE-CONF-002" {
		t.Fatalf("wrong reference kind: %+v", f)
	}
	if f := check(t, a, base, position(t, "POS-NEW", "OU-CREDIT-CONTROL", "1.0.0")); len(f) != 1 || f[0].RuleID != "spec-version" {
		t.Fatalf("stale specification version: %+v", f)
	}
	replaced := position(t, "POS-CREDIT-MANAGER", "OU-NONE", "1.2.0")
	if f := check(t, a, base, replaced); len(f) != 1 || f[0].DocumentID != "POS-CREDIT-MANAGER" {
		t.Fatalf("a replacement is judged in place of the stored document: %+v", f)
	}
	unit, err := (corpus.Parser{}).Parse([]byte(`{"charterSpecVersion":"1.2.0","namespace":"harbor.example","kind":"OrganizationUnit","id":"OU-NEW","enterpriseId":"ENT-HARBOR","name":"New","lifecycleState":"active","parentUnitId":"OU-FINANCE"}`))
	if err != nil {
		t.Fatal(err)
	}
	if f := check(t, a, base, position(t, "POS-NEW", "OU-NEW", "1.2.0"), unit); len(f) != 0 {
		t.Fatalf("documents written together resolve one another: %+v", f)
	}
	twice := check(t, a, base, unit, unit)
	if len(twice) != 1 || !strings.Contains(twice[0].Message, "twice") {
		t.Fatalf("duplicate key in one write: %+v", twice)
	}
	invalid := check(t, a, base, &model.Document{Envelope: model.Envelope{CharterSpecVersion: "1.2.0", Namespace: "harbor.example", ID: "X", Kind: "Unknown"}, Value: map[string]any{}})
	if len(invalid) == 0 || invalid[0].Class != ClassStructural {
		t.Fatalf("unknown kind: %+v", invalid)
	}
}
