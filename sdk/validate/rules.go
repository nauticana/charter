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
	}
	set := RuleSet{}
	for _, r := range all {
		set[r.ID()] = r
	}
	return set, nil
}
