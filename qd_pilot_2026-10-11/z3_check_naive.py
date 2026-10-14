#!/usr/bin/env python3
"""Formal (not just boundary-sampled) check of the two fresh-agent-authored naive policies
(naive_cedar/policy.cedar, naive_opa/policy.rego) against spec_v1.md's real rule, reusing
find_divergence() from rp_checker.py (the same general Z3 R_P checker behind Corollary 1,
2026-10-10) unmodified -- not a new checker. Both naive policies were read directly and
confirmed to transcribe the exact same cap/fraction constants and amount<=cap*fraction
structure as the real rule (no swap/reversal/off-by-one found by inspection or by the 70-case
empirical sweep already run via the real Cedar/OPA engines); this script encodes that same
structure in Z3 to get a definitive, exhaustive (all integers in domain, not just 70 sampled
boundary points) verdict, not an assumption.
"""
import sys, os

sys.path.insert(0, os.path.expanduser("~/pqc-hardhat/experiments/portability"))
from rp_checker import find_divergence
from z3 import Int, And, If

# Five roles from spec_v1.md (extends rp_checker_tier.py's 3-role REAL_ROLE_CAP to match the
# pilot spec, which also covers gerente/gerente_senior at the same dollar caps as
# manager/director respectively -- the real system's RoleMapping target values).
ROLE_CAP = {1: 10000, 2: 500000, 3: 2000000, 4: 500000, 5: 2000000}
ROLE_NAME = {1: "clerk", 2: "manager", 3: "director", 4: "gerente", 5: "gerente_senior"}
FRACTION_BPS = {0: 0, 1: 1000, 2: 5000, 3: 10000}


def _cap(role_idx):
    return If(role_idx == 1, ROLE_CAP[1],
           If(role_idx == 2, ROLE_CAP[2],
           If(role_idx == 3, ROLE_CAP[3],
           If(role_idx == 4, ROLE_CAP[4], ROLE_CAP[5]))))


def _fraction(tier, table):
    return If(tier == 0, table[0], If(tier == 1, table[1], If(tier == 2, table[2], table[3])))


def real_gamma_d(tier, role_idx, amount):
    return And(tier >= 1, amount * 10000 <= _cap(role_idx) * _fraction(tier, FRACTION_BPS))


def domain_constraints(tier, role_idx, amount):
    return [tier >= 0, tier <= 3, role_idx >= 1, role_idx <= 5, amount >= 1, amount <= 20_000_000]


def naive_opa_gamma_d(tier, role_idx, amount):
    """Independent structural transcription of naive_opa/policy.rego (read 2026-10-11), NOT a
    reuse of real_gamma_d's Z3 object: caps dict (same literals), tier_fraction_num/den =
    {1:(1,10), 2:(1,2), 3:(1,1)}, allow iff tier in [1,3] and amount*den(tier) <= cap*num(tier)
    (the file's own exact-integer-arithmetic form to avoid float rounding). Tier 0/out-of-range
    falls through Rego's `default allow = false`, encoded here as the tier<=3 conjunct."""
    cap = _cap(role_idx)
    num = If(tier == 1, 1, If(tier == 2, 1, 1))  # tier_fraction_num = {1:1,2:1,3:1}
    den = If(tier == 1, 10, If(tier == 2, 2, 1))  # tier_fraction_den = {1:10,2:2,3:1}
    return And(tier >= 1, tier <= 3, amount * den <= cap * num)


def naive_cedar_gamma_d(tier, role_idx, amount):
    """Independent structural transcription of naive_cedar/policy.cedar (read 2026-10-11), NOT
    a reuse of real_gamma_d's Z3 object: the file's own nested per-role if/else chain with
    precomputed (cap*fraction) literals -- clerk {1:1000,2:5000,3:10000}, manager/gerente
    {1:50000,2:250000,3:500000}, director/gerente_senior {1:200000,2:1000000,3:2000000} --
    unmatched role or tier not in {1,2,3} falls to the file's own `else 0` threshold."""
    def threshold_for_role(cap_t1, cap_t2, cap_t3):
        return If(tier == 1, cap_t1, If(tier == 2, cap_t2, If(tier == 3, cap_t3, 0)))

    threshold = If(role_idx == 1, threshold_for_role(1000, 5000, 10000),          # clerk
                If(role_idx == 2, threshold_for_role(50000, 250000, 500000),      # manager
                If(role_idx == 3, threshold_for_role(200000, 1000000, 2000000),   # director
                If(role_idx == 4, threshold_for_role(50000, 250000, 500000),      # gerente
                   threshold_for_role(200000, 1000000, 2000000)))))               # gerente_senior
    return And(tier >= 1, amount <= threshold)


def run(name, naive_fn):
    tier, role_idx, amount = Int("tier"), Int("role_idx"), Int("amount")
    domain = domain_constraints(tier, role_idx, amount)
    real_g = real_gamma_d(tier, role_idx, amount)
    naive_g = naive_fn(tier, role_idx, amount)
    result = find_divergence(domain, real_g, naive_g, [tier, role_idx, amount])
    print(f"{name}: widening_found={result['widening_found']} "
          f"over_conservative_found={result['over_conservative_found']}")
    if result["widening_found"]:
        print(f"  WIDENING WITNESS: {result['widening_witness']}")
    if result["over_conservative_found"]:
        print(f"  OVER-CONSERVATIVE WITNESS: {result['over_conservative_witness']}")
    return result


if __name__ == "__main__":
    out = {
        "naive_opa": run("naive_opa (destination_authz.allow)", naive_opa_gamma_d),
        "naive_cedar": run("naive_cedar (permit when-clause)", naive_cedar_gamma_d),
    }
    import json
    json.dump(out, open(os.path.join(os.path.dirname(os.path.abspath(__file__)),
                                       "z3_check_results.json"), "w"), indent=2)
