# Business Processes and Tasks

This chapter connects business outcomes to the work performed by humans, agents, and enterprise systems.

## Core concepts

A `ValueStream` describes how value is realized. A `BusinessProcess` organizes work toward a business outcome. A `Task` is an assignable unit of work. A `ProcessInstance` and `TaskInstance` represent execution context without redefining the underlying process architecture. An `OutcomeMeasure` makes the outcome of a value stream, process, or task measurable.

## Requirements

- **CHR-PROC-001:** Every process and task MUST have a stable identifier and declared business outcome.
- **CHR-PROC-002:** Every governed task MUST identify its accountable responsibility and permitted performer kinds. Performers are human or agent identities; an external system acts only through a binding invoked by a performer and is never itself accountable.
- **CHR-PROC-003:** Process hierarchy, sequence, dependency, and responsibility assignment MUST be modeled as distinct relationships.
- **CHR-PROC-004:** An implementation MUST support customer-defined process names, aliases, and hierarchies without requiring adoption of a vendor taxonomy.
- **CHR-PROC-005:** Execution state MUST NOT silently modify the normative definition of a process or task.
- **CHR-PROC-006:** A task assignment MUST declare whether the performer observes, recommends, prepares, approves, or executes.
- **CHR-PROC-007:** Process exceptions MUST identify the affected task or process instance and the required disposition or escalation path.
- **CHR-PROC-008:** Participation levels are ordered `observes` < `recommends` < `prepares` < `performs` < `executes`, where `performs` carries out the work and `executes` also commits its effect on the performer's own authority. `approves` decides on another actor's work, and `supports` and `occupies` relate an identity to a position; none of them is a level. Work performed under an assignment MUST NOT exceed the assignment's level, and a narrower assignment or delegation MAY only keep or lower the level it derives from. Raising an agent's level is a new assignment, evidenced as a governed action, that MUST NOT take effect retroactively. An agent assignment at the `executes` level SHOULD declare the end of its validity, the window in which that autonomy is granted.
- **CHR-PROC-009:** An outcome measure MUST declare the value stream, process, or task it measures, its unit, and whether an increase or a decrease is the improvement. Its baseline MUST state the value observed over a closed window and cite the evidence it was measured from. When a measure of a task, or of the process the task belongs to, is in effect on the day an agent's assignment begins, the agent MUST NOT perform that task above `recommends` under that assignment unless the measure has a baseline whose window ended before that day.

Reference process classifications in examples are non-normative and do not form a closed taxonomy.
