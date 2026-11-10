package keel

import (
	"fmt"

	"github.com/nauticana/charter/sdk/model"
)

// codeList maps the words of one specification enumeration to the codes of the keel constant list that stores them.
type codeList struct {
	id    string
	codes map[string]string
	words map[string]string
	order []string
}

func newCodes(id string, wordCodePairs ...string) *codeList {
	c := &codeList{id: id, codes: map[string]string{}, words: map[string]string{}}
	for i := 0; i+1 < len(wordCodePairs); i += 2 {
		word, code := wordCodePairs[i], wordCodePairs[i+1]
		c.codes[word], c.words[code] = code, word
		c.order = append(c.order, code)
	}
	return c
}

func (c *codeList) code(word string) (string, error) {
	if code, ok := c.codes[word]; ok {
		return code, nil
	}
	return "", fmt.Errorf("%w: %q is not in %s", ErrUnrepresentable, word, c.id)
}

func (c *codeList) word(code string) (string, error) {
	if word, ok := c.words[code]; ok {
		return word, nil
	}
	return "", fmt.Errorf("%w: code %q is not in %s", ErrDocumentIncomplete, code, c.id)
}

const externalKindCode = "XO"

var kindCodes = newCodes("charter_kind",
	string(model.KindEnterprise), "EN", string(model.KindOrganizationUnit), "OU", string(model.KindPositionType), "PT",
	string(model.KindPosition), "PO", string(model.KindRole), "RL", string(model.KindResponsibility), "RS",
	string(model.KindOrganizationRelationship), "OR", string(model.KindAssignment), "AS", string(model.KindHumanIdentity), "HI",
	string(model.KindAgentIdentity), "AI", string(model.KindAgentDefinition), "AD", string(model.KindAgentRuntime), "AR",
	string(model.KindAuthorityGrant), "AG", string(model.KindApproval), "AP", string(model.KindSodConstraint), "SC",
	string(model.KindSodException), "SX", string(model.KindValueStream), "VS", string(model.KindBusinessProcess), "BP",
	string(model.KindTask), "TK", string(model.KindProcessRelationship), "PR", string(model.KindProcessInstance), "PI",
	string(model.KindTaskInstance), "TI", string(model.KindCapabilityContract), "CC", string(model.KindArchitectureState), "AT",
	string(model.KindGap), "GP", string(model.KindRoadmapItem), "RM", string(model.KindOutcomeMeasure), "OM",
	string(model.KindEnterpriseSystem), "ES", string(model.KindSystemProfile), "SP", string(model.KindCapabilityBinding), "CB",
	string(model.KindDataBinding), "DB", string(model.KindAuthorityBinding), "AB", string(model.KindEventBinding), "EB",
	string(model.KindBindingConformance), "BC", string(model.KindInformationDefinition), "ID",
	string(model.KindInformationGovernancePolicy), "IG", string(model.KindMediationProfile), "MP",
	string(model.KindMediationDecision), "MD", string(model.KindEvidenceRecord), "ER", string(model.KindActionRecord), "AC",
	string(model.KindExceptionRecord), "XR", string(model.KindEscalation), "EL", string(model.KindEvidenceBundle), "BU",
	string(model.KindWorkClassification), "WC", string(model.KindWorkConcept), "WK", string(model.KindConformanceClaim), "CL",
	"", externalKindCode)

