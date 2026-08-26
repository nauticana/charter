# Organization SDK

`Provider` resolves enterprise, unit, position-type, position, role, responsibility, relationship, and assignment documents (`BaseProvider` over any `corpus.Source`). `Queries` derives effective-dated views without inferring one relationship from another: assignments of a subject or to a target, human position occupants, root-to-unit paths, and `Coverage`, which distinguishes unassigned, human-performed, agent-supported, and agent-executed responsibilities. This package supplies contracts and reusable views only; it does not manage organizations.
