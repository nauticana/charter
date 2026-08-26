package keel

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/capability"
)

const DefaultInvocationMetric = "charter_invocations_total"

// MetricsInvoker counts governed invocations by capability, status, and requirement so denials and failures stay observable (CHR-SEC-007).
type MetricsInvoker struct {
	Next    capability.Invoker
	Metrics port.MetricsRecorder
	Name    string
}

var _ capability.Invoker = (*MetricsInvoker)(nil)

func (m *MetricsInvoker) Invoke(ctx context.Context, inv capability.Invocation) capability.Result {
	if m.Next == nil {
		return capability.Result{Status: capability.StatusDenied, Reason: "metrics invoker has no next invoker", Requirement: "CHR-AUTH-010"}
	}
	res := m.Next.Invoke(ctx, inv)
	if m.Metrics == nil {
		return res
	}
	name := m.Name
	if name == "" {
		name = DefaultInvocationMetric
	}
	err := m.Metrics.RecordMetric(ctx, port.MetricMeasurement{Name: name, Help: "Governed capability invocations by status", Kind: port.MetricCounter, Value: 1,
		Labels: map[string]string{"capability": inv.CapabilityID.ID, "status": string(res.Status), "requirement": res.Requirement}})
	if err != nil {
		res.Err = errors.Join(res.Err, fmt.Errorf("metrics: %w", err))
	}
	return res
}
