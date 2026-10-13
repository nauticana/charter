# Process SDK

`DefinitionProvider` and `InstanceProvider` keep definitions and execution context distinct (`BaseProvider` implements both). `Graph` reads hierarchy, sequence, and dependency relationships as declared. `BaseContextResolver` resolves a task instance and fails closed on inconsistent definitions, performer, participation, assignment, due time (`ErrDue`), or outcome-measure baseline (`ErrBaseline`); `Resolve` checks a changed instance before it is recorded.

`Level` and `Exceeds` order participation (CHR-PROC-008); `NeedsBaseline` and `Unbaselined` apply CHR-PROC-009. For CHR-PROC-010, `Assign` refuses a closed instance (`ErrClosed`), `Late` reports a missed start deadline, and `LateException` addresses the exception to the task's accountable authority, refusing an instance that is not late (`ErrNotLate`).