var (
	lifecycleCodes         = newCodes("charter_lifecycle_state", "proposed", "P", "planned", "L", "active", "A", "suspended", "S", "retired", "R")
	identityLifecycleCodes = newCodes("charter_identity_lifecycle_state", "proposed", "P", "active", "A", "suspended", "S", "retired", "R")
	adoptionCodes          = newCodes("charter_adoption", "adopted", "A", "tailored", "T")
	operatorCodes          = newCodes("charter_limit_operator", "lt", "LT", "lte", "LE", "eq", "EQ", "gte", "GE", "gt", "GT")
	supportCodes           = newCodes("charter_feature_support", "supported", "S", "partial", "P", "unsupported", "U")
	jsonValueTypeCodes     = newCodes("charter_json_value_type", "object", "O", "array", "A", "string", "S", "number", "N", "boolean", "B", "null", "Z")
	jsonTypeCodes          = newCodes("charter_json_type", "object", "O", "array", "A", "string", "S", "number", "N", "integer", "I", "boolean", "B", "null", "Z")
	propertyTypeCodes      = newCodes("charter_json_property_type", "object", "O", "array", "A", "string", "S", "number", "N", "integer", "I", "boolean", "B")
	applicatorCodes        = newCodes("charter_json_applicator",
		"properties", "PR", "patternProperties", "PP", "additionalProperties", "AP", "propertyNames", "PN", "dependentSchemas", "DS",
		"unevaluatedProperties", "UP", "items", "IT", "prefixItems", "PI", "contains", "CT", "unevaluatedItems", "UI", "allOf", "AL",
		"anyOf", "AN", "oneOf", "OO", "not", "NT", "if", "IF", "then", "TH", "else", "EL", "$defs", "DF", "contentSchema", "CS")
	relationshipCodes      = newCodes("charter_relationship_code", "reports-to", "R", "deputy-of", "D", "coordinates-with", "C")
	participationCodes     = newCodes("charter_participation", "occupies", "OC", "supports", "SU", "observes", "OB", "recommends", "RC", "prepares", "PP", "approves", "AP", "performs", "PF", "executes", "EX")
	taskParticipationCodes = newCodes("charter_task_participation", "observes", "OB", "recommends", "RC", "prepares", "PP", "approves", "AP", "executes", "EX")
	validityModeCodes      = newCodes("charter_validity_mode", "expires-at", "E", "open-ended", "O")
	processRelationCodes   = newCodes("charter_process_relationship_type", "parent-child", "C", "precedes", "P", "depends-on", "D")
	performerKindCodes     = newCodes("charter_performer_kind", "human", "H", "agent", "A")
	operationClassCodes    = newCodes("charter_operation_class", "read", "R", "propose", "P", "approve", "A", "execute", "E")
	predicateLanguageCodes = newCodes("charter_predicate_language", "cel", "C")
	processStateCodes      = newCodes("charter_process_instance_state", "proposed", "P", "running", "R", "suspended", "S", "completed", "C", "cancelled", "X", "failed", "F")
	taskStateCodes         = newCodes("charter_task_instance_state", "pending", "P", "running", "R", "suspended", "S", "completed", "C", "cancelled", "X", "failed", "F", "escalated", "E")
	stateTypeCodes         = newCodes("charter_state_type", "baseline", "B", "target", "T")
	gapCodes               = newCodes("charter_gap_classification", "missing", "M", "changed", "C", "retired", "R")
	roadmapStatusCodes     = newCodes("charter_roadmap_status", "proposed", "P", "approved", "A", "implementing", "I", "validating", "V", "completed", "C", "cancelled", "X")
	improvementCodes       = newCodes("charter_improvement", "increase", "I", "decrease", "D")
	deliveryCodes          = newCodes("charter_delivery_semantics", "at-most-once", "M", "at-least-once", "L", "exactly-once", "E")
	conformanceResultCodes = newCodes("charter_conformance_result", "conforming", "C", "partial", "P", "nonconforming", "N")
	cardinalityCodes       = newCodes("charter_cardinality", "one", "O", "many", "M")
	evidenceCategoryCodes  = newCodes("charter_evidence_category", "observed-fact", "O", "agent-assertion", "A", "human-decision", "H", "external-response", "X", "derived-conclusion", "D")
	conflictCodes          = newCodes("charter_conflict_behavior", "fail-closed", "C", "apply-declared-precedence", "P")
	artifactKindCodes      = newCodes("charter_artifact_kind", "source-data", "S", "action-evidence", "A", "prompt", "P", "intermediate-artifact", "I", "cache", "C", "external-reference", "E")
	mediatorPartyCodes     = newCodes("charter_mediator_party", "enterprise", "E", "platform", "P", "third-party", "T")
	enforcementCodes       = newCodes("charter_enforcement_mode", "enforce", "E", "shadow", "S")
	failurePolicyCodes     = newCodes("charter_failure_policy", "fail-closed", "C", "fail-open", "O", "observe", "B")
	budgetUnitCodes        = newCodes("charter_budget_unit", "millisecond", "M", "second", "S")
	mediationOutcomeCodes  = newCodes("charter_mediation_outcome", "allow", "A", "deny", "D", "modify", "M", "ask", "K", "defer", "F", "not-decided", "N")
	declaredOutcomeCodes   = newCodes("charter_declared_outcome", "allow", "A", "deny", "D", "modify", "M", "ask", "K", "defer", "F")
	availabilityCodes      = newCodes("charter_mediator_availability", "available", "A", "unavailable", "U", "budget-exceeded", "B", "undecidable", "N")
	notDecidedCodes        = newCodes("charter_not_decided_reason", "out-of-scope", "S", "excluded", "E", "not-sampled", "N", "mediator-unavailable", "U")
	provenanceCodes        = newCodes("charter_attribute_provenance", "authenticated", "A", "asserted", "S")
	permittedEffectCodes   = newCodes("charter_permitted_effect", "tighten-only", "T", "relax-or-tighten", "R")
	attributeEffectCodes   = newCodes("charter_attribute_effect", "tightened", "T", "relaxed", "R", "none", "N")
	obligationTargetCodes  = newCodes("charter_obligation_target", "act", "A", "decision", "D")
	assessmentKindCodes    = newCodes("charter_assessment_kind", "evaluation", "E", "feedback", "F")
	assignmentBasisCodes   = newCodes("charter_assignment_basis", "sole", "S", "matched", "M", "classified", "C", "default", "D")
	dispositionCodes       = newCodes("charter_disposition", "executed", "E", "business-error", "B", "denied", "D", "failed", "F", "unknown", "U")
	effectCodes            = newCodes("charter_effect", "complete", "C", "partial", "P", "absent", "A", "unobserved", "U")
	unknownCauseCodes      = newCodes("charter_unknown_cause", "observation-failed", "O", "unevidenced", "U", "contradictory", "C")
	authorityResultCodes   = newCodes("charter_authority_result", "allowed", "A", "denied", "D", "missing", "M", "error", "E")
	approvalResultCodes    = newCodes("charter_approval_result", "approved", "A", "rejected", "R", "not-required", "N", "missing", "M", "stale", "S")
	informationResultCodes = newCodes("charter_information_result", "allowed", "A", "denied", "D", "error", "E")
	conditionResultCodes   = newCodes("charter_condition_result", "satisfied", "S", "violated", "V", "unknown", "U")
	limitResultCodes       = newCodes("charter_limit_result", "within", "W", "exceeded", "E")
	verificationCodes      = newCodes("charter_verification_type", "structural", "S", "semantic", "M", "runtime-behavioral", "R")
	ruleResultCodes        = newCodes("charter_rule_result", "pass", "P", "fail", "F", "not-tested", "N")
)

