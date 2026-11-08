package validate

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

func TestEvidenceRequirementsRule(t *testing.T) {
	at := time.Date(2026, 6, 18, 17, 15, 0, 0, time.UTC)
	info := model.InformationDefinition{Envelope: envelope(model.KindInformationDefinition, "INFO"), FactAuthority: &model.FactAuthority{MaxAgeSeconds: 3600},
		Shape: &model.InformationShape{Properties: []model.ShapeProperty{{Name: "eligible", Type: "integer"}}, Identity: []string{"eligible"}}}
	item := func(attribute string, age int64) model.EvidenceRequirement {
		return model.EvidenceRequirement{InformationID: &model.Ref{ID: "INFO"}, Attribute: attribute, Required: true, MaxAgeSeconds: age}
	}
	fact := func(id string, validAt time.Time) model.EvidenceRecord {
		r := model.EvidenceRecord{Envelope: envelope(model.KindEvidenceRecord, id), Category: model.CategoryObservedFact, Content: "c", RecordedAt: validAt,
			Fact: &model.Fact{Subject: model.ObjectRef{Kind: "Order", ID: "O", External: true}, InformationID: model.Ref{ID: "INFO"}, Attribute: "eligible", Value: 40, ValidAt: validAt}}
		return r
	}
	contract := func(reqs ...model.EvidenceRequirement) model.CapabilityContract {
		return model.CapabilityContract{Envelope: envelope(model.KindCapabilityContract, "CAP"), EvidenceRequirements: reqs}
	}
	action := func(cites ...string) model.ActionRecord {
		a := model.ActionRecord{Envelope: envelope(model.KindActionRecord, "ACT"), CapabilityID: model.Ref{ID: "CAP"}, Disposition: model.DispositionExecuted, ActionTime: at}
		for _, id := range cites {
			a.EvidenceRecordIDs = append(a.EvidenceRecordIDs, model.Ref{ID: id})
		}
		return a
	}
	rule := EvidenceRequirementsRule{AbstractRule{"CHR-RULE-CAP-003", nil}}
	for name, tc := range map[string]struct {
		docs []any
		want string
	}{
		"current fact cited":      {[]any{info, contract(item("eligible", 900)), fact("F", at.Add(-time.Minute)), action("F")}, ""},
		"prose only":              {[]any{info, contract(model.EvidenceRequirement{Statement: "invocation"}), action()}, ""},
		"optional item uncited":   {[]any{info, contract(model.EvidenceRequirement{InformationID: &model.Ref{ID: "INFO"}, Attribute: "eligible"}), action()}, ""},
		"no fact cited":           {[]any{info, contract(item("eligible", 0)), action()}, "cites no fact"},
		"fact older than item":    {[]any{info, contract(item("eligible", 60)), fact("F", at.Add(-2*time.Minute)), action("F")}, "stale"},
		"item wider than bound":   {[]any{info, contract(item("eligible", 7200))}, "wider than the 3600"},
		"no shape":                {[]any{model.InformationDefinition{Envelope: envelope(model.KindInformationDefinition, "INFO")}, contract(item("eligible", 0))}, "declares no shape"},
		"attribute outside shape": {[]any{info, model.Task{Envelope: envelope(model.KindTask, "TASK"), EvidenceRequirements: []model.EvidenceRequirement{item("blocked", 0)}}}, "shape does not declare"},
		"not executed":            {[]any{info, contract(item("eligible", 0)), func() model.ActionRecord { a := action(); a.Disposition = model.DispositionDenied; return a }()}, ""},
	} {
		findings := rule.Validate(effectCorpus(t, tc.docs...))
		switch {
		case tc.want == "" && len(findings) != 0:
			t.Errorf("%s: %+v", name, findings)
		case tc.want != "" && (len(findings) != 1 || !strings.Contains(findings[0].Message, tc.want)):
			t.Errorf("%s: %+v, want %q", name, findings, tc.want)
		}
	}
}

func TestExternalIdentifiersRule(t *testing.T) {
	id := func(profile, external string, validity *model.Validity) model.ExternalIdentifier {
		return model.ExternalIdentifier{SystemProfileID: model.Ref{ID: profile}, ExternalID: external, Validity: validity}
	}
	span := func(from, to model.Date) *model.Validity { return &model.Validity{From: from, To: to} }
	rule := ExternalIdentifiersRule{AbstractRule{"CHR-RULE-ENT-007", nil}}
	for name, tc := range map[string]struct {
		ids  []model.ExternalIdentifier
		want int
	}{
		"successive":        {[]model.ExternalIdentifier{id("S4", "A", span("2024-01-01", "2025-12-31")), id("S4", "B", span("2026-01-01", ""))}, 0},
		"different systems": {[]model.ExternalIdentifier{id("S4", "A", nil), id("HR", "B", nil)}, 0},
		"both open":         {[]model.ExternalIdentifier{id("S4", "A", nil), id("S4", "B", nil)}, 1},
		"one open":          {[]model.ExternalIdentifier{id("S4", "A", span("2024-01-01", "2025-12-31")), id("S4", "B", nil)}, 1},
		"touching":          {[]model.ExternalIdentifier{id("S4", "A", span("2024-01-01", "2026-01-01")), id("S4", "B", span("2026-01-01", ""))}, 1},
		"reversed validity": {[]model.ExternalIdentifier{id("S4", "A", span("2026-01-01", "2025-01-01"))}, 1},
		"none":              {nil, 0},
	} {
		h := model.HumanIdentity{Envelope: envelope(model.KindHumanIdentity, "H")}
		h.ExternalIdentifiers = tc.ids
		if got := len(rule.Validate(effectCorpus(t, h))); got != tc.want {
			t.Errorf("%s: %d findings, want %d", name, got, tc.want)
		}
	}
}

