# Information SDK

`Provider` resolves information definitions and governance policies (`BaseProvider` over any `corpus.Source`). `BaseEvaluator` decides permitted use, retention and deletion rules, and access constraints from the policies a definition declares; conflicting policies fail closed unless every one of them opts into declared precedence. `CheckShape`, `Resolve`, and `CheckMappings` check an information shape and the paths of a binding against it, and `JSONSchema` is the schema an instance of a shape satisfies.
