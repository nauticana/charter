package keel

import "github.com/nauticana/charter/sdk/model"

// kindSpecs maps every stored document kind to its tables.
var kindSpecs = map[model.Kind]*kindSpec{}

func register(k *kindSpec) {
	code, err := kindCodes.code(string(k.kind))
	if err != nil {
		panic(err)
	}
	k.code = code
	k.envelope = envelopeOf(k.label)
	kindSpecs[k.kind] = k
}

func mutable(kind model.Kind, root *tableSpec) *kindSpec { return &kindSpec{kind: kind, root: root} }

func versioned(kind model.Kind, label string, identity, root *tableSpec) *kindSpec {
	return &kindSpec{kind: kind, label: label, identity: identity, root: root}
}

func (k *kindSpec) needs(paths ...string) *kindSpec {
	k.objects = append(k.objects, paths...)
	return k
}

var (
	positionOrUnit = []arc{on(model.KindPosition, "accountable_position_id"), on(model.KindOrganizationUnit, "accountable_unit_id")}
	humanOrAgent   = func(human, agent string) []arc {
		return []arc{on(model.KindHumanIdentity, human), on(model.KindAgentIdentity, agent)}
	}
)

func init() {
	// Enterprise.
	register(mutable(model.KindEnterprise, tbl("enterprise", "id").sub(sourceEvidence())))
	register(mutable(model.KindOrganizationUnit, tbl("organization_unit", "id").
		with(ref("parentUnitId", "parent_unit_id")).
		sub(externalIdentifiers(), sourceEvidence())))
	register(mutable(model.KindPositionType, tbl("position_type", "id").
		with(text("purpose", "purpose")).
		sub(externalIdentifiers(), sourceEvidence())))
	register(mutable(model.KindPosition, tbl("position", "id").
		with(ref("organizationUnitId", "organization_unit_id"), ref("positionTypeId", "position_type_id")).
		sub(externalIdentifiers(), sourceEvidence())))
	register(mutable(model.KindRole, tbl("organization_role", "id").
		sub(refSet("responsibilityIds", "organization_role_responsibility", "role_id", "responsibility_id"),
			externalIdentifiers(), sourceEvidence())))
	register(mutable(model.KindResponsibility, tbl("responsibility", "id").
		with(text("expectedOutcome", "expected_outcome"), arcOf("accountable", positionOrUnit...)).
		sub(refSet("conceptIds", "responsibility_concept", "responsibility_id", "work_concept_id"), sourceEvidence())))
	register(mutable(model.KindOrganizationRelationship, tbl("organization_relationship", "id").
		with(codeOrText("relationshipType", "relationship_code", "custom_relationship_type", relationshipCodes),
			object("subject", "subject_id"), object("object", "object_id")).
		sub(sourceEvidence())))
	register(mutable(model.KindAssignment, tbl("organization_assignment", "id").
		with(arcOf("subject", humanOrAgent("subject_human_id", "subject_agent_id")...),
			arcOf("target", on(model.KindPosition, "target_position_id"), on(model.KindResponsibility, "target_responsibility_id"),
				on(model.KindRole, "target_role_id"), outside("WorkContext", "target_work_context_id")),
			code("participation", "participation", participationCodes), text("condition", "condition"), text("reason", "reason"),
			object("applicability.classifier", "classifier_id"), boolean("applicability.default", "applies_by_default")).
		sub(objects("applicability.criteria", tbl("organization_assignment_criterion", "assignment_id").keyed("dimension").
			with(text("dimension", "dimension")).
			sub(scalars("values", tbl("organization_assignment_criterion_value", "assignment_id", "dimension").keyed("criterion_value"),
				text("", "criterion_value")).required())),
			sourceEvidence())))

	// Agent.
	register(mutable(model.KindHumanIdentity, tbl("human_identity", "id").
		with(object("lifecycleAuthority", "lifecycle_authority_id")).
		sub(lifecycleHistory(), externalIdentifiers())))
	register(mutable(model.KindAgentIdentity, tbl("agent_identity", "id").
		with(object("lifecycleAuthority", "lifecycle_authority_id")).
		sub(lifecycleHistory())))
	definitionOwner := []string{"agent_definition_id", "version_no"}
	register(versioned(model.KindAgentDefinition, "definitionVersion",
		tbl("enterprise_agent_definition", "id").with(ref("agentIdentityId", "agent_identity_id").once()),
		tbl("enterprise_agent_definition_version", definitionOwner...).
			with(text("purpose", "purpose"), arcOf("accountable", positionOrUnit...)).
			sub(scalars("responsibilityIds", tbl("enterprise_agent_definition_responsibility", definitionOwner...).keyed("responsibility_id"),
				ref("", "responsibility_id")).required(),
				scalars("capabilityIds", tbl("enterprise_agent_definition_capability", definitionOwner...).keyed("capability_contract_id"),
					ref("", "capability_contract_id")).required(),
				statements("triggers", "enterprise_agent_definition_trigger", definitionOwner, "statement").required(),
				statements("requiredInputs", "enterprise_agent_definition_required_input", definitionOwner, "statement").required(),
				statements("expectedOutcomes", "enterprise_agent_definition_expected_outcome", definitionOwner, "statement").required(),
				statements("policies", "enterprise_agent_definition_policy", definitionOwner, "statement").required(),
				statements("escalationConditions", "enterprise_agent_definition_escalation_condition", definitionOwner, "statement").required(),
				statements("confidenceBoundaries", "enterprise_agent_definition_confidence_boundary", definitionOwner, "statement").required(),
				limits("operatingLimits"))))
	register(mutable(model.KindAgentRuntime, tbl("enterprise_agent_runtime", "id").
		with(pin("agentDefinitionId", "agent_definition_id", "definitionVersion", "agent_definition_version_no").once(),
			runtimeAgent("agentIdentityId"), object("operator", "operator_id")).
		sub(limits("operatingLimits"))))
	register(mutable(model.KindAuthorityGrant, tbl("authority_grant", "id").
		with(arcOf("actor", humanOrAgent("actor_human_id", "actor_agent_id")...), ref("capabilityId", "capability_contract_id"),
			text("resourceScope", "resource_scope"), text("organizationalContext", "organizational_context"),
			object("delegation.delegator", "delegator_id"), integer("delegation.maxRedelegationDepth", "max_redelegation_depth"),
			integer("certification.intervalDays", "certification_interval_days"), stamp("certification.certifiedAt", "certified_at"),
			arcOf("certification.certifiedBy", on(model.KindHumanIdentity, "certified_by_id")),
			ref("certification.evidenceRecordId", "certification_evidence_id")).
		sub(limits("limits").required())))
	register(mutable(model.KindApproval, tbl("authority_approval", "id").
		with(arcOf("approver", humanOrAgent("approver_human_id", "approver_agent_id")...), ref("responsibilityId", "responsibility_id"),
			approved("approvedAction"), text("materialInputsDigest", "material_inputs_digest"), stamp("issuedAt", "issued_at"),
			code("validityMode", "validity_mode", validityModeCodes), stamp("expiresAt", "expires_at"),
			text("openEndedJustification", "open_ended_justification"), ref("sodExceptionId", "sod_exception_id")).
		sub(scalars("subjectRefs", tbl("authority_approval_subject", "approval_id").keyed("subject_id"), object("", "subject_id")),
			limits("limits").required(),
			statements("conditions", "authority_approval_condition", []string{"approval_id"}, "statement").required())))
	register(mutable(model.KindSodConstraint, tbl("sod_constraint", "id").
		with(text("scope", "scope")).
		sub(split("constrainedActions", "sod_constraint_id",
			into(model.KindCapabilityContract, "sod_constrained_capability", "capability_contract_id"),
			into(model.KindResponsibility, "sod_constrained_responsibility", "responsibility_id")).required())))
	register(mutable(model.KindSodException, tbl("sod_exception", "id").
		with(ref("sodConstraintId", "sod_constraint_id"), arcOf("beneficiary", humanOrAgent("beneficiary_human_id", "beneficiary_agent_id")...),
			arcOf("grantor", on(model.KindHumanIdentity, "grantor_id")), text("reason", "reason"),
			text("compensatingReview", "compensating_review"), stamp("revokedAt", "revoked_at"))))

	// Work.
	register(mutable(model.KindWorkClassification, tbl("work_classification", "id").
		with(text("publisher", "publisher"), text("version", "classification_version"), text("url", "url"), text("license", "license"))))
	register(mutable(model.KindWorkConcept, tbl("work_concept", "id").
		with(ref("classificationId", "work_classification_id"), text("externalId", "external_id"), text("preferredLabel", "preferred_label")).
		sub(objects("labels", tbl("work_concept_label", "work_concept_id").keyed("language", "label").
			with(text("language", "language"), text("label", "label"))),
			refSet("broaderConceptIds", "work_concept_broader", "work_concept_id", "broader_concept_id"),
			refSet("relatedConceptIds", "work_concept_related", "work_concept_id", "related_concept_id"))))
}
