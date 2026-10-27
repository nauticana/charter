package capability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/information"
	"github.com/nauticana/charter/sdk/model"
)

const dataSchemaURL = "urn:charter:data-schema"

// checkData validates a value against a contract's input or output schema; no schema accepts anything (CHR-CAP-011).
func (i *BaseInvoker) checkData(ctx context.Context, namespace string, ds *model.DataSchema, value any) error {
	if ds == nil {
		return nil
	}
	s, err := i.dataSchema(ctx, namespace, ds)
	if err != nil {
		return err
	}
	v, err := jsonValue(value)
	if err != nil {
		return err
	}
	return s.Validate(v)
}

// dataSchema compiles the shape or embedded JSON Schema a contract names; references to other schemas are refused.
func (i *BaseInvoker) dataSchema(ctx context.Context, namespace string, ds *model.DataSchema) (*jsonschema.Schema, error) {
	var doc any = ds.JSONSchema
	if ds.InformationID != nil {
		if i.Definitions == nil {
			return nil, errors.New("the schema names information but no information provider is composed")
		}
		def, err := i.Definitions.Definition(ctx, namespace, *ds.InformationID)
		if err != nil {
			return nil, err
		}
		if def.Shape == nil {
			return nil, fmt.Errorf("information %s has no shape", def.ID)
		}
		doc = information.JSONSchema(*def.Shape)
	}
	v, err := jsonValue(doc)
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.UseLoader(refusingLoader{})
	if err := c.AddResource(dataSchemaURL, v); err != nil {
		return nil, err
	}
	return c.Compile(dataSchemaURL)
}

type refusingLoader struct{}

func (refusingLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("schema reference %s is refused", url)
}

func jsonValue(v any) (any, error) {
	b, ok := v.(json.RawMessage)
	if !ok {
		var err error
		if b, err = json.Marshal(v); err != nil {
			return nil, err
		}
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}

// checkPreconditions decides each predicate precondition on an observation taken for a new attempt, after its key is
// claimed so a replay returns the settled result; violated or unknown releases the claim and denies before the
// transport is called (CHR-CAP-012).
func (r *invocationRun) checkPreconditions(ctx context.Context) (Result, bool) {
	var preds []model.Precondition
	for _, p := range r.contract.Preconditions {
		if p.Predicate != nil {
			preds = append(preds, p)
		}
	}
	if len(preds) == 0 {
		return Result{}, true
	}
	inv := r.inv
	obs, err := r.invoker.Observer.Observe(ctx, binding.ObservationRequest{Capability: inv.CapabilityID, ContractVersion: r.contract.ContractVersion,
		Inputs: inv.Inputs, IdempotencyKey: inv.IdempotencyKey, SubjectRefs: inv.SubjectRefs, Preconditions: preds})
	if err != nil {
		obs = binding.Observation{}
	}
	r.cite(obs.EvidenceRecordIDs)
	for _, p := range preds {
		e := model.PreconditionEvaluation{PreconditionID: p.ID, ObservedAt: inv.At, EvidenceRecordIDs: obs.EvidenceRecordIDs}
		if err != nil {
			e.Result, e.Reason = model.PostconditionUnknown, "observation failed: "+err.Error()
		} else {
			e.Result, e.Reason = r.decide(ctx, *p.Predicate, obs)
		}
		e.Reason = clip(e.Reason)
		r.result.Preconditions = append(r.result.Preconditions, e)
	}
	for _, e := range r.result.Preconditions {
		if e.Result != model.PostconditionSatisfied {
			r.ledger(ctx, r.contract.Idempotency.Mutating, true, Ledger.Release)
			return r.deny(ctx, fmt.Sprintf("precondition %s is %s: %s", e.PreconditionID, e.Result, e.Reason), "CHR-CAP-012"), false
		}
	}
	return Result{}, true
}

// decide evaluates a predicate on observed facts; facts without observed-fact evidence, or an undecidable predicate,
// are unknown.
func (r *invocationRun) decide(ctx context.Context, p model.Predicate, obs binding.Observation) (string, string) {
	if len(obs.EvidenceRecordIDs) == 0 {
		return model.PostconditionUnknown, "the observed facts carry no observed-fact evidence"
	}
	holds, err := r.invoker.Predicates.Evaluate(ctx, p, r.inv.Inputs, obs.Facts)
	switch {
	case err != nil:
		return model.PostconditionUnknown, "predicate undecided: " + err.Error()
	case holds:
		return model.PostconditionSatisfied, "predicate holds on the observed facts"
	}
	return model.PostconditionViolated, "predicate does not hold on the observed facts"
}

// predicates reports whether the contract declares any predicate, and whether a precondition does.
func predicates(c model.CapabilityContract) (declared, pre bool) {
	for _, p := range c.Preconditions {
		pre = pre || p.Predicate != nil
	}
	declared = pre
	for _, p := range c.Postconditions {
		declared = declared || p.Predicate != nil
	}
	return declared, pre
}
