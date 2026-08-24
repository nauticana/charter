# Information Governance Example

Status: Conceptual, non-normative example

| Information | Business meaning | Owner or steward | Classification | Policy |
|---|---|---|---|---|
| `INFO-ORDER-EXCEPTION` | Condition preventing an accepted order from proceeding | `OU-SALES-OPERATIONS` | Internal | `POL-ORDER-EXCEPTION-INFORMATION` |
| `INFO-CREDIT-STATUS-SUMMARY` | Current decision-relevant credit state, not the full customer file | `OU-CREDIT-CONTROL` | Confidential | `POL-ORDER-EXCEPTION-INFORMATION` |

`POL-ORDER-EXCEPTION-INFORMATION` permits use only to resolve the identified exception, for governed approval and execution, and for audit; it fails closed on conflicts.

Capabilities expose only the minimum fields needed for their operation. Access to a broader source does not permit the agent to use unrelated customer or employee information.

The proposal records material sources and transformations, including currency conversion and inventory subtraction. The SAP DataBinding declares omitted fields, unit or precision changes, identifier mappings, and SAP configuration assumptions. If classification, precision, fields, or governance boundaries change during transformation, the change and decision impact are explicit.

The policy retains action evidence for 7 years, prompts for 90 days, and cache entries for 24 hours, each with its own deletion rule. Deletion of a cache does not delete required evidence; retention of evidence does not authorize indefinite retention of every intermediate artifact. Conflicting rules fail closed unless an approved precedence rule resolves them.
