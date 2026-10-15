// hardened_cedar/policy.cedar -- full-knowledge upper-bound baseline (author = me) for
// spec_v1.md's obligation/evidence-strength rule. Scope deliberately unconstrained
// (principal, action, resource all wildcard) so this policy matches ANY request entities
// regardless of what namespace the harness constructs -- all logic lives in `when` over
// context, avoiding the entity-namespace mismatch bug found in the Group 1 pilot.
permit(principal, action, resource)
when {
    context.amount > 0 &&
    ((context.role == "clerk" && context.amount <= 10000 &&
        (context.amount <= 5000 || context.evidence_type == "dual_approval" || context.evidence_type == "combined")) ||
     (context.role == "manager" && context.amount <= 500000 &&
        (context.amount <= 250000 || context.evidence_type == "dual_approval" || context.evidence_type == "combined")) ||
     (context.role == "director" && context.amount <= 2000000 &&
        (context.amount <= 1000000 || context.evidence_type == "dual_approval" || context.evidence_type == "combined")) ||
     (context.role == "gerente" && context.amount <= 500000 &&
        (context.amount <= 250000 || context.evidence_type == "dual_approval" || context.evidence_type == "combined")) ||
     (context.role == "gerente_senior" && context.amount <= 2000000 &&
        (context.amount <= 1000000 || context.evidence_type == "dual_approval" || context.evidence_type == "combined")))
};
