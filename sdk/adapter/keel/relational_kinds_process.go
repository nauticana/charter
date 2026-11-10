package keel

import "github.com/nauticana/charter/sdk/model"

func init() {
	// Process.
	register(mutable(model.KindValueStream, tbl("value_stream", "id").
		with(text("businessOutcome", "business_outcome")).
		sub(sourceEvidence())))
	processOwner := []string{"business_process_id", "version_no"}
	register(versioned(model.KindBusinessProcess, "processDefinitionVersion", tbl("business_process", "id"),
		tbl("business_process_version", processOwner...).
			with(text("businessOutcome", "business_outcome"), ref("valueStreamId", "value_stream_id"), ref("parentProcessId", "parent_process_id")).
			sub(statements("aliases", "business_process_alias", processOwner, "alias"), sourceEvidence())))
	taskOwner := []string{"task_id", "version_no"}
	register(versioned(model.KindTask, "taskDefinitionVersion", tbl("process_task", "id"),
		tbl("process_task_version", taskOwner...).
			with(ref("processId", "business_process_id"), text("businessOutcome", "business_outcome"),
				ref("accountableResponsibilityId", "accountable_responsibility_id")).
			sub(codeSet("permittedPerformerKinds", "process_task_performer_kind", taskOwner, "performer_kind", performerKindCodes).required(),
				codeSet("permittedParticipation", "process_task_participation", taskOwner, "participation", taskParticipationCodes),
				evidenceRequirements(),
				scalars("conceptIds", tbl("process_task_concept", taskOwner...).keyed("work_concept_id"), ref("", "work_concept_id")),
				sourceEvidence())))
	processOrTask := func(processCol, taskCol string) []arc {
		return []arc{on(model.KindBusinessProcess, processCol), on(model.KindTask, taskCol)}
	}
	register(mutable(model.KindProcessRelationship, tbl("process_relationship", "id").
		with(code("relationshipType", "relationship_type", processRelationCodes),
			arcOf("subject", processOrTask("subject_process_id", "subject_task_id")...),
			arcOf("object", processOrTask("object_process_id", "object_task_id")...)).
		sub(sourceEvidence())))
	register(mutable(model.KindProcessInstance, tbl("process_instance", "id").
		with(pin("processId", "business_process_id", "processDefinitionVersion", "business_process_version_no"),
			code("state", "state", processStateCodes), stamp("createdAt", "created_at"), stamp("startedAt", "started_at"),
			stamp("endedAt", "ended_at")).
		sub(scalars("contextRefs", tbl("process_instance_context", "process_instance_id").keyed("context_id"), object("", "context_id")))))
	register(mutable(model.KindTaskInstance, tbl("process_task_instance", "id").
		with(ref("processInstanceId", "process_instance_id"), pin("taskId", "task_id", "taskDefinitionVersion", "task_version_no"),
			object("performer", "performer_id"), ref("assignmentId", "assignment_id"),
			code("participation", "participation", taskParticipationCodes), code("state", "state", taskStateCodes),
			stamp("createdAt", "created_at"), stamp("dueAt", "due_at"), stamp("startedAt", "started_at"), stamp("endedAt", "ended_at"),
			text("outcome", "outcome"))))
	contractOwner := []string{"capability_contract_id", "version_no"}
	predicate := []field{code("predicate.language", "predicate_language", predicateLanguageCodes), text("predicate.expression", "predicate_expression")}
	register(versioned(model.KindCapabilityContract, "contractVersion", tbl("capability_contract", "id"),
		tbl("capability_contract_version", contractOwner...).
			with(text("purpose", "purpose"), code("operationClass", "operation_class", operationClassCodes),
				text("requiredAuthority", "required_authority"),
				ref("inputSchema.informationId", "input_information_id"), schemaOf("inputSchema.jsonSchema", "input_schema_id"),
				ref("outputSchema.informationId", "output_information_id"), schemaOf("outputSchema.jsonSchema", "output_schema_id"),
				boolean("constraints.approvalRequired", "approval_required"), boolean("idempotency.mutating", "mutating"),
				text("idempotency.retrySemantics", "retry_semantics"), boolean("idempotency.retryWhenEffectAbsent", "retry_when_effect_absent")).
			sub(statements("inputs", "capability_contract_input", contractOwner, "statement").required(),
				statements("outputs", "capability_contract_output", contractOwner, "statement").required(),
				objects("preconditions", tbl("capability_contract_precondition", contractOwner...).ordered("ordinal").
					proseOr("statement", "precondition_key").
					with(append([]field{text("id", "precondition_key"), text("statement", "statement")}, predicate...)...)).required(),
				scalars("outcomes", tbl("capability_contract_outcome", contractOwner...).ordered("ordinal").keyed("outcome"),
					text("", "outcome")).required(),
				scalars("businessErrors", tbl("capability_contract_business_error", contractOwner...).ordered("ordinal").keyed("business_error"),
					text("", "business_error")).required(),
				objects("postconditions", tbl("capability_contract_postcondition", contractOwner...).ordered("ordinal").keyed("postcondition_key").
					with(append([]field{text("id", "postcondition_key"), text("statement", "statement"),
						boolean("verificationRequired", "verification_required"), text("violationBusinessError", "violation_business_error")},
						predicate...)...).
					sub(scalars("outcomes", tbl("capability_contract_postcondition_outcome", "capability_contract_id", "version_no", "postcondition_key").
						keyed("outcome"), text("", "outcome")))),
				scalars("constraints.sodConstraintIds", tbl("capability_contract_sod_constraint", contractOwner...).keyed("sod_constraint_id"),
					ref("", "sod_constraint_id")).required(),
				limits("constraints.limits").required(),
				evidenceRequirements().required(),
				scalars("conceptIds", tbl("capability_contract_concept", contractOwner...).keyed("work_concept_id"), ref("", "work_concept_id")),
				sourceEvidence())).needs("constraints", "idempotency"))
	register(mutable(model.KindArchitectureState, tbl("architecture_state", "id").
		with(code("stateType", "state_type", stateTypeCodes), object("decisionAuthority", "decision_authority_id")).
		sub(sourceEvidence())))
	register(mutable(model.KindGap, tbl("architecture_gap", "id").
		with(ref("baselineStateId", "baseline_state_id"), ref("targetStateId", "target_state_id"),
			code("classification", "classification", gapCodes)).
		sub(scalars("comparedElements", tbl("architecture_gap_compared_element", "gap_id").keyed("element_id"), object("", "element_id")),
			sourceEvidence())))
	register(mutable(model.KindRoadmapItem, tbl("architecture_roadmap_item", "id").
		with(object("owner", "owner_id"), code("status", "status", roadmapStatusCodes)).
		sub(refSet("addressesGapIds", "architecture_roadmap_item_gap", "roadmap_item_id", "gap_id").required(),
			statements("dependencies", "architecture_roadmap_item_dependency", []string{"roadmap_item_id"}, "statement"),
			sourceEvidence())))
	register(mutable(model.KindOutcomeMeasure, tbl("outcome_measure", "id").
		with(arcOf("subject", on(model.KindValueStream, "subject_value_stream_id"), on(model.KindBusinessProcess, "subject_process_id"),
			on(model.KindTask, "subject_task_id")),
			code("improvement", "improvement", improvementCodes), text("unit", "unit"), text("currency", "currency_code"),
			exponent("currencyExponent"), money("target", "target_amount", "target_quantity"),
			money("baseline.value", "baseline_amount", "baseline_quantity"),
			day("baseline.window.from", "baseline_from"), day("baseline.window.to", "baseline_to")).
		sub(refSet("baseline.evidenceRecordIds", "outcome_measure_baseline_evidence", "outcome_measure_id", "evidence_record_id"))))

	// Binding.
	register(mutable(model.KindEnterpriseSystem, tbl("enterprise_system", "id")))
	register(mutable(model.KindSystemProfile, tbl("system_profile", "id").
		with(ref("enterpriseSystemId", "enterprise_system_id"), text("systemVersion", "system_version"),
			text("authenticationMethod", "authentication_method")).
		sub(statements("supportedFeatures", "system_profile_feature", []string{"system_profile_id"}, "feature").required(),
			statements("constraints", "system_profile_constraint", []string{"system_profile_id"}, "statement").required())))
	capabilityBindingOwner := []string{"capability_binding_id", "version_no"}
	register(versioned(model.KindCapabilityBinding, "bindingVersion", tbl("system_capability_binding", "id"),
		tbl("system_capability_binding_version", capabilityBindingOwner...).
			with(ref("systemProfileId", "system_profile_id"), ref("capabilityId", "capability_contract_id"), text("operation", "operation"),
				text("mappings.inputs", "input_mapping"), text("mappings.outputs", "output_mapping"),
				text("mappings.businessErrors", "business_error_mapping"), text("mappings.authorityChecks", "authority_check_mapping"),
				text("mappings.idempotency", "idempotency_mapping"), text("mappings.evidence", "evidence_mapping")).
			sub(featureSupport("featureSupport"),
				objects("lossyMappings", tbl("system_capability_binding_lossy_mapping", capabilityBindingOwner...).ordered("ordinal").
					with(text("mapping", "mapping"), text("effect", "effect"))))))
	dataBindingOwner := []string{"data_binding_id", "version_no"}
	register(versioned(model.KindDataBinding, "bindingVersion", tbl("system_data_binding", "id"),
		tbl("system_data_binding_version", dataBindingOwner...).
			with(ref("systemProfileId", "system_profile_id"), ref("informationDefinitionId", "information_definition_id")).
			sub(featureSupport("featureSupport"),
				objects("fieldMappings", tbl("system_data_binding_field_mapping", dataBindingOwner...).ordered("ordinal").
					with(text("charterPath", "charter_path"), text("externalPath", "external_path"), text("transformation", "transformation"))).required(),
				objects("lossyTransformations", tbl("system_data_binding_lossy_transformation", dataBindingOwner...).ordered("ordinal").
					with(text("transformation", "transformation"), text("effectOnValidation", "effect_on_validation"),
						text("effectOnDecisions", "effect_on_decisions"), text("effectOnEvidence", "effect_on_evidence"))))))
	register(versioned(model.KindAuthorityBinding, "bindingVersion", tbl("system_authority_binding", "id"),
		tbl("system_authority_binding_version", "authority_binding_id", "version_no").
			with(ref("systemProfileId", "system_profile_id"), ref("capabilityId", "capability_contract_id"),
				text("externalEnforcement", "external_enforcement"), constant("unpreservableConstraintBehavior", "fail-closed")).
			sub(featureSupport("featureSupport"))))
	register(versioned(model.KindEventBinding, "bindingVersion", tbl("system_event_binding", "id"),
		tbl("system_event_binding_version", "event_binding_id", "version_no").
			with(ref("systemProfileId", "system_profile_id"), text("externalEvent", "external_event"), text("charterTrigger", "charter_trigger"),
				code("deliverySemantics", "delivery_semantics", deliveryCodes)).
			sub(featureSupport("featureSupport"))))
	register(mutable(model.KindBindingConformance, tbl("system_binding_conformance", "id").
		with(refArcOf("bindingId", on(model.KindCapabilityBinding, "capability_binding_id"), on(model.KindDataBinding, "data_binding_id"),
			on(model.KindAuthorityBinding, "authority_binding_id"), on(model.KindEventBinding, "event_binding_id")),
			text("conformanceProfile", "conformance_profile"), text("implementationVersion", "implementation_version"),
			code("result", "result", conformanceResultCodes), day("resultDate", "result_date")).
		sub(featureSupport("evaluatedFeatures"), testedRules(), verificationTypes())))

	// Information.
	register(mutable(model.KindInformationDefinition, tbl("information_definition", "id").
		with(text("businessMeaning", "business_meaning"), object("ownerOrSteward", "owner_or_steward_id"),
			text("classification", "classification"), integer("factAuthority.maxAgeSeconds", "fact_max_age_seconds")).
		sub(refSet("governancePolicyIds", "information_definition_policy", "information_definition_id", "information_governance_policy_id").required(),
			objects("shape.properties", tbl("information_property", "information_definition_id").ordered("ordinal").keyed("property_name").
				with(text("name", "property_name"), code("type", "json_type", propertyTypeCodes), boolean("required", "required"),
					text("classification", "classification"))),
			scalars("shape.identity", tbl("information_identity_property", "information_definition_id").ordered("ordinal").keyed("property_name"),
				text("", "property_name")),
			objects("shape.relationships", tbl("information_relationship", "information_definition_id").ordered("ordinal").keyed("relationship_name").
				with(text("name", "relationship_name"), ref("targetInformationId", "target_information_id"),
					code("cardinality", "cardinality", cardinalityCodes))),
			objects("factAuthority.ranks", tbl("information_fact_source", "information_definition_id").ordered("rank_no").
				with(ref("systemProfileId", "system_profile_id"), code("category", "evidence_category", evidenceCategoryCodes))))))
	register(mutable(model.KindInformationGovernancePolicy, tbl("information_governance_policy", "id").
		with(integer("precedence", "precedence"), code("conflictBehavior", "conflict_behavior", conflictCodes)).
		sub(scalars("scope", tbl("information_policy_scope", "information_governance_policy_id").keyed("scope_object_id"),
			object("", "scope_object_id")).required(),
			scalars("permittedUses", tbl("information_policy_use", "information_governance_policy_id").keyed("permitted_use"),
				text("", "permitted_use")).required(),
			statements("accessConstraints", "information_policy_access_constraint", []string{"information_governance_policy_id"}, "statement").required(),
			objects("retentionAndDeletion", tbl("information_retention_rule", "information_governance_policy_id").ordered("ordinal").
				with(code("artifactKind", "artifact_kind", artifactKindCodes), text("retentionRule", "retention_rule"),
					text("deletionRule", "deletion_rule"))).required())))
}
