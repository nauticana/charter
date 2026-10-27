package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// The standing of a fact at a decision time; only a current fact may be relied on.
const (
	FactCurrent    = "current"
	FactStale      = "stale"
	FactSuperseded = "superseded"
	FactOutranked  = "outranked"
	FactConflicted = "conflicted"
)

// FactState is a fact record with its standing at a decision time and the rule that decided it.
type FactState struct {
	Record model.EvidenceRecord
	State  string
	Rule   string
}

// CurrentFacts judges, per attribute, every fact about subject and def's information recorded and valid at or before
// at. A superseded fact or one older than def's maxAge never counts; among the rest only the highest-ranked source
// counts, and a disagreement at that rank is a conflict, never settled by recency (CHR-EVID-014, CHR-EVID-015).
func (q Queries) CurrentFacts(ctx context.Context, ownerNamespace string, subject model.ObjectRef, def model.InformationDefinition, at time.Time) ([]FactState, error) {
	all, err := q.Provider.Records(ctx)
	if err != nil {
		return nil, err
	}
	about, info := corpus.ObjectKeyOf(ownerNamespace, subject), corpus.DocumentKey{Namespace: def.Namespace, ID: def.ID}
	concerns := func(r model.EvidenceRecord) bool {
		f := r.Fact
		return f.Subject.Kind == subject.Kind && corpus.ObjectKeyOf(r.Namespace, f.Subject) == about && corpus.KeyOf(r.Namespace, f.InformationID) == info
	}
	facts := map[corpus.DocumentKey]model.Fact{}
	var out []FactState
	for _, r := range all {
		if r.Fact != nil && !r.RecordedAt.After(at) && !r.Fact.ValidAt.After(at) && concerns(r) {
			out = append(out, FactState{Record: r})
			facts[corpus.DocumentKey{Namespace: r.Namespace, ID: r.ID}] = *r.Fact
		}
	}
	// A correction without a fact retracts what it supersedes; one with a fact replaces only the same attribute.
	superseded := map[corpus.DocumentKey]bool{}
	for _, r := range all {
		if r.Supersedes == nil || r.RecordedAt.After(at) {
			continue
		}
		key := corpus.KeyOf(r.Namespace, *r.Supersedes)
		if prior, ok := facts[key]; ok && (r.Fact == nil || concerns(r) && r.Fact.Attribute == prior.Attribute) {
			superseded[key] = true
		}
	}
	var authority model.FactAuthority
	if def.FactAuthority != nil {
		authority = *def.FactAuthority
	}
	maxAge := time.Duration(authority.MaxAgeSeconds) * time.Second
	best, values := map[string]int{}, map[string][]string{}
	for k := range out {
		r := out[k].Record
		switch {
		case superseded[corpus.DocumentKey{Namespace: r.Namespace, ID: r.ID}]:
			out[k].State, out[k].Rule = FactSuperseded, "CHR-EVID-004: a later record supersedes it"
		case maxAge > 0 && at.Sub(r.Fact.ValidAt) > maxAge:
			out[k].State, out[k].Rule = FactStale, fmt.Sprintf("CHR-EVID-014: valid at %s, more than %s before the decision", r.Fact.ValidAt.Format(time.RFC3339), maxAge)
		default:
			if rank, ok := best[r.Fact.Attribute]; !ok || rankOf(authority, def.Namespace, r) < rank {
				best[r.Fact.Attribute] = rankOf(authority, def.Namespace, r)
			}
		}
	}
	for _, s := range out {
		if r := s.Record; s.State == "" && rankOf(authority, def.Namespace, r) == best[r.Fact.Attribute] {
			if v := canonical(r.Fact.Value); !slices.Contains(values[r.Fact.Attribute], v) {
				values[r.Fact.Attribute] = append(values[r.Fact.Attribute], v)
			}
		}
	}
	for k := range out {
		r := out[k].Record
		switch {
		case out[k].State != "":
		case rankOf(authority, def.Namespace, r) > best[r.Fact.Attribute]:
			out[k].State, out[k].Rule = FactOutranked, "CHR-EVID-015: a higher-ranked source states this attribute"
		case len(values[r.Fact.Attribute]) > 1:
			out[k].State, out[k].Rule = FactConflicted, "CHR-EVID-015: sources of the highest present rank disagree"
		default:
			out[k].State, out[k].Rule = FactCurrent, "current at the highest present rank"
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Record.Fact, out[j].Record.Fact
		if a.Attribute != b.Attribute {
			return a.Attribute < b.Attribute
		}
		return a.ValidAt.Before(b.ValidAt)
	})
	return out, nil
}

// rankOf is the position of the first source matching the record, or one past the last for an unranked source.
func rankOf(a model.FactAuthority, defNamespace string, r model.EvidenceRecord) int {
	for k, s := range a.Ranks {
		switch {
		case s.Category != "" && s.Category == r.Category:
			return k
		case s.SystemProfileID != nil && r.Fact.SystemProfileID != nil &&
			corpus.KeyOf(defNamespace, *s.SystemProfileID) == corpus.KeyOf(r.Namespace, *r.Fact.SystemProfileID):
			return k
		}
	}
	return len(a.Ranks)
}

func canonical(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
