package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/validate"
)

func TestRefCodec(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want model.Ref
	}{
		{`"POS-1"`, model.Ref{ID: "POS-1"}},
		{`{"id":"POS-1","namespace":"other.example"}`, model.Ref{Namespace: "other.example", ID: "POS-1"}},
	} {
		var r model.Ref
		if err := json.Unmarshal([]byte(tc.raw), &r); err != nil || r != tc.want {
			t.Errorf("%s: %+v %v", tc.raw, r, err)
		}
		if out, err := json.Marshal(r); err != nil || string(out) != tc.raw {
			t.Errorf("%s: marshalled %s %v", tc.raw, out, err)
		}
	}
	if err := json.Unmarshal([]byte(`{"id":7}`), new(model.Ref)); err == nil {
		t.Error("non-string id accepted")
	}
}

// typed decodes a document into the struct for its kind, so the round trip below covers every Harbor kind.
func typed(d *model.Document) (any, error) {
	decode := map[model.Kind]func() (any, error){
		model.KindEnterprise:                  func() (any, error) { return corpus.Decode[model.Enterprise](d) },
		model.KindOrganizationUnit:            func() (any, error) { return corpus.Decode[model.OrganizationUnit](d) },
		model.KindPositionType:                func() (any, error) { return corpus.Decode[model.PositionType](d) },
		model.KindPosition:                    func() (any, error) { return corpus.Decode[model.Position](d) },
		model.KindRole:                        func() (any, error) { return corpus.Decode[model.Role](d) },
		model.KindResponsibility:              func() (any, error) { return corpus.Decode[model.Responsibility](d) },
		model.KindAssignment:                  func() (any, error) { return corpus.Decode[model.Assignment](d) },
		model.KindOrganizationRelationship:    func() (any, error) { return corpus.Decode[model.OrganizationRelationship](d) },
		model.KindHumanIdentity:               func() (any, error) { return corpus.Decode[model.HumanIdentity](d) },
		model.KindAgentIdentity:               func() (any, error) { return corpus.Decode[model.AgentIdentity](d) },
		model.KindAgentDefinition:             func() (any, error) { return corpus.Decode[model.AgentDefinition](d) },
		model.KindAgentRuntime:                func() (any, error) { return corpus.Decode[model.AgentRuntime](d) },
		model.KindAuthorityGrant:              func() (any, error) { return corpus.Decode[model.AuthorityGrant](d) },
		model.KindApproval:                    func() (any, error) { return corpus.Decode[model.Approval](d) },
		model.KindSodConstraint:               func() (any, error) { return corpus.Decode[model.SodConstraint](d) },
		model.KindValueStream:                 func() (any, error) { return corpus.Decode[model.ValueStream](d) },
		model.KindBusinessProcess:             func() (any, error) { return corpus.Decode[model.BusinessProcess](d) },
		model.KindTask:                        func() (any, error) { return corpus.Decode[model.Task](d) },
		model.KindProcessRelationship:         func() (any, error) { return corpus.Decode[model.ProcessRelationship](d) },
		model.KindProcessInstance:             func() (any, error) { return corpus.Decode[model.ProcessInstance](d) },
		model.KindTaskInstance:                func() (any, error) { return corpus.Decode[model.TaskInstance](d) },
		model.KindCapabilityContract:          func() (any, error) { return corpus.Decode[model.CapabilityContract](d) },
		model.KindArchitectureState:           func() (any, error) { return corpus.Decode[model.ArchitectureState](d) },
		model.KindGap:                         func() (any, error) { return corpus.Decode[model.Gap](d) },
		model.KindRoadmapItem:                 func() (any, error) { return corpus.Decode[model.RoadmapItem](d) },
		model.KindEnterpriseSystem:            func() (any, error) { return corpus.Decode[model.EnterpriseSystem](d) },
		model.KindSystemProfile:               func() (any, error) { return corpus.Decode[model.SystemProfile](d) },
		model.KindCapabilityBinding:           func() (any, error) { return corpus.Decode[model.CapabilityBinding](d) },
		model.KindDataBinding:                 func() (any, error) { return corpus.Decode[model.DataBinding](d) },
		model.KindAuthorityBinding:            func() (any, error) { return corpus.Decode[model.AuthorityBinding](d) },
		model.KindEventBinding:                func() (any, error) { return corpus.Decode[model.EventBinding](d) },
		model.KindBindingConformance:          func() (any, error) { return corpus.Decode[model.BindingConformance](d) },
		model.KindActionRecord:                func() (any, error) { return corpus.Decode[model.ActionRecord](d) },
		model.KindEvidenceRecord:              func() (any, error) { return corpus.Decode[model.EvidenceRecord](d) },
		model.KindEvidenceBundle:              func() (any, error) { return corpus.Decode[model.EvidenceBundle](d) },
		model.KindExceptionRecord:             func() (any, error) { return corpus.Decode[model.ExceptionRecord](d) },
		model.KindEscalation:                  func() (any, error) { return corpus.Decode[model.Escalation](d) },
		model.KindInformationDefinition:       func() (any, error) { return corpus.Decode[model.InformationDefinition](d) },
		model.KindInformationGovernancePolicy: func() (any, error) { return corpus.Decode[model.InformationGovernancePolicy](d) },
		model.KindConformanceClaim:            func() (any, error) { return corpus.Decode[model.ConformanceClaim](d) },
		model.KindMediationProfile:            func() (any, error) { return corpus.Decode[model.MediationProfile](d) },
		model.KindMediationDecision:           func() (any, error) { return corpus.Decode[model.MediationDecision](d) },
	}
	return decode[d.Kind]()
}

// TestTypedModelsRoundTripHarbor proves the typed structs neither add nor drop normative fields: every Harbor document
// decodes, re-encodes to the same JSON value, and still validates against its schema.
func TestTypedModelsRoundTripHarbor(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	structural, err := validate.NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[model.Kind]bool{}
	for _, d := range c.Documents() {
		seen[d.Kind] = true
		v, err := typed(d)
		if err != nil {
			t.Errorf("%s: %v", d.ID, err)
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			t.Errorf("%s: %v", d.ID, err)
			continue
		}
		var original, again any
		if err := json.Unmarshal(d.Raw, &original); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &again); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original, again) {
			t.Errorf("%s: typed round trip changed the document\n original: %s\n again:    %s", d.ID, d.Raw, raw)
		}
		parsed, err := (corpus.Parser{}).Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range structural.ValidateDocument(parsed) {
			t.Errorf("%s: %s %s", d.ID, f.Path, f.Message)
		}
	}
	if len(seen) < 35 {
		t.Errorf("Harbor covers only %d kinds", len(seen))
	}
}
