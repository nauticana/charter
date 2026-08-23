# Information Governance Example

Status: Conceptual, non-normative example

| Information | Business meaning | Owner or steward | Classification | Permitted use |
|---|---|---|---|---|
| Order exception | Condition preventing an accepted order from proceeding | Sales Operations | Internal | Resolve the identified order exception |
| Inventory availability | Quantity eligible for allocation at an observed time | Production Planning | Internal | Evaluate fulfillment options |
| Credit-status summary | Current decision-relevant credit state, not the full customer file | Credit Control | Confidential | Prepare or decide the assigned credit exception |
| Resolution proposal | Alternatives, recommendation, assumptions, and cited facts | Sales Operations | Internal | Governed approval and execution |
| Approval record | Human decision bound to action and inputs | Credit Control | Confidential | Authorization, evidence, and audit |

Capabilities expose only the minimum fields needed for their operation. Access to a broader source does not permit the agent to use unrelated customer or employee information.

The proposal records material sources and transformations, including currency conversion and inventory subtraction. The HOS DataBinding declares its minute-level expiry precision. If classification, precision, fields, or governance boundaries change during transformation, the change and decision impact are explicit.

The example retention policy applies separately to prompts, intermediate proposals, cache entries, external references, action evidence, and approval records. Deletion of a cache does not delete required evidence; retention of evidence does not authorize indefinite retention of every intermediate artifact. Conflicting rules fail closed unless an approved precedence rule resolves them.
