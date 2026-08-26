# Identity SDK

`Provider` resolves human and agent identities (`BaseProvider` over any `corpus.Source`); `Resolver` turns identity references into credential-independent `Actor`s and establishes lifecycle state at action time (`AbstractResolver` over any Provider, `BaseResolver` over a corpus). `Lifecycle.StateAt` and `ActiveAt` fail closed on missing, unordered, or inconsistent history.
