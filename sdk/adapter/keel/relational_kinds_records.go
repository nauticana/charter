package keel

import "github.com/nauticana/charter/sdk/model"

func init() {
	// Mediation.
	profileOwner := "mediation_profile_id"
	register(mutable(model.KindMediationProfile, tbl("mediation_profile", "id").
		with(text("stage", "stage"), code("mediator.party", "mediator_party", mediatorPartyCodes), text("mediator.mediatorId", "mediator_key"),
			text("mediator.name", "mediator_name"), code("enforcementMode", "enforcement_mode", enforcementCodes),
			code("failurePolicy", "failure_policy", failurePolicyCodes), number("decisionBudget.value", "decision_budget_value"),
			code("decisionBudget.unit", "decision_budget_unit", budgetUnitCodes), number("coverage.submittedShare", "coverage_submitted_share"),
			text("evidenceObligation.retention", "evidence_retention")).
		sub(refSet("scope.agentIdentityIds", "mediation_scope_agent", profileOwner, "agent_identity_id"),
			refSet("scope.capabilityIds", "mediation_scope_capability", profileOwner, "capability_contract_id"),
			refSet("scope.informationIds", "mediation_scope_information", profileOwner, "information_definition_id"),
			codeSet("declaredOutcomes", "mediation_declared_outcome", []string{profileOwner}, "outcome", declaredOutcomeCodes).required(),
			objects("coverage.exclusions", tbl("mediation_coverage_exclusion", profileOwner).keyed("subject_id").
				with(object("subject", "subject_id"), text("reason", "reason"))),
			objects("attributeTrust", tbl("mediation_attribute_trust", profileOwner).keyed("attribute").
				with(text("attribute", "attribute"), code("provenance", "provenance", provenanceCodes),
					code("permittedEffect", "permitted_effect", permittedEffectCodes))).required(),
			refSet("evidenceObligation.informationIds", "mediation_evidence_information", profileOwner, "information_definition_id"),
			split("boundaryControlIds", profileOwner,
				into(model.KindAuthorityGrant, "mediation_boundary_grant", "authority_grant_id"),
				into(model.KindSodConstraint, "mediation_boundary_sod_constraint", "sod_constraint_id"),
				into(model.KindInformationGovernancePolicy, "mediation_boundary_policy", "information_governance_policy_id")))).
		needs("scope", "evidenceObligation"))
	decisionOwner := "mediation_decision_id"
	obligations := func(path, table string) listSpec {
		return objects(path, tbl(table, decisionOwner).ordered("ordinal").
			with(text("obligation", "obligation"), text("appliedTo", "applied_to"), code("targetKind", "target_kind", obligationTargetCodes),
				constant("authorityExpanded", false)))
	}
	decision := mutable(model.KindMediationDecision, tbl("mediation_decision", "id").
		with(ref("mediationProfileId", "mediation_profile_id"), text("submissionId", "submission_key"),
			arcOf("actor", humanOrAgent("actor_human_id", "actor_agent_id")...),
			ref("runtimeContext.runtimeInstanceId", "runtime_instance_id"), text("runtimeContext.executionContextId", "execution_context_id"),
			text("stage", "stage"), stamp("decidedAt", "decided_at"), constant("actEffective", false), boolean("enforced", "enforced"),
			code("outcome", "outcome", mediationOutcomeCodes), code("mediatorAvailability", "mediator_availability", availabilityCodes),
			code("appliedFailurePolicy", "applied_failure_policy", failurePolicyCodes),
			code("notDecidedReason", "not_decided_reason", notDecidedCodes), text("subjectReason", "subject_reason"),
			text("auditReference", "audit_reference"), text("handoff.destination", "handoff_destination"),
			text("handoff.request", "handoff_request"), ref("supersedes", "supersedes_id"), ref("actionRecordId", "action_record_id")).
		sub(refSet("subject.capabilityIds", "mediation_subject_capability", decisionOwner, "capability_contract_id").required(),
			refSet("subject.informationIds", "mediation_subject_information", decisionOwner, "information_definition_id").required(),
			obligations("obligationsApplied", "mediation_applied_obligation"),
			obligations("obligationsProposed", "mediation_proposed_obligation"),
			objects("decisionAttributes", tbl("mediation_decision_attribute", decisionOwner).keyed("attribute").
				with(text("attribute", "attribute"), code("provenance", "provenance", provenanceCodes),
					code("effect", "effect", attributeEffectCodes))),
			refSet("evidenceRecordIds", "mediation_decision_evidence", decisionOwner, "evidence_record_id"))).
		needs("subject", "runtimeContext")
	decision.once = true
	decision.actorRuntime = [2]string{"actor_agent_id", "runtime_instance_id"}
	register(decision)

	// Evidence.
	integrity := []field{text("integrity.method", "integrity_method"), text("integrity.value", "integrity_value"),
		stamp("integrity.verifiedAt", "integrity_verified_at")}
	record := "evidence_record_id"
	register(appendOnly(mutable(model.KindEvidenceRecord, tbl("evidence_record", "id").
		with(append([]field{code("category", "category", evidenceCategoryCodes), text("content", "content"),
			stamp("recordedAt", "recorded_at"), ref("supersedes", "supersedes_id")}, integrity...)...).
		sub(objects("sources", tbl("evidence_record_source", record).ordered("ordinal").proseOr("statement", "locator").
			with(text("locator", "locator"), text("mediaType", "media_type"), text("version", "artifact_version"), text("digest", "digest"))),
			statements("transformations", "evidence_record_transformation", []string{record}, "statement"),
			one("assessment", tbl("evidence_assessment", record).
				with(code("assessmentKind", "assessment_kind", assessmentKindCodes), object("subject", "subject_id"),
					text("subjectDigest", "subject_digest"), object("judge", "judge_id"), text("judgeVersion", "judge_version"),
					text("caseSet.id", "case_set_key"), text("caseSet.version", "case_set_version"), text("rating", "rating"),
					text("reason", "reason")).
				sub(objects("scores", tbl("evidence_assessment_score", record).keyed("metric").
					with(text("metric", "metric"), number("value", "score"))))),
			one("fact", tbl("evidence_fact", record).
				with(object("subject", "subject_id"), ref("informationId", "information_definition_id"), text("attribute", "attribute"),
					jsonOf("value", "json_value_id"), stamp("validAt", "valid_at"), ref("systemProfileId", "system_profile_id"))),
			one("measurement", tbl("evidence_measurement", record).
				with(ref("measureId", "outcome_measure_id"), measured("value"), day("window.from", "window_from"),
					day("window.to", "window_to"), text("processDefinitionVersion", "process_definition_version")).
				sub(refSet("actionRecordIds", "evidence_measurement_action", record, "action_record_id")))))))
	action := "action_record_id"
	conditionEvaluations := func(path, table, keyCol, evidenceTable string) listSpec {
		return objects(path, tbl(table, action).keyed(keyCol).
			with(text(path[:len(path)-len("Evaluations")]+"Id", keyCol), code("result", "result", conditionResultCodes),
				text("reason", "reason"), stamp("observedAt", "observed_at")).
			sub(scalars("evidenceRecordIds", tbl(evidenceTable, action, keyCol).keyed("evidence_record_id"), ref("", "evidence_record_id"))))
	}
	register(appendOnly(mutable(model.KindActionRecord, tbl("action_record", "id").
		with(actor("actor", "actor_human_id"), ref("runtimeContext.runtimeInstanceId", "runtime_instance_id"),
			text("runtimeContext.executionContextId", "execution_context_id"), ref("assignmentId", "assignment_id"),
			code("assignmentBasis", "assignment_basis", assignmentBasisCodes), ref("responsibilityId", "responsibility_id"),
			ref("capabilityId", "capability_contract_id"), text("materialInputsDigest", "material_inputs_digest"),
			code("operationClass", "operation_class", operationClassCodes), stamp("actionTime", "action_time"), text("outcome", "outcome"),
			code("disposition", "disposition", dispositionCodes), code("effect", "effect", effectCodes),
			code("unknownCause", "unknown_cause", unknownCauseCodes), ref("reconcilesActionId", "reconciles_action_id"),
			ref("sodExceptionId", "sod_exception_id")).
		sub(scalars("subjectRefs", tbl("action_record_subject", action).keyed("subject_id"), object("", "subject_id")),
			objects("authorityEvaluations", tbl("action_record_authority_evaluation", action).ordered("ordinal").
				with(ref("authorityGrantId", "authority_grant_id"), code("result", "result", authorityResultCodes), text("reason", "reason"))),
			objects("approvalEvaluations", tbl("action_record_approval_evaluation", action).ordered("ordinal").
				with(ref("approvalId", "approval_id"), approved("approvedAction"), code("result", "result", approvalResultCodes),
					text("reason", "reason"))),
			objects("informationEvaluations", tbl("action_record_information_evaluation", action).ordered("ordinal").
				with(ref("informationId", "information_definition_id"), text("purpose", "purpose"),
					code("result", "result", informationResultCodes), text("reason", "reason"))),
			refSet("approvalIds", "action_record_approval", action, "approval_id"),
			refSet("evidenceRecordIds", "action_record_evidence", action, "evidence_record_id"),
			conditionEvaluations("preconditionEvaluations", "action_record_precondition_evaluation", "precondition_key",
				"action_record_precondition_evidence"),
			conditionEvaluations("postconditionEvaluations", "action_record_postcondition_evaluation", "postcondition_key",
				"action_record_postcondition_evidence"),
			objects("operatingLimitEvaluations", tbl("action_record_limit_evaluation", action).keyed("limit_kind").
				with(text("limitKind", "limit_kind"), code("result", "result", limitResultCodes), text("reason", "reason"))))).
		needs("runtimeContext")))
	register(appendOnly(mutable(model.KindExceptionRecord, tbl("exception_record", "id").
		with(text("violatedExpectation", "violated_expectation"), object("affected", "affected_id"), text("disposition", "disposition"),
			object("escalationTarget", "escalation_target_id")))))
	register(appendOnly(mutable(model.KindEscalation, tbl("escalation_record", "id").
		with(text("reason", "reason"), object("recipient", "recipient_id"), text("requestedDecision", "requested_decision"),
			text("urgency", "urgency"), text("resultingDisposition", "resulting_disposition")))))
	register(appendOnly(mutable(model.KindEvidenceBundle, tbl("evidence_bundle", "id").
		with(append([]field{object("subject", "subject_id"), text("assuranceProfile", "assurance_profile")}, integrity...)...).
		sub(split("recordIds", "evidence_bundle_id",
			into(model.KindEvidenceRecord, "evidence_bundle_evidence_record", "evidence_record_id"),
			into(model.KindActionRecord, "evidence_bundle_action_record", "action_record_id"),
			into(model.KindExceptionRecord, "evidence_bundle_exception_record", "exception_record_id"),
			into(model.KindEscalation, "evidence_bundle_escalation_record", "escalation_id")).required()))))

	// Conformance.
	register(mutable(model.KindConformanceClaim, tbl("conformance_claim", "id").
		with(object("implementation", "implementation_id"), text("conformanceProfile", "conformance_profile"),
			text("implementationVersion", "implementation_version"), code("result", "result", conformanceResultCodes),
			day("resultDate", "result_date")).
		sub(objects("results", tbl("conformance_claim_result", "conformance_claim_id").keyed("rule_id", "verification_type").
			with(text("ruleId", "rule_id"), code("verificationType", "verification_type", verificationCodes),
				code("result", "result", ruleResultCodes), text("details", "details"))).required(),
			testedRules(), verificationTypes())))
}

// appendOnly marks an evidence kind: written once, never updated.
func appendOnly(k *kindSpec) *kindSpec { k.evidence = true; return k }
