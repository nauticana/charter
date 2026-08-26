package validate

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nauticana/charter/sdk/agent"
	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

// Subject is the runtime under test for runtime-behavioral rules. It composes its governed runtime over the
// scenario's documents; the harness scripts the external transport and reads back the evidence the runtime records.
type Subject interface {
	Compose(ctx context.Context, documents corpus.Source, transport binding.Executor) (Runtime, error)
}

// Runtime is what a Subject exposes: admission of work, governed invocation, and read access to recorded evidence.
type Runtime struct {
	Admission agent.Admission
	Invoker   capability.Invoker
	Evidence  evidence.Provider
}

// AdapterSubject is the system adapter under test for system-adapter rules: it realizes the capability binding the
// scenario names over a vendor endpoint the harness scripts, so its request, response, and error mapping is what is judged.
type AdapterSubject interface {
	Realize(ctx context.Context, documents corpus.Source, ownerNamespace string, binding model.Ref, vendor VendorEndpoint) (binding.Executor, error)
}

// VendorEndpoint is the scripted external system an adapter calls with its mapped request.
type VendorEndpoint interface {
	Call(ctx context.Context, payload any) (result any, err error)
}

// Subjects are the implementations under test: a runtime for agent-runtime rules, an adapter for system-adapter rules.
type Subjects struct {
	Runtime Subject
	Adapter AdapterSubject
}

type SubjectKind string

const (
	SubjectRuntime SubjectKind = "runtime"
	SubjectAdapter SubjectKind = "adapter"
)

func (s Subjects) Has(kind SubjectKind) bool {
	if kind == SubjectAdapter {
		return s.Adapter != nil
	}
	return s.Runtime != nil
}

// Scenario drives a subject: a base context, invocation, or request that steps override field by field, each step with
// a scripted transport or vendor outcome and the decision, requirement, and evidence the subject must produce.
type Scenario struct {
	Namespace  string         `yaml:"namespace"`
	Enterprise string         `yaml:"enterprise"`
	Binding    string         `yaml:"binding"`
	Context    ContextSpec    `yaml:"context"`
	Invocation InvocationSpec `yaml:"invocation"`
	Request    RequestSpec    `yaml:"request"`
	Steps      []Step         `yaml:"steps"`
}

// RequestSpec is a capability request as an adapter sees it.
type RequestSpec struct {
	Capability       string          `yaml:"capability"`
	ContractVersion  string          `yaml:"contract_version"`
	Inputs           any             `yaml:"inputs"`
	IdempotencyKey   string          `yaml:"idempotency_key"`
	RequiredFeatures []string        `yaml:"required_features"`
	Subjects         []ObjectRefSpec `yaml:"subjects"`
}

// VendorSpec scripts the external system's reply: a result the adapter must map, or an error whose class is
// not-executed (nothing was sent) or, by default, a failure after sending.
type VendorSpec struct {
	Result any    `yaml:"result"`
	Error  string `yaml:"error"`
	Class  string `yaml:"class"`
}

type ContextSpec struct {
	Identity           string    `yaml:"identity"`
	Runtime            string    `yaml:"runtime"`
	ExecutionContextID string    `yaml:"execution_context_id"`
	Definition         string    `yaml:"definition"`
	DefinitionVersion  string    `yaml:"definition_version"`
	Assignment         string    `yaml:"assignment"`
	Participation      string    `yaml:"participation"`
	Capability         string    `yaml:"capability"`
	At                 time.Time `yaml:"at"`
}

type ObjectRefSpec struct {
	Kind      string `yaml:"kind"`
	ID        string `yaml:"id"`
	Namespace string `yaml:"namespace"`
	External  bool   `yaml:"external"`
}

type MeasureSpec struct {
	Value            any    `yaml:"value"`
	Unit             string `yaml:"unit"`
	Currency         string `yaml:"currency"`
	CurrencyExponent int    `yaml:"currency_exponent"`
}

