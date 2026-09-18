# Mediation of Governed Acts

Status: Version 1.1.0

A mediation point is a place where a governed act is submitted for a decision before it takes effect, and where the decision is made by a party other than the actor performing the act. Platforms that host agents increasingly offer such a callout so that an enterprise can govern acts it does not itself execute.

Charter models what a mediated decision is, what it must carry, and how it relates to authority, information governance, and evidence. It does not define a transport, a payload encoding, an authentication scheme, or a detection technique. Those belong to a binding.

## Relationship to other chapters

Mediation is an additional control surface, not a replacement for one. Authority is still evaluated at the action boundary (`CHR-SEC-003`), information use is still evaluated against declared purpose (`CHR-INFO-007`), and a mediated decision is evidenced with the attribution every governed attempt requires (`CHR-EVID-010`).

Mediation is where `CHR-SEC-004` becomes enforceable: untrusted instructions and retrieved content reach a mediation point as subject material, and the decision must not let that material widen what the actor may do.

## Model

- A `MediationProfile` declares a mediation point: the stage it mediates, its scope, the outcomes it may issue, its enforcement mode, its failure policy, its decision budget, and the coverage it claims.
- A `MediationDecision` records one decision at that point, including the actor and the capabilities and information definitions relevant to the submitted act. An agent actor is identified by `actor`; empty capability or information lists explicitly mean none applies.
- A mediator is the party that evaluates the submitted act. It may be the enterprise itself, the hosting platform, or a third party.
- A submitted act is the act awaiting decision. Charter provides the common stage names `model-invocation`, `model-result`, `tool-invocation`, `tool-result`, `retrieval`, `memory-read`, `memory-write`, `session-start`, `session-end`, `turn-start`, and `turn-end`. A binding may declare another stage name for a distinct host or platform hook; the profile and decision must use the same name.
- An enforcement mode is `enforce`, where the decision governs whether the act proceeds, or `shadow`, where the decision is reached and evidenced but the act proceeds regardless. Shadow mode exists so that a mediation point can be observed before it is trusted to block.
- An obligation is a duty imposed on the act or on handling the decision: redaction, notification, recording, spend caps, or step-up approval. Applied obligations record work actually done; proposed obligations record what an unenforced decision would have required.

## Requirements

- **CHR-MED-001:** A mediation point MUST declare the stage it mediates, the outcomes it may issue, its enforcement mode, its failure policy, its decision budget, and the evidence it produces.
- **CHR-MED-002:** A mediated decision MUST be reached before the submitted act takes effect. An act that has already taken effect MUST NOT be evidenced as mediated. A decision MUST record whether it was enforced; under `shadow` mode no decision is enforced, and an unenforced `deny` or `modify` MUST NOT be evidenced as having prevented or altered the act.
- **CHR-MED-003:** A mediation outcome MUST be one of `allow`, `deny`, `modify`, `ask`, or `defer`, and MUST be one the profile declares. An outcome a profile does not declare MUST be treated as an unavailable mediator rather than as an allow.
- **CHR-MED-004:** A non-allow outcome MUST carry a reference for audit and a reason for the subject that states what the subject can change or what decision is pending, without disclosing detection internals whose disclosure would defeat the control. `ask` and `defer` MUST identify the destination and request for the next decision.
- **CHR-MED-005:** A failure policy MUST be declared as `fail-closed`, `fail-open`, or `observe`. When a mediator is unavailable, exceeds its budget, or returns an undecidable result, the declared policy MUST be applied and recorded, and the act MUST NOT be evidenced as an allow decision.
- **CHR-MED-006:** A mediation point MUST declare any act it leaves undecided, including sampling and exclusions, and an act permitted without a decision MUST be distinguishable in evidence from one allowed by a decision.
- **CHR-MED-007:** An attribute whose provenance is asserted by the mediated runtime, the client, or the submitted material MUST NOT relax a constraint. Only attributes whose provenance is authenticated may do so. Any attribute MAY be the basis for tightening one.
- **CHR-MED-008:** Any outcome MAY carry obligations, including decision-handling duties after a `deny`. A `modify` outcome MUST carry at least one applied obligation if enforced, or at least one proposed obligation if unenforced. An obligation MUST NOT expand the actor's authority. An applied obligation on the act MUST NOT be claimed for an unenforced decision, and where an applied obligation alters submitted material, the altered material MUST be what the act carries onward.
- **CHR-MED-009:** A mediated decision MUST be evidenced with the same attribution as a governed action, MUST state its outcome explicitly, and MUST identify the profile, mediator, actor, and any capability and information definitions relevant to the submitted act.
- **CHR-MED-010:** Repeated submission of the same act to the same mediation profile MUST NOT produce contradictory evidence: a re-submission MUST either reach the same outcome or explicitly supersede an earlier decision for that submission and profile. A superseding decision's `decidedAt` MUST be strictly later than the decision it supersedes.
- **CHR-MED-011:** Mediation MUST NOT be the only enforcement of an authority, separation-of-duties, or information-governance constraint that is required at the action boundary.

## Bindings

A binding maps this model to a concrete interface: how an act is submitted, how the mediator is authenticated, how outcomes are encoded, and how budget and failure are signalled on the wire. Binding requirements in `CHR-BIND-*` apply, in particular the fail-closed obligation in `CHR-BIND-006` and the declared-feature obligation in `CHR-BIND-002`: a binding whose interface cannot express an outcome MUST NOT declare it.

A platform interface that offers only `allow` and `deny` is a conforming binding of a profile that declares only those outcomes. A profile declaring `modify` cannot be bound to it.
