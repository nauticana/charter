package evidence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestCurrentFacts(t *testing.T) {
	at := time.Date(2026, 6, 18, 18, 0, 0, 0, time.UTC)
	fact := func(id, category, profile, attribute string, value any, validAt time.Time, extra string) string {
		source := ""
		if profile != "" {
			source = `,"systemProfileId":"` + profile + `"`
		}
		return fmt.Sprintf(`{"charterSpecVersion":"1.6.0","namespace":"t","kind":"EvidenceRecord","id":%q,"category":%q,"content":"c","recordedAt":%q%s,`+
			`"fact":{"subject":{"kind":"SalesOrderItem","id":"ORD-1-10","external":true},"informationId":"INFO","attribute":%q,"value":%v,"validAt":%q%s}}`,
			id, category, validAt.Format(time.RFC3339), extra, attribute, value, validAt.Format(time.RFC3339), source)
	}
	c := corpus.New()
	for _, raw := range []string{
		fact("SAP-OLD", model.CategoryObservedFact, "SAP", "eligible", 40, at.Add(-2*time.Hour), ""),
		fact("SAP-NEW", model.CategoryObservedFact, "SAP", "eligible", 38, at.Add(-time.Hour), `,"supersedes":"SAP-OLD"`),
		fact("HUMAN", model.CategoryHumanDecision, "", "eligible", 30, at.Add(-time.Minute), ""),
		fact("WMS-A", model.CategoryObservedFact, "WMS", "blocked", "true", at.Add(-time.Minute), ""),
		fact("WMS-B", model.CategoryObservedFact, "WMS", "blocked", "false", at.Add(-time.Minute), ""),
		fact("AGENT", model.CategoryAgentAssertion, "", "price", 10, at.Add(-time.Minute), ""),
		fact("LATER", model.CategoryObservedFact, "SAP", "eligible", 41, at.Add(time.Minute), ""),
		fact("PRICE", model.CategoryObservedFact, "SAP", "price", 12, at.Add(-time.Minute), `,"supersedes":"HUMAN"`),
		`{"charterSpecVersion":"1.6.0","namespace":"t","kind":"EvidenceRecord","id":"RETRACT","category":"human-decision","content":"the WMS block reading was wrong",` +
			`"recordedAt":"` + at.Format(time.RFC3339) + `","supersedes":"WMS-B"}`,
	} {
		d, err := (corpus.Parser{}).Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	def := model.InformationDefinition{Envelope: model.Envelope{Namespace: "t", ID: "INFO"}, FactAuthority: &model.FactAuthority{
		Ranks: []model.FactSource{{SystemProfileID: &model.Ref{ID: "SAP"}}, {Category: model.CategoryHumanDecision}}, MaxAgeSeconds: 3600}}
	q := Queries{Provider: NewBaseProvider(c)}
	subject := model.ObjectRef{Kind: "SalesOrderItem", ID: "ORD-1-10", External: true}
	states := func(at time.Time, def model.InformationDefinition) map[string]string {
		facts, err := q.CurrentFacts(context.Background(), "t", subject, def, at)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, f := range facts {
			out[f.Record.ID] = f.State
		}
		return out
	}
	// PRICE names HUMAN as superseded but states another attribute, so HUMAN stands; RETRACT, stating no fact, retracts WMS-B.
	want := map[string]string{"SAP-OLD": FactSuperseded, "SAP-NEW": FactCurrent, "HUMAN": FactOutranked, "WMS-A": FactCurrent, "WMS-B": FactSuperseded,
		"AGENT": FactOutranked, "PRICE": FactCurrent}
	if got := states(at, def); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("at the decision: %v, want %v", got, want)
	}
	// One instant past maxAge the SAP fact is stale, the human decision is the highest present rank, and a fact not yet
	// valid is not considered.
	got := states(at.Add(time.Second), def)
	if _, early := got["LATER"]; got["SAP-NEW"] != FactStale || got["HUMAN"] != FactCurrent || early {
		t.Errorf("past maxAge: %v", got)
	}
	// Before the correction was recorded the original fact was current.
	if got := states(at.Add(-90*time.Minute), def); got["SAP-OLD"] != FactCurrent || len(got) != 1 {
		t.Errorf("before the correction: %v", got)
	}
	// Without declared authority every source ranks alike, so the human decision conflicts with SAP.
	if got := states(at, model.InformationDefinition{Envelope: def.Envelope}); got["SAP-NEW"] != FactConflicted || got["HUMAN"] != FactConflicted {
		t.Errorf("unranked: %v", got)
	}
}
