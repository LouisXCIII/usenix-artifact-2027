// hardened_cedar/policy.cedar -- full-knowledge upper-bound baseline (author = me) for
// spec_v1.md's cross-vocabulary role-mapping rule. Unconstrained scope; fail-closed for
// any source_role not in the five-row table, matching Cedar's implicit default-deny.
permit(principal, action, resource)
when {
    context.amount > 0 &&
    ((context.source_role == "gerente" && context.amount <= 500000) ||
     (context.source_role == "gerente_senior" && context.amount <= 2000000) ||
     (context.source_role == "funcionario" && context.amount <= 10000) ||
     (context.source_role == "diretor" && context.amount <= 2000000) ||
     (context.source_role == "supervisor" && context.amount <= 500000))
};