type InvocationSpec struct {
	Actor                 ObjectRefSpec          `yaml:"actor"`
	Runtime               string                 `yaml:"runtime"`
	ExecutionContextID    string                 `yaml:"execution_context_id"`
	Assignment            string                 `yaml:"assignment"`
	Responsibility        string                 `yaml:"responsibility"`
	Capability            string                 `yaml:"capability"`
	ContractVersion       string                 `yaml:"contract_version"`
	SystemProfile         string                 `yaml:"system_profile"`
	RequiredFeatures      []string               `yaml:"required_features"`
	ResourceScope         string                 `yaml:"resource_scope"`
	OrganizationalContext string                 `yaml:"organizational_context"`
	Measures              map[string]MeasureSpec `yaml:"measures"`
	ApprovedAction        string                 `yaml:"approved_action"`
	MaterialInputsDigest  string                 `yaml:"material_inputs_digest"`
	Subjects              []ObjectRefSpec        `yaml:"subjects"`
	IdempotencyKey        string                 `yaml:"idempotency_key"`
	InformationUses       []InformationUseSpec   `yaml:"information_uses"`
	At                    time.Time              `yaml:"at"`
}

type InformationUseSpec struct {
	Information string `yaml:"information"`
	Purpose     string `yaml:"purpose"`
}

// TransportSpec scripts the external outcome of one invocation; Error is unknown, not-executed, or text treated as unknown.
type TransportSpec struct {
	Outcome       string `yaml:"outcome"`
	BusinessError string `yaml:"business_error"`
	Reference     string `yaml:"reference"`
	Error         string `yaml:"error"`
}

// Expectation lists what the subject must produce; empty fields are not compared. Result is the admission result for
// runtime steps and the response class (outcome, business-error, unknown, not-executed) for adapter steps.
type Expectation struct {
	Result         string `yaml:"result"`
	Status         string `yaml:"status"`
	Requirement    string `yaml:"requirement"`
	Authority      string `yaml:"authority"`
	Approval       string `yaml:"approval"`
	Outcome        string `yaml:"outcome"`
	BusinessError  string `yaml:"business_error"`
	Reference      string `yaml:"reference"`
	TransportCalls *int   `yaml:"transport_calls"`
	VendorCalls    *int   `yaml:"vendor_calls"`
	Evidence       *bool  `yaml:"evidence"`
	Escalated      *bool  `yaml:"escalated"`
	Recipient      string `yaml:"recipient"`
}

// Step admits a context, invokes a capability, or sends a request to an adapter; absent ones are zero Nodes.
type Step struct {
	Name      string         `yaml:"name"`
	Admit     yaml.Node      `yaml:"admit"`
	Invoke    yaml.Node      `yaml:"invoke"`
	Request   yaml.Node      `yaml:"request"`
	Transport *TransportSpec `yaml:"transport"`
	Vendor    *VendorSpec    `yaml:"vendor"`
	Expect    Expectation    `yaml:"expect"`
}

// BehavioralRule executes a fixture's scenario against the subject its Kind names and reports every deviation.
type BehavioralRule struct {
	AbstractRule
	Kind SubjectKind
}

type BehavioralRuleSet map[string]BehavioralRule

