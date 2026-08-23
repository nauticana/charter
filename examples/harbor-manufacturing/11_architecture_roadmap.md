# Architecture States and Roadmap

Status: Conceptual, non-normative example

## Baseline

`ARCH-BASELINE-2026-Q1` describes an observed current state: Sales Operations coordinates exceptions through messages and HOS screens; approval evidence is inconsistently linked to execution; retries after timeouts are manual; and process-level outcome measures are incomplete. The state is time-bounded and remains historical after later changes.

## Target

`ARCH-TARGET-2026-Q4` describes an intended state proposed by the Sales Operations Manager and subject to Harbor architecture and risk approval. It introduces the governed Order Exception Coordinator, stable capability contracts, explicit authority evaluation, bound approvals, idempotent execution, and linked evidence while retaining human credit decisions.

## Gap and roadmap

`GAP-ORDER-EXCEPTION-EVIDENCE` compares the baseline and target and classifies missing consistent links among facts, proposals, approvals, executions, and outcomes.

`ROADMAP-GOVERNED-EXCEPTION-PILOT` addresses that gap. Sales Operations owns it; dependencies include approved agent definition, identity controls, capability and binding review, evidence retention, security tests, and operational suspension procedures. Its statuses progress through proposed, approved, implementing, validating, and completed or cancelled.

The target does not imply that every gap needs an agent. For example, improving approval retention could be achieved by a process or system-control change independent of agent automation.
