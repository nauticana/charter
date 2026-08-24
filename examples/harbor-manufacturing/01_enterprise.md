# Harbor Manufacturing Enterprise

Harbor Manufacturing (`ENT-HARBOR`) contains three functional organization units and one supporting unit beneath its root organization unit `OU-HARBOR`:

- Sales Operations (`OU-SALES-OPERATIONS`) owns customer-order intake and exception coordination; Order Management is its child unit.
- Production (`OU-PRODUCTION`) owns planning and fulfillment readiness; Production Planning is its child unit.
- Finance (`OU-FINANCE`) owns credit policy and financial control; Credit Control is its child unit.
- Platform Operations (`OU-PLATFORM-OPERATIONS`) operates agent runtimes without holding business authority.

```mermaid
flowchart TB
    Enterprise["Harbor Manufacturing<br/>Enterprise ENT-HARBOR"]

    subgraph HarborGroup["OU-HARBOR"]
        Harbor["Harbor Manufacturing Organization<br/>OrganizationUnit"]
    end

    subgraph SalesGroup["OU-SALES-OPERATIONS"]
        direction LR
        Sales["Sales Operations<br/>OrganizationUnit"]
        SalesManager["Sales Operations Manager<br/>Position POS-SALES-OPERATIONS-MANAGER"]
        Sales --> SalesManager
    end

    subgraph OrderManagementGroup["OU-ORDER-MANAGEMENT"]
        direction LR
        OrderManagement["Order Management<br/>OrganizationUnit"]
        OrderSpecialist["Order Specialist<br/>Position POS-ORDER-SPECIALIST"]
        OrderManagement --> OrderSpecialist
    end

    subgraph ProductionGroup["OU-PRODUCTION"]
        Production["Production<br/>OrganizationUnit"]
    end

    subgraph ProductionPlanningGroup["OU-PRODUCTION-PLANNING"]
        direction LR
        ProductionPlanning["Production Planning<br/>OrganizationUnit"]
        PlanningManager["Production Planning Manager<br/>Position PP-MGR-01"]
        PlannerOne["Production Planner<br/>Position PP-PLN-01"]
        PlannerTwo["Production Planner<br/>Position PP-PLN-02"]
        ProductionPlanning --> PlanningManager
        ProductionPlanning --> PlannerOne
        ProductionPlanning --> PlannerTwo
    end

    subgraph FinanceGroup["OU-FINANCE"]
        Finance["Finance<br/>OrganizationUnit"]
    end

    subgraph CreditControlGroup["OU-CREDIT-CONTROL"]
        direction LR
        CreditControl["Credit Control<br/>OrganizationUnit"]
        CreditManager["Credit Manager<br/>Position POS-CREDIT-MANAGER"]
        CreditControl --> CreditManager
    end

    subgraph PlatformGroup["OU-PLATFORM-OPERATIONS"]
        Platform["Platform Operations<br/>OrganizationUnit"]
    end

    Enterprise ==> HarborGroup
    HarborGroup ==> SalesGroup
    HarborGroup ==> ProductionGroup
    HarborGroup ==> FinanceGroup
    HarborGroup ==> PlatformGroup
    SalesGroup ==> OrderManagementGroup
    ProductionGroup ==> ProductionPlanningGroup
    FinanceGroup ==> CreditControlGroup

    classDef enterprise fill:#eceff1,stroke:#455a64,stroke-width:2px
    classDef organizationUnit fill:#e8f1ff,stroke:#245b8a,stroke-width:2px
    classDef position fill:#fff3d6,stroke:#9a6700,stroke-width:1px
    class Enterprise enterprise
    class Harbor,Sales,OrderManagement,Production,ProductionPlanning,Finance,CreditControl,Platform organizationUnit
    class SalesManager,OrderSpecialist,PlanningManager,PlannerOne,PlannerTwo,CreditManager position
```

Each subgraph represents one `OrganizationUnit` and contains the `Position` objects that belong to it. Thick edges are the `parentUnitId` tree; thin edges are `organizationUnitId` placement. The tree edges target subgraph IDs rather than internal organization-unit nodes because Mermaid ignores a subgraph's local direction when one of its internal nodes has an external edge.

Each position belongs to exactly one organization unit but remains a different type of Charter object. The two Production Planner positions share a display name and keep distinct identifiers.

## Position types across organization units

`PT-FUNCTIONAL-MANAGER` is a `PositionType` that classifies the Sales Operations Manager and Credit Manager positions. Those positions belong to different organization units and retain different identifiers.

```mermaid
flowchart TB
    PositionType["Functional Manager<br/>PositionType PT-FUNCTIONAL-MANAGER"]
    SalesManager["Sales Operations Manager<br/>Position POS-SALES-OPERATIONS-MANAGER"]
    CreditManager["Credit Manager<br/>Position POS-CREDIT-MANAGER"]

    PositionType -.->|classifies| SalesManager
    PositionType -.->|classifies| CreditManager
```

The shared type means the positions have the same broad organizational nature. It does not place them in the same organization unit, make them report to one another, or copy roles, responsibilities, assignments, or authority between them. Those relationships remain explicit on the individual positions and related Charter objects.

## Production Planning position assignments