func NewBehavioralRuleSet() BehavioralRuleSet {
	all := []BehavioralRule{
		{AbstractRule{"CHR-RULE-RT-001", []string{"CHR-AUTH-002", "CHR-AUTH-008", "CHR-AUTH-010", "CHR-SEC-003", "CHR-EVID-002"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-RT-002", []string{"CHR-AUTH-009"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-RT-003", []string{"CHR-CAP-004", "CHR-SEC-008"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-RT-004", []string{"CHR-CAP-001", "CHR-CAP-005", "CHR-SEC-007"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-RT-005", []string{"CHR-AGENT-005", "CHR-AGENT-007"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-RT-006", []string{"CHR-ID-005", "CHR-SEC-009"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-RT-007", []string{"CHR-INFO-002", "CHR-INFO-007", "CHR-INFO-008"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-RT-008", []string{"CHR-AGENT-004", "CHR-EVID-006", "CHR-EVID-007"}}, SubjectRuntime},
		{AbstractRule{"CHR-RULE-SA-001", []string{"CHR-BIND-002", "CHR-BIND-006", "CHR-BIND-009"}}, SubjectAdapter},
		{AbstractRule{"CHR-RULE-SA-002", []string{"CHR-CAP-005", "CHR-BIND-005", "CHR-SEC-007", "CHR-SEC-008"}}, SubjectAdapter},
		{AbstractRule{"CHR-RULE-SA-003", []string{"CHR-CAP-001", "CHR-CAP-007"}}, SubjectAdapter},
		{AbstractRule{"CHR-RULE-SA-004", []string{"CHR-CAP-007", "CHR-BIND-001"}}, SubjectAdapter},
	}
	set := BehavioralRuleSet{}
	for _, r := range all {
		set[r.ID()] = r
	}
	return set
}

func (r BehavioralRule) Run(ctx context.Context, subjects Subjects, documents corpus.Source, sc *Scenario) []Finding {
	if r.Kind == SubjectAdapter {
		return r.runAdapter(ctx, subjects.Adapter, documents, sc)
	}
	return r.runRuntime(ctx, subjects.Runtime, documents, sc)
}

func (r BehavioralRule) runRuntime(ctx context.Context, subject Subject, documents corpus.Source, sc *Scenario) []Finding {
	if subject == nil {
		return []Finding{r.behavioral("", "no runtime under test")}
	}
	transport := &scriptedTransport{}
	rt, err := subject.Compose(ctx, documents, transport)
	if err != nil {
		return []Finding{r.behavioral("", "compose: "+err.Error())}
	}
	var out []Finding
	for i, step := range sc.Steps {
		for _, msg := range r.runStep(ctx, rt, transport, sc, step) {
			out = append(out, r.behavioral(stepName(i, step), msg))
		}
	}
	return out
}

func (r BehavioralRule) runAdapter(ctx context.Context, adapter AdapterSubject, documents corpus.Source, sc *Scenario) []Finding {
	if adapter == nil {
		return []Finding{r.behavioral("", "no adapter under test")}
	}
	vendor := &scriptedVendor{}
	executor, err := adapter.Realize(ctx, documents, sc.Namespace, model.Ref{ID: sc.Binding}, vendor)
	if err != nil {
		return []Finding{r.behavioral("", "realize: "+err.Error())}
	}
	if executor == nil {
		return []Finding{r.behavioral("", "realize: adapter returned no executor")}
	}
	var out []Finding
	for i, step := range sc.Steps {
		for _, msg := range r.runAdapterStep(ctx, executor, vendor, sc, step) {
			out = append(out, r.behavioral(stepName(i, step), msg))
		}
	}
	return out
}

func (r BehavioralRule) runAdapterStep(ctx context.Context, executor binding.Executor, vendor *scriptedVendor, sc *Scenario, step Step) []string {
	if step.Request.Kind == 0 {
		return []string{"adapter step declares no request"}
	}
	spec, err := sc.Request.override(&step.Request)
	if err != nil {
		return []string{"request: " + err.Error()}
	}
	if step.Vendor != nil {
		vendor.script(*step.Vendor)
	}
	resp, err := executor.Execute(ctx, spec.request())
	var mismatches []string
	expect := func(label, got, want string) {
		if want != "" && got != want {
			mismatches = append(mismatches, fmt.Sprintf("%s %q, want %q", label, got, want))
		}
	}
	e := step.Expect
	expect("result", responseClass(resp, err), e.Result)
	expect("outcome", resp.Outcome, e.Outcome)
	expect("business error", resp.BusinessError, e.BusinessError)
	expect("reference", resp.ExternalReference, e.Reference)
	if e.VendorCalls != nil && vendor.calls != *e.VendorCalls {
		mismatches = append(mismatches, fmt.Sprintf("vendor called %d times, want %d", vendor.calls, *e.VendorCalls))
	}
	if len(mismatches) > 0 && err != nil {
		mismatches = append(mismatches, "adapter said: "+err.Error())
	}
	return mismatches
}

// responseClass names what an adapter reported: a declared outcome, a declared business error, an unknown outcome,
// proven non-execution, or an unclassified failure, which no conforming adapter may return.
func responseClass(resp binding.Response, err error) string {
	switch {
	case err == nil && resp.BusinessError != "":
		return "business-error"
	case err == nil:
		return "outcome"
	case errors.Is(err, binding.ErrNotExecuted):
		return "not-executed"
	case errors.Is(err, binding.ErrOutcomeUnknown):
		return "unknown"
	}
	return "unclassified"
}

func stepName(i int, step Step) string {
	if step.Name != "" {
		return step.Name
	}
	return fmt.Sprintf("step %d", i+1)
}

func (r BehavioralRule) runStep(ctx context.Context, rt Runtime, transport *scriptedTransport, sc *Scenario, step Step) []string {
	var mismatches []string
	expect := func(label string, got, want string) {
		if want != "" && got != want {
			mismatches = append(mismatches, fmt.Sprintf("%s %q, want %q", label, got, want))
		}
	}
	e := step.Expect
	switch {
	case step.Admit.Kind != 0:
		if rt.Admission == nil {
			return []string{"subject exposes no admission"}
		}
		spec := sc.Context
		if err := step.Admit.Decode(&spec); err != nil {
			return []string{"admit: " + err.Error()}
		}
		d := rt.Admission.Admit(ctx, spec.context(sc))
		expect("result", string(d.Result)+" ("+d.Reason+")", withReason(e.Result, d))
		expect("requirement", d.Requirement, e.Requirement)
	case step.Invoke.Kind != 0:
		if rt.Invoker == nil {
			return []string{"subject exposes no invoker"}
		}
		spec, err := sc.Invocation.override(&step.Invoke)
		if err != nil {
			return []string{"invoke: " + err.Error()}
		}
		if step.Transport != nil {
			transport.script(*step.Transport)
		}
		res := rt.Invoker.Invoke(ctx, spec.invocation(sc))
		expect("status", string(res.Status), e.Status)
		expect("requirement", res.Requirement, e.Requirement)
		expect("authority", string(res.Authority.Result), e.Authority)
		expect("approval", string(res.Approval.Result), e.Approval)
		expect("outcome", res.Outcome, e.Outcome)
		if e.TransportCalls != nil && transport.calls != *e.TransportCalls {
			mismatches = append(mismatches, fmt.Sprintf("transport called %d times, want %d", transport.calls, *e.TransportCalls))
		}
		if e.Evidence != nil {
			recorded := res.Action != nil
			if recorded && rt.Evidence != nil {
				_, err := rt.Evidence.Action(ctx, res.Action.Namespace, model.Ref{ID: res.Action.ID})
				recorded = err == nil
			}
			if recorded != *e.Evidence {
				mismatches = append(mismatches, fmt.Sprintf("evidence recorded %v, want %v", recorded, *e.Evidence))
			}
		}
		if e.Escalated != nil || e.Recipient != "" {
			var escalation *model.Escalation
			if res.Escalation != nil && rt.Evidence != nil {
				if stored, err := rt.Evidence.Escalation(ctx, res.Escalation.Namespace, *res.Escalation); err == nil {
					escalation = &stored
				}
			}
			if e.Escalated != nil && (escalation != nil) != *e.Escalated {
				mismatches = append(mismatches, fmt.Sprintf("escalation recorded %v, want %v", escalation != nil, *e.Escalated))
			}
			if e.Recipient != "" && (escalation == nil || escalation.Recipient.ID != e.Recipient) {
				mismatches = append(mismatches, fmt.Sprintf("escalation recipient %v, want %s", escalation, e.Recipient))
			}
		}
		if len(mismatches) > 0 && res.Reason != "" {
			mismatches = append(mismatches, "runtime said: "+res.Reason)
		}
	default:
		return []string{"step declares neither admit nor invoke"}
	}
	return mismatches
}

// withReason keeps the comparison on the result while the finding still shows the runtime's reason.
func withReason(want string, d agent.Decision) string {
	if want == "" {
		return ""
	}
	if string(d.Result) == want {
		return string(d.Result) + " (" + d.Reason + ")"
	}
	return want
}

func (r AbstractRule) behavioral(step, msg string) Finding {
	return Finding{RuleID: r.id, Requirements: r.reqs, Class: ClassBehavioral, Path: step, Message: msg}
}

// override applies a step's fields over the base invocation; a step's measures replace the base measures entirely.
func (s InvocationSpec) override(node *yaml.Node) (InvocationSpec, error) {
	spec := s
	spec.RequiredFeatures = slices.Clone(s.RequiredFeatures)
	spec.Subjects = slices.Clone(s.Subjects)
	spec.InformationUses = slices.Clone(s.InformationUses)
	spec.Measures = maps.Clone(s.Measures)
	var keys map[string]any
	if err := node.Decode(&keys); err != nil {
		return spec, err
	}
	if _, ok := keys["measures"]; ok {
		spec.Measures = nil
	}
	return spec, node.Decode(&spec)
}

// override applies a step's fields over the base request; a step's inputs replace the base inputs entirely.
func (s RequestSpec) override(node *yaml.Node) (RequestSpec, error) {
	spec := s
	spec.RequiredFeatures = slices.Clone(s.RequiredFeatures)
	spec.Subjects = slices.Clone(s.Subjects)
	var keys map[string]any
	if err := node.Decode(&keys); err != nil {
		return spec, err
	}
	if _, ok := keys["inputs"]; ok {
		spec.Inputs = nil
	}
	return spec, node.Decode(&spec)
}

func (s RequestSpec) request() binding.Request {
	req := binding.Request{Capability: model.Ref{ID: s.Capability}, ContractVersion: s.ContractVersion, Inputs: s.Inputs,
		IdempotencyKey: s.IdempotencyKey, RequiredFeatures: s.RequiredFeatures}
	for _, subject := range s.Subjects {
		req.SubjectRefs = append(req.SubjectRefs, subject.ref())
	}
	return req
}

func (s InvocationSpec) invocation(sc *Scenario) capability.Invocation {
	inv := capability.Invocation{
		Namespace: sc.Namespace, EnterpriseID: model.Ref{ID: sc.Enterprise}, Actor: s.Actor.ref(),
		Runtime:      model.RuntimeContext{RuntimeInstanceID: model.Ref{ID: s.Runtime}, ExecutionContextID: s.ExecutionContextID},
		CapabilityID: model.Ref{ID: s.Capability}, ContractVersion: s.ContractVersion, RequiredFeatures: s.RequiredFeatures,
		ResourceScope: s.ResourceScope, OrganizationalContext: s.OrganizationalContext, ApprovedAction: s.ApprovedAction,
		MaterialInputsDigest: s.MaterialInputsDigest, IdempotencyKey: s.IdempotencyKey, At: s.At,
	}
	if s.Assignment != "" {
		inv.AssignmentID = &model.Ref{ID: s.Assignment}
	}
	if s.Responsibility != "" {
		inv.ResponsibilityID = &model.Ref{ID: s.Responsibility}
	}
	if s.SystemProfile != "" {
		inv.SystemProfileID = &model.Ref{ID: s.SystemProfile}
	}
	if len(s.Measures) > 0 {
		inv.Measures = map[string]authority.Measure{}
		for k, m := range s.Measures {
			inv.Measures[k] = authority.Measure{Value: m.Value, Unit: m.Unit, Currency: m.Currency, CurrencyExponent: m.CurrencyExponent}
		}
	}
	for _, subject := range s.Subjects {
		inv.SubjectRefs = append(inv.SubjectRefs, subject.ref())
	}
	for _, use := range s.InformationUses {
		inv.InformationUses = append(inv.InformationUses, capability.InformationUse{Information: model.Ref{ID: use.Information}, Purpose: use.Purpose})
	}
	return inv
}

func (c ContextSpec) context(sc *Scenario) agent.ExecutionContext {
	ec := agent.ExecutionContext{Namespace: sc.Namespace, Identity: model.Ref{ID: c.Identity}, Runtime: model.Ref{ID: c.Runtime},
		ExecutionContextID: c.ExecutionContextID, Definition: model.Ref{ID: c.Definition}, DefinitionVersion: c.DefinitionVersion,
		Assignment: model.Ref{ID: c.Assignment}, Participation: c.Participation, At: c.At}
	if c.Capability != "" {
		ec.Capability = &model.Ref{ID: c.Capability}
	}
	return ec
}

func (o ObjectRefSpec) ref() model.ObjectRef {
	return model.ObjectRef{Kind: model.Kind(o.Kind), ID: o.ID, Namespace: o.Namespace, External: o.External}
}

type scriptedVendor struct {
	calls  int
	result any
	err    error
}

var _ VendorEndpoint = (*scriptedVendor)(nil)

func (v *scriptedVendor) Call(context.Context, any) (any, error) {
	v.calls++
	return v.result, v.err
}

func (v *scriptedVendor) script(spec VendorSpec) {
	v.result, v.err = spec.Result, nil
	switch {
	case spec.Error == "":
	case spec.Class == "not-executed":
		v.err = fmt.Errorf("%w: %s", binding.ErrNotExecuted, spec.Error)
	default:
		v.err = errors.New(spec.Error)
	}
}

type scriptedTransport struct {
	calls int
	resp  binding.Response
	err   error
}

var _ binding.Executor = (*scriptedTransport)(nil)

func (t *scriptedTransport) Execute(context.Context, binding.Request) (binding.Response, error) {
	t.calls++
	return t.resp, t.err
}

func (t *scriptedTransport) script(spec TransportSpec) {
	t.resp = binding.Response{Outcome: spec.Outcome, BusinessError: spec.BusinessError, ExternalReference: spec.Reference}
	switch spec.Error {
	case "":
		t.err = nil
	case "unknown":
		t.err = fmt.Errorf("%w: scripted", binding.ErrOutcomeUnknown)
	case "not-executed":
		t.err = fmt.Errorf("%w: scripted", binding.ErrNotExecuted)
	default:
		t.err = errors.New(spec.Error)
	}
}