func TestMeasurementRule(t *testing.T) {
	window := model.Validity{From: "2026-01-01", To: "2026-03-31"}
	measure := func(currency string, baseline *model.Baseline) model.OutcomeMeasure {
		m := model.OutcomeMeasure{Envelope: envelope(model.KindOutcomeMeasure, "OM"), Currency: currency, Baseline: baseline}
		if currency == "" {
			m.Unit = "hour"
		}
		return m
	}
	record := func(value string, w model.Validity, actions ...string) model.EvidenceRecord {
		m := &model.Measurement{MeasureID: model.Ref{ID: "OM"}, Value: json.Number(value), Window: w}
		for _, id := range actions {
			m.ActionRecordIDs = append(m.ActionRecordIDs, model.Ref{ID: id})
		}
		return model.EvidenceRecord{Envelope: envelope(model.KindEvidenceRecord, "EVR"), Category: model.CategoryDerivedConclusion, Content: "c", Measurement: m}
	}
	action := func(disposition, effect string) model.ActionRecord {
		return model.ActionRecord{Envelope: envelope(model.KindActionRecord, "ACT"), Disposition: disposition, Effect: effect}
	}
	rule := MeasurementRule{AbstractRule{"CHR-RULE-PROC-006", nil}}
	for name, tc := range map[string]struct {
		docs []any
		want int
	}{
		"complete action":     {[]any{measure("", nil), record("26", window, "ACT"), action(model.DispositionExecuted, model.EffectComplete)}, 0},
		"effect not stated":   {[]any{measure("", nil), record("26", window, "ACT"), action(model.DispositionExecuted, "")}, 0},
		"partial effect":      {[]any{measure("", nil), record("26", window, "ACT"), action(model.DispositionExecuted, model.EffectPartial)}, 1},
		"unknown disposition": {[]any{measure("", nil), record("26", window, "ACT"), action(model.DispositionUnknown, "")}, 1},
		"reversed window":     {[]any{measure("", nil), record("26", model.Validity{From: "2026-03-31", To: "2026-01-01"})}, 1},
		"currency integer":    {[]any{measure("USD", nil), record("185000", window)}, 0},
		"currency fraction":   {[]any{measure("USD", nil), record("1850.5", window)}, 1},
		"baseline agrees":     {[]any{measure("", &model.Baseline{Value: "26", Window: window, EvidenceRecordIDs: []model.Ref{{ID: "EVR"}}}), record("26", window)}, 0},
		"baseline disagrees":  {[]any{measure("", &model.Baseline{Value: "25", Window: window, EvidenceRecordIDs: []model.Ref{{ID: "EVR"}}}), record("26", window)}, 1},
	} {
		if got := len(rule.Validate(effectCorpus(t, tc.docs...))); got != tc.want {
			t.Errorf("%s: %d findings, want %d: %+v", name, got, tc.want, rule.Validate(effectCorpus(t, tc.docs...)))
		}
	}
}

func TestWorkConceptRule(t *testing.T) {
	concept := func(id, classification, external string, broader ...string) model.WorkConcept {
		w := model.WorkConcept{Envelope: envelope(model.KindWorkConcept, id), ClassificationID: model.Ref{ID: classification}, ExternalID: external, PreferredLabel: id}
		for _, b := range broader {
			w.BroaderConceptIDs = append(w.BroaderConceptIDs, model.Ref{ID: b})
		}
		return w
	}
	rule := WorkConceptRule{AbstractRule{"CHR-RULE-ENT-008", nil}}
	for name, tc := range map[string]struct {
		docs []any
		want int
	}{
		"distinct":            {[]any{concept("A", "C1", "1"), concept("B", "C1", "2", "A")}, 0},
		"same id other class": {[]any{concept("A", "C1", "1"), concept("B", "C2", "1")}, 0},
		"duplicate":           {[]any{concept("A", "C1", "1"), concept("B", "C1", "1")}, 1},
		"broader elsewhere":   {[]any{concept("A", "C1", "1"), concept("B", "C2", "2", "A")}, 1},
		"broader is itself":   {[]any{concept("A", "C1", "1", "A")}, 1},
		"broader unresolved":  {[]any{concept("A", "C1", "1", "NONE")}, 0},
	} {
		if got := len(rule.Validate(effectCorpus(t, tc.docs...))); got != tc.want {
			t.Errorf("%s: %d findings, want %d", name, got, tc.want)
		}
	}
}
