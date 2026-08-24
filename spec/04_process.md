# Business Processes and Tasks

Status: Draft, unreleased

This chapter connects business outcomes to the work performed by humans, agents, and enterprise systems.

## Core concepts

A `ValueStream` describes how value is realized. A `BusinessProcess` organizes work toward a business outcome. A `Task` is an assignable unit of work. A `ProcessInstance` and `TaskInstance` represent execution context without redefining the underlying process architecture.

## Requirements

- **CHR-PROC-001:** Every process and task MUST have a stable identifier and declared business outcome.
- **CHR-PROC-002:** Every governed task MUST identify its accountable responsibility and permitted performer kinds. Performers are human or agent identities; an external system acts only through a binding invoked by a performer and is never itself accountable.
- **CHR-PROC-003:** Process hierarchy, sequence, dependency, and responsibility assignment MUST be modeled as distinct relationships.
- **CHR-PROC-004:** An implementation MUST support customer-defined process names, aliases, and hierarchies without requiring adoption of a vendor taxonomy.
- **CHR-PROC-005:** Execution state MUST NOT silently modify the normative definition of a process or task.
- **CHR-PROC-006:** A task assignment MUST declare whether the performer observes, recommends, prepares, approves, or executes.
- **CHR-PROC-007:** Process exceptions MUST identify the affected task or process instance and the required disposition or escalation path.

Reference process classifications in examples are non-normative and do not form a closed taxonomy.
