#!/usr/bin/env python3
"""Exhaustive Z3 check for Group 4 (cross-vocabulary role mapping), reusing
find_divergence() from rp_checker.py unmodified. source_role is abstracted to role_idx in
{0,1,2,3,4,5}: 0 = the single equivalence class of EVERY unmapped string (sound, since
neither naive_cedar nor naive_opa nor the real rule distinguishes between different
unmapped strings -- all fail the same five-entry table lookup identically), 1-5 = the
five known table rows."""
import sys, os
sys.path.insert(0, os.path.expanduser("~/pqc-hardhat/experiments/portability"))
from rp_checker import find_divergence
from z3 import Int, And, If, Or

# role_idx: 1=gerente, 2=gerente_senior, 3=funcionario, 4=diretor, 5=supervisor, 0=unmapped
CAP = {1: 500000, 2: 2000000, 3: 10000, 4: 2000000, 5: 500000}


def _cap(role_idx):
    return If(role_idx == 1, CAP[1], If(role_idx == 2, CAP[2], If(role_idx == 3, CAP[3],
           If(role_idx == 4, CAP[4], CAP[5]))))


def real_gamma_d(role_idx, amount):
    return And(role_idx != 0, amount > 0, amount <= _cap(role_idx))


def domain_constraints(role_idx, amount):
    return [role_idx >= 0, role_idx <= 5, amount >= -10, amount <= 5_000_000]


def naive_cedar_gamma_d(role_idx, amount):
    """Independent transcription of naive_cedar/policy.cedar's single permit when-clause
    (read 2026-10-11): OR of 5 (source_role==X && amount>0 && amount<=cap) clauses -- any
    role_idx==0 (unmapped) satisfies none of the 5 disjuncts, so it is DENY by Cedar's
    implicit default-deny, matched here by role_idx!=0 being required structurally (the
    disjunction below is simply never true for role_idx==0)."""
    return Or(
        And(role_idx == 1, amount > 0, amount <= CAP[1]),
        And(role_idx == 2, amount > 0, amount <= CAP[2]),
        And(role_idx == 3, amount > 0, amount <= CAP[3]),
        And(role_idx == 4, amount > 0, amount <= CAP[4]),
        And(role_idx == 5, amount > 0, amount <= CAP[5]),
    )


def naive_opa_gamma_d(role_idx, amount):
    """Independent transcription of naive_opa/policy.rego's allow rule (read 2026-10-11):
    cap := role_caps[input.source_role] (undefined/fails for role_idx==0) AND amount>0 AND
    amount<=cap."""
    return And(role_idx != 0, amount > 0, amount <= _cap(role_idx))


def run(name, naive_fn):
    role_idx, amount = Int("role_idx"), Int("amount")
    domain = domain_constraints(role_idx, amount)
    real_g = real_gamma_d(role_idx, amount)
    naive_g = naive_fn(role_idx, amount)
    result = find_divergence(domain, real_g, naive_g, [role_idx, amount])
    print(f"{name}: widening_found={result['widening_found']} "
          f"over_conservative_found={result['over_conservative_found']}")
    if result["widening_found"]:
        print(f"  WIDENING WITNESS: {result['widening_witness']}")
    if result["over_conservative_found"]:
        print(f"  OVER-CONSERVATIVE WITNESS: {result['over_conservative_witness']}")
    return result


if __name__ == "__main__":
    out = {
        "naive_opa": run("naive_opa (authz.rolemap.allow)", naive_opa_gamma_d),
        "naive_cedar": run("naive_cedar (permit when-clause)", naive_cedar_gamma_d),
    }
    import json
    json.dump(out, open(os.path.join(os.path.dirname(os.path.abspath(__file__)),
                                       "z3_check_g4_results.json"), "w"), indent=2)
