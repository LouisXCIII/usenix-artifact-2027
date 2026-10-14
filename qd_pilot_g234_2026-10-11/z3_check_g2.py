#!/usr/bin/env python3
"""Exhaustive Z3 check for Group 2 (obligation/evidence-strength), reusing find_divergence()
from rp_checker.py unmodified -- not a new checker. evidence_type is abstracted to its
STRENGTH value (0-4): sound because every policy in this pilot (real rule, naive_cedar,
naive_opa) only ever examines evidence_type through its table-looked-up strength and the
single comparison strength>=3, never through the raw string identity, so any two evidence_type
strings sharing a strength are interchangeable for every function checked here."""
import sys, os
sys.path.insert(0, os.path.expanduser("~/pqc-hardhat/experiments/portability"))
from rp_checker import find_divergence
from z3 import Int, And, If

ROLE_NAME = {1: "clerk", 2: "manager", 3: "director", 4: "gerente", 5: "gerente_senior"}
CAP = {1: 10000, 2: 500000, 3: 2000000, 4: 500000, 5: 2000000}
HALF = {1: 5000, 2: 250000, 3: 1000000, 4: 250000, 5: 1000000}


def _cap(role_idx):
    return If(role_idx == 1, CAP[1], If(role_idx == 2, CAP[2], If(role_idx == 3, CAP[3],
           If(role_idx == 4, CAP[4], CAP[5]))))


def _half(role_idx):
    return If(role_idx == 1, HALF[1], If(role_idx == 2, HALF[2], If(role_idx == 3, HALF[3],
           If(role_idx == 4, HALF[4], HALF[5]))))


def real_gamma_d(role_idx, amount, strength):
    return And(amount > 0, amount <= _cap(role_idx),
               If(amount <= _half(role_idx), True, strength >= 3) == True)


def domain_constraints(role_idx, amount, strength):
    return [role_idx >= 1, role_idx <= 5, amount >= -10, amount <= 5_000_000,
            strength >= 0, strength <= 4]


def naive_cedar_gamma_d(role_idx, amount, strength):
    """Independent transcription of naive_cedar/policy.cedar's 10 permit rules (read
    2026-10-11): per role, an evidence-free grant up to half-cap OR an evidence-gated grant
    (strength>=3, i.e. dual_approval/combined) from half-cap up to the full cap."""
    cap, half = _cap(role_idx), _half(role_idx)
    return And(amount > 0,
               If(amount <= half, amount <= cap,
                  And(amount <= cap, strength >= 3)))


def naive_opa_gamma_d(role_idx, amount, strength):
    """Independent transcription of naive_opa/policy.rego's two allow rules (read 2026-10-11):
    cap_map/half_cap_map lookup + strength_map lookup (else=0), same case split as above."""
    cap, half = _cap(role_idx), _half(role_idx)
    return And(amount > 0, amount <= cap,
               If(amount <= half, True, strength >= 3))


def run(name, naive_fn):
    role_idx, amount, strength = Int("role_idx"), Int("amount"), Int("strength")
    domain = domain_constraints(role_idx, amount, strength)
    real_g = real_gamma_d(role_idx, amount, strength)
    naive_g = naive_fn(role_idx, amount, strength)
    result = find_divergence(domain, real_g, naive_g, [role_idx, amount, strength])
    print(f"{name}: widening_found={result['widening_found']} "
          f"over_conservative_found={result['over_conservative_found']}")
    if result["widening_found"]:
        print(f"  WIDENING WITNESS: {result['widening_witness']}")
    if result["over_conservative_found"]:
        print(f"  OVER-CONSERVATIVE WITNESS: {result['over_conservative_witness']}")
    return result


if __name__ == "__main__":
    out = {
        "naive_opa": run("naive_opa (authz.obligation.allow)", naive_opa_gamma_d),
        "naive_cedar": run("naive_cedar (permit when-clauses)", naive_cedar_gamma_d),
    }
    import json
    json.dump(out, open(os.path.join(os.path.dirname(os.path.abspath(__file__)),
                                       "z3_check_g2_results.json"), "w"), indent=2)
