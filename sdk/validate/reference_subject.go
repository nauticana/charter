package validate

import (
	"context"

	"github.com/nauticana/charter/sdk/agent"
	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/information"
	"github.com/nauticana/charter/sdk/organization"
)

// ReferenceSubject is the SDK's own runtime: the invoker, admission, evaluators, ledger, and evidence sink composed over
// the scenario documents. It is the subject the manifest runs against unless an implementation supplies its own.
type ReferenceSubject struct{}

var _ Subject = ReferenceSubject{}

func (ReferenceSubject) Compose(_ context.Context, documents corpus.Source, transport binding.Executor) (Runtime, error) {
	sink := evidence.NewBaseMemorySink()
	ids := &capability.BaseCounterIDs{Prefix: "RT-"}
	agents, org := agent.NewBaseProvider(documents), organization.NewBaseProvider(documents)
	invoker := &capability.AbstractInvoker{
		Catalog:     capability.NewBaseCatalog(documents),
		Identities:  identity.NewBaseResolver(documents),
		Authority:   &authority.AbstractEvaluator{Source: authority.NewDocumentGrantSource(documents)},
		Approvals:   &authority.AbstractApprovalGate{Source: authority.NewDocumentApprovalSource(documents)},
		Sod:         capability.NewBaseSodChecker(documents, evidence.NewBaseProvider(sink.Store)),
		Information: &information.BaseEvaluator{Provider: information.NewBaseProvider(documents)},
		Bindings:    binding.NewBaseProvider(documents),
		Transport:   transport,
		Ledger:      capability.NewBaseMemoryLedger(),
		Evidence:    sink,
		Escalation:  &capability.BaseEscalator{Recipients: &capability.BaseAccountableRecipient{Agents: agents, Organization: org}, IDs: ids},
		IDs:         ids,
	}
	admission := &agent.BaseAdmission{Agents: agents, Assignments: org}
	return Runtime{Admission: admission, Invoker: invoker, Evidence: evidence.NewBaseProvider(sink.Store)}, nil
}
