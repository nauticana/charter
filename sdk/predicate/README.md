# Predicate Package

`Evaluator` decides a contract predicate over the invocation input and observed facts; an error means undecided, which the invoker treats as unknown. `CEL` evaluates Common Expression Language predicates with the variables `input` and `facts`, comparing numbers across integer and floating-point types, within a cost limit (`DefaultCELCostLimit` unless `CostLimit` is set).