// allCodeLists is every list the stores read and write, for the test that keeps them equal to the seed.
var allCodeLists = []*codeList{kindCodes, lifecycleCodes, identityLifecycleCodes, adoptionCodes, operatorCodes, supportCodes,
	jsonValueTypeCodes, jsonTypeCodes, propertyTypeCodes, applicatorCodes, relationshipCodes, participationCodes,
	taskParticipationCodes, validityModeCodes, processRelationCodes, performerKindCodes, operationClassCodes,
	predicateLanguageCodes, processStateCodes, taskStateCodes, stateTypeCodes, gapCodes, roadmapStatusCodes, improvementCodes,
	deliveryCodes, conformanceResultCodes, cardinalityCodes, evidenceCategoryCodes, conflictCodes, artifactKindCodes,
	mediatorPartyCodes, enforcementCodes, failurePolicyCodes, budgetUnitCodes, mediationOutcomeCodes, declaredOutcomeCodes,
	availabilityCodes, notDecidedCodes, provenanceCodes, permittedEffectCodes, attributeEffectCodes, obligationTargetCodes,
	assessmentKindCodes, assignmentBasisCodes, dispositionCodes, effectCodes, unknownCauseCodes, authorityResultCodes,
	approvalResultCodes, informationResultCodes, conditionResultCodes, limitResultCodes, verificationCodes, ruleResultCodes}