The following focused example distinguishes stable positions from the human and agent identities assigned to support them. It shows human succession, an agent supporting two positions, and a position that is currently unfilled. `REL-PP-PLN-01-REPORTS-TO-PP-MGR-01` records the reporting line as a separate `OrganizationRelationship`.

```mermaid
flowchart LR
    direction LR
    ProductionPlanningUnit["Production Planning<br/>OrganizationUnit"]
    PlanningManagerPosition["Production Planning Manager<br/>Position PP-MGR-01"]
    PlannerPositionOne["Production Planner<br/>Position PP-PLN-01"]
    PlannerPositionTwo["Production Planner<br/>Position PP-PLN-02"]

    ProductionPlanningUnit ==> PlanningManagerPosition
    ProductionPlanningUnit ==> PlannerPositionOne
    ProductionPlanningUnit ==> PlannerPositionTwo

    Morgan["Morgan Lee<br/>HUMAN-MORGAN-LEE"]
    Priya["Priya Shah<br/>HUMAN-PRIYA-SHAH"]
    Jordan["Jordan Kim<br/>HUMAN-JORDAN-KIM"]
    DataQualityAgent["Planning Data Quality Assistant<br/>AGENT-PLANNING-DATA-QUALITY"]

    MorganAssignment["ASGN-MORGAN-PP-MGR<br/>occupies · 2024-01-01 onward"]
    PriyaAssignment["ASGN-PRIYA-PP-PLN-01<br/>occupies · 2024-01-01 to 2025-12-31"]
    JordanAssignment["ASGN-JORDAN-PP-PLN-01<br/>occupies · 2026-01-01 onward"]
    DataQualityAssignmentOne["ASGN-DQA-PP-PLN-01<br/>supports · 2026-04-01 onward"]
    DataQualityAssignmentTwo["ASGN-DQA-PP-PLN-02<br/>supports · 2026-04-01 onward"]

    PlanningManagerPosition --- MorganAssignment --- Morgan
    PlannerPositionOne --- PriyaAssignment --- Priya
    PlannerPositionOne --- JordanAssignment --- Jordan
    PlannerPositionOne --- DataQualityAssignmentOne --- DataQualityAgent
    PlannerPositionTwo --- DataQualityAssignmentTwo --- DataQualityAgent

    classDef organizationUnit fill:#e8f1ff,stroke:#245b8a,stroke-width:2px
    classDef position fill:#fff3d6,stroke:#9a6700,stroke-width:1px
    classDef humanIdentity fill:#e8f5e9,stroke:#2e7d32,stroke-width:1px
    classDef agentIdentity fill:#e0f7fa,stroke:#00796b,stroke-width:2px
    classDef assignment fill:#f3e5f5,stroke:#7b1fa2,stroke-width:1px
    class ProductionPlanningUnit organizationUnit
    class PlanningManagerPosition,PlannerPositionOne,PlannerPositionTwo position
    class Morgan,Priya,Jordan humanIdentity
    class DataQualityAgent agentIdentity
    class MorganAssignment,PriyaAssignment,JordanAssignment,DataQualityAssignmentOne,DataQualityAssignmentTwo assignment
```

`PP-PLN-01` remains the same position while its human occupant changes from Priya Shah to Jordan Kim. The Planning Data Quality Assistant supports both planner positions through two distinct assignments, preserving separate context and authority for each position; it never becomes a position, displaces a human occupant, or inherits a position's authority. `PP-PLN-02` has no human occupant and is vacant for human occupancy even though it has an agent-support assignment. Because assignments are separate, effective-dated objects, identities never become part of the stable organization-unit or position structure.

The Sales Operations Manager position is accountable for resolving blocked customer orders. The Order Specialist position investigates individual exceptions. The Credit Manager position owns credit-policy decisions. The Production Planner position `PP-PLN-01` confirms material and capacity feasibility. Responsibilities remain attached to positions when the assigned people change.

## Roles and responsibilities

Roles group related responsibilities; they are not positions, identities, or external-system permission sets. Harbor uses the following objects in the order-exception scenario without adding nodes to the organization chart:

| Object | Kind | Expected outcome | Accountable position |
|---|---|---|---|
| `ROLE-ORDER-EXCEPTION-SUPPORT` | Role | Groups coordination responsibilities used across an exception | Not applicable; accountability belongs to each responsibility |
| `RESP-COORDINATE-ORDER-EXCEPTION` | Responsibility | A blocked order reaches an authorized disposition or documented escalation | Sales Operations Manager |
| `RESP-INVESTIGATE-ORDER-EXCEPTION` | Responsibility | Required facts and feasible alternatives are identified | Order Specialist |
| `RESP-APPROVE-CREDIT-EXCEPTION` | Responsibility | Credit risk is explicitly accepted or rejected within policy | Credit Manager |
| `RESP-CONFIRM-FULFILLMENT-FEASIBILITY` | Responsibility | Material and capacity feasibility is current and evidenced | Production Planner |

`ROLE-ORDER-EXCEPTION-SUPPORT` contains the coordination and investigation responsibilities. Role membership does not transfer the Credit Manager's approval authority, and a similarly named SAP business role or authorization does not automatically confer Charter authority.
