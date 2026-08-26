# Invalid Fixtures

Each directory isolates one condition that must fail one active rule. Every document is schema-valid, so the failure comes from the semantic rule alone; `fixture.yaml` names the rule, the expected result, and why the set is invalid. Scenario fixtures for runtime-behavioral rules instead describe an invocation or context a conforming runtime must refuse, and the decision, requirement, and evidence it must produce.
