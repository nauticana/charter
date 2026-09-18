package validate

// RuleSet holds the active core-model rules keyed by id.
type RuleSet map[string]Rule

func NewRuleSet() (RuleSet, error) {
	meta, err := NewSchemaMeta()
	if err != nil {
		return nil, err
	}
	all := []Rule{
		UnitTreeRule{AbstractRule{"CHR-RULE-ENT-001", []string{"CHR-ENT-008"}}},
		PositionPlacementRule{AbstractRule{"CHR-RULE-ENT-002", []string{"CHR-ENT-009"}}},
		ValidityOrderedRule{AbstractRule{"CHR-RULE-ENT-003", []string{"CHR-ENT-005"}}},
		HumanOccupancyRule{AbstractRule{"CHR-RULE-ENT-004", []string{"CHR-ENT-011"}}},
		ReferencesResolveRule{AbstractRule{"CHR-RULE-CONF-001", []string{"CHR-CONF-012"}}, meta},
		ActiveActorRule{AbstractRule{"CHR-RULE-ID-001", []string{"CHR-ID-005", "CHR-ID-009"}}},
		GrantEffectiveRule{AbstractRule{"CHR-RULE-AUTH-001", []string{"CHR-AUTH-005", "CHR-EVID-002"}}},
		ApprovalValidRule{AbstractRule{"CHR-RULE-AUTH-002", []string{"CHR-AUTH-009"}}},
		ReferenceKindsRule{AbstractRule{"CHR-RULE-CONF-002", []string{"CHR-CONF-012"}}, meta},
		EnterpriseBoundaryRule{AbstractRule{"CHR-RULE-ENT-005", []string{"CHR-ENT-001", "CHR-SEC-006"}}, meta},
		PerformerRule{AbstractRule{"CHR-RULE-PROC-001", []string{"CHR-PROC-002", "CHR-PROC-006"}}},
		InstanceAssignmentRule{AbstractRule{"CHR-RULE-PROC-002", []string{"CHR-PROC-002", "CHR-ENT-005"}}},
		InstanceDefinitionRule{AbstractRule{"CHR-RULE-PROC-003", []string{"CHR-PROC-005"}}},
		GapStatesRule{AbstractRule{"CHR-RULE-ARCH-001", []string{"CHR-ARCH-004", "CHR-ARCH-006"}}},
		PolicyScopeRule{AbstractRule{"CHR-RULE-INFO-001", []string{"CHR-INFO-002"}}},
		SupersessionRule{AbstractRule{"CHR-RULE-EVID-001", []string{"CHR-EVID-004"}}},
		RuntimeActorRule{AbstractRule{"CHR-RULE-AGENT-001", []string{"CHR-AGENT-006"}}},
		DeclaredCapabilityRule{AbstractRule{"CHR-RULE-AGENT-002", []string{"CHR-AGENT-003"}}},
		SodApprovalRule{AbstractRule{"CHR-RULE-AUTH-003", []string{"CHR-AUTH-007"}}},
		RecordedOutcomeRule{AbstractRule{"CHR-RULE-CAP-001", []string{"CHR-CAP-001", "CHR-EVID-010"}}},
		DeclaredOutcomeRule{AbstractRule{"CHR-RULE-MED-001", []string{"CHR-MED-001", "CHR-MED-003", "CHR-MED-009"}}},
		UnavailableMediatorRule{AbstractRule{"CHR-RULE-MED-002", []string{"CHR-MED-002", "CHR-MED-005", "CHR-MED-006"}}},
		AttributeProvenanceRule{AbstractRule{"CHR-RULE-MED-003", []string{"CHR-MED-007", "CHR-SEC-004"}}},
		RepeatedSubmissionRule{AbstractRule{"CHR-RULE-MED-004", []string{"CHR-MED-010"}}},
	}
	set := RuleSet{}
	for _, r := range all {
		set[r.ID()] = r
	}
	return set, nil
}
